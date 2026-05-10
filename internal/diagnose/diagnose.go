package diagnose

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/midagedev/dogtap/internal/bundle"
	"github.com/midagedev/dogtap/internal/contract"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/report"
)

type Options struct {
	BaseURL                string
	OutputDir              string
	Limit                  int
	Expectations           Expectations
	WorkflowContracts      []contract.Definition
	FailOnWorkflowContract bool
	Filter                 bundle.Request
	Client                 *http.Client
}

type Request struct {
	Limit                       int                   `json:"limit,omitempty"`
	Expect                      Expectations          `json:"expect,omitempty"`
	Expectations                Expectations          `json:"expectations,omitempty"`
	WorkflowContract            contract.Definition   `json:"workflowContract,omitempty"`
	WorkflowContracts           []contract.Definition `json:"workflowContracts,omitempty"`
	UseDefaultWorkflowContracts bool                  `json:"useDefaultWorkflowContracts,omitempty"`
	Filter                      bundle.Request        `json:"filter,omitempty"`
}

type Expectations struct {
	NonEmpty     bool     `json:"nonEmpty,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	PayloadKinds []string `json:"payloadKinds,omitempty"`
	Services     []string `json:"services,omitempty"`
	Sessions     []string `json:"sessions,omitempty"`
	Traces       []string `json:"traces,omitempty"`
	Cases        []string `json:"cases,omitempty"`
	Routes       []string `json:"routes,omitempty"`
	Metrics      []string `json:"metrics,omitempty"`
	Endpoints    []string `json:"endpoints,omitempty"`
}

type Snapshot struct {
	CreatedAt         time.Time             `json:"createdAt"`
	BaseURL           string                `json:"baseUrl,omitempty"`
	Limit             int                   `json:"limit"`
	Filter            bundle.Request        `json:"filter"`
	Health            map[string]string     `json:"healthz"`
	Readiness         map[string]string     `json:"readyz"`
	Events            []event.EventEnvelope `json:"events"`
	Report            report.Report         `json:"report"`
	DebugBundle       bundle.DebugBundle    `json:"debugBundle"`
	Metrics           string                `json:"metrics"`
	Assertions        AssertionReport       `json:"assertions"`
	WorkflowContracts []contract.Result     `json:"workflowContracts,omitempty"`
}

type SnapshotInput struct {
	CreatedAt         time.Time
	BaseURL           string
	Request           Request
	Health            map[string]string
	Readiness         map[string]string
	Events            []event.EventEnvelope
	Report            report.Report
	DebugBundle       bundle.DebugBundle
	Metrics           string
	Probes            map[string]bool
	WorkflowContracts []contract.Definition
}

type Artifact struct {
	Name        string
	Filename    string
	ContentType string
	StatusCode  int
	Body        []byte
}

type Result struct {
	CreatedAt         time.Time         `json:"createdAt"`
	BaseURL           string            `json:"baseUrl"`
	OutputDir         string            `json:"outputDir"`
	Files             map[string]File   `json:"files"`
	Assertions        AssertionReport   `json:"assertions"`
	WorkflowContracts []contract.Result `json:"workflowContracts,omitempty"`
	RequestError      []RequestError    `json:"requestErrors,omitempty"`
}

type File struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	StatusCode int    `json:"statusCode,omitempty"`
}

type RequestError struct {
	Name       string `json:"name"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	StatusCode int    `json:"statusCode,omitempty"`
	Error      string `json:"error"`
}

type AssertionReport struct {
	Status       string        `json:"status"`
	Summary      CheckSummary  `json:"summary"`
	Observed     Observed      `json:"observed"`
	Expectations Expectations  `json:"expectations"`
	Checks       []CheckResult `json:"checks"`
}

type CheckSummary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type Observed struct {
	Total        int            `json:"total"`
	Sources      map[string]int `json:"sources"`
	PayloadKinds map[string]int `json:"payloadKinds"`
	Services     map[string]int `json:"services"`
	Sessions     map[string]int `json:"sessions"`
	Traces       map[string]int `json:"traces"`
	Cases        map[string]int `json:"cases"`
	Routes       map[string]int `json:"routes"`
	Metrics      map[string]int `json:"metrics"`
	Endpoints    map[string]int `json:"endpoints"`
	Validation   map[string]int `json:"validation"`
	Recent       []RecentEvent  `json:"recent"`
}

