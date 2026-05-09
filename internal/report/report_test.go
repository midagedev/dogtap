package report

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/event"
)

func TestMarkdownIncludesSummaryFindingsAndEvents(t *testing.T) {
	r := Report{
		CreatedAt: time.Date(2026, 5, 8, 10, 11, 12, 0, time.UTC),
		Summary: Summary{
			Total:  2,
			Passed: 1,
			Failed: 1,
			Fatal:  1,
		},
		Events: []event.EventEnvelope{
			{
				ID:       "rum-1",
				Source:   event.SourceRUM,
				Endpoint: "/rum",
				Method:   "POST",
				Normalized: event.NormalizedTelemetry{
					Source:  event.SourceRUM,
					Service: "web",
					Env:     "local",
					Route:   "/checkout",
				},
				Validation: event.ValidationResult{
					Status:  "fail",
					Summary: "1 validation rule(s) failed",
					Rules: []event.ValidationRuleResult{
						{
							RuleID:    "required.rum.workspaceId",
							Severity:  "error",
							Status:    "fail",
							Message:   "required field is missing",
							FieldPath: "workspaceId",
						},
					},
				},
			},
			{
				ID:       "logs-1",
				Source:   event.SourceLogs,
				Endpoint: "/api/v2/logs",
				Method:   "POST",
				Normalized: event.NormalizedTelemetry{
					Source:  event.SourceLogs,
					Service: "api",
					Env:     "local",
					TraceID: "trace-1",
				},
				Validation: event.ValidationResult{Status: "pass", Summary: "validation passed"},
			},
		},
	}

	got := string(r.Markdown())
	for _, want := range []string{
		"# Dogtap Validation Report",
		"- Created: 2026-05-08T10:11:12Z",
		"| Total | Passed | Failed | Fatal | Warnings |",
		"| 2 | 1 | 1 | 1 | 0 |",
		"### RUM `rum-1`",
		"required.rum.workspaceId",
		"workspaceId",
		"| logs-1 | logs | pass | api | local | - | trace-1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Markdown report missing %q:\n%s", want, got)
		}
	}
}

func TestMarkdownEscapesTableCells(t *testing.T) {
	r := Report{
		CreatedAt: time.Date(2026, 5, 8, 10, 11, 12, 0, time.UTC),
		Summary:   Summary{Total: 1, Failed: 1, Fatal: 1},
		Events: []event.EventEnvelope{
			{
				ID:       "logs-1",
				Source:   event.SourceLogs,
				Endpoint: "/api/v2/logs",
				Method:   "POST",
				Validation: event.ValidationResult{
					Status: "fail",
					Rules: []event.ValidationRuleResult{
						{
							RuleID:   "pii.body.token",
							Severity: "fatal",
							Status:   "fail",
							Message:  "sensitive body token detected",
							Evidence: "token|value\nnext line",
						},
					},
				},
			},
		},
	}

	got := string(r.Markdown())
	if !strings.Contains(got, "token\\|value<br>next line") {
		t.Fatalf("Markdown report did not escape table cell:\n%s", got)
	}
}

func TestResolveFormatInfersMarkdownOnlyForMarkdownExtensions(t *testing.T) {
	tests := []struct {
		name       string
		format     Format
		outputPath string
		want       Format
	}{
		{name: "md", format: FormatAuto, outputPath: "dogtap-report.md", want: FormatMarkdown},
		{name: "markdown", format: FormatAuto, outputPath: "dogtap-report.markdown", want: FormatMarkdown},
		{name: "json", format: FormatAuto, outputPath: "dogtap-report.json", want: FormatJSON},
		{name: "no extension", format: FormatAuto, outputPath: "dogtap-report", want: FormatJSON},
		{name: "explicit json", format: FormatJSON, outputPath: "dogtap-report.md", want: FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveFormat(tt.format, tt.outputPath); got != tt.want {
				t.Fatalf("ResolveFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFaroReplaySourceInference(t *testing.T) {
	for _, path := range []string{
		"fixtures/faro/workflow.json",
		"fixtures/collect/browser.json",
	} {
		if got := sourceFromPath(path); got != event.SourceFaro {
			t.Fatalf("sourceFromPath(%q) = %q, want %q", path, got, event.SourceFaro)
		}
	}
	if got := endpointFor(event.SourceFaro); got != "/collect" {
		t.Fatalf("endpointFor(faro) = %q, want /collect", got)
	}
}

func TestReplayUsesFixtureMetadataRequest(t *testing.T) {
	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "rum-browser.json")
	body := `[
		{"service":"web-frontend","version":"g1-fixture","usr":{"id":"user-1"},"context":{"account":{"id":"account-1"},"workspace":{"id":"workspace-1"}}}
	]`
	if err := os.WriteFile(fixturePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := `{
		"replayRequest": {
			"method": "POST",
			"path": "/datadog-intake-proxy",
			"query": "ddforward=%2Fapi%2Fv2%2Frum%3Fddtags%3Denv%253Alocal%252Cservice%253Aweb-frontend%252Cversion%253Ag1-fixture",
			"headers": {"Content-Type": "text/plain;charset=UTF-8"}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "rum-browser.meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Replay(context.Background(), config.Default(), []string{fixturePath})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Failed != 0 || len(report.Events) != 1 {
		t.Fatalf("unexpected report: %#v", report.Summary)
	}
	got := report.Events[0].Normalized
	if got.Env != "local" || got.Service != "web-frontend" || got.Version != "g1-fixture" {
		t.Fatalf("fixture metadata was not applied: %#v", got)
	}
}
