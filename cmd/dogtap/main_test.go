package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/midagedev/dogtap/internal/report"
)

func TestVersionCommandPrintsBuildMetadata(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"version"}); err != nil {
			t.Fatalf("run version: %v", err)
		}
	})

	for _, expected := range []string{"dogtap dev", "commit: none", "built: unknown"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected %q in version output, got:\n%s", expected, output)
		}
	}
}

func TestReplayOutputInfersMarkdownFromOutputExtension(t *testing.T) {
	clearDogtapEnv(t)
	fixture := writePassingRUMFixture(t)
	output := filepath.Join(t.TempDir(), "dogtap-report.md")

	if err := run([]string{"replay", "-output", output, fixture}); err != nil {
		t.Fatalf("run replay: %v", err)
	}

	got := readFile(t, output)
	if !strings.HasPrefix(got, "# Dogtap Validation Report\n") {
		t.Fatalf("expected Markdown report, got:\n%s", got)
	}
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("Markdown output should not be JSON:\n%s", got)
	}
}

func TestReplayOutputPreservesJSONDefault(t *testing.T) {
	clearDogtapEnv(t)
	fixture := writePassingRUMFixture(t)
	output := filepath.Join(t.TempDir(), "dogtap-report.out")

	if err := run([]string{"replay", "-output", output, fixture}); err != nil {
		t.Fatalf("run replay: %v", err)
	}

	got := readFile(t, output)
	if !json.Valid([]byte(got)) {
		t.Fatalf("expected JSON report, got:\n%s", got)
	}
	if !strings.Contains(got, `"summary"`) {
		t.Fatalf("expected JSON summary, got:\n%s", got)
	}
}

func TestReplayOutputFormatOverridesExtension(t *testing.T) {
	clearDogtapEnv(t)
	fixture := writePassingRUMFixture(t)
	output := filepath.Join(t.TempDir(), "dogtap-report.txt")

	if err := run([]string{"replay", "-format", "markdown", "-output", output, fixture}); err != nil {
		t.Fatalf("run replay: %v", err)
	}

	got := readFile(t, output)
	if !strings.HasPrefix(got, "# Dogtap Validation Report\n") {
		t.Fatalf("expected Markdown report, got:\n%s", got)
	}
}

func TestReplayWritesReportBeforeReturningValidationFailure(t *testing.T) {
	clearDogtapEnv(t)
	dir := t.TempDir()
	fixture := filepath.Join(dir, "rum-missing-context.json")
	if err := os.WriteFile(fixture, []byte(`{"service":"web","env":"local","usr":{"id":"u1"}}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	output := filepath.Join(dir, "dogtap-report.md")

	err := run([]string{"replay", "-output", output, fixture})
	if !errors.Is(err, report.ErrValidationFailed) {
		t.Fatalf("run replay error = %v, want validation failure", err)
	}

	got := readFile(t, output)
	if !strings.Contains(got, "required.rum.accountId") || !strings.Contains(got, "required.rum.workspaceId") {
		t.Fatalf("expected validation findings in report, got:\n%s", got)
	}
}

func TestDiagnoseCommandWritesBundle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/readyz":
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		case "/api/events":
			_, _ = w.Write([]byte(`[{
				"id":"rum-1",
				"receivedAt":"2026-05-09T12:00:00Z",
				"source":"rum",
				"payloadKind":"event",
				"endpoint":"/datadog-intake-proxy",
				"method":"POST",
				"normalized":{"source":"rum","service":"web","env":"local","sessionId":"session-1"},
				"validation":{"status":"pass"}
			}]`))
		case "/api/reports/latest":
			_, _ = w.Write([]byte(`{"summary":{"total":1,"passed":1,"failed":0},"events":[]}`))
		case "/api/debug-bundles":
			_, _ = w.Write([]byte(`{"summary":{"total":1},"events":[]}`))
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("dogtap_store_events 1\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	output := filepath.Join(t.TempDir(), "diagnostics")

	err := run([]string{
		"diagnose",
		"-base-url", server.URL,
		"-output", output,
		"-expect-non-empty",
		"-expect-source", "rum",
		"-expect-session", "session-1",
	})
	if err != nil {
		t.Fatalf("run diagnose: %v", err)
	}

	assertions := readFile(t, filepath.Join(output, "assertions.json"))
	if !strings.Contains(assertions, `"status": "pass"`) {
		t.Fatalf("expected passing assertions, got:\n%s", assertions)
	}
	if !strings.Contains(readFile(t, filepath.Join(output, "summary.md")), "session:session-1") {
		t.Fatalf("expected session assertion in summary")
	}
}

func writePassingRUMFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rum-pass.json")
	body := []byte(`{
  "service": "web",
  "env": "local",
  "usr": {"id": "user-1"},
  "context": {
    "account": {"id": "account-1"},
    "workspace": {"id": "workspace-1"}
  }
}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func clearDogtapEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"DOGTAP_MODE",
		"DOGTAP_HTTP_ADDR",
		"DOGTAP_APM_ADDR",
		"DOGTAP_OTLP_HTTP_ADDR",
		"DOGTAP_GRPC_ADDR",
		"DOGTAP_STORAGE_MAX_EVENTS",
		"DOGTAP_STORAGE_KIND",
		"DOGTAP_STORAGE_PATH",
		"DOGTAP_STORAGE_TTL",
		"DOGTAP_ALLOW_RAW_PAYLOADS",
	} {
		t.Setenv(key, "")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