type RecentEvent struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	PayloadKind string `json:"payloadKind,omitempty"`
	Endpoint    string `json:"endpoint"`
	Service     string `json:"service,omitempty"`
	SessionID   string `json:"sessionId,omitempty"`
	TraceID     string `json:"traceId,omitempty"`
	Route       string `json:"route,omitempty"`
	Status      string `json:"status,omitempty"`
	ReceivedAt  string `json:"receivedAt"`
}

type CheckResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Matched int    `json:"matched,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

func Collect(ctx context.Context, opt Options) (Result, error) {
	opt = normalizeOptions(opt)
	if err := os.MkdirAll(opt.OutputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("%w: create diagnostics dir: %v", report.ErrTool, err)
	}

	result := Result{
		CreatedAt: time.Now().UTC(),
		BaseURL:   opt.BaseURL,
		OutputDir: opt.OutputDir,
		Files:     map[string]File{},
	}

	_, healthOK := collectGET(ctx, opt, &result, "healthz", "/healthz", "healthz.json")
	_, readyOK := collectGET(ctx, opt, &result, "readyz", "/readyz", "readyz.json")
	eventsBody, eventsOK := collectGET(ctx, opt, &result, "events", "/api/events?limit="+url.QueryEscape(fmt.Sprint(opt.Limit)), "events.json")
	_, reportOK := collectGET(ctx, opt, &result, "report", "/api/reports/latest", "report.json")
	_, metricsOK := collectGET(ctx, opt, &result, "metrics", "/metrics", "metrics.txt")
	debugOK := collectDebugBundle(ctx, opt, &result)

	var events []event.EventEnvelope
	if eventsOK {
		if err := json.Unmarshal(eventsBody, &events); err != nil {
			result.RequestError = append(result.RequestError, RequestError{
				Name:  "events",
				URL:   opt.BaseURL + "/api/events",
				Error: "decode events.json: " + err.Error(),
			})
			eventsOK = false
		}
	}

	result.Assertions = BuildAssertions(events, opt.Expectations, map[string]bool{
		"healthz":      healthOK,
		"readyz":       readyOK,
		"events":       eventsOK,
		"report":       reportOK,
		"metrics":      metricsOK,
		"debug-bundle": debugOK,
	})
	result.WorkflowContracts = contract.EvaluateAll(opt.WorkflowContracts, events)

	writeJSONFile(&result, "assertions", "assertions.json", result.Assertions, http.StatusOK)
	if len(result.WorkflowContracts) > 0 {
		writeJSONFile(&result, "workflow-contracts", "workflow-contracts.json", result.WorkflowContracts, http.StatusOK)
	}
	writeTextFile(&result, "summary", "summary.md", RenderSummary(result), http.StatusOK)
	writeManifestFile(&result)

	if result.Assertions.Status == "fail" {
		return result, report.ErrValidationFailed
	}
	if opt.FailOnWorkflowContract && workflowContractsFailed(result.WorkflowContracts) {
		return result, report.ErrValidationFailed
	}
	return result, nil
}

func normalizeOptions(opt Options) Options {
	if strings.TrimSpace(opt.BaseURL) == "" {
		opt.BaseURL = "http://127.0.0.1:8080"
	}
	opt.BaseURL = strings.TrimRight(opt.BaseURL, "/")
	if opt.Limit <= 0 {
		opt.Limit = 200
	}
	if strings.TrimSpace(opt.OutputDir) == "" {
		opt.OutputDir = filepath.Join(".dogtap", "diagnostics", time.Now().UTC().Format("20060102T150405Z"))
	}
	if opt.Client == nil {
		opt.Client = &http.Client{Timeout: 5 * time.Second}
	}
	opt.Expectations = normalizeExpectations(opt.Expectations)
	opt.WorkflowContracts = normalizeWorkflowDefinitions(opt.WorkflowContracts)
	return opt
}

func NormalizeRequest(req Request, defaultLimit int) Request {
	if expectationIsEmpty(req.Expect) {
		req.Expect = req.Expectations
	}
	req.Expect = normalizeExpectations(req.Expect)
	if req.Limit <= 0 && req.Filter.Limit > 0 {
		req.Limit = req.Filter.Limit
	}
	if req.Limit <= 0 {
		req.Limit = defaultLimit
	}
	if req.Limit <= 0 {
		req.Limit = 200
	}
	if req.Filter.Limit <= 0 {
		req.Filter.Limit = req.Limit
	}
	req.WorkflowContracts = normalizeWorkflowDefinitions(req.WorkflowContracts)
	if !workflowDefinitionIsEmpty(req.WorkflowContract) {
		req.WorkflowContracts = append([]contract.Definition{contract.Normalize(req.WorkflowContract)}, req.WorkflowContracts...)
	}
	req.WorkflowContract = contract.Definition{}
	if req.UseDefaultWorkflowContracts && len(req.WorkflowContracts) == 0 {
		req.WorkflowContracts = contract.DefaultDashboardContracts()
	}
	req.Expectations = Expectations{}
	return req
}

func NewSnapshot(input SnapshotInput) Snapshot {
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	req := NormalizeRequest(input.Request, len(input.Events))
	probes := input.Probes
	if probes == nil {
		probes = map[string]bool{}
	}
	return Snapshot{
		CreatedAt:         createdAt,
		BaseURL:           strings.TrimRight(input.BaseURL, "/"),
		Limit:             req.Limit,
		Filter:            req.Filter,
		Health:            input.Health,
		Readiness:         input.Readiness,
		Events:            input.Events,
		Report:            input.Report,
		DebugBundle:       input.DebugBundle,
		Metrics:           input.Metrics,
		Assertions:        BuildAssertions(input.Events, req.Expect, probes),
		WorkflowContracts: contract.EvaluateAll(append(input.WorkflowContracts, req.WorkflowContracts...), input.Events),
	}
}

func SnapshotArtifacts(snapshot Snapshot, outputDir string) []Artifact {
	artifacts := []Artifact{
		jsonArtifact("healthz", "healthz.json", snapshot.Health),
		jsonArtifact("readyz", "readyz.json", snapshot.Readiness),
		jsonArtifact("events", "events.json", snapshot.Events),
		jsonArtifact("report", "report.json", snapshot.Report),
		jsonArtifact("debug-bundle", "debug-bundle.json", snapshot.DebugBundle),
		textArtifact("metrics", "metrics.txt", snapshot.Metrics),
		jsonArtifact("assertions", "assertions.json", snapshot.Assertions),
	}
	if len(snapshot.WorkflowContracts) > 0 {
		artifacts = append(artifacts, jsonArtifact("workflow-contracts", "workflow-contracts.json", snapshot.WorkflowContracts))
	}

	files := make(map[string]File, len(artifacts)+2)
	for _, artifact := range artifacts {
		files[artifact.Name] = File{
			Path:       artifactPath(outputDir, artifact.Filename),
			Content:    artifact.ContentType,
			StatusCode: artifact.StatusCode,
		}
	}
	files["summary"] = File{Path: artifactPath(outputDir, "summary.md"), Content: "text/markdown", StatusCode: http.StatusOK}
	files["manifest"] = File{Path: artifactPath(outputDir, "manifest.json"), Content: "application/json", StatusCode: http.StatusOK}

	result := Result{
		CreatedAt:         snapshot.CreatedAt,
		BaseURL:           snapshot.BaseURL,
		OutputDir:         outputDir,
		Files:             files,
		Assertions:        snapshot.Assertions,
		WorkflowContracts: snapshot.WorkflowContracts,
	}

	return append(artifacts,
		textArtifact("summary", "summary.md", RenderSummary(result)),
		jsonArtifact("manifest", "manifest.json", result),
	)
}

func expectationIsEmpty(exp Expectations) bool {
	return !exp.NonEmpty &&
		len(exp.Sources) == 0 &&
		len(exp.PayloadKinds) == 0 &&
		len(exp.Services) == 0 &&
		len(exp.Sessions) == 0 &&
		len(exp.Traces) == 0 &&
		len(exp.Cases) == 0 &&
		len(exp.Routes) == 0 &&
		len(exp.Metrics) == 0 &&
		len(exp.Endpoints) == 0
}

func artifactPath(outputDir, filename string) string {
	if strings.TrimSpace(outputDir) == "" {
		return filename
	}
	return filepath.Join(outputDir, filename)
}

func jsonArtifact(name, filename string, value any) Artifact {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		body = []byte(`{"error":"marshal diagnostics artifact"}`)
	}
	return Artifact{
		Name:        name,
		Filename:    filename,
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
		Body:        append(body, '\n'),
	}
}

func textArtifact(name, filename, value string) Artifact {
	return Artifact{
		Name:        name,
		Filename:    filename,
		ContentType: contentLabel("text/plain", filename),
		StatusCode:  http.StatusOK,
		Body:        []byte(value),
	}
}

func normalizeExpectations(exp Expectations) Expectations {
	exp.Sources = normalizeList(exp.Sources)
	exp.PayloadKinds = normalizeList(exp.PayloadKinds)
	exp.Services = normalizeList(exp.Services)
	exp.Sessions = normalizeList(exp.Sessions)
	exp.Traces = normalizeList(exp.Traces)
	exp.Cases = normalizeList(exp.Cases)
	exp.Routes = normalizeList(exp.Routes)
	exp.Metrics = normalizeList(exp.Metrics)
	exp.Endpoints = normalizeList(exp.Endpoints)
	return exp
}

func normalizeWorkflowDefinitions(defs []contract.Definition) []contract.Definition {
	out := make([]contract.Definition, 0, len(defs))
	for _, def := range defs {
		if workflowDefinitionIsEmpty(def) {
			continue
		}
		out = append(out, contract.Normalize(def))
	}
	return out
}

func workflowDefinitionIsEmpty(def contract.Definition) bool {
	return strings.TrimSpace(def.Name) == "" && len(def.Checks) == 0
}

func workflowContractsFailed(results []contract.Result) bool {
	for _, result := range results {
		if result.Status == "fail" {
			return true
		}
	}
	return false
}

func normalizeList(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func collectGET(ctx context.Context, opt Options, result *Result, name, path, filename string) ([]byte, bool) {
	return collectHTTP(ctx, opt, result, name, http.MethodGet, path, "application/json", nil, filename)
}

func collectDebugBundle(ctx context.Context, opt Options, result *Result) bool {
	filter := opt.Filter
	if filter.Limit <= 0 {
		filter.Limit = opt.Limit
	}
	body, err := json.Marshal(filter)
	if err != nil {
		result.RequestError = append(result.RequestError, RequestError{Name: "debug-bundle", Error: err.Error()})
		return false
	}
	_, ok := collectHTTP(ctx, opt, result, "debug-bundle", http.MethodPost, "/api/debug-bundles", "application/json", bytes.NewReader(body), "debug-bundle.json")
	return ok
}

func collectHTTP(ctx context.Context, opt Options, result *Result, name, method, path, contentType string, body io.Reader, filename string) ([]byte, bool) {
	fullURL := opt.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		result.RequestError = append(result.RequestError, RequestError{Name: name, Method: method, URL: fullURL, Error: err.Error()})
		return nil, false
	}
	if contentType != "" && body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := opt.Client.Do(req)
	if err != nil {
		result.RequestError = append(result.RequestError, RequestError{Name: name, Method: method, URL: fullURL, Error: err.Error()})
		return nil, false
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.RequestError = append(result.RequestError, RequestError{Name: name, Method: method, URL: fullURL, StatusCode: resp.StatusCode, Error: err.Error()})
		return nil, false
	}

	writeBytesFile(result, name, filename, responseBody, resp.StatusCode, resp.Header.Get("Content-Type"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.RequestError = append(result.RequestError, RequestError{
			Name:       name,
			Method:     method,
			URL:        fullURL,
			StatusCode: resp.StatusCode,
			Error:      strings.TrimSpace(string(responseBody)),
		})
		return responseBody, false
	}
	return responseBody, true
}

func writeJSONFile(result *Result, name, filename string, value any, statusCode int) {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		body = []byte(`{"error":"marshal diagnostics file"}`)
	}
	writeBytesFile(result, name, filename, append(body, '\n'), statusCode, "application/json")
}

func writeManifestFile(result *Result) {
	result.Files["manifest"] = File{
		Path:       filepath.Join(result.OutputDir, "manifest.json"),
		Content:    "application/json",
		StatusCode: http.StatusOK,
	}
	writeJSONFile(result, "manifest", "manifest.json", result, http.StatusOK)
}

func writeTextFile(result *Result, name, filename, value string, statusCode int) {
	writeBytesFile(result, name, filename, []byte(value), statusCode, "text/plain")
}

func writeBytesFile(result *Result, name, filename string, body []byte, statusCode int, contentType string) {
	path := filepath.Join(result.OutputDir, filename)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		result.RequestError = append(result.RequestError, RequestError{Name: name, Error: "write " + filename + ": " + err.Error()})
		return
	}
	result.Files[name] = File{Path: path, Content: contentLabel(contentType, filename), StatusCode: statusCode}
}

func contentLabel(contentType, filename string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType != "" {
		return mediaType
	}
	switch filepath.Ext(filename) {
	case ".json":
		return "application/json"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func BuildAssertions(events []event.EventEnvelope, expectations Expectations, probes map[string]bool) AssertionReport {
	observed := observe(events)
	checks := make([]CheckResult, 0, 8+len(expectations.Sources)+len(expectations.PayloadKinds))

	checks = append(checks, probeCheck("dogtap:healthz", probes["healthz"], "Dogtap health endpoint responded.", "Dogtap health endpoint did not respond. Check that Dogtap is running and that -base-url points at the HTTP port."))
	checks = append(checks, probeCheck("dogtap:readyz", probes["readyz"], "Dogtap readiness endpoint responded.", "Dogtap readiness endpoint did not respond. Check server startup logs and bound ports."))
	checks = append(checks, probeCheck("dogtap:events-api", probes["events"], "Dogtap events API returned retained events.", "Dogtap events API could not be read. Check /api/events and storage health."))
	checks = append(checks, probeCheck("dogtap:report-api", probes["report"], "Dogtap latest report API returned validation data.", "Dogtap report API could not be read. Check /api/reports/latest."))
	checks = append(checks, probeCheck("dogtap:metrics", probes["metrics"], "Dogtap metrics endpoint responded.", "Dogtap metrics endpoint did not respond. Check /metrics on the Dogtap HTTP port."))
	checks = append(checks, probeCheck("dogtap:debug-bundle", probes["debug-bundle"], "Dogtap debug bundle API returned filtered evidence.", "Dogtap debug bundle API could not be read. Check /api/debug-bundles."))

	if expectations.NonEmpty {
		checks = append(checks, countCheck("events:non-empty", observed.Total, "Dogtap retained events.", "Dogtap has no retained events. Trigger the app workflow again, then verify SDK endpoint config and network reachability from the app container/browser to Dogtap."))
	}

	addExpectedChecks(&checks, "source", expectations.Sources, observed.Sources)
	addExpectedChecks(&checks, "payload-kind", expectations.PayloadKinds, observed.PayloadKinds)
	addExpectedChecks(&checks, "service", expectations.Services, observed.Services)
	addExpectedChecks(&checks, "session", expectations.Sessions, observed.Sessions)
	addExpectedTraceChecks(&checks, expectations.Traces, observed.Traces)
	addExpectedChecks(&checks, "case", expectations.Cases, observed.Cases)
	addExpectedChecks(&checks, "route", expectations.Routes, observed.Routes)
	addExpectedChecks(&checks, "metric", expectations.Metrics, observed.Metrics)
	addExpectedChecks(&checks, "endpoint", expectations.Endpoints, observed.Endpoints)

	summary := CheckSummary{Total: len(checks)}
	for _, check := range checks {
		if check.Status == "pass" {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}
	status := "pass"
	if summary.Failed > 0 {
		status = "fail"
	}
	return AssertionReport{
		Status:       status,
		Summary:      summary,
		Observed:     observed,
		Expectations: expectations,
		Checks:       checks,
	}
}

func observe(events []event.EventEnvelope) Observed {
	observed := Observed{
		Total:        len(events),
		Sources:      map[string]int{},
		PayloadKinds: map[string]int{},
		Services:     map[string]int{},
		Sessions:     map[string]int{},
		Traces:       map[string]int{},
		Cases:        map[string]int{},
		Routes:       map[string]int{},
		Metrics:      map[string]int{},
		Endpoints:    map[string]int{},
		Validation:   map[string]int{},
		Recent:       make([]RecentEvent, 0, min(10, len(events))),
	}
	for index, e := range events {
		addObserved(observed.Sources, string(e.Source))
		addObserved(observed.PayloadKinds, e.PayloadKind)
		addObserved(observed.Services, e.Normalized.Service)
		addObserved(observed.Sessions, e.Normalized.SessionID)
		addObserved(observed.Traces, e.Normalized.TraceID)
		if canonical := canonicalTraceID(e.Normalized.TraceID); canonical != "" && canonical != e.Normalized.TraceID {
			addObserved(observed.Traces, canonical)
		}
		addObserved(observed.Cases, e.Normalized.CaseID)
		addObserved(observed.Routes, e.Normalized.Route)
		addObserved(observed.Endpoints, e.Endpoint)
		addObserved(observed.Validation, e.Validation.Status)
		if e.Details != nil {
			for _, metric := range e.Details.Metrics {
				addObserved(observed.Metrics, metric.Name)
			}
		}
		if index < 10 {
			observed.Recent = append(observed.Recent, RecentEvent{
				ID:          e.ID,
				Source:      string(e.Source),
				PayloadKind: e.PayloadKind,
				Endpoint:    e.Endpoint,
				Service:     e.Normalized.Service,
				SessionID:   e.Normalized.SessionID,
				TraceID:     e.Normalized.TraceID,
				Route:       e.Normalized.Route,
				Status:      e.Validation.Status,
				ReceivedAt:  e.ReceivedAt.UTC().Format(time.RFC3339),
			})
		}
	}
	return observed
}

func addObserved(values map[string]int, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	values[value]++
}

func canonicalTraceID(value string) string {
	raw := strings.TrimSpace(value)
	lower := strings.ToLower(raw)
	if raw == "" || raw == "0" {
		return ""
	}
	if isDecimal(raw) {
		parsed, ok := new(big.Int).SetString(raw, 10)
		if !ok {
			return ""
		}
		return leftPad(parsed.Text(16), 32)
	}
	if isHex(lower) {
		return leftPad(lower, 32)
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		switch len(decoded) {
		case 16:
			return hex.EncodeToString(decoded)
		case 8:
			return leftPad(hex.EncodeToString(decoded), 32)
		}
	}
	return ""
}

func isDecimal(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func isHex(value string) bool {
	if value == "" || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func leftPad(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return strings.Repeat("0", width-len(value)) + value
}

func probeCheck(id string, ok bool, passMessage, failHint string) CheckResult {
	if ok {
		return CheckResult{ID: id, Status: "pass", Message: passMessage, Matched: 1}
	}
	return CheckResult{ID: id, Status: "fail", Message: strings.TrimSuffix(failHint, "."), Hint: failHint}
}

func countCheck(id string, count int, passMessage, failHint string) CheckResult {
	if count > 0 {
		return CheckResult{ID: id, Status: "pass", Message: passMessage, Matched: count}
	}
	return CheckResult{ID: id, Status: "fail", Message: "Expected evidence was not observed.", Hint: failHint}
}

func addExpectedChecks(checks *[]CheckResult, kind string, expected []string, observed map[string]int) {
	for _, value := range expected {
		id := kind + ":" + value
		count := observed[value]
		if count > 0 {
			*checks = append(*checks, CheckResult{
				ID:      id,
				Status:  "pass",
				Message: fmt.Sprintf("Observed expected %s %q.", kind, value),
				Matched: count,
			})
			continue
		}
		*checks = append(*checks, CheckResult{
			ID:      id,
			Status:  "fail",
			Message: fmt.Sprintf("Expected %s %q was not observed.", kind, value),
			Hint:    missingHint(kind, value),
		})
	}
}

func addExpectedTraceChecks(checks *[]CheckResult, expected []string, observed map[string]int) {
	for _, value := range expected {
		id := "trace:" + value
		count := observed[value]
		if canonical := canonicalTraceID(value); count == 0 && canonical != "" && canonical != value {
			count += observed[canonical]
		}
		if count > 0 {
			*checks = append(*checks, CheckResult{
				ID:      id,
				Status:  "pass",
				Message: fmt.Sprintf("Observed expected trace %q.", value),
				Matched: count,
			})
			continue
		}
		*checks = append(*checks, CheckResult{
			ID:      id,
			Status:  "fail",
			Message: fmt.Sprintf("Expected trace %q was not observed.", value),
			Hint:    missingHint("trace", value),
		})
	}
}

func missingHint(kind, value string) string {
	switch kind {
	case "source":
		return missingSourceHint(value)
	case "payload-kind":
		return missingPayloadKindHint(value)
	case "service":
		return "Check unified service tag configuration: DD_SERVICE for Datadog tracers/logs, OTEL_SERVICE_NAME or service.name resource attributes for OpenTelemetry, and frontend RUM service names."
	case "session":
		return "RUM/Faro reached Dogtap without the expected session id, or the browser workflow did not run. Check browser network calls, sampling, SDK initialization order, and session replay/RUM session configuration."
	case "trace":
		return "Check trace exporter routing and ID correlation. For Datadog tracers use DD_TRACE_AGENT_URL or DD_AGENT_HOST/DD_TRACE_AGENT_PORT. For OTLP use OTEL_EXPORTER_OTLP_ENDPOINT and the correct grpc/http protocol."
	case "case":
		return "Check workflow context propagation. The app should attach case/workflow identifiers to RUM context, logs, traces, or OTLP attributes before the action emits telemetry."
	case "route":
		return "Check route normalization and framework instrumentation. Backend spans/logs should include stable route templates, not only concrete URLs."
	case "metric":
		return "Check OTLP metrics exporter configuration, export interval, and endpoint. Dogtap does not currently receive DogStatsD metrics unless they are bridged through OTLP."
	case "endpoint":
		return "Check that the SDK, tracer, proxy, or collector is pointed at the expected Dogtap intake endpoint and that container networking can reach it."
	default:
		return "Check Dogtap events.json and debug-bundle.json for nearby telemetry from the same workflow."
	}
}

func missingSourceHint(source string) string {
	switch source {
	case "rum":
		return "Check frontend runtime config for the Datadog RUM proxy URL, browser network calls to /datadog-intake-proxy or /rum, CORS/proxy rules, and whether the workflow ran in a browser context."
	case "faro":
		return "Check Faro SDK collectorUrl, /faro or /collect routing, browser network failures, and remember native Faro support is smoke-level unless routed through Alloy to OTLP."
	case "logs":
		return "Check how logs are delivered. Dogtap does not tail containers like the Datadog Agent; route logs through /api/v2/logs, /v1/input, OTLP /v1/logs, or a log-forwarder bridge."
	case "apm":
		return "Check Datadog tracer agent settings: DD_TRACE_AGENT_URL, DD_AGENT_HOST, DD_TRACE_AGENT_PORT, and whether the tracer starts before the app workflow."
	case "otlp":
		return "Check OTEL_EXPORTER_OTLP_ENDPOINT, protocol selection, and whether traces/logs/metrics exporters are enabled. Use 4317 for gRPC or 4318 for HTTP."
	default:
		return "Check endpoint routing and Dogtap events.json for any nearby unknown-source payloads."
	}
}

func missingPayloadKindHint(kind string) string {
	switch kind {
	case "replay":
		return "Check session replay enablement, replay sample rate, Browser SDK replay intake path /api/v2/replay through the proxy, multipart forwarding, and whether user consent/session sampling allowed replay capture."
	case "metric":
		return "Check OTLP metrics exporter enablement, export interval, and endpoint. Dogtap does not currently accept DogStatsD directly."
	case "trace":
		return "Check Datadog APM or OTLP trace exporter configuration and the Dogtap trace intake port/path."
	case "log":
		return "Check structured log forwarding. If logs only go to stdout, add a log forwarder bridge or send HTTP/OTLP logs to Dogtap."
	case "event":
		return "Check the browser SDK event intake endpoint and network calls."
	default:
		return "Check events.json for the source and endpoint that arrived, then compare payloadKind with the expected telemetry type."
	}
}

func RenderSummary(result Result) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Dogtap Diagnostics")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Created: %s\n", result.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- Base URL: `%s`\n", result.BaseURL)
	fmt.Fprintf(&b, "- Status: `%s`\n", result.Assertions.Status)
	if len(result.WorkflowContracts) > 0 {
		workflowStatus := "pass"
		if workflowContractsFailed(result.WorkflowContracts) {
			workflowStatus = "fail"
		}
		fmt.Fprintf(&b, "- Workflow contracts: `%s`\n", workflowStatus)
	}
	fmt.Fprintf(&b, "- Events: `%d`\n", result.Assertions.Observed.Total)
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Files")
	fmt.Fprintln(&b)
	for _, name := range sortedFileNames(result.Files) {
		file := result.Files[name]
		fmt.Fprintf(&b, "- `%s`: `%s`\n", name, file.Path)
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Observed")
	fmt.Fprintln(&b)
	writeMapSummary(&b, "Sources", result.Assertions.Observed.Sources)
	writeMapSummary(&b, "Payload Kinds", result.Assertions.Observed.PayloadKinds)
	writeMapSummary(&b, "Services", result.Assertions.Observed.Services)
	writeMapSummary(&b, "Sessions", result.Assertions.Observed.Sessions)
	writeMapSummary(&b, "Traces", result.Assertions.Observed.Traces)
	writeMapSummary(&b, "Cases", result.Assertions.Observed.Cases)
	writeMapSummary(&b, "Routes", result.Assertions.Observed.Routes)
	writeMapSummary(&b, "Metrics", result.Assertions.Observed.Metrics)
	writeMapSummary(&b, "Endpoints", result.Assertions.Observed.Endpoints)
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Checks")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Status | Check | Matched | Message | Hint |")
	fmt.Fprintln(&b, "| --- | --- | ---: | --- | --- |")
	for _, check := range result.Assertions.Checks {
		fmt.Fprintf(
			&b,
			"| %s | `%s` | %d | %s | %s |\n",
			check.Status,
			markdownCell(check.ID),
			check.Matched,
			markdownCell(check.Message),
			markdownCell(check.Hint),
		)
	}
	if len(result.WorkflowContracts) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## Workflow Contracts")
		fmt.Fprintln(&b)
		for _, workflow := range result.WorkflowContracts {
			fmt.Fprintf(
				&b,
				"- `%s`: `%s` (%d passed, %d failed)\n",
				markdownCell(workflow.Name),
				workflow.Status,
				workflow.Summary.Passed,
				workflow.Summary.Failed,
			)
		}
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "| Status | Contract | Check | Matched | Message | Hint |")
		fmt.Fprintln(&b, "| --- | --- | --- | ---: | --- | --- |")
		for _, workflow := range result.WorkflowContracts {
			for _, check := range workflow.Checks {
				fmt.Fprintf(
					&b,
					"| %s | `%s` | `%s` | %d | %s | %s |\n",
					check.Status,
					markdownCell(workflow.Name),
					markdownCell(check.ID),
					check.Matched,
					markdownCell(check.Message),
					markdownCell(check.Hint),
				)
			}
		}
	}
	return b.String()
}

func sortedFileNames(files map[string]File) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeMapSummary(b *strings.Builder, title string, values map[string]int) {
	if len(values) == 0 {
		fmt.Fprintf(b, "- %s: `none`\n", title)
		return
	}
	parts := make([]string, 0, len(values))
	for _, key := range sortedKeys(values) {
		parts = append(parts, fmt.Sprintf("`%s=%d`", key, values[key]))
	}
	fmt.Fprintf(b, "- %s: %s\n", title, strings.Join(parts, ", "))
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
