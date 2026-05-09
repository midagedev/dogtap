package bundle

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

type Request struct {
	Source      event.Source `json:"source,omitempty"`
	PayloadKind string       `json:"payloadKind,omitempty"`
	Service     string       `json:"service,omitempty"`
	Env         string       `json:"env,omitempty"`
	UserID      string       `json:"userId,omitempty"`
	AccountID   string       `json:"accountId,omitempty"`
	WorkspaceID string       `json:"workspaceId,omitempty"`
	CaseID      string       `json:"caseId,omitempty"`
	TraceID     string       `json:"traceId,omitempty"`
	SessionID   string       `json:"sessionId,omitempty"`
	ViewID      string       `json:"viewId,omitempty"`
	Route       string       `json:"route,omitempty"`
	Status      string       `json:"status,omitempty"`
	Limit       int          `json:"limit,omitempty"`
}

type DebugBundle struct {
	BundleID           string                `json:"bundleId"`
	CreatedAt          time.Time             `json:"createdAt"`
	Filter             Request               `json:"filter"`
	Summary            Summary               `json:"summary"`
	Events             []event.EventEnvelope `json:"events"`
	ValidationFailures []FailureSummary      `json:"validationFailures"`
	DatadogQueries     []DatadogQuery        `json:"datadogQueries"`
	RedactionReport    RedactionReport       `json:"redactionReport"`
}

type Summary struct {
	Total    int            `json:"total"`
	BySource map[string]int `json:"bySource"`
	Passed   int            `json:"passed"`
	Failed   int            `json:"failed"`
}

type FailureSummary struct {
	EventID   string `json:"eventId"`
	Source    string `json:"source"`
	RuleID    string `json:"ruleId"`
	Severity  string `json:"severity"`
	FieldPath string `json:"fieldPath,omitempty"`
	Message   string `json:"message"`
}

type DatadogQuery struct {
	Label string `json:"label"`
	Query string `json:"query"`
	URL   string `json:"url"`
}

type RedactionReport struct {
	RawEventCount      int    `json:"rawEventCount"`
	RedactedEventCount int    `json:"redactedEventCount"`
	Note               string `json:"note"`
}

func New(req Request, events []event.EventEnvelope) DebugBundle {
	summary := Summary{Total: len(events), BySource: map[string]int{}}
	failures := make([]FailureSummary, 0)
	queries := map[string]DatadogQuery{}
	redaction := RedactionReport{
		Note: "Raw payload availability follows the current Dogtap mode and event retention policy.",
	}

	for _, e := range events {
		summary.BySource[string(e.Source)]++
		if e.Validation.Status == "fail" {
			summary.Failed++
		} else {
			summary.Passed++
		}
		if e.RawBody == "" {
			redaction.RedactedEventCount++
		} else {
			redaction.RawEventCount++
		}
		for _, rule := range e.Validation.Rules {
			if rule.Status != "fail" {
				continue
			}
			failures = append(failures, FailureSummary{
				EventID:   e.ID,
				Source:    string(e.Source),
				RuleID:    rule.RuleID,
				Severity:  rule.Severity,
				FieldPath: rule.FieldPath,
				Message:   rule.Message,
			})
		}
		for _, query := range queriesFor(e) {
			queries[query.Label+"="+query.Query] = query
		}
	}

	queryList := make([]DatadogQuery, 0, len(queries))
	for _, query := range queries {
		queryList = append(queryList, query)
	}
	sort.Slice(queryList, func(i, j int) bool {
		return queryList[i].Label < queryList[j].Label
	})

	return DebugBundle{
		BundleID:           newID(),
		CreatedAt:          time.Now().UTC(),
		Filter:             req,
		Summary:            summary,
		Events:             events,
		ValidationFailures: failures,
		DatadogQueries:     queryList,
		RedactionReport:    redaction,
	}
}

func queriesFor(e event.EventEnvelope) []DatadogQuery {
	n := e.Normalized
	parts := make([]string, 0, 6)
	add := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, key+":"+quoteIfNeeded(value))
		}
	}
	add("service", n.Service)
	add("env", n.Env)
	add("version", n.Version)

	out := make([]DatadogQuery, 0, 5)
	if len(parts) > 0 {
		out = append(out, datadogQuery("Service context", strings.Join(parts, " ")))
	}
	if n.TraceID != "" {
		out = append(out, datadogQuery("Trace ID", "trace_id:"+quoteIfNeeded(n.TraceID)))
	}
	if n.UserID != "" {
		out = append(out, datadogQuery("User ID", "@usr.id:"+quoteIfNeeded(n.UserID)))
	}
	if n.WorkspaceID != "" {
		out = append(out, datadogQuery("Workspace ID", "@workspace.id:"+quoteIfNeeded(n.WorkspaceID)))
	}
	if n.CaseID != "" {
		out = append(out, datadogQuery("Case ID", "@case_id:"+quoteIfNeeded(n.CaseID)))
	}
	return out
}

func datadogQuery(label, query string) DatadogQuery {
	return DatadogQuery{
		Label: label,
		Query: query,
		URL:   "https://app.datadoghq.com/logs?query=" + url.QueryEscape(query),
	}
}

func quoteIfNeeded(value string) string {
	if strings.ContainsAny(value, " /{}") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("bundle-%d", time.Now().UnixNano())
	}
	return "bundle-" + hex.EncodeToString(b[:])
}
