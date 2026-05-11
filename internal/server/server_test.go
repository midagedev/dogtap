package server

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"

	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/diagnose"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/store"
)

func TestRUMIntakeStoresEvent(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{
		"service":"web",
		"env":"local",
		"version":"dev",
		"usr":{"id":"user-1"},
		"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}},
		"view":{"url_path":"/cases/123"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/events?source=rum", nil)
	listRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(listRec, listReq)

	var events []event.EventEnvelope
	if err := json.Unmarshal(listRec.Body.Bytes(), &events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Normalized.UserID != "user-1" || events[0].Validation.Status != "pass" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
	if events[0].RawBody == "" {
		t.Fatalf("local mode should retain raw body")
	}
}

func TestLogsIntakeDecodesGzip(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write([]byte(`{"message":"hello","ddtags":"service:api,env:local,version:dev"}`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs", &gz)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Normalized.Service != "api" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestUnsupportedContentTypeFails(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs", bytes.NewBufferString("not a supported log wire format"))
	req.Header.Set("Content-Type", "application/x-dogtap-unknown")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unsupported content type") {
		t.Fatalf("expected useful unsupported content type error: %s", rec.Body.String())
	}
}

func TestRUMProxySupportsBrowserCORS(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(http.MethodOptions, "/datadog-intake-proxy", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow origin = %q, want *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("allow methods = %q, want POST", got)
	}

	postReq := httptest.NewRequest(http.MethodPost, "/datadog-intake-proxy", bytes.NewBufferString(`{"service":"web","env":"local","usr":{"id":"user-1"},"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}},"view":{"url_path":"/"}}`))
	postReq.Header.Set("Origin", "http://localhost:3000")
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()

	app.Handler().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusAccepted {
		t.Fatalf("got status %d, want %d: %s", postRec.Code, http.StatusAccepted, postRec.Body.String())
	}
	if got := postRec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("post allow origin = %q, want *", got)
	}
}

func TestFaroCollectEndpointStoresSDKPayload(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{
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
	}`
	req := httptest.NewRequest(http.MethodPost, "/collect/faro-smoke", bytes.NewBufferString(body))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Faro-Session-Id", "faro-session-1")
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(strings.ToLower(got), "x-api-key") {
		t.Fatalf("allow headers = %q, want x-api-key", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(strings.ToLower(got), "x-faro-session-id") {
		t.Fatalf("allow headers = %q, want x-faro-session-id", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(strings.ToLower(got), "x-faro-session-status") {
		t.Fatalf("expose headers = %q, want x-faro-session-status", got)
	}
	events, err := app.store.List(req.Context(), store.Query{Source: event.SourceFaro})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d Faro events, want 1", len(events))
	}
	if events[0].PayloadKind != "event" || events[0].Normalized.Service != "faro-smoke-frontend" || events[0].Normalized.UserID != "faro-user-1" || events[0].Validation.Status != "pass" {
		t.Fatalf("unexpected Faro event: %#v", events[0])
	}

	exactReq := httptest.NewRequest(http.MethodOptions, "/collect", nil)
	exactReq.Header.Set("Origin", "http://localhost:3000")
	exactReq.Header.Set("Access-Control-Request-Method", "POST")
	exactReq.Header.Set("Access-Control-Request-Headers", "content-type,x-faro-session-id")
	exactRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(exactRec, exactReq)
	if exactRec.Code != http.StatusNoContent {
		t.Fatalf("exact /collect preflight status = %d, want %d", exactRec.Code, http.StatusNoContent)
	}
}

func TestRUMReplayProxyStoresReplayWithoutRequiredRUMContext(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{"records":[{"type":4,"timestamp":1000,"data":{"href":"http://localhost/cloud/"}},{"type":2,"timestamp":1100,"data":{"node":{"type":0}}}]}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/datadog-intake-proxy?ddforward=%2Fapi%2Fv2%2Freplay%3Fddtags%3Denv%253Alocal%252Cservice%253Aweb-frontend%252Cversion%253Alocal",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].PayloadKind != "replay" {
		t.Fatalf("payload kind = %q, want replay", events[0].PayloadKind)
	}
	if events[0].Validation.Status != "pass" {
		t.Fatalf("replay payload should not fail RUM required-context rules: %#v", events[0].Validation)
	}
}

func TestRUMReplayDirectEndpointStoresReplayDetails(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{"records":[{"type":4,"timestamp":1000,"data":{"href":"http://localhost/cloud/"}},{"type":3,"timestamp":1100,"data":{"source":2,"text":"click"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/replay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{PayloadKind: "replay"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d replay events, want 1", len(events))
	}
	if events[0].Details == nil || events[0].Details.Replay == nil || events[0].Details.Replay.RecordCount != 2 {
		t.Fatalf("missing replay details: %#v", events[0].Details)
	}
}

func TestOTLPMetricsStoresMetricDetails(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{
		"resourceMetrics": [{
			"resource": {
				"attributes": [
					{"key":"service.name","value":{"stringValue":"api-service"}},
					{"key":"deployment.environment","value":{"stringValue":"local"}},
					{"key":"service.version","value":{"stringValue":"e2e"}}
				]
			},
			"scopeMetrics": [{
				"metrics": [{
					"name": "http.server.request.duration",
					"unit": "ms",
					"gauge": {
						"dataPoints": [{
							"asDouble": 42.5,
							"timeUnixNano": "1778206500000000000",
							"attributes": [
								{"key":"http.route","value":{"stringValue":"/api/cloud/cases"}},
								{"key":"http.response.status_code","value":{"intValue":"200"}}
							]
						}]
					}
				}]
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{PayloadKind: "metric"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d metric events, want 1", len(events))
	}
	if events[0].Normalized.Service != "api-service" || events[0].Normalized.Route != "/api/cloud/cases" || events[0].Normalized.StatusCode != 200 {
		t.Fatalf("unexpected normalized metric event: %#v", events[0].Normalized)
	}
	if events[0].Details == nil || len(events[0].Details.Metrics) != 1 {
		t.Fatalf("missing metric details: %#v", events[0].Details)
	}
	metric := events[0].Details.Metrics[0]
	if metric.Name != "http.server.request.duration" || metric.Value != 42.5 || metric.Route != "/api/cloud/cases" {
		t.Fatalf("unexpected metric detail: %#v", metric)
	}
}

func TestOTLPGRPCMetricsStoresMetricDetails(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := &collectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricsv1.ResourceMetrics{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						otelStringAttribute("service.name", "api-service"),
						otelStringAttribute("deployment.environment", "local"),
						otelStringAttribute("service.version", "e2e"),
					},
				},
				ScopeMetrics: []*metricsv1.ScopeMetrics{
					{
						Metrics: []*metricsv1.Metric{
							{
								Name: "http.server.request.duration",
								Unit: "ms",
								Data: &metricsv1.Metric_Gauge{
									Gauge: &metricsv1.Gauge{
										DataPoints: []*metricsv1.NumberDataPoint{
											{
												Attributes: []*commonv1.KeyValue{
													otelStringAttribute("http.route", "/api/cloud/cases"),
													otelIntAttribute("http.response.status_code", 200),
												},
												Value: &metricsv1.NumberDataPoint_AsDouble{AsDouble: 42.5},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := app.ingestGRPC(context.Background(), "otlp-grpc-metrics", req); err != nil {
		t.Fatal(err)
	}
	events, err := app.store.List(context.Background(), store.Query{PayloadKind: "metric"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d metric events, want 1", len(events))
	}
	if events[0].PayloadKind != "metric" || events[0].Details == nil || len(events[0].Details.Metrics) != 1 {
		t.Fatalf("missing grpc metric details: %#v", events[0])
	}
	if events[0].Normalized.Service != "api-service" || events[0].Normalized.Route != "/api/cloud/cases" {
		t.Fatalf("unexpected normalized grpc metric event: %#v", events[0].Normalized)
	}
}

func TestDatadogLogsSearchCompatibilityReturnsRetainedLogs(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/logs",
		bytes.NewBufferString(`{"message":"login failed","ddtags":"service:api,env:local,version:dev","trace_id":"trace-1","span_id":"span-1","status":"error"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	searchReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/logs/events/search",
		bytes.NewBufferString(`{"filter":{"query":"service:api @trace_id:trace-1 login"},"page":{"limit":5},"sort":"-timestamp"}`),
	)
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var got struct {
		Data []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Message    string         `json:"message"`
				Service    string         `json:"service"`
				Attributes map[string]any `json:"attributes"`
			} `json:"attributes"`
		} `json:"data"`
		Meta struct {
			Status string `json:"status"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Meta.Status != "done" || len(got.Data) != 1 {
		t.Fatalf("unexpected search response: %#v", got)
	}
	if got.Data[0].Type != "log" || got.Data[0].Attributes.Service != "api" || got.Data[0].Attributes.Message != "login failed" {
		t.Fatalf("unexpected log event: %#v", got.Data[0])
	}
	if got.Data[0].Attributes.Attributes["trace_id"] != "trace-1" {
		t.Fatalf("missing trace id in Datadog-compatible attributes: %#v", got.Data[0].Attributes.Attributes)
	}
}

func TestDatadogLogsSearchMatchesStructuredLogFields(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/logs",
		bytes.NewBufferString(`{
			"message":"login failed",
			"status":"error",
			"service":"api",
			"env":"local",
			"version":"dev",
			"trace_id":"trace-1",
			"span_id":"span-1",
			"route":"/api/login",
			"http.method":"POST",
			"http.status_code":500,
			"request_id":"req-1",
			"correlation_id":"corr-1"
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	searchReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/logs/events/search",
		bytes.NewBufferString(`{"filter":{"query":"service:api env:local @http.status_code:500 @http.method:POST @endpoint:/api/v2/logs @request_id:req-1 @correlation_id:corr-1 @payload_kind:log @validation.status:pass"},"page":{"limit":5}}`),
	)
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var got struct {
		Data []struct {
			Attributes struct {
				Attributes map[string]any `json:"attributes"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 {
		t.Fatalf("unexpected structured log search response: %#v", got)
	}
	attrs := got.Data[0].Attributes.Attributes
	if attrs["http.method"] != "POST" || attrs["request_id"] != "req-1" || attrs["correlation_id"] != "corr-1" {
		t.Fatalf("missing structured attributes: %#v", attrs)
	}
	if attrs["http.status_code"].(float64) != 500 {
		t.Fatalf("missing http status code: %#v", attrs)
	}
	if _, exists := attrs["raw_body"]; exists {
		t.Fatalf("Datadog-compatible log attributes should not expose raw body: %#v", attrs)
	}
}

func TestDatadogLogsSearchMatchesQuotedPhraseAndPathValues(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/logs",
		bytes.NewBufferString(`{
			"message":"billing subscription upgrade confirm",
			"status":"info",
			"service":"api",
			"env":"local",
			"version":"dev",
			"trace_id":"trace-quoted-1",
			"route":"/account/v2/workspace-plan-subscriptions/upgrade/confirm",
			"http.method":"POST",
			"http.status_code":200
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	searchReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/logs/events/search",
		bytes.NewBufferString(`{"filter":{"query":"service:api @route:\"/account/v2/workspace-plan-subscriptions/upgrade/confirm\" \"billing subscription upgrade confirm\""},"page":{"limit":5}}`),
	)
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var got struct {
		Data []struct {
			Attributes struct {
				Message    string         `json:"message"`
				Attributes map[string]any `json:"attributes"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 {
		t.Fatalf("expected quoted route and phrase query to match, got %#v", got)
	}
	if got.Data[0].Attributes.Message != "billing subscription upgrade confirm" {
		t.Fatalf("unexpected message: %#v", got.Data[0].Attributes)
	}
	if got.Data[0].Attributes.Attributes["route"] != "/account/v2/workspace-plan-subscriptions/upgrade/confirm" {
		t.Fatalf("missing quoted route match attributes: %#v", got.Data[0].Attributes.Attributes)
	}
}

func TestDatadogRUMSearchCompatibilityReturnsRetainedRUM(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{
		"service":"web",
		"env":"local",
		"version":"dev",
		"session":{"id":"session-1"},
		"view":{"id":"view-1","url_path":"/login"},
		"usr":{"id":"user-1"},
		"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}}
	}`))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	searchReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/rum/events/search",
		bytes.NewBufferString(`{"filter":{"query":"service:web @session.id:session-1 @usr.id:user-1"},"page":{"limit":5}}`),
	)
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var got struct {
		Data []struct {
			Type       string `json:"type"`
			Attributes struct {
				Service string `json:"service"`
				Session struct {
					ID string `json:"id"`
				} `json:"session"`
				Usr struct {
					ID string `json:"id"`
				} `json:"usr"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0].Type != "rum" {
		t.Fatalf("unexpected rum response: %#v", got)
	}
	if got.Data[0].Attributes.Service != "web" || got.Data[0].Attributes.Session.ID != "session-1" || got.Data[0].Attributes.Usr.ID != "user-1" {
		t.Fatalf("unexpected rum attributes: %#v", got.Data[0].Attributes)
	}
}

func TestDatadogSpansSearchCompatibilityReturnsRetainedSpans(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0.5/traces",
		bytes.NewBufferString(`[[{"trace_id":"trace-1","span_id":"span-1","parent_id":"0","service":"api","name":"web.request","resource":"GET /login","duration":1000000}]]`),
	)
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	searchReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/spans/events/search",
		bytes.NewBufferString(`{"data":{"attributes":{"filter":{"query":"service:api trace_id:trace-1"},"page":{"limit":5},"sort":"-timestamp"}}}`),
	)
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var got struct {
		Data []struct {
			Type       string `json:"type"`
			Attributes struct {
				Service      string `json:"service"`
				TraceID      string `json:"trace_id"`
				SpanID       string `json:"span_id"`
				ResourceName string `json:"resource_name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0].Type != "span" {
		t.Fatalf("unexpected spans response: %#v", got)
	}
	if got.Data[0].Attributes.Service != "api" || got.Data[0].Attributes.TraceID != "trace-1" || got.Data[0].Attributes.SpanID != "span-1" || got.Data[0].Attributes.ResourceName != "GET /login" {
		t.Fatalf("unexpected span attributes: %#v", got.Data[0].Attributes)
	}
}

func TestDatadogSpansSearchMatchesTraceIDAlias(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0.5/traces",
		bytes.NewBufferString(`[[{"trace_id":"0000000000000000000000000000007b","span_id":"span-1","service":"api","name":"web.request","resource":"GET /login"}]]`),
	)
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	searchReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/spans/events/search",
		bytes.NewBufferString(`{"data":{"attributes":{"filter":{"query":"service:api trace_id:123"},"page":{"limit":5}}}}`),
	)
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var got struct {
		Data []struct {
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0].Type != "span" {
		t.Fatalf("expected trace-id alias match, got %#v", got)
	}
}

func TestDatadogMetricQueryCompatibilityReturnsTimeseries(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{
		"resourceMetrics": [{
			"resource": {"attributes": [
				{"key":"service.name","value":{"stringValue":"api-service"}},
				{"key":"deployment.environment","value":{"stringValue":"local"}},
				{"key":"service.version","value":{"stringValue":"e2e"}}
			]},
			"scopeMetrics": [{
				"metrics": [{
					"name": "http.server.request.duration",
					"unit": "ms",
					"gauge": {"dataPoints": [{
						"asDouble": 42.5,
						"timeUnixNano": "1778206500000000000",
						"attributes": [
							{"key":"http.route","value":{"stringValue":"/login"}},
							{"key":"http.request.method","value":{"stringValue":"POST"}},
							{"key":"http.response.status_code","value":{"intValue":"200"}}
						]
					}]}
				}]
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	queryReq := httptest.NewRequest(http.MethodGet, "/api/v1/query?from=0&to=9999999999&query=avg:http.server.request.duration%7Bhttp.route:%2Flogin,http.request.method:POST,http.response.status_code:200%7D", nil)
	queryRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(queryRec, queryReq)

	if queryRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", queryRec.Code, queryRec.Body.String())
	}
	var got struct {
		Status string `json:"status"`
		Series []struct {
			Metric    string   `json:"metric"`
			Scope     string   `json:"scope"`
			Pointlist [][]any  `json:"pointlist"`
			EventIDs  []string `json:"dogtap_event_ids"`
		} `json:"series"`
	}
	if err := json.Unmarshal(queryRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "ok" || len(got.Series) != 1 {
		t.Fatalf("unexpected metric query response: %#v", got)
	}
	if got.Series[0].Metric != "http.server.request.duration" || !strings.Contains(got.Series[0].Scope, "service:api-service") || len(got.Series[0].Pointlist) != 1 {
		t.Fatalf("unexpected metric series: %#v", got.Series[0])
	}
	if !strings.Contains(got.Series[0].Scope, "http.request.method:POST") || !strings.Contains(got.Series[0].Scope, "http.response.status_code:200") {
		t.Fatalf("metric scope did not retain OTLP point tags: %#v", got.Series[0])
	}
	if len(got.Series[0].EventIDs) != 1 {
		t.Fatalf("missing dogtap event ids: %#v", got.Series[0])
	}
	if got.Series[0].Pointlist[0][1].(float64) != 42.5 {
		t.Fatalf("unexpected metric value: %#v", got.Series[0].Pointlist)
	}

	quotedScopeReq := httptest.NewRequest(http.MethodGet, "/api/v1/query?from=0&to=9999999999&query=avg:http.server.request.duration%7Bhttp.route:%22%2Flogin%22,http.request.method:POST,http.response.status_code:200%7D", nil)
	quotedScopeRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(quotedScopeRec, quotedScopeReq)
	if quotedScopeRec.Code != http.StatusOK {
		t.Fatalf("quoted scope status %d: %s", quotedScopeRec.Code, quotedScopeRec.Body.String())
	}
	var quotedScopeGot struct {
		Status string `json:"status"`
		Series []struct {
			Metric    string  `json:"metric"`
			Pointlist [][]any `json:"pointlist"`
		} `json:"series"`
	}
	if err := json.Unmarshal(quotedScopeRec.Body.Bytes(), &quotedScopeGot); err != nil {
		t.Fatal(err)
	}
	if quotedScopeGot.Status != "ok" || len(quotedScopeGot.Series) != 1 || len(quotedScopeGot.Series[0].Pointlist) != 1 {
		t.Fatalf("expected quoted metric scope to match, got %#v", quotedScopeGot)
	}
}

func otelStringAttribute(key string, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		},
	}
}

func otelIntAttribute(key string, value int64) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_IntValue{IntValue: value},
		},
	}
}

func TestForwardModeDoesNotStoreRawBodyByDefault(t *testing.T) {
	app := newTestApp(t, config.ModeForward)
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"local","usr":{"id":"u"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if events[0].RawBody != "" {
		t.Fatalf("forward mode should not store raw body")
	}
}

func TestForwardingStoresResultWithoutRawBody(t *testing.T) {
	var hits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.Header.Get("DD-API-KEY"); got != "test-api-key" {
			t.Fatalf("forwarded DD-API-KEY = %q, want test-api-key", got)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()

	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Forwarding.Enabled = true
	cfg.Forwarding.APIKey = "test-api-key"
	cfg.Forwarding.TargetBaseURL = upstream.URL
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs", bytes.NewBufferString(`{"message":"hello","ddtags":"service:api,env:local"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}

	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].RawBody != "" {
		t.Fatalf("forward mode should not store raw body")
	}
	if events[0].Forwarding.Status != "success" || events[0].Forwarding.Target == "" {
		t.Fatalf("unexpected forwarding result: %#v", events[0].Forwarding)
	}
	if strings.Contains(mustJSON(t, events[0].Forwarding), "test-api-key") {
		t.Fatalf("forwarding result leaked API key: %#v", events[0].Forwarding)
	}
	if hits.Load() != 1 {
		t.Fatalf("got %d upstream hits, want 1", hits.Load())
	}
}

func TestForwardingUnsupportedSourceIsRecorded(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Forwarding.Enabled = true
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/v0.5/traces", bytes.NewBufferString(`[[{"service":"api","env":"local","version":"dev"}]]`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if events[0].Forwarding.Status != "unsupported" {
		t.Fatalf("unexpected forwarding result: %#v", events[0].Forwarding)
	}
}

func TestDebugBundleIncludesFailuresAndQueries(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{"service":"web","env":"local","version":"dev","usr":{"id":"user-1"},"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"},"case":{"id":"case-1"}},"view":{"url_path":"/cases/case-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	failReq := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"local"}`))
	failReq.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), failReq)

	bundleReq := httptest.NewRequest(http.MethodPost, "/api/debug-bundles", bytes.NewBufferString(`{"service":"web"}`))
	bundleRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(bundleRec, bundleReq)

	if bundleRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", bundleRec.Code, bundleRec.Body.String())
	}
	var got struct {
		Summary struct {
			Total  int `json:"total"`
			Failed int `json:"failed"`
		} `json:"summary"`
		ValidationFailures []struct {
			RuleID string `json:"ruleId"`
		} `json:"validationFailures"`
		DatadogQueries []struct {
			Label string `json:"label"`
			Query string `json:"query"`
		} `json:"datadogQueries"`
	}
	if err := json.Unmarshal(bundleRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary.Total != 2 || got.Summary.Failed != 1 {
		t.Fatalf("unexpected bundle summary: %#v", got.Summary)
	}
	if len(got.ValidationFailures) == 0 {
		t.Fatalf("expected validation failures in debug bundle")
	}
	if len(got.DatadogQueries) == 0 {
		t.Fatalf("expected datadog query hints in debug bundle")
	}
}

func TestDebugBundleSupportsSessionAndPayloadKindFilters(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	for _, body := range []string{
		`{"service":"web","env":"local","session":{"id":"session-1"},"usr":{"id":"user-1"},"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}}}`,
		`{"service":"web","env":"local","session":{"id":"session-2"},"usr":{"id":"user-2"},"context":{"account":{"id":"acct-2"},"workspace":{"id":"ws-2"}}}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Handler().ServeHTTP(httptest.NewRecorder(), req)
	}

	bundleReq := httptest.NewRequest(
		http.MethodPost,
		"/api/debug-bundles",
		bytes.NewBufferString(`{"source":"rum","payloadKind":"rum","sessionId":"session-1"}`),
	)
	bundleRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(bundleRec, bundleReq)

	if bundleRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", bundleRec.Code, bundleRec.Body.String())
	}
	var got struct {
		Summary struct {
			Total int `json:"total"`
		} `json:"summary"`
		Events []event.EventEnvelope `json:"events"`
	}
	if err := json.Unmarshal(bundleRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary.Total != 1 || len(got.Events) != 1 {
		t.Fatalf("unexpected filtered bundle: %#v", got.Summary)
	}
	if got.Events[0].Normalized.SessionID != "session-1" {
		t.Fatalf("unexpected session in bundle: %#v", got.Events[0].Normalized)
	}
}

func TestDiagnosticsAPIReturnsScopedAssertions(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{
		"service":"web",
		"env":"local",
		"version":"dev",
		"usr":{"id":"user-1"},
		"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"},"case":{"id":"case-1"}},
		"view":{"url_path":"/cases/case-1"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	diagReq := httptest.NewRequest(http.MethodPost, "/api/diagnostics", bytes.NewBufferString(`{
		"limit": 50,
		"filter": {"service":"web"},
		"expect": {
			"nonEmpty": true,
			"sources": ["rum"],
			"services": ["web"],
			"routes": ["/cases/case-1"],
			"cases": ["case-1"],
			"endpoints": ["/rum"]
		}
	}`))
	diagReq.Header.Set("Content-Type", "application/json")
	diagRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(diagRec, diagReq)

	if diagRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", diagRec.Code, diagRec.Body.String())
	}
	var got diagnose.Snapshot
	if err := json.Unmarshal(diagRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Assertions.Status != "pass" {
		t.Fatalf("diagnostics assertions = %s: %#v", got.Assertions.Status, got.Assertions.Checks)
	}
	if len(got.Events) != 1 || got.DebugBundle.Summary.Total != 1 {
		t.Fatalf("expected scoped diagnostics event and bundle: events=%d bundle=%#v", len(got.Events), got.DebugBundle.Summary)
	}
	if !strings.Contains(got.Metrics, "dogtap_store_events 1") {
		t.Fatalf("expected metrics in diagnostics response:\n%s", got.Metrics)
	}
}

func TestDiagnosticsAPIReturnsWorkflowContractsWithoutChangingAssertions(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	body := `{
		"service":"web",
		"env":"local",
		"version":"dev",
		"usr":{"id":"user-1"},
		"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}},
		"view":{"url_path":"/login"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	diagReq := httptest.NewRequest(http.MethodPost, "/api/diagnostics", bytes.NewBufferString(`{
		"expect": {"nonEmpty": true, "sources": ["rum"]},
		"workflowContract": {
			"name": "login-workflow",
			"checks": [{
				"id": "login-rum-user",
				"type": "event",
				"source": "rum",
				"fields": ["userId"]
			}]
		}
	}`))
	diagReq.Header.Set("Content-Type", "application/json")
	diagRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(diagRec, diagReq)

	if diagRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", diagRec.Code, diagRec.Body.String())
	}
	var got diagnose.Snapshot
	if err := json.Unmarshal(diagRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Assertions.Status != "pass" {
		t.Fatalf("diagnostics assertions = %s: %#v", got.Assertions.Status, got.Assertions.Checks)
	}
	if len(got.WorkflowContracts) != 1 || got.WorkflowContracts[0].Status != "pass" {
		t.Fatalf("unexpected workflow contract result: %#v", got.WorkflowContracts)
	}
}

func TestDiagnosticsArchiveIncludesWorkflowContractWhenSupplied(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{
		"service":"web",
		"env":"local",
		"usr":{"id":"user-1"},
		"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}}
	}`))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	archiveReq := httptest.NewRequest(http.MethodPost, "/api/diagnostics/archive", bytes.NewBufferString(`{
		"workflowContract": {
			"name": "login-workflow",
			"checks": [{
				"id": "backend-log",
				"type": "log-message",
				"source": "logs"
			}]
		}
	}`))
	archiveReq.Header.Set("Content-Type", "application/json")
	archiveRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(archiveRec, archiveReq)

	if archiveRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", archiveRec.Code, archiveRec.Body.String())
	}
	reader, err := zip.NewReader(bytes.NewReader(archiveRec.Body.Bytes()), int64(archiveRec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = string(body)
	}
	if !strings.Contains(files["workflow-contracts.json"], `"status": "fail"`) {
		t.Fatalf("archive missing failing workflow contract:\n%s", files["workflow-contracts.json"])
	}
	if !strings.Contains(files["summary.md"], "## Workflow Contracts") {
		t.Fatalf("summary missing workflow contract section:\n%s", files["summary.md"])
	}
}

func TestDiagnosticsArchiveReturnsAgentReadableFiles(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{
		"service":"web",
		"env":"local",
		"usr":{"id":"user-1"},
		"context":{"account":{"id":"acct-1"},"workspace":{"id":"ws-1"}}
	}`))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	archiveReq := httptest.NewRequest(http.MethodPost, "/api/diagnostics/archive", bytes.NewBufferString(`{
		"expect": {"nonEmpty": true, "sources": ["rum"], "services": ["web"]}
	}`))
	archiveReq.Header.Set("Content-Type", "application/json")
	archiveRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(archiveRec, archiveReq)

	if archiveRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", archiveRec.Code, archiveRec.Body.String())
	}
	if got := archiveRec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("content type = %q, want application/zip", got)
	}

	reader, err := zip.NewReader(bytes.NewReader(archiveRec.Body.Bytes()), int64(archiveRec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = string(body)
	}
	for _, name := range []string{
		"healthz.json",
		"readyz.json",
		"events.json",
		"report.json",
		"debug-bundle.json",
		"metrics.txt",
		"assertions.json",
		"summary.md",
		"manifest.json",
	} {
		if _, ok := files[name]; !ok {
			t.Fatalf("archive missing %s; got %v", name, files)
		}
	}
	if !strings.Contains(files["summary.md"], "source:rum") || !strings.Contains(files["assertions.json"], `"status": "pass"`) {
		t.Fatalf("archive diagnostics missing expected assertion evidence:\n%s\n%s", files["summary.md"], files["assertions.json"])
	}
}

func TestMetricsExposeRetainedEvents(t *testing.T) {
	app := newTestApp(t, config.ModeLocal)
	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"local"}`))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(httptest.NewRecorder(), req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", metricsRec.Code, metricsRec.Body.String())
	}
	body := metricsRec.Body.String()
	for _, want := range []string{
		"dogtap_store_events 1",
		`dogtap_events_by_source{source="rum"} 1`,
		`dogtap_events_by_validation{status="fail"} 1`,
		"dogtap_validation_failures 1",
		"dogtap_intake_accepted_total 1",
		"dogtap_forwarding_payloads_total 0",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, body)
		}
	}
}

func TestSamplingDropsWithoutPersistenceOrForwarding(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeTee
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	samplingRate := 0.0
	cfg.Safety.SamplingRate = &samplingRate
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"prod"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "sampled_out") {
		t.Fatalf("expected sampled_out response: %s", rec.Body.String())
	}
	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("sampled out payload should not be persisted: %#v", events)
	}

	metricsRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricsRec.Body.String(), "dogtap_intake_sample_drops_total 1") {
		t.Fatalf("metrics missing sample drop:\n%s", metricsRec.Body.String())
	}
}

func TestSamplingDoesNotSkipForwarding(t *testing.T) {
	var hits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()

	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Forwarding.Enabled = true
	cfg.Forwarding.APIKey = "test-api-key"
	cfg.Forwarding.TargetBaseURL = upstream.URL
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	samplingRate := 0.0
	cfg.Safety.SamplingRate = &samplingRate
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs", bytes.NewBufferString(`{"message":"hello","ddtags":"service:api,env:prod"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	if hits.Load() != 1 {
		t.Fatalf("forwarding should still run for sampled-out Dogtap copies, hits = %d", hits.Load())
	}
	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("sampled out payload should not be persisted: %#v", events)
	}
	if !strings.Contains(rec.Body.String(), `"status":"dropped"`) || !strings.Contains(rec.Body.String(), `"forwarding"`) {
		t.Fatalf("expected sampled response with forwarding metadata: %s", rec.Body.String())
	}
}

func TestProductionQueueFullDropsDogtapCopyFailOpen(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeTee
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	cfg.Safety.QueueMaxInFlight = 1
	samplingRate := 1.0
	cfg.Safety.SamplingRate = &samplingRate
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	blocking := newBlockingStore()
	app.store = blocking

	done := make(chan struct{})
	go func() {
		defer close(done)
		req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"prod"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Errorf("first request got status %d: %s", rec.Code, rec.Body.String())
		}
	}()
	<-blocking.entered

	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"prod"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("production queue-full should fail open with 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "queue_full") {
		t.Fatalf("expected queue_full response: %s", rec.Body.String())
	}

	close(blocking.release)
	<-done
	if got := blocking.count(); got != 1 {
		t.Fatalf("stored events = %d, want only first admitted event", got)
	}

	metricsRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricsRec.Body.String(), "dogtap_intake_backpressure_drops_total 1") {
		t.Fatalf("metrics missing backpressure drop:\n%s", metricsRec.Body.String())
	}
}

func TestProductionStorageFailureDropsDogtapCopyFailOpen(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	samplingRate := 1.0
	cfg.Safety.SamplingRate = &samplingRate
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	app.store = failingStore{}

	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"prod","usr":{"id":"u"},"context":{"account":{"id":"a"},"workspace":{"id":"w"}}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("production storage failure should fail open with 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "storage_error") {
		t.Fatalf("expected storage_error response: %s", rec.Body.String())
	}

	metricsRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricsRec.Body.String(), "dogtap_intake_storage_drops_total 1") {
		t.Fatalf("metrics missing storage drop:\n%s", metricsRec.Body.String())
	}
}

