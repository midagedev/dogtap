package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/midagedev/dogtap/internal/bundle"
	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/diagnose"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/report"
	"github.com/midagedev/dogtap/internal/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) error {
	if len(args) == 0 {
		args = []string{"serve"}
	}

	switch args[0] {
	case "serve":
		return serve(args[1:])
	case "replay":
		return replay(args[1:])
	case "diagnose":
		return diagnoseLive(args[1:])
	case "version":
		fmt.Printf("dogtap %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to dogtap YAML config")
	if err := fs.Parse(args); err != nil {
		return config.ErrInvalid
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := server.New(cfg)
	if err != nil {
		return err
	}

	slog.Info("starting dogtap", "http", cfg.Server.HTTPAddr, "apm", cfg.Server.APMAddr, "otlp_http", cfg.Server.OTLPHTTPAddr, "mode", cfg.Mode)
	return app.Run(ctx)
}

func diagnoseLive(args []string) error {
	fs := flag.NewFlagSet("diagnose", flag.ContinueOnError)
	baseURL := fs.String("base-url", envDefault("DOGTAP_DIAG_BASE_URL", "http://127.0.0.1:8080"), "Dogtap HTTP base URL")
	outputDir := fs.String("output", envDefault("DOGTAP_ARTIFACT_DIR", defaultDiagnosticsDir()), "diagnostics output directory")
	limit := fs.Int("limit", 200, "maximum retained events to collect")
	expectNonEmpty := fs.Bool("expect-non-empty", false, "fail if Dogtap has no retained events")
	expectSource := fs.String("expect-source", "", "comma-separated sources expected in retained events")
	expectPayloadKind := fs.String("expect-payload-kind", "", "comma-separated payload kinds expected in retained events")
	expectService := fs.String("expect-service", "", "comma-separated services expected in retained events")
	expectSession := fs.String("expect-session", "", "comma-separated session IDs expected in retained events")
	expectTrace := fs.String("expect-trace", "", "comma-separated trace IDs expected in retained events")
	expectCase := fs.String("expect-case", "", "comma-separated case IDs expected in retained events")
	expectRoute := fs.String("expect-route", "", "comma-separated routes expected in retained events")
	expectMetric := fs.String("expect-metric", "", "comma-separated metric names expected in retained events")
	expectEndpoint := fs.String("expect-endpoint", "", "comma-separated intake endpoints expected in retained events")
	filterSource := fs.String("filter-source", "", "debug bundle source filter")
	filterPayloadKind := fs.String("filter-payload-kind", "", "debug bundle payload kind filter")
	filterService := fs.String("filter-service", "", "debug bundle service filter")
	filterEnv := fs.String("filter-env", "", "debug bundle env filter")
	filterUserID := fs.String("filter-user-id", "", "debug bundle user ID filter")
	filterAccountID := fs.String("filter-account-id", "", "debug bundle account ID filter")
	filterWorkspaceID := fs.String("filter-workspace-id", "", "debug bundle workspace ID filter")
	filterCaseID := fs.String("filter-case-id", "", "debug bundle case ID filter")
	filterTraceID := fs.String("filter-trace-id", "", "debug bundle trace ID filter")
	filterSessionID := fs.String("filter-session-id", "", "debug bundle session ID filter")
	filterViewID := fs.String("filter-view-id", "", "debug bundle view ID filter")
	filterRoute := fs.String("filter-route", "", "debug bundle route filter")
	filterStatus := fs.String("filter-status", "", "debug bundle validation status filter")
	if err := fs.Parse(args); err != nil {
		return config.ErrInvalid
	}

	result, err := diagnose.Collect(context.Background(), diagnose.Options{
		BaseURL:   *baseURL,
		OutputDir: *outputDir,
		Limit:     *limit,
		Expectations: diagnose.Expectations{
			NonEmpty:     *expectNonEmpty,
			Sources:      splitCSV(*expectSource),
			PayloadKinds: splitCSV(*expectPayloadKind),
			Services:     splitCSV(*expectService),
			Sessions:     splitCSV(*expectSession),
			Traces:       splitCSV(*expectTrace),
			Cases:        splitCSV(*expectCase),
			Routes:       splitCSV(*expectRoute),
			Metrics:      splitCSV(*expectMetric),
			Endpoints:    splitCSV(*expectEndpoint),
		},
		Filter: bundle.Request{
			Source:      event.Source(strings.TrimSpace(*filterSource)),
			PayloadKind: strings.TrimSpace(*filterPayloadKind),
			Service:     strings.TrimSpace(*filterService),
			Env:         strings.TrimSpace(*filterEnv),
			UserID:      strings.TrimSpace(*filterUserID),
			AccountID:   strings.TrimSpace(*filterAccountID),
			WorkspaceID: strings.TrimSpace(*filterWorkspaceID),
			CaseID:      strings.TrimSpace(*filterCaseID),
			TraceID:     strings.TrimSpace(*filterTraceID),
			SessionID:   strings.TrimSpace(*filterSessionID),
			ViewID:      strings.TrimSpace(*filterViewID),
			Route:       strings.TrimSpace(*filterRoute),
			Status:      strings.TrimSpace(*filterStatus),
		},
	})
	fmt.Printf("Dogtap diagnostics: %s\n", result.Assertions.Status)
	fmt.Printf("Output: %s\n", result.OutputDir)
	fmt.Printf("Summary: %s\n", filepath.Join(result.OutputDir, "summary.md"))
	return err
}

func replay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to dogtap YAML config")
	outputPath := fs.String("output", "", "optional report output path")
	outputFormat := fs.String("format", string(report.FormatAuto), "report format: auto, json, markdown")
	if err := fs.Parse(args); err != nil {
		return config.ErrInvalid
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("%w: replay requires at least one fixture path", report.ErrTool)
	}
	format, err := report.ParseFormat(*outputFormat)
	if err != nil {
		return fmt.Errorf("%w: %v", config.ErrInvalid, err)
	}
	format = report.ResolveFormat(format, *outputPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	r, err := report.Replay(context.Background(), cfg, fs.Args())
	if err != nil {
		return err
	}
	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, r.Render(format), 0o644); err != nil {
			return fmt.Errorf("%w: write report: %v", report.ErrTool, err)
		}
		if r.HasFailures() {
			return report.ErrValidationFailed
		}
		return nil
	}
	fmt.Println(string(r.Render(format)))
	if r.HasFailures() {
		return report.ErrValidationFailed
	}
	return nil
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func defaultDiagnosticsDir() string {
	return filepath.Join(".dogtap", "diagnostics", time.Now().UTC().Format("20060102T150405Z"))
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, report.ErrValidationFailed):
		return 1
	case errors.Is(err, config.ErrInvalid):
		return 2
	case errors.Is(err, server.ErrStart):
		return 3
	case errors.Is(err, report.ErrTool):
		return 4
	default:
		return 1
	}
}
