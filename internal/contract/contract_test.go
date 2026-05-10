package contract

import (
	"os"
	"path/filepath"
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
