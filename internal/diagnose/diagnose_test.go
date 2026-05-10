package diagnose

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/contract"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/report"
)

func TestCollectWritesAgentReadableBundle(t *testing.T) {
	events := []event.EventEnvelope{
		{
			ID:          "rum-1",
			ReceivedAt:  time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
			Source:      event.SourceRUM,
			PayloadKind: "event",
			Endpoint:    "/datadog-intake-proxy",
			Method:      http.MethodPost,
			Normalized: event.NormalizedTelemetry{
				Source:    event.SourceRUM,
				Service:   "web-frontend",
				Env:       "local",
				SessionID: "session-123",
				CaseID:    "case-123",
				Route:     "/cases/case-123",
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
		{
			ID:          "metric-1",
			ReceivedAt:  time.Date(2026, 5, 9, 12, 0, 1, 0, time.UTC),
			Source:      event.SourceOTLP,
			PayloadKind: "metric",
			Endpoint:    "/v1/metrics",
			Method:      http.MethodPost,
			Details: &event.TelemetryDetails{
				Metrics: []event.MetricEntry{{Name: "http.server.request.duration", Service: "api-service"}},
			},
			Normalized: event.NormalizedTelemetry{
				Source:  event.SourceOTLP,
				Service: "api-service",
				Env:     "local",
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
	}
	server := diagnosticsServer(t, events)
	output := filepath.Join(t.TempDir(), "diagnostics")

	result, err := Collect(context.Background(), Options{
		BaseURL:   server.URL,
		OutputDir: output,
		Expectations: Expectations{
			NonEmpty:     true,
			Sources:      []string{"rum", "otlp"},
			PayloadKinds: []string{"metric"},
			Services:     []string{"web-frontend", "api-service"},
			Sessions:     []string{"session-123"},
			Metrics:      []string{"http.server.request.duration"},
		},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Assertions.Status != "pass" {
		t.Fatalf("assertions status = %s", result.Assertions.Status)
	}

	for _, name := range []string{
		"manifest.json",
		"healthz.json",
		"readyz.json",
		"events.json",
		"report.json",
		"metrics.txt",
		"debug-bundle.json",
		"assertions.json",
		"summary.md",
	} {
		if _, err := os.Stat(filepath.Join(output, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	summary := readFile(t, filepath.Join(output, "summary.md"))
	if !strings.Contains(summary, "source:rum") || !strings.Contains(summary, "metric:http.server.request.duration") {
		t.Fatalf("summary missing checks:\n%s", summary)
	}
}

func TestCollectWritesWorkflowContractArtifactWithoutChangingAssertions(t *testing.T) {
	events := []event.EventEnvelope{
		{
			ID:          "rum-1",
			ReceivedAt:  time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
			Source:      event.SourceRUM,
			PayloadKind: "event",
			Endpoint:    "/datadog-intake-proxy",
			Method:      http.MethodPost,
			Normalized: event.NormalizedTelemetry{
				Source:    event.SourceRUM,
				Service:   "web-frontend",
				SessionID: "session-123",
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
	}
	server := diagnosticsServer(t, events)
	output := filepath.Join(t.TempDir(), "diagnostics")

	result, err := Collect(context.Background(), Options{
		BaseURL:   server.URL,
		OutputDir: output,
		WorkflowContracts: []contract.Definition{{
			Name: "login-workflow",
			Checks: []contract.CheckDefinition{{
				ID:     "backend-log",
				Type:   "log-message",
				Source: "logs",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Assertions.Status != "pass" {
		t.Fatalf("assertions status = %s, want pass", result.Assertions.Status)
	}
	if len(result.WorkflowContracts) != 1 || result.WorkflowContracts[0].Status != "fail" {
		t.Fatalf("unexpected workflow contracts: %#v", result.WorkflowContracts)
	}
	if _, err := os.Stat(filepath.Join(output, "workflow-contracts.json")); err != nil {
		t.Fatalf("expected workflow-contracts.json: %v", err)
	}
	summary := readFile(t, filepath.Join(output, "summary.md"))
	if !strings.Contains(summary, "## Workflow Contracts") || !strings.Contains(summary, "backend-log") {
		t.Fatalf("summary missing workflow contract section:\n%s", summary)
	}
}

func TestCollectCanFailOnWorkflowContractWhenRequested(t *testing.T) {
	server := diagnosticsServer(t, nil)
	output := filepath.Join(t.TempDir(), "diagnostics")

	_, err := Collect(context.Background(), Options{
		BaseURL:                server.URL,
		OutputDir:              output,
		FailOnWorkflowContract: true,
		WorkflowContracts: []contract.Definition{{
			Name: "login-workflow",
			Checks: []contract.CheckDefinition{{
				ID:     "login-rum",
				Type:   "event",
				Source: "rum",
			}},
		}},
	})
	if !errors.Is(err, report.ErrValidationFailed) {
		t.Fatalf("Collect error = %v, want validation failure", err)
	}
}

func TestCollectFailsWithActionableHintForMissingExpectation(t *testing.T) {
	server := diagnosticsServer(t, []event.EventEnvelope{
		{
			ID:          "rum-1",
			ReceivedAt:  time.Now().UTC(),
			Source:      event.SourceRUM,
			PayloadKind: "event",
			Endpoint:    "/datadog-intake-proxy",
			Method:      http.MethodPost,
			Normalized: event.NormalizedTelemetry{
				Source:  event.SourceRUM,
				Service: "web-frontend",
				Env:     "local",
			},
			Validation: event.ValidationResult{Status: "pass"},
		},
	})
	output := filepath.Join(t.TempDir(), "diagnostics")

	result, err := Collect(context.Background(), Options{
		BaseURL:   server.URL,
		OutputDir: output,
		Expectations: Expectations{
			Sources:      []string{"logs"},
			PayloadKinds: []string{"replay"},
		},
	})
	if !errors.Is(err, report.ErrValidationFailed) {
		t.Fatalf("Collect error = %v, want validation failure", err)
	}
	if result.Assertions.Status != "fail" || result.Assertions.Summary.Failed == 0 {
		t.Fatalf("unexpected assertions: %#v", result.Assertions.Summary)
	}

	assertions := readFile(t, filepath.Join(output, "assertions.json"))
	for _, want := range []string{"source:logs", "payload-kind:replay", "Dogtap does not tail containers", "session replay", "rootCauses", "backend-logs-not-forwarded", "browser-telemetry-not-reaching-dogtap"} {
		if !strings.Contains(assertions, want) {
			t.Fatalf("assertions missing %q:\n%s", want, assertions)
		}
	}
	summary := readFile(t, filepath.Join(output, "summary.md"))
	for _, want := range []string{"## Likely Causes", "backend-logs-not-forwarded", "browser-telemetry-not-reaching-dogtap"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q:\n%s", want, summary)
		}
	}
}

func TestBuildAssertionsMatchesTraceAliases(t *testing.T) {
	report := BuildAssertions(
		[]event.EventEnvelope{
			{
				ID:     "trace-1",
				Source: event.SourceOTLP,
				Normalized: event.NormalizedTelemetry{
					Source:  event.SourceOTLP,
					TraceID: "000000000000000000000000075bcd15",
				},
				Validation: event.ValidationResult{Status: "pass"},
			},
		},
		Expectations{Traces: []string{"123456789"}},
		map[string]bool{},
	)

	if report.Status != "fail" {
		t.Fatalf("probe checks should still fail without probe evidence: %#v", report)
	}
	var traceCheck *CheckResult
	for index := range report.Checks {
		if report.Checks[index].ID == "trace:123456789" {
			traceCheck = &report.Checks[index]
			break
		}
	}
	if traceCheck == nil || traceCheck.Status != "pass" {
		t.Fatalf("expected trace alias check to pass: %#v", report.Checks)
	}
}

func TestBuildAssertionsClassifiesOTLPAndEndpointFailures(t *testing.T) {
	report := BuildAssertions(
		[]event.EventEnvelope{
			{
				ID:          "rum-1",
				Source:      event.SourceRUM,
				PayloadKind: "event",
				Endpoint:    "/datadog-intake-proxy",
				Normalized: event.NormalizedTelemetry{
					Source: event.SourceRUM,
				},
				Validation: event.ValidationResult{Status: "pass"},
			},
		},
		Expectations{Sources: []string{"otlp"}, Endpoints: []string{"/v1/traces"}},
		map[string]bool{"healthz": true, "readyz": true, "events": true, "report": true, "metrics": true, "debug-bundle": true},
	)

	got := map[string]bool{}
	for _, cause := range report.RootCauses {
		got[cause.ID] = true
	}
	for _, want := range []string{"otel-exporter-not-reaching-dogtap", "endpoint-routing-mismatch"} {
		if !got[want] {
			t.Fatalf("root causes missing %q: %#v", want, report.RootCauses)
		}
	}
}

func diagnosticsServer(t *testing.T, events []event.EventEnvelope) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, events)
	})
	mux.HandleFunc("GET /api/reports/latest", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, report.FromEvents(events))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("dogtap_store_events 2\n"))
	})
	mux.HandleFunc("POST /api/debug-bundles", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, map[string]any{"summary": map[string]int{"total": len(events)}, "events": events})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
