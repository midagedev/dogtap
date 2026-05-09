package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/midagedev/dogtap/internal/report"
)

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