func TestDatadogUnavailableIsRecordedAndIntakeStaysAccepted(t *testing.T) {
	var hits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Forwarding.Enabled = true
	cfg.Forwarding.APIKey = "test-api-key"
	cfg.Forwarding.TargetBaseURL = upstream.URL
	cfg.Forwarding.MaxAttempts = 2
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	samplingRate := 1.0
	cfg.Safety.SamplingRate = &samplingRate
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs", bytes.NewBufferString(`{"message":"hello","ddtags":"service:api,env:prod"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}

	events, err := app.store.List(req.Context(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	gotForwarding := events[0].Forwarding
	if gotForwarding.Status != "dropped" || gotForwarding.ErrorClass != "upstream_status" || gotForwarding.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unexpected forwarding result: %#v", gotForwarding)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("upstream hits = %d, want 2 bounded attempts", got)
	}
}

func TestConfigEndpointDoesNotExposeAPIKey(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Forwarding.Enabled = true
	cfg.Forwarding.APIKey = "test-api-key"
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "test-api-key") || strings.Contains(body, "apiKey") {
		t.Fatalf("config endpoint leaked API key material: %s", body)
	}
}

func TestFileStoragePersistsRedactedEnvelope(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Storage.Kind = "file"
	cfg.Storage.Path = t.TempDir() + "/events.json"
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"message":"owner@example.com failed login","password":"plain-secret","ddtags":"service:api,env:local"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs?access_token=query-secret", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer header-secret")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}

	persisted, err := os.ReadFile(cfg.Storage.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(persisted)
	for _, leaked := range []string{"owner@example.com", "plain-secret", "query-secret", "header-secret", "Bearer"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("persisted event leaked %q:\n%s", leaked, text)
		}
	}
	if strings.Contains(text, `"rawBody":`) {
		t.Fatalf("forward mode should not persist rawBody:\n%s", text)
	}
	if !strings.Contains(text, "***REDACTED***") {
		t.Fatalf("persisted event should include redaction markers:\n%s", text)
	}
}

