package intake

import (
	"bytes"
	"compress/zlib"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/midagedev/dogtap/internal/event"
)

func TestNormalizeRUMContext(t *testing.T) {
	n := Normalize(event.SourceRUM, map[string]any{
		"service": "web",
		"env":     "local",
		"usr": map[string]any{
			"id": "user-1",
		},
		"context": map[string]any{
			"account":   map[string]any{"id": "account-1"},
			"workspace": map[string]any{"id": "workspace-1"},
			"case":      map[string]any{"id": "case-1"},
		},
	})
	if n.UserID != "user-1" || n.AccountID != "account-1" || n.WorkspaceID != "workspace-1" || n.CaseID != "case-1" {
		t.Fatalf("unexpected normalized context: %#v", n)
	}
}

func TestNormalizeRUMBatchTraceContextFromResourceEvent(t *testing.T) {
	n := Normalize(event.SourceRUM, []any{
		map[string]any{
			"type":    "view",
			"service": "web-frontend",
			"env":     "local",
			"session": map[string]any{"id": "session-1"},
			"view":    map[string]any{"url_path": "/cloud"},
		},
		map[string]any{
			"type": "resource",
			"resource": map[string]any{
				"method":      "POST",
				"status_code": 0,
				"url":         "https://www.google-analytics.com/g/collect",
			},
		},
		map[string]any{
			"type": "resource",
			"_dd": map[string]any{
				"trace_id": "123456789",
				"span_id":  "987654321",
			},
			"resource": map[string]any{
				"method":      "GET",
				"status_code": 404,
				"url":         "https://localhost:8080/api/cloud/__dogtap-log-probe?debug=true",
			},
		},
	})

	if n.TraceID != "123456789" || n.SpanID != "987654321" {
		t.Fatalf("unexpected RUM trace normalization: %#v", n)
	}
	if n.Route != "/api/cloud/__dogtap-log-probe" || n.Method != "GET" || n.StatusCode != 404 {
		t.Fatalf("unexpected RUM resource normalization: %#v", n)
	}
	if n.Service != "web-frontend" || n.Env != "local" || n.SessionID != "session-1" {
		t.Fatalf("unexpected RUM batch context normalization: %#v", n)
	}
}

