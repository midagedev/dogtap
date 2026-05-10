package main

import (
	"context"
	"encoding/json"
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
	"github.com/midagedev/dogtap/internal/contract"
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
	case "contract":
		return contractCommand(args[1:])
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
	workflowContractPaths := stringListFlag{}
	fs.Var(&workflowContractPaths, "workflow-contract", "path to a workflow contract YAML/JSON file; repeatable")
	failOnWorkflowContract := fs.Bool("fail-on-workflow-contract", false, "return a validation failure when any workflow contract fails")
	if err := fs.Parse(args); err != nil {
		return config.ErrInvalid
	}
	workflowContracts, err := loadWorkflowContracts(workflowContractPaths)
	if err != nil {
		return err
	}

	result, err := diagnose.Collect(context.Background(), diagnose.Options{
		BaseURL:                *baseURL,
		OutputDir:              *outputDir,
		Limit:                  *limit,
		WorkflowContracts:      workflowContracts,
		FailOnWorkflowContract: *failOnWorkflowContract,
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
	if len(result.WorkflowContracts) > 0 {
		fmt.Printf("Workflow contracts: %s\n", workflowContractStatus(result.WorkflowContracts))
	}
	fmt.Printf("Output: %s\n", result.OutputDir)
	fmt.Printf("Summary: %s\n", filepath.Join(result.OutputDir, "summary.md"))
	return err
}

func contractCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: contract requires a subcommand", config.ErrInvalid)
	}
	switch args[0] {
	case "validate":
		return validateContractFiles(args[1:])
	default:
		return fmt.Errorf("%w: unknown contract subcommand %q", config.ErrInvalid, args[0])
	}
}

func validateContractFiles(args []string) error {
	fs := flag.NewFlagSet("contract validate", flag.ContinueOnError)
	outputFormat := fs.String("format", "text", "output format: text, json")
	if err := fs.Parse(args); err != nil {
		return config.ErrInvalid
	}
	paths := fs.Args()
	if len(paths) == 0 {
		return fmt.Errorf("%w: contract validate requires at least one path", config.ErrInvalid)
	}

	reports := make([]contract.ValidationReport, 0, len(paths))
	for _, path := range paths {
		reports = append(reports, contract.ValidateFile(path))
	}

	switch strings.TrimSpace(*outputFormat) {
	case "", "text":
		fmt.Print(renderContractValidationReports(reports))
	case "json":
		body, err := json.MarshalIndent(reports, "", "  ")
		if err != nil {
			return fmt.Errorf("%w: render contract validation report: %v", report.ErrTool, err)
		}
		fmt.Println(string(body))
	default:
		return fmt.Errorf("%w: unsupported contract validate format %q", config.ErrInvalid, *outputFormat)
	}

	if hasContractValidationFailures(reports) {
		return fmt.Errorf("%w: workflow contract validation failed", report.ErrValidationFailed)
	}
	return nil
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

type stringListFlag []string

func (f *stringListFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*f = append(*f, value)
	return nil
}

func loadWorkflowContracts(paths []string) ([]contract.Definition, error) {
	contracts := make([]contract.Definition, 0, len(paths))
	for _, path := range paths {
		def, err := contract.LoadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", config.ErrInvalid, err)
		}
		contracts = append(contracts, def)
	}
	return contracts, nil
}

func workflowContractStatus(results []contract.Result) string {
	for _, result := range results {
		if result.Status == "fail" {
			return "fail"
		}
	}
	return "pass"
}

func hasContractValidationFailures(reports []contract.ValidationReport) bool {
	for _, validationReport := range reports {
		if validationReport.Status == "fail" {
			return true
		}
	}
	return false
}

func renderContractValidationReports(reports []contract.ValidationReport) string {
	var b strings.Builder
	for _, validationReport := range reports {
		fmt.Fprintf(&b, "%s: %s\n", validationReport.Path, validationReport.Status)
		for _, validationIssue := range validationReport.Issues {
			field := validationIssue.Field
			if field == "" {
				field = "file"
			}
			if validationIssue.CheckID != "" {
				fmt.Fprintf(&b, "  - %s (%s): %s\n", field, validationIssue.CheckID, validationIssue.Message)
			} else {
				fmt.Fprintf(&b, "  - %s: %s\n", field, validationIssue.Message)
			}
		}
	}
	return b.String()
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