func TestSQLiteStoragePersistsRedactedEnvelope(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Storage.Kind = "sqlite"
	cfg.Storage.Path = t.TempDir() + "/events.db"
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"message":"owner@example.com failed login","password":"plain-secret","ddtags":"service:api,env:local"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs?access_token=query-secret", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer header-secret")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got status %d: %s", rec.Code, rec.Body.String())
	}
	if closer, ok := app.store.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}
	}

	reopened, err := store.NewSQLite(cfg.Storage.Path, cfg.Storage.MaxEvents, cfg.Storage.TTL)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	events, err := reopened.List(context.Background(), store.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].RawBody != "" {
		t.Fatalf("forward mode should not persist rawBody: %#v", events[0])
	}
	text := mustJSON(t, events)
	for _, leaked := range []string{"owner@example.com", "plain-secret", "query-secret", "header-secret", "Bearer"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("sqlite persisted event leaked %q:\n%s", leaked, text)
		}
	}
	if !strings.Contains(text, "***REDACTED***") {
		t.Fatalf("sqlite persisted event should include redaction markers:\n%s", text)
	}
}

func TestSQLiteStorageFailureDropsDogtapCopyFailOpen(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = config.ModeForward
	cfg.Storage.Kind = "sqlite"
	cfg.Storage.Path = t.TempDir() + "/events.db"
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	samplingRate := 1.0
	cfg.Safety.SamplingRate = &samplingRate
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := app.store.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/rum", bytes.NewBufferString(`{"service":"web","env":"prod","usr":{"id":"u"},"context":{"account":{"id":"a"},"workspace":{"id":"w"}}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("production storage failure should fail open with 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "storage_error") {
		t.Fatalf("expected storage_error response: %s", rec.Body.String())
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func newTestApp(t *testing.T, mode config.Mode) *App {
	t.Helper()
	cfg := config.Default()
	cfg.Mode = mode
	cfg.Server.HTTPAddr = ""
	cfg.Server.APMAddr = ""
	cfg.Server.OTLPHTTPAddr = ""
	cfg.Server.GRPCAddr = ""
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

type blockingStore struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	events  []event.EventEnvelope
}

type failingStore struct{}

func (failingStore) Add(context.Context, event.EventEnvelope) error {
	return errors.New("store unavailable")
}

func (failingStore) List(context.Context, store.Query) ([]event.EventEnvelope, error) {
	return nil, nil
}

func (failingStore) Get(context.Context, string) (event.EventEnvelope, bool, error) {
	return event.EventEnvelope{}, false, nil
}

func newBlockingStore() *blockingStore {
	return &blockingStore{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (s *blockingStore) Add(_ context.Context, e event.EventEnvelope) error {
	s.once.Do(func() {
		close(s.entered)
	})
	<-s.release
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *blockingStore) List(_ context.Context, _ store.Query) ([]event.EventEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]event.EventEnvelope, len(s.events))
	copy(out, s.events)
	return out, nil
}

func (s *blockingStore) Get(_ context.Context, id string) (event.EventEnvelope, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.events {
		if e.ID == id {
			return e, true, nil
		}
	}
	return event.EventEnvelope{}, false, nil
}

func (s *blockingStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}
