package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

func TestFrontendBackendReadinessPassesWithRepresentativeSignals(t *testing.T) {
	result := Evaluate(FrontendBackendReadiness(), representativeEvents())

	if result.Status != "pass" {
		t.Fatalf("status = %s, want pass: %#v", result.Status, result.Checks)
	}
	if result.Summary.Total != 6 || result.Summary.Failed != 0 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
}

func TestTraceCorrelationMatchesDecimalAndHexAliases(t *testing.T) {
	result := Evaluate(Definition{
		Name: "login-workflow",
		Checks: []CheckDefinition{
			{
				ID:   "browser-to-api-trace",
				Type: "trace-correlation",
				From: Selector{
					Source: "rum",
					Fields: []string{"sessionId", "traceId"},
				},
				To: Selector{
					PayloadKind: "trace",
					Service:     "api-service",
				},
			},
		},
	}, representativeEvents())

	if result.Status != "pass" {
		t.Fatalf("status = %s, want pass: %#v", result.Status, result.Checks)
	}
	check := result.Checks[0]
	if check.Matched != 1 || len(check.TraceIDs) != 1 || check.TraceIDs[0] != "000000000000000000000000075bcd15" {
		t.Fatalf("unexpected trace correlation check: %#v", check)
	}
}

func TestMissingRequiredSignalFailsWithHint(t *testing.T) {
	result := Evaluate(Definition{
		Name: "login-workflow",
		Checks: []CheckDefinition{
			{
				ID:      "login-log",
				Type:    "log-message",
				Source:  "logs",
				Pattern: "login succeeded",
				Hint:    "Check backend log forwarding.",
			},
		},
	}, representativeEvents()[:1])

	if result.Status != "fail" {
		t.Fatalf("status = %s, want fail", result.Status)
	}
	check := result.Checks[0]
	if check.Status != "fail" || check.Hint != "Check backend log forwarding." {
		t.Fatalf("unexpected check: %#v", check)
	}
}

func TestFailedEventCheckIncludesSelectorAlternatives(t *testing.T) {
	result := Evaluate(Definition{
		Name: "login-workflow",
		Checks: []CheckDefinition{{
			ID:     "login-rum-context",
			Type:   "event",
			Source: "rum",
			Route:  "/login",
			Fields: []string{"accountId", "userId"},
		}},
	}, representativeEvents())

	check := result.Checks[0]
	if check.Status != "fail" || len(check.Selectors) != 1 {
		t.Fatalf("unexpected check: %#v", check)
	}
	selector := check.Selectors[0]
	if selector.Criteria.Source != "rum" || selector.Criteria.Route != "/login" || selector.Matched != 0 {
		t.Fatalf("unexpected selector: %#v", selector)
	}
	if len(selector.Alternatives) == 0 || selector.Alternatives[0].EventID != "rum-1" {
		t.Fatalf("expected rum-1 as closest alternative: %#v", selector.Alternatives)
	}
	if !hasString(selector.Alternatives[0].MissingFields, "accountId") || !hasString(selector.Alternatives[0].PresentFields, "userId") {
		t.Fatalf("unexpected alternative fields: %#v", selector.Alternatives[0])
	}
}

func TestFailedLogCheckIncludesPatternAlternative(t *testing.T) {
	result := Evaluate(Definition{
		Name: "login-workflow",
		Checks: []CheckDefinition{{
			ID:      "login-log",
			Type:    "log-message",
			Source:  "logs",
			Pattern: "login succeeded",
		}},
	}, representativeEvents())

	check := result.Checks[0]
	if check.Status != "fail" || len(check.Selectors) != 1 {
		t.Fatalf("unexpected check: %#v", check)
	}
	selector := check.Selectors[0]
	if selector.Pattern != "login succeeded" || selector.Matched != 1 {
		t.Fatalf("unexpected selector: %#v", selector)
	}
	if len(selector.Alternatives) == 0 || selector.Alternatives[0].EventID != "log-1" || !hasString(selector.Alternatives[0].Differences, "log pattern did not match") {
		t.Fatalf("unexpected log alternatives: %#v", selector.Alternatives)
	}
}

func TestFailedTraceCorrelationIncludesFromAndToSelectors(t *testing.T) {
	result := Evaluate(Definition{
		Name: "login-workflow",
		Checks: []CheckDefinition{{
			ID:   "browser-to-billing-trace",
			Type: "trace-correlation",
			From: Selector{
				Source: "rum",
				Fields: []string{"traceId"},
			},
			To: Selector{
				PayloadKind: "trace",
				Service:     "billing-service",
			},
		}},
	}, representativeEvents())

	check := result.Checks[0]
	if check.Status != "fail" || len(check.Selectors) != 2 {
		t.Fatalf("unexpected check: %#v", check)
	}
	if check.Selectors[0].Label != "from" || check.Selectors[0].Matched != 1 {
		t.Fatalf("unexpected from selector: %#v", check.Selectors[0])
	}
	if check.Selectors[1].Label != "to" || check.Selectors[1].Matched != 0 {
		t.Fatalf("unexpected to selector: %#v", check.Selectors[1])
	}
	if len(check.Selectors[1].Alternatives) == 0 || check.Selectors[1].Alternatives[0].EventID != "trace-1" {
		t.Fatalf("expected trace-1 as closest to alternative: %#v", check.Selectors[1].Alternatives)
	}
}

