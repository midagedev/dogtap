package forwarding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/midagedev/dogtap/internal/event"
)

func TestForwardLogsSuccessRecordsAccountingAndOmitsAPIKey(t *testing.T) {
	const apiKey = "dd-api-secret-test-value"
	type observedRequest struct {
		apiKey        string
		authorization string
		cookie        string
		path          string
	}
	observed := make(chan observedRequest, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed <- observedRequest{
			apiKey:        r.Header.Get("DD-API-KEY"),
			authorization: r.Header.Get("Authorization"),
			cookie:        r.Header.Get("Cookie"),
			path:          r.URL.Path,
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	forwarder := newTestForwarder(t, Config{
		Mode:          ModeForward,
		APIKey:        apiKey,
		TargetBaseURL: server.URL,
		Retry:         RetryPolicy{MaxAttempts: 3},
	})

	result := forwarder.Forward(context.Background(), Payload{
		Kind: KindLogs,
		Body: []byte(`{"message":"hello","service":"api"}`),
		Header: http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"Bearer inbound-secret"},
			"Cookie":        []string{"session=inbound-secret"},
		},
	})

	if result.Status != "success" {
		t.Fatalf("got status %q, want success: %#v", result.Status, result)
	}
	if result.StatusCode != http.StatusAccepted {
		t.Fatalf("got status code %d, want %d", result.StatusCode, http.StatusAccepted)
	}
	if result.RetryCount != 0 {
		t.Fatalf("got retry count %d, want 0", result.RetryCount)
	}
	got := <-observed
	if got.path != "/api/v2/logs" {
		t.Fatalf("got path %q, want /api/v2/logs", got.path)
	}
	if got.apiKey != apiKey {
		t.Fatalf("logs forwarding did not set the Datadog API key header")
	}
	if got.authorization != "" || got.cookie != "" {
		t.Fatalf("forwarded inbound secret headers: authorization=%q cookie=%q", got.authorization, got.cookie)
	}
	assertResultDoesNotContain(t, result, apiKey)

	stats := forwarder.Stats()
	if stats != (Stats{Payloads: 1, Attempts: 1, Successes: 1}) {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestForwardRUMRetriesThenSucceeds(t *testing.T) {
	var hits atomic.Int64
	var badPath atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/rum" {
			badPath.Add(1)
		}
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	forwarder := newTestForwarder(t, Config{
		Mode:          ModeForward,
		TargetBaseURL: server.URL,
		Retry:         RetryPolicy{MaxAttempts: 3},
	})

	result := forwarder.Forward(context.Background(), Payload{
		Kind: KindRUM,
		Body: []byte(`{"type":"view","application":{"id":"app-1"}}`),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	})

	if result.Status != "success" {
		t.Fatalf("got status %q, want success: %#v", result.Status, result)
	}
	if result.RetryCount != 1 {
		t.Fatalf("got retry count %d, want 1", result.RetryCount)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("got %d upstream attempts, want 2", got)
	}
	if got := badPath.Load(); got != 0 {
		t.Fatalf("got %d requests with unexpected path", got)
	}
	stats := forwarder.Stats()
	if stats != (Stats{Payloads: 1, Attempts: 2, Retries: 1, Successes: 1}) {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestForwardReplayUsesReplayEndpoint(t *testing.T) {
	observedPath := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedPath <- r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	forwarder := newTestForwarder(t, Config{
		Mode:          ModeForward,
		TargetBaseURL: server.URL,
	})

	result := forwarder.Forward(context.Background(), Payload{
		Kind: KindReplay,
		Body: []byte(`{"records":[{"type":4}]}`),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	})

	if result.Status != "success" {
		t.Fatalf("got status %q, want success: %#v", result.Status, result)
	}
	if got := <-observedPath; got != "/api/v2/replay" {
		t.Fatalf("got path %q, want /api/v2/replay", got)
	}
}

func TestForwardRUMDropsAfterHardBoundedRetries(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	forwarder := newTestForwarder(t, Config{
		Mode:          ModeForward,
		TargetBaseURL: server.URL,
		Retry:         RetryPolicy{MaxAttempts: 99},
	})

	result := forwarder.Forward(context.Background(), Payload{
		Kind: KindRUM,
		Body: []byte(`{"type":"error"}`),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	})

	if got := hits.Load(); got != hardMaxAttempts {
		t.Fatalf("got %d upstream attempts, want hard bound %d", got, hardMaxAttempts)
	}
	if result.Status != "dropped" {
		t.Fatalf("got status %q, want dropped: %#v", result.Status, result)
	}
	if result.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("got status code %d, want %d", result.StatusCode, http.StatusServiceUnavailable)
	}
	if result.RetryCount != hardMaxAttempts-1 {
		t.Fatalf("got retry count %d, want %d", result.RetryCount, hardMaxAttempts-1)
	}
	if result.ErrorClass != "upstream_status" {
		t.Fatalf("got error class %q, want upstream_status", result.ErrorClass)
	}
	stats := forwarder.Stats()
	if stats != (Stats{Payloads: 1, Attempts: hardMaxAttempts, Retries: hardMaxAttempts - 1, Failures: 1, Drops: 1}) {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestForwardLogsWithoutAPIKeyDropsWithoutHTTPAttempt(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		hits.Add(1)
	}))
	defer server.Close()

	forwarder := newTestForwarder(t, Config{
		Mode:          ModeForward,
		TargetBaseURL: server.URL,
		Retry:         RetryPolicy{MaxAttempts: 3},
	})

	result := forwarder.Forward(context.Background(), Payload{
		Kind: KindLogs,
		Body: []byte(`{"message":"hello"}`),
	})

	if result.Status != "dropped" {
		t.Fatalf("got status %q, want dropped: %#v", result.Status, result)
	}
	if result.ErrorClass != "missing_api_key" {
		t.Fatalf("got error class %q, want missing_api_key", result.ErrorClass)
	}
	if got := hits.Load(); got != 0 {
		t.Fatalf("got %d upstream attempts, want 0", got)
	}
	stats := forwarder.Stats()
	if stats != (Stats{Payloads: 1, Failures: 1, Drops: 1}) {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func newTestForwarder(t *testing.T, cfg Config) *Forwarder {
	t.Helper()
	forwarder, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return forwarder
}

func assertResultDoesNotContain(t *testing.T, result event.ForwardingResult, secret string) {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("ForwardingResult persisted API key: %s", encoded)
	}
}
