package intake

import (
	"testing"

	"github.com/midagedev/dogtap/internal/event"
)

func TestLogDetailsDecodeOTLPLogRecords(t *testing.T) {
	decoded := map[string]any{
		"resourceLogs": []any{
			map[string]any{
				"scopeLogs": []any{
					map[string]any{
						"logRecords": []any{
							map[string]any{
								"timeUnixNano": "1778341581124954115",
								"severityText": "warn",
								"traceId":      "trace-1",
								"body": map[string]any{
									"stringValue": "[GET] /api/cloud/__dogtap-log-probe-1 - [No Resource Found]",
								},
							},
							map[string]any{
								"severityText": "info",
								"body": map[string]any{
									"stringValue": "backend request completed",
								},
							},
						},
					},
				},
			},
		},
	}

	entries := logDetails(decoded, event.NormalizedTelemetry{Timestamp: "fallback-time", TraceID: "fallback-trace"})
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d: %#v", len(entries), entries)
	}
	if entries[0].Message != "[GET] /api/cloud/__dogtap-log-probe-1 - [No Resource Found]" {
		t.Fatalf("unexpected first message: %#v", entries[0])
	}
	if entries[0].Level != "WARN" {
		t.Fatalf("unexpected first level: %#v", entries[0])
	}
	if entries[0].Timestamp != "1778341581124954115" {
		t.Fatalf("unexpected first timestamp: %#v", entries[0])
	}
	if entries[0].TraceID != "trace-1" {
		t.Fatalf("unexpected first trace id: %#v", entries[0])
	}
	if entries[1].Message != "backend request completed" {
		t.Fatalf("unexpected second message: %#v", entries[1])
	}
}

func TestLogDetailsPreserveStructuredFields(t *testing.T) {
	entries := logDetails(
		map[string]any{
			"message":          "login failed",
			"status":           "error",
			"trace_id":         "trace-1",
			"span_id":          "span-1",
			"route":            "/api/login",
			"http.method":      "POST",
			"http.status_code": 500,
			"account_id":       "acct-1",
			"workspace_id":     "ws-1",
			"request_id":       "req-1",
			"correlation_id":   "corr-1",
		},
		event.NormalizedTelemetry{
			Service:    "api",
			Env:        "local",
			Version:    "dev",
			UserID:     "user-1",
			CaseID:     "case-1",
			StatusCode: 500,
		},
	)

	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d: %#v", len(entries), entries)
	}
	entry := entries[0]
	if entry.TraceID != "trace-1" || entry.SpanID != "span-1" || entry.Route != "/api/login" || entry.Method != "POST" || entry.StatusCode != 500 {
		t.Fatalf("missing structured request fields: %#v", entry)
	}
	if entry.Service != "api" || entry.Env != "local" || entry.Version != "dev" {
		t.Fatalf("missing service tags: %#v", entry)
	}
	if entry.UserID != "user-1" || entry.AccountID != "acct-1" || entry.WorkspaceID != "ws-1" || entry.CaseID != "case-1" {
		t.Fatalf("missing context fields: %#v", entry)
	}
	if entry.RequestID != "req-1" || entry.CorrelationID != "corr-1" {
		t.Fatalf("missing correlation fields: %#v", entry)
	}
}
