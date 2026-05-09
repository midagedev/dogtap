package report

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/intake"
	"github.com/midagedev/dogtap/internal/validation"
)

var (
	ErrValidationFailed = errors.New("dogtap validation failed")
	ErrTool             = errors.New("dogtap replay tool error")
)

type Report struct {
	CreatedAt time.Time             `json:"createdAt"`
	Summary   Summary               `json:"summary"`
	Events    []event.EventEnvelope `json:"events"`
}

type Summary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Fatal    int `json:"fatal"`
	Warnings int `json:"warnings"`
}

type Format string

const (
	FormatAuto     Format = "auto"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

func ParseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(FormatAuto):
		return FormatAuto, nil
	case string(FormatJSON):
		return FormatJSON, nil
	case string(FormatMarkdown), "md":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unsupported report format %q", value)
	}
}

func ResolveFormat(format Format, outputPath string) Format {
	if format != FormatAuto {
		return format
	}
	switch strings.ToLower(filepath.Ext(outputPath)) {
	case ".md", ".markdown":
		return FormatMarkdown
	default:
		return FormatJSON
	}
}

func FromEvents(events []event.EventEnvelope) Report {
	summary := Summary{Total: len(events)}
	for _, e := range events {
		if e.Validation.Status == "fail" {
			summary.Failed++
		} else {
			summary.Passed++
		}
		for _, rule := range e.Validation.Rules {
			if rule.Status != "fail" {
				continue
			}
			switch rule.Severity {
			case "fatal":
				summary.Fatal++
			case "warning":
				summary.Warnings++
			}
		}
	}
	return Report{CreatedAt: time.Now().UTC(), Summary: summary, Events: events}
}

func (r Report) JSON() []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return []byte(`{"error":"marshal report"}`)
	}
	return b
}

func (r Report) Markdown() []byte {
	var b strings.Builder
	fmt.Fprintln(&b, "# Dogtap Validation Report")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Created: %s\n", r.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- Status: %s\n", reportStatus(r))
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Summary")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Total | Passed | Failed | Fatal | Warnings |")
	fmt.Fprintln(&b, "| ---: | ---: | ---: | ---: | ---: |")
	fmt.Fprintf(&b, "| %d | %d | %d | %d | %d |\n", r.Summary.Total, r.Summary.Passed, r.Summary.Failed, r.Summary.Fatal, r.Summary.Warnings)
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Findings")
	fmt.Fprintln(&b)
	if !writeFindings(&b, r.Events) {
		fmt.Fprintln(&b, "No failing validation rules.")
		fmt.Fprintln(&b)
	}

	fmt.Fprintln(&b, "## Events")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| ID | Source | Status | Service | Env | Route | Trace |")
	fmt.Fprintln(&b, "| --- | --- | --- | --- | --- | --- | --- |")
	for _, e := range r.Events {
		fmt.Fprintf(
			&b,
			"| %s | %s | %s | %s | %s | %s | %s |\n",
			markdownCell(e.ID),
			markdownCell(string(e.Source)),
			markdownCell(e.Validation.Status),
			markdownCell(e.Normalized.Service),
			markdownCell(e.Normalized.Env),
			markdownCell(e.Normalized.Route),
			markdownCell(e.Normalized.TraceID),
		)
	}
	return []byte(b.String())
}

func (r Report) Render(format Format) []byte {
	switch format {
	case FormatMarkdown:
		return r.Markdown()
	default:
		return r.JSON()
	}
}

func (r Report) HasFailures() bool {
	return r.Summary.Failed > 0 || r.Summary.Fatal > 0
}

func reportStatus(r Report) string {
	if r.HasFailures() {
		return "failed"
	}
	return "passed"
}

func writeFindings(b *strings.Builder, events []event.EventEnvelope) bool {
	wrote := false
	for _, e := range events {
		rules := failingRules(e.Validation.Rules)
		if len(rules) == 0 {
			continue
		}
		wrote = true
		fmt.Fprintf(b, "### %s %s\n\n", strings.ToUpper(string(e.Source)), inlineCode(e.ID))
		fmt.Fprintf(b, "- Endpoint: %s %s\n", inlineCode(e.Method), inlineCode(e.Endpoint))
		fmt.Fprintf(b, "- Service: %s\n", inlineCode(e.Normalized.Service))
		fmt.Fprintln(b)
		fmt.Fprintln(b, "| Rule | Severity | Field | Message | Evidence |")
		fmt.Fprintln(b, "| --- | --- | --- | --- | --- |")
		for _, rule := range rules {
			fmt.Fprintf(
				b,
				"| %s | %s | %s | %s | %s |\n",
				markdownCell(rule.RuleID),
				markdownCell(rule.Severity),
				markdownCell(rule.FieldPath),
				markdownCell(rule.Message),
				markdownCell(rule.Evidence),
			)
		}
		fmt.Fprintln(b)
	}
	return wrote
}