func TestNoSensitiveValuesFindsLowercaseEmailBearerAndJWT(t *testing.T) {
	events := []event.EventEnvelope{
		{
			ID:     "email",
			Source: event.SourceRUM,
			Normalized: event.NormalizedTelemetry{
				UserID: "dev@example.com",
			},
		},
		{
			ID:     "bearer",
			Source: event.SourceLogs,
			Details: &event.TelemetryDetails{
				Logs: []event.LogEntry{{Message: "Authorization: Bearer abcdefghijklmnopqrstuvwxyz"}},
			},
		},
		{
			ID:     "jwt",
			Source: event.SourceLogs,
			Details: &event.TelemetryDetails{
				Logs: []event.LogEntry{{Message: "token eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.signature"}},
			},
		},
	}

	result := Evaluate(Definition{
		Name: "privacy",
		Checks: []CheckDefinition{{
			ID:   "no-sensitive-values",
			Type: "no-sensitive-values",
		}},
	}, events)

	if result.Status != "fail" {
		t.Fatalf("status = %s, want fail: %#v", result.Status, result.Checks)
	}
	if got := result.Checks[0].Matched; got != 3 {
		t.Fatalf("matched = %d, want 3", got)
	}
}

func TestBundledContractTemplatesLoad(t *testing.T) {
	dir := filepath.Join("..", "..", "configs", "contracts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected bundled contract templates")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		def, err := LoadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("load %s: %v", entry.Name(), err)
		}
		if def.Name == "" || len(def.Checks) == 0 {
			t.Fatalf("template %s is missing name or checks: %#v", entry.Name(), def)
		}
		report := ValidateFile(filepath.Join(dir, entry.Name()))
		if report.Status != "pass" {
			t.Fatalf("template %s should validate: %#v", entry.Name(), report.Issues)
		}
		seen := map[string]bool{}
		for _, check := range def.Checks {
			if check.ID == "" || check.Type == "" {
				t.Fatalf("template %s has incomplete check: %#v", entry.Name(), check)
			}
			if seen[check.ID] {
				t.Fatalf("template %s has duplicate check id %q", entry.Name(), check.ID)
			}
			seen[check.ID] = true
		}
	}
}

func TestValidateFileRejectsAuthoringErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	if err := os.WriteFile(path, []byte(`name: broken-workflow
checks:
  - id: duplicate
    type: event
    source: browser
    payloadKind: span
    routeRegex: "["
    fields:
      - session
      - sessionId
      - sessionId
    from:
      source: rum
  - id: duplicate
    type: unknown-check
  - id: trace-top-level-selector
    type: trace-correlation
    source: rum
    from:
      source: rum
    to:
      payloadKind: trace
  - id: sensitive-selector
    type: no-sensitive-values
    source: logs
    pattern: password
    metric: auth.failures
`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	report := ValidateFile(path)

	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	messages := validationMessages(report)
	for _, expected := range []string{
		`unsupported source "browser"`,
		`unsupported payload kind "span"`,
		"invalid regex",
		`unsupported selector field "session"`,
		`duplicate selector field "sessionId"`,
		"from selectors are only supported on trace-correlation checks",
		`duplicate check id "duplicate"`,
		`unsupported check type "unknown-check"`,
		"trace-correlation checks use from/to selectors and do not support top-level selectors",
		"no-sensitive-values checks inspect all visible values and do not support selectors",
		"no-sensitive-values checks inspect all visible values and do not support pattern",
		"metric is only supported on metric checks",
	} {
		if !strings.Contains(messages, expected) {
			t.Fatalf("expected %q in validation issues:\n%s", expected, messages)
		}
	}
}

func TestValidateFileRejectsMissingNameAndChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	if err := os.WriteFile(path, []byte(`description: incomplete workflow
checks: []
`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	report := ValidateFile(path)

	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	messages := validationMessages(report)
	for _, expected := range []string{"contract name is required", "at least one check is required"} {
		if !strings.Contains(messages, expected) {
			t.Fatalf("expected %q in validation issues:\n%s", expected, messages)
		}
	}
}

func TestValidateFileRejectsTrailingJSONValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.json")
	if err := os.WriteFile(path, []byte(`{"name":"one","checks":[{"id":"rum","type":"event"}]}
{"name":"two","checks":[{"id":"rum","type":"event"}]}`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	report := ValidateFile(path)

	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	if !strings.Contains(validationMessages(report), "multiple JSON values are not supported") {
		t.Fatalf("expected trailing JSON issue, got: %#v", report.Issues)
	}
}

func TestValidateFileRejectsMultipleYAMLDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	if err := os.WriteFile(path, []byte(`name: one
checks:
  - id: rum
    type: event
---
name: two
checks:
  - id: rum
    type: event
`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	report := ValidateFile(path)

	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	if !strings.Contains(validationMessages(report), "multiple YAML documents are not supported") {
		t.Fatalf("expected multiple YAML document issue, got: %#v", report.Issues)
	}
}

func TestValidateFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	if err := os.WriteFile(path, []byte(`name: broken-workflow
checks:
  - id: rum-event
    type: event
    routeRegexp: "/login"
`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	report := ValidateFile(path)

	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	if !strings.Contains(validationMessages(report), "field routeRegexp not found") {
		t.Fatalf("expected unknown field issue, got: %#v", report.Issues)
	}
}

func TestValidateFileAllowsSchemaHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	if err := os.WriteFile(path, []byte(`$schema: ../../schemas/workflow-contract.schema.json
name: schema-hint-workflow
checks:
  - id: rum-event
    type: event
    source: rum
`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	report := ValidateFile(path)

	if report.Status != "pass" {
		t.Fatalf("status = %s, want pass: %#v", report.Status, report.Issues)
	}
}

func validationMessages(report ValidationReport) string {
	messages := []string{}
	for _, issue := range report.Issues {
		messages = append(messages, issue.Message)
	}
	return strings.Join(messages, "\n")
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func representativeEvents() []event.EventEnvelope {
	ts := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	return []event.EventEnvelope{
		{
			ID:          "rum-1",
			ReceivedAt:  ts,
			Source:      event.SourceRUM,
			PayloadKind: "event",
			Endpoint:    "/datadog-intake-proxy",
			Normalized: event.NormalizedTelemetry{
				Source:    event.SourceRUM,
				Service:   "web-frontend",
				UserID:    "user-1",
				SessionID: "session-1",
				TraceID:   "123456789",
				Route:     "/login",
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
		{
			ID:          "replay-1",
			ReceivedAt:  ts.Add(time.Second),
			Source:      event.SourceRUM,
			PayloadKind: "replay",
			Endpoint:    "/api/v2/replay",
			Normalized: event.NormalizedTelemetry{
				Source:    event.SourceRUM,
				Service:   "web-frontend",
				UserID:    "user-1",
				SessionID: "session-1",
			},
			Details: &event.TelemetryDetails{
				Replay: &event.ReplayDetail{RecordCount: 2},
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
		{
			ID:          "log-1",
			ReceivedAt:  ts.Add(2 * time.Second),
			Source:      event.SourceLogs,
			PayloadKind: "log",
			Endpoint:    "/api/v2/logs",
			Normalized: event.NormalizedTelemetry{
				Source:  event.SourceLogs,
				Service: "api-service",
			},
			Details: &event.TelemetryDetails{
				Logs: []event.LogEntry{{Level: "info", Message: "login completed", TraceID: "000000000000000000000000075bcd15"}},
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
		{
			ID:          "trace-1",
			ReceivedAt:  ts.Add(3 * time.Second),
			Source:      event.SourceAPM,
			PayloadKind: "trace",
			Endpoint:    "/v0.5/traces",
			Normalized: event.NormalizedTelemetry{
				Source:  event.SourceAPM,
				Service: "api-service",
				TraceID: "000000000000000000000000075bcd15",
				Route:   "/api/login",
			},
			Details: &event.TelemetryDetails{
				Trace: &event.TraceDetail{
					TraceID: "000000000000000000000000075bcd15",
					Spans: []event.SpanDetail{{
						TraceID: "000000000000000000000000075bcd15",
						SpanID:  "span-1",
						Name:    "POST /api/login",
						Service: "api-service",
					}},
				},
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
		{
			ID:          "metric-1",
			ReceivedAt:  ts.Add(4 * time.Second),
			Source:      event.SourceOTLP,
			PayloadKind: "metric",
			Endpoint:    "/v1/metrics",
			Normalized: event.NormalizedTelemetry{
				Source:  event.SourceOTLP,
				Service: "api-service",
				Route:   "/api/login",
			},
			Details: &event.TelemetryDetails{
				Metrics: []event.MetricEntry{{Name: "http.server.request.duration", Service: "api-service", Route: "/api/login", Value: 42}},
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
	}
}