func TestCaptureFaroPayloads(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantKind    string
		wantMetric  string
		wantMessage string
	}{
		{
			name:     "event",
			wantKind: "event",
			body: `{
				"meta": {
					"app": {"name": "faro-smoke-frontend", "version": "dev", "environment": "local"},
					"user": {
						"id": "faro-user-1",
						"attributes": {
							"accountId": "faro-account-1",
							"workspaceId": "faro-workspace-1",
							"caseId": "faro-case-1"
						}
					},
					"session": {"id": "faro-session-1"},
					"page": {"url": "http://localhost/faro"}
				},
				"events": [{
					"name": "faro.workflow.run",
					"attributes": {"route": "/faro", "caseId": "faro-case-1"},
					"timestamp": "2026-05-09T12:00:00Z"
				}]
			}`,
		},
		{
			name:       "metric",
			wantKind:   "metric",
			wantMetric: "faro.workflow.duration",
			body: `{
				"meta": {
					"app": {"name": "faro-smoke-frontend", "version": "dev", "environment": "local"},
					"user": {
						"id": "faro-user-1",
						"attributes": {
							"accountId": "faro-account-1",
							"workspaceId": "faro-workspace-1",
							"caseId": "faro-case-1"
						}
					},
					"session": {"id": "faro-session-1"},
					"page": {"url": "http://localhost/faro"}
				},
				"measurements": [{
					"type": "faro.workflow.duration",
					"values": {"duration": 42.5},
					"timestamp": "2026-05-09T12:00:01Z",
					"context": {"route": "/faro"}
				}]
			}`,
		},
		{
			name:        "log",
			wantKind:    "log",
			wantMessage: "Faro workflow log",
			body: `{
				"meta": {
					"app": {"name": "faro-smoke-frontend", "version": "dev", "environment": "local"},
					"user": {
						"id": "faro-user-1",
						"attributes": {
							"accountId": "faro-account-1",
							"workspaceId": "faro-workspace-1",
							"caseId": "faro-case-1"
						}
					},
					"session": {"id": "faro-session-1"},
					"page": {"url": "http://localhost/faro"}
				},
				"logs": [{
					"message": "Faro workflow log",
					"level": "info",
					"context": {"route": "/faro"},
					"timestamp": "2026-05-09T12:00:02Z"
				}]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/collect/faro-smoke", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			result, err := CaptureRequest(req, CaptureOptions{
				Source:           event.SourceFaro,
				AllowRawPayloads: false,
				MaxBodyBytes:     1 << 20,
				ForwardMode:      "local",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Event.PayloadKind != tt.wantKind {
				t.Fatalf("payload kind = %q, want %q", result.Event.PayloadKind, tt.wantKind)
			}
			n := result.Event.Normalized
			if n.Service != "faro-smoke-frontend" || n.Env != "local" || n.Version != "dev" || n.UserID != "faro-user-1" || n.AccountID != "faro-account-1" || n.WorkspaceID != "faro-workspace-1" || n.CaseID != "faro-case-1" || n.Route != "/faro" || n.SessionID != "faro-session-1" {
				t.Fatalf("unexpected Faro normalization: %#v", n)
			}
			if tt.wantMetric != "" {
				if result.Event.Details == nil || len(result.Event.Details.Metrics) == 0 || result.Event.Details.Metrics[0].Name != tt.wantMetric || result.Event.Details.Metrics[0].Value != 42.5 {
					t.Fatalf("unexpected Faro metric details: %#v", result.Event.Details)
				}
			}
			if tt.wantMessage != "" {
				if result.Event.Details == nil || len(result.Event.Details.Logs) == 0 || result.Event.Details.Logs[0].Message != tt.wantMessage {
					t.Fatalf("unexpected Faro log details: %#v", result.Event.Details)
				}
			}
		})
	}
}

func TestNormalizeFaroTraceContext(t *testing.T) {
	decoded := map[string]any{
		"meta": map[string]any{
			"app":     map[string]any{"environment": "local", "version": "dev"},
			"session": map[string]any{"id": "faro-session-1"},
		},
		"traces": map[string]any{
			"resourceSpans": []any{
				map[string]any{
					"resource": map[string]any{
						"attributes": []any{
							attribute("service.name", "faro-trace-frontend"),
						},
					},
					"scopeSpans": []any{
						map[string]any{
							"spans": []any{
								map[string]any{
									"traceId":      "trace-1",
									"spanId":       "span-1",
									"parentSpanId": "parent-1",
									"attributes": []any{
										attribute("http.request.method", "GET"),
										attribute("http.response.status_code", 202),
										attribute("url.path", "/api/faro"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	n := Normalize(event.SourceFaro, decoded)
	if n.Service != "faro-trace-frontend" || n.TraceID != "trace-1" || n.SpanID != "span-1" || n.ParentSpanID != "parent-1" {
		t.Fatalf("unexpected Faro trace normalization: %#v", n)
	}
	if n.Method != "GET" || n.StatusCode != 202 || n.Route != "/api/faro" || n.SessionID != "faro-session-1" {
		t.Fatalf("unexpected Faro HTTP/session normalization: %#v", n)
	}
}

func TestCaptureRUMTextBatchAndProxyTags(t *testing.T) {
	body := bytes.NewBufferString(`{"service":"web-frontend","version":"g1-fixture","usr":{"id":"user-1"},"context":{"account":{"id":"account-1"},"workspace":{"id":"workspace-1"}}}` + "\n" +
		`{"service":"web-frontend","version":"g1-fixture","usr":{"id":"user-1"},"context":{"account":{"id":"account-1"},"workspace":{"id":"workspace-1"}}}`)
	req := httptest.NewRequest(
		http.MethodPost,
		"/datadog-intake-proxy?ddforward=%2Fapi%2Fv2%2Frum%3Fddtags%3Denv%253Alocal%252Cservice%253Aweb-frontend%252Cversion%253Ag1-fixture%26dd-api-key%3Dfixture",
		body,
	)
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")

	result, err := CaptureRequest(req, CaptureOptions{
		Source:           event.SourceRUM,
		AllowRawPayloads: false,
		MaxBodyBytes:     1 << 20,
		ForwardMode:      "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Event.Decoded.([]any); !ok {
		t.Fatalf("RUM text batch should decode as JSON array, got %#v", result.Event.Decoded)
	}
	if result.Event.Normalized.Env != "local" || result.Event.Normalized.Service != "web-frontend" || result.Event.Normalized.Version != "g1-fixture" {
		t.Fatalf("unexpected request tag normalization: %#v", result.Event.Normalized)
	}
	if result.ValidationEvent.RawBody == "" {
		t.Fatalf("validation event should retain decoded body for policy checks")
	}
	if result.Event.RawBody != "" {
		t.Fatalf("stored event should not retain raw body when raw payloads are disabled")
	}
}

func TestCaptureRUMSessionReplayPayload(t *testing.T) {
	body := bytes.NewBufferString(`{"records":[{"type":4,"timestamp":1000,"data":{"href":"http://localhost/cloud/"}},{"type":2,"timestamp":1100,"data":{"node":{"type":0}}}]}`)
	req := httptest.NewRequest(
		http.MethodPost,
		"/datadog-intake-proxy?ddforward=%2Fapi%2Fv2%2Freplay%3Fddtags%3Denv%253Alocal%252Cservice%253Aweb-frontend%252Cversion%253Alocal%26dd-api-key%3Dfixture",
		body,
	)
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")

	result, err := CaptureRequest(req, CaptureOptions{
		Source:           event.SourceRUM,
		AllowRawPayloads: false,
		MaxBodyBytes:     1 << 20,
		ForwardMode:      "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Event.PayloadKind != "replay" {
		t.Fatalf("payload kind = %q, want replay", result.Event.PayloadKind)
	}
	if result.Event.Normalized.Env != "local" || result.Event.Normalized.Service != "web-frontend" {
		t.Fatalf("unexpected request tag normalization: %#v", result.Event.Normalized)
	}
	decoded, ok := result.Event.Decoded.(map[string]any)
	if !ok {
		t.Fatalf("decoded = %#v, want object", result.Event.Decoded)
	}
	replay, ok := decoded["replay"].(map[string]any)
	if !ok {
		t.Fatalf("missing replay summary: %#v", decoded)
	}
	if replay["format"] != "json" || replay["frames"] != 2 {
		t.Fatalf("unexpected replay summary: %#v", replay)
	}
	if result.Event.Details == nil || result.Event.Details.Replay == nil || result.Event.Details.Replay.RecordCount != 2 {
		t.Fatalf("missing replay detail: %#v", result.Event.Details)
	}
}

func TestCaptureRUMSessionReplayMultipartPayload(t *testing.T) {
	var segment bytes.Buffer
	zw := zlib.NewWriter(&segment)
	if _, err := zw.Write([]byte(`[{"type":4,"timestamp":1000,"data":{"href":"http://localhost/cloud/"}},{"type":3,"timestamp":1050,"data":{"source":2,"text":"click"}}]`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipartWriter(t, &body)
	eventPart := mustCreatePart(t, writer, "event", "", "application/json")
	if _, err := eventPart.Write([]byte(`{"session":{"id":"session-1"},"view":{"id":"view-1"},"start":"1000","end":"1050","records_count":2,"raw_segment_size":180,"compressed_segment_size":120}`)); err != nil {
		t.Fatal(err)
	}
	segmentPart := mustCreatePart(t, writer, "segment", "session-1-1000", "application/octet-stream")
	if _, err := segmentPart.Write(segment.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/replay", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	result, err := CaptureRequest(req, CaptureOptions{
		Source:           event.SourceRUM,
		AllowRawPayloads: false,
		MaxBodyBytes:     1 << 20,
		ForwardMode:      "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Event.PayloadKind != "replay" {
		t.Fatalf("payload kind = %q, want replay", result.Event.PayloadKind)
	}
	if result.Event.Normalized.SessionID != "session-1" || result.Event.Normalized.ViewID != "view-1" {
		t.Fatalf("unexpected replay normalization: %#v", result.Event.Normalized)
	}
	if result.Event.Details == nil || result.Event.Details.Replay == nil {
		t.Fatalf("missing replay detail: %#v", result.Event.Details)
	}
	replay := result.Event.Details.Replay
	if replay.Format != "multipart" || replay.RecordCount != 2 || replay.SegmentBytes == 0 || replay.SegmentFilename != "session-1-1000" {
		t.Fatalf("unexpected replay detail: %#v", replay)
	}
	decoded, ok := result.Event.Decoded.(map[string]any)
	if !ok {
		t.Fatalf("decoded = %#v, want object", result.Event.Decoded)
	}
	if _, ok := decoded["records"].([]any); !ok {
		t.Fatalf("expected decoded replay records: %#v", decoded)
	}
}

func TestNormalizeTags(t *testing.T) {
	n := Normalize(event.SourceLogs, map[string]any{
		"ddtags":  "service:api,env:local,version:dev",
		"message": "hello",
	})
	if n.Service != "api" || n.Env != "local" || n.Version != "dev" {
		t.Fatalf("unexpected tag normalization: %#v", n)
	}
}

func attribute(key string, value any) map[string]any {
	encoded := map[string]any{}
	switch typed := value.(type) {
	case string:
		encoded["stringValue"] = typed
	case int:
		encoded["intValue"] = typed
	default:
		encoded["stringValue"] = typed
	}
	return map[string]any{
		"key":   key,
		"value": encoded,
	}
}

func multipartWriter(t *testing.T, body *bytes.Buffer) *multipart.Writer {
	t.Helper()
	return multipart.NewWriter(body)
}

func mustCreatePart(t *testing.T, writer *multipart.Writer, name string, filename string, contentType string) io.Writer {
	t.Helper()
	header := textproto.MIMEHeader{}
	disposition := `form-data; name="` + name + `"`
	if filename != "" {
		disposition += `; filename="` + filename + `"`
	}
	header.Set("Content-Disposition", disposition)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	return part
}