func failingRules(rules []event.ValidationRuleResult) []event.ValidationRuleResult {
	out := make([]event.ValidationRuleResult, 0, len(rules))
	for _, rule := range rules {
		if rule.Status == "fail" {
			out = append(out, rule)
		}
	}
	return out
}

func inlineCode(value string) string {
	if strings.TrimSpace(value) == "" {
		return "`-`"
	}
	return "`" + strings.ReplaceAll(value, "`", "'") + "`"
}

func markdownCell(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}

func Replay(ctx context.Context, cfg config.Config, paths []string) (Report, error) {
	validator := validation.New(cfg.Validation)
	events := make([]event.EventEnvelope, 0, len(paths))
	validationEvents := make([]event.EventEnvelope, 0, len(paths))
	for _, path := range paths {
		source := sourceFromPath(path)
		body, err := os.ReadFile(path)
		if err != nil {
			return Report{}, fmt.Errorf("%w: read fixture %s: %v", ErrTool, path, err)
		}
		req, err := fixtureRequest(ctx, path, source, body)
		if err != nil {
			return Report{}, err
		}
		result, err := intake.CaptureRequest(req, intake.CaptureOptions{
			Source:           source,
			AllowRawPayloads: cfg.RawPayloadsAllowed(),
			MaxBodyBytes:     cfg.Security.MaxBodyBytes,
			ForwardMode:      string(cfg.Mode),
		})
		if err != nil {
			return Report{}, fmt.Errorf("%w: capture fixture %s: %v", ErrTool, path, err)
		}
		result.Event.Validation = validator.Validate(result.ValidationEvent)
		events = append(events, result.Event)
		validationEvent := result.ValidationEvent
		validationEvent.Validation = result.Event.Validation
		validationEvents = append(validationEvents, validationEvent)
	}
	validationEvents = validator.ValidateBatch(validationEvents)
	for i := range events {
		events[i].Validation = validationEvents[i].Validation
	}
	return FromEvents(events), nil
}

func fixtureRequest(ctx context.Context, path string, source event.Source, body []byte) (*http.Request, error) {
	endpoint := endpointFor(source)
	reader := bytes.NewReader(body)
	req, err := http.NewRequestWithContext(ctx, methodFor(source), "http://dogtap.local"+endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("%w: build fixture request: %v", ErrTool, err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".gz":
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")
		if _, err := gzip.NewReader(bytes.NewReader(body)); err != nil {
			return nil, fmt.Errorf("%w: invalid gzip fixture %s: %v", ErrTool, path, err)
		}
	case ".txt":
		req.Header.Set("Content-Type", "text/plain")
	default:
		req.Header.Set("Content-Type", "application/json")
	}
	if err := applyFixtureMetadata(req, path); err != nil {
		return nil, err
	}
	return req, nil
}

func applyFixtureMetadata(req *http.Request, path string) error {
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".meta.json"
	b, err := os.ReadFile(metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%w: read fixture metadata %s: %v", ErrTool, metaPath, err)
	}
	var meta struct {
		ReplayRequest struct {
			Method  string            `json:"method"`
			Path    string            `json:"path"`
			Query   string            `json:"query"`
			Headers map[string]string `json:"headers"`
		} `json:"replayRequest"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return fmt.Errorf("%w: parse fixture metadata %s: %v", ErrTool, metaPath, err)
	}
	if meta.ReplayRequest.Method != "" {
		req.Method = meta.ReplayRequest.Method
	}
	if meta.ReplayRequest.Path != "" {
		req.URL.Path = meta.ReplayRequest.Path
	}
	if meta.ReplayRequest.Query != "" {
		req.URL.RawQuery = meta.ReplayRequest.Query
	}
	for key, value := range meta.ReplayRequest.Headers {
		req.Header.Set(key, value)
	}
	return nil
}

func sourceFromPath(path string) event.Source {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "faro"), strings.Contains(lower, "collect"):
		return event.SourceFaro
	case strings.Contains(lower, "otlp"):
		return event.SourceOTLP
	case strings.Contains(lower, "rum"):
		return event.SourceRUM
	case strings.Contains(lower, "log"):
		return event.SourceLogs
	case strings.Contains(lower, "apm"), strings.Contains(lower, "trace"):
		return event.SourceAPM
	default:
		return event.SourceUnknown
	}
}

func endpointFor(source event.Source) string {
	switch source {
	case event.SourceRUM:
		return "/rum"
	case event.SourceLogs:
		return "/api/v2/logs"
	case event.SourceAPM:
		return "/v0.5/traces"
	case event.SourceOTLP:
		return "/v1/traces"
	case event.SourceFaro:
		return "/collect"
	default:
		return "/unknown"
	}
}

func methodFor(source event.Source) string {
	if source == event.SourceAPM {
		return http.MethodPut
	}
	return http.MethodPost
}
