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
