package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/midagedev/dogtap/internal/config"
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
