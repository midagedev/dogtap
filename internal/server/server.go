package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/midagedev/dogtap/internal/bundle"
	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/diagnose"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/forwarding"
	"github.com/midagedev/dogtap/internal/intake"
	"github.com/midagedev/dogtap/internal/report"
	"github.com/midagedev/dogtap/internal/store"
	"github.com/midagedev/dogtap/internal/validation"
	"github.com/midagedev/dogtap/web"
)

var ErrStart = errors.New("dogtap server failed to start")

type App struct {
	cfg       config.Config
	store     store.Store
	validator validation.Validator
	forwarder *forwarding.Forwarder
	safety    *safetyController
	assets    http.Handler
}

type safetyController struct {
	cfg               config.SafetyConfig
	mode              config.Mode
	sampleMu          sync.Mutex
	sampleCredit      float64
	inFlight          atomic.Int64
	accepted          atomic.Int64
	sampleDrops       atomic.Int64
	backpressureDrops atomic.Int64
	storageDrops      atomic.Int64
}

func New(cfg config.Config) (*App, error) {
	publicBasePath, err := config.NormalizePublicBasePath(cfg.Server.PublicBasePath)
	if err != nil {
		return nil, err
	}
	cfg.Server.PublicBasePath = publicBasePath
	assets, err := dashboardHandler()
	if err != nil {
		return nil, err
	}
	eventStore, err := newStore(cfg)
	if err != nil {
		return nil, err
	}
	forwarder, err := newForwarder(cfg)
	if err != nil {
		return nil, err
	}
	return &App{
		cfg:       cfg,
		store:     eventStore,
		validator: validation.New(cfg.Validation),
		forwarder: forwarder,
		safety:    newSafetyController(cfg),
		assets:    assets,
	}, nil
}

func newSafetyController(cfg config.Config) *safetyController {
	return &safetyController{cfg: cfg.Safety, mode: cfg.Mode, sampleCredit: 1}
}

func newStore(cfg config.Config) (store.Store, error) {
	switch cfg.Storage.Kind {
	case "memory":
		return store.NewMemory(cfg.Storage.MaxEvents, cfg.Storage.TTL), nil
	case "file":
		return store.NewFile(cfg.Storage.Path, cfg.Storage.MaxEvents, cfg.Storage.TTL)
	case "sqlite":
		return store.NewSQLite(cfg.Storage.Path, cfg.Storage.MaxEvents, cfg.Storage.TTL)
	default:
		return nil, fmt.Errorf("unsupported storage kind %q", cfg.Storage.Kind)
	}
}

func newForwarder(cfg config.Config) (*forwarding.Forwarder, error) {
	mode := forwarding.ModeDisabled
	if cfg.Forwarding.Enabled {
		switch cfg.Mode {
		case config.ModeForward:
			mode = forwarding.ModeForward
		case config.ModeTee:
			mode = forwarding.ModeTee
		case config.ModeRedactOnly:
			mode = forwarding.ModeRedactOnly
		case config.ModeLocal:
			mode = forwarding.ModeForward
		default:
			mode = forwarding.ModeDisabled
		}
	}
	return forwarding.New(forwarding.Config{
		Mode:          mode,
		Site:          cfg.Forwarding.Site,
		APIKey:        cfg.Forwarding.APIKey,
		TargetBaseURL: cfg.Forwarding.TargetBaseURL,
		Retry: forwarding.RetryPolicy{
			MaxAttempts: cfg.Forwarding.MaxAttempts,
			Backoff:     cfg.Forwarding.Backoff,
		},
		Timeout: cfg.Forwarding.Timeout,
	})
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.Close(); err != nil {
			slog.Warn("close event store", "error", err)
		}
	}()
	servers := a.httpServers()
	errCh := make(chan error, len(servers)+1)

	for _, srv := range servers {
		srv := srv
		go func() {
			slog.Info("listening", "addr", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("%w: %s: %v", ErrStart, srv.Addr, err)
			}
		}()
	}

	if a.cfg.Server.GRPCAddr != "" {
		go func() {
			if err := a.runGRPC(ctx); err != nil {
				errCh <- err
			}
		}()
	}

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("%w: shutdown %s: %v", ErrStart, srv.Addr, err)
		}
	}
	return nil
}

func (a *App) Close() error {
	closer, ok := a.store.(interface{ Close() error })
	if !ok {
		return nil
	}
	return closer.Close()
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	a.registerCommon(mux)
	a.registerIntake(mux, "/rum", event.SourceRUM)
	a.registerIntake(mux, "/datadog-intake-proxy", event.SourceRUM)
	a.registerIntake(mux, "/api/v2/replay", event.SourceRUM)
	a.registerIntake(mux, "/v1/input", event.SourceLogs)
	a.registerIntake(mux, "/api/v2/logs", event.SourceLogs)
	a.registerIntake(mux, "/v0.3/traces", event.SourceAPM)
	a.registerIntake(mux, "/v0.4/traces", event.SourceAPM)
	a.registerIntake(mux, "/v0.5/traces", event.SourceAPM)
	a.registerIntake(mux, "/v1/traces", event.SourceOTLP)
	a.registerIntake(mux, "/v1/logs", event.SourceOTLP)
	a.registerIntake(mux, "/v1/metrics", event.SourceOTLP)
	a.registerIntake(mux, "/faro", event.SourceFaro)
	a.registerIntake(mux, "/collect", event.SourceFaro)
	a.registerIntake(mux, "/collect/", event.SourceFaro)
	mux.Handle("/", a.assets)
	return a.withPublicBasePath(mux)
}

func (a *App) httpServers() []*http.Server {
	common := a.Handler()
	servers := []*http.Server{{Addr: a.cfg.Server.HTTPAddr, Handler: common}}
	if a.cfg.Server.APMAddr != "" && a.cfg.Server.APMAddr != a.cfg.Server.HTTPAddr {
		apmMux := http.NewServeMux()
		a.registerCommon(apmMux)
		a.registerIntake(apmMux, "/v0.3/traces", event.SourceAPM)
		a.registerIntake(apmMux, "/v0.4/traces", event.SourceAPM)
		a.registerIntake(apmMux, "/v0.5/traces", event.SourceAPM)
		servers = append(servers, &http.Server{Addr: a.cfg.Server.APMAddr, Handler: a.withPublicBasePath(apmMux)})
	}
	if a.cfg.Server.OTLPHTTPAddr != "" && a.cfg.Server.OTLPHTTPAddr != a.cfg.Server.HTTPAddr {
		otlpMux := http.NewServeMux()
		a.registerCommon(otlpMux)
		a.registerIntake(otlpMux, "/v1/traces", event.SourceOTLP)
		a.registerIntake(otlpMux, "/v1/logs", event.SourceOTLP)
		a.registerIntake(otlpMux, "/v1/metrics", event.SourceOTLP)
		servers = append(servers, &http.Server{Addr: a.cfg.Server.OTLPHTTPAddr, Handler: a.withPublicBasePath(otlpMux)})
	}
	return servers
}

func (a *App) withPublicBasePath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := publicBasePathForRequest(r, a.cfg.Server.PublicBasePath)
		if prefix == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == prefix {
			http.Redirect(w, r, prefix+"/", http.StatusPermanentRedirect)
			return
		}
		if !strings.HasPrefix(r.URL.Path, prefix+"/") {
			next.ServeHTTP(w, r)
			return
		}
		clone := new(http.Request)
		*clone = *r
		clone.URL = cloneURL(r)
		clone.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		if clone.URL.Path == "" {
			clone.URL.Path = "/"
		}
		if clone.URL.RawPath != "" {
			clone.URL.RawPath = strings.TrimPrefix(clone.URL.RawPath, prefix)
			if clone.URL.RawPath == "" {
				clone.URL.RawPath = "/"
			}
		}
		next.ServeHTTP(w, clone)
	})
}

func (a *App) registerCommon(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /metrics", a.handleMetrics)
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, _ *http.Request) {
		safe := a.cfg
		writeJSON(w, http.StatusOK, safe)
	})
	mux.HandleFunc("GET /api/events", a.handleListEvents)
	mux.HandleFunc("GET /api/events/", a.handleGetEvent)
	mux.HandleFunc("GET /api/validation/failures", a.handleValidationFailures)
	mux.HandleFunc("GET /api/reports/latest", a.handleLatestReport)
	mux.HandleFunc("POST /api/debug-bundles", a.handleCreateDebugBundle)
	mux.HandleFunc("POST /api/diagnostics", a.handleCreateDiagnostics)
	mux.HandleFunc("POST /api/diagnostics/archive", a.handleCreateDiagnosticsArchive)
	mux.HandleFunc("POST /api/v2/logs/events/search", a.handleDatadogLogsSearch)
	mux.HandleFunc("POST /api/v2/rum/events/search", a.handleDatadogRUMSearch)
	mux.HandleFunc("POST /api/v2/spans/events/search", a.handleDatadogSpansSearch)
	mux.HandleFunc("GET /api/v1/query", a.handleDatadogMetricQuery)
	mux.HandleFunc("POST /api/replay", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "use dogtap replay for fixture replay"})
	})
}

func (a *App) registerIntake(mux *http.ServeMux, pattern string, source event.Source) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		setIntakeCORSHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		admission, release := a.safety.admit()
		if !admission.Accepted {
			writeJSON(w, a.dropStatus(admission.Reason), map[string]any{
				"status": admission.Status,
				"reason": admission.Reason,
			})
			return
		}
		defer release()
		result, err := intake.CaptureRequest(r, intake.CaptureOptions{
			Source:           source,
			AllowRawPayloads: a.cfg.RawPayloadsAllowed(),
			MaxBodyBytes:     a.cfg.Security.MaxBodyBytes,
			ForwardMode:      string(a.cfg.Mode),
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result.Event.Validation = a.validator.Validate(result.ValidationEvent)
		result.Event.Forwarding = a.forward(r.Context(), source, result)
		if sampled := a.safety.sample(); !sampled.Accepted {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"id":         result.Event.ID,
				"source":     result.Event.Source,
				"status":     sampled.Status,
				"reason":     sampled.Reason,
				"forwarding": result.Event.Forwarding,
			})
			return
		}
		if err := a.store.Add(r.Context(), result.Event); err != nil {
			a.handleStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"id":         result.Event.ID,
			"source":     result.Event.Source,
			"validation": result.Event.Validation,
		})
	})
}

func setIntakeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, content-encoding, dd-api-key, dd-evp-origin, dd-evp-origin-version, x-api-key, x-datadog-origin, x-datadog-parent-id, x-datadog-sampling-priority, x-datadog-trace-id, x-faro-session-id")
	w.Header().Set("Access-Control-Expose-Headers", "x-faro-session-status")
	w.Header().Set("Access-Control-Max-Age", "600")
}

type safetyDecision struct {
	Accepted bool
	Status   string
	Reason   string
}

func (s *safetyController) admit() (safetyDecision, func()) {
	if s == nil || s.cfg.QueueMaxInFlight <= 0 {
		return safetyDecision{Accepted: true, Status: "accepted"}, func() {}
	}
	current := s.inFlight.Add(1)
	if current > int64(s.cfg.QueueMaxInFlight) {
		s.inFlight.Add(-1)
		s.backpressureDrops.Add(1)
		return safetyDecision{
			Accepted: false,
			Status:   "dropped",
			Reason:   "queue_full",
		}, func() {}
	}
	return safetyDecision{Accepted: true, Status: "accepted"}, func() {
		s.inFlight.Add(-1)
	}
}

func (s *safetyController) sample() safetyDecision {
	rate := s.samplingRate()
	if rate >= 1 {
		s.accepted.Add(1)
		return safetyDecision{Accepted: true, Status: "accepted"}
	}
	if rate <= 0 {
		s.sampleDrops.Add(1)
		return safetyDecision{Accepted: false, Status: "dropped", Reason: "sampled_out"}
	}
	s.sampleMu.Lock()
	s.sampleCredit += rate
	accepted := s.sampleCredit >= 1
	if accepted {
		s.sampleCredit -= 1
	}
	s.sampleMu.Unlock()
	if accepted {
		s.accepted.Add(1)
		return safetyDecision{Accepted: true, Status: "accepted"}
	}
	s.sampleDrops.Add(1)
	return safetyDecision{Accepted: false, Status: "dropped", Reason: "sampled_out"}
}

func (s *safetyController) samplingRate() float64 {
	if s.cfg.SamplingRate != nil {
		return *s.cfg.SamplingRate
	}
	switch s.mode {
	case config.ModeForward, config.ModeTee, config.ModeRedactOnly:
		return 0.1
	default:
		return 1
	}
}

func (a *App) dropStatus(reason string) int {
	if reason != "queue_full" {
		return http.StatusAccepted
	}
	switch a.cfg.Mode {
	case config.ModeForward, config.ModeTee, config.ModeRedactOnly:
		return http.StatusAccepted
	default:
		return http.StatusServiceUnavailable
	}
}

func (a *App) handleStoreError(w http.ResponseWriter, err error) {
	if a.safety != nil {
		a.safety.storageDrops.Add(1)
	}
	switch a.cfg.Mode {
	case config.ModeForward, config.ModeTee, config.ModeRedactOnly:
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "dropped",
			"reason": "storage_error",
		})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

func (a *App) forward(ctx context.Context, source event.Source, result intake.CaptureResult) event.ForwardingResult {
	kind, ok := forwardingKind(source, result.Event.PayloadKind)
	if !ok {
		if a.cfg.Forwarding.Enabled {
			return event.ForwardingResult{
				Mode:         string(a.cfg.Mode),
				Attempted:    false,
				Status:       "unsupported",
				ErrorClass:   "unsupported_source",
				ErrorMessage: "forwarding is not implemented for " + string(source),
			}
		}
		return result.Event.Forwarding
	}
	return a.forwarder.Forward(ctx, forwarding.Payload{
		Kind:        kind,
		Body:        result.ForwardBody,
		Header:      result.ForwardHeader,
		ForwardPath: result.ForwardPath,
	})
}

func forwardingKind(source event.Source, payloadKind string) (forwarding.Kind, bool) {
	switch source {
	case event.SourceRUM:
		if payloadKind == "replay" {
			return forwarding.KindReplay, true
		}
		return forwarding.KindRUM, true
	case event.SourceLogs:
		return forwarding.KindLogs, true
	default:
		return "", false
	}
}

func (a *App) handleListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	events, err := a.store.List(r.Context(), store.Query{
		Source:      event.Source(q.Get("source")),
		PayloadKind: q.Get("payloadKind"),
		Service:     q.Get("service"),
		Env:         q.Get("env"),
		UserID:      q.Get("userId"),
		AccountID:   q.Get("accountId"),
		WorkspaceID: q.Get("workspaceId"),
		CaseID:      q.Get("caseId"),
		TraceID:     q.Get("traceId"),
		SpanID:      q.Get("spanId"),
		SessionID:   q.Get("sessionId"),
		ViewID:      q.Get("viewId"),
		Route:       q.Get("route"),
		Status:      q.Get("status"),
		Limit:       limit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *App) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/events/")
	e, ok, err := a.store.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *App) handleValidationFailures(w http.ResponseWriter, r *http.Request) {
	events, err := a.store.List(r.Context(), store.Query{Status: "fail", Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	failures := make([]event.EventEnvelope, 0, len(events))
	for _, e := range events {
		if e.Validation.Status == "fail" {
			failures = append(failures, e)
		}
	}
	writeJSON(w, http.StatusOK, failures)
}

func (a *App) handleLatestReport(w http.ResponseWriter, r *http.Request) {
	events, err := a.store.List(r.Context(), store.Query{Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, report.FromEvents(events))
}

func (a *App) handleCreateDiagnostics(w http.ResponseWriter, r *http.Request) {
	snapshot, ok := a.diagnosticsSnapshot(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (a *App) handleCreateDiagnosticsArchive(w http.ResponseWriter, r *http.Request) {
	snapshot, ok := a.diagnosticsSnapshot(w, r)
	if !ok {
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, artifact := range diagnose.SnapshotArtifacts(snapshot, "") {
		header := &zip.FileHeader{
			Name:   artifact.Filename,
			Method: zip.Deflate,
		}
		header.SetModTime(snapshot.CreatedAt)
		header.SetMode(0o644)
		writer, err := zw.CreateHeader(header)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := writer.Write(artifact.Body); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if err := zw.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="dogtap-diagnostics.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func (a *App) diagnosticsSnapshot(w http.ResponseWriter, r *http.Request) (diagnose.Snapshot, bool) {
	req, ok := decodeDiagnosticsRequest(w, r)
	if !ok {
		return diagnose.Snapshot{}, false
	}
	req = diagnose.NormalizeRequest(req, a.cfg.Storage.MaxEvents)
	events, err := a.store.List(r.Context(), diagnosticsQuery(req))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return diagnose.Snapshot{}, false
	}
	filter := req.Filter
	filter.Limit = req.Limit
	debugBundle := bundle.New(filter, events)
	snapshot := diagnose.NewSnapshot(diagnose.SnapshotInput{
		CreatedAt:   time.Now().UTC(),
		BaseURL:     requestBaseURL(r, a.cfg.Server.PublicBasePath),
		Request:     req,
		Health:      map[string]string{"status": "ok"},
		Readiness:   map[string]string{"status": "ready"},
		Events:      events,
		Report:      report.FromEvents(events),
		DebugBundle: debugBundle,
		Metrics:     a.metricsText(events),
		Probes: map[string]bool{
			"healthz":      true,
			"readyz":       true,
			"events":       true,
			"report":       true,
			"metrics":      true,
			"debug-bundle": true,
		},
	})
	return snapshot, true
}

func decodeDiagnosticsRequest(w http.ResponseWriter, r *http.Request) (diagnose.Request, bool) {
	if r.Body == nil || r.ContentLength == 0 {
		return diagnose.Request{}, true
	}
	var req diagnose.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return diagnose.Request{}, true
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid diagnostics request"})
		return diagnose.Request{}, false
	}
	return req, true
}

func diagnosticsQuery(req diagnose.Request) store.Query {
	filter := req.Filter
	return store.Query{
		Source:      filter.Source,
		PayloadKind: filter.PayloadKind,
		Service:     filter.Service,
		Env:         filter.Env,
		UserID:      filter.UserID,
		AccountID:   filter.AccountID,
		WorkspaceID: filter.WorkspaceID,
		CaseID:      filter.CaseID,
		TraceID:     filter.TraceID,
		SessionID:   filter.SessionID,
		ViewID:      filter.ViewID,
		Route:       filter.Route,
		Status:      filter.Status,
		Limit:       req.Limit,
	}
}

func requestBaseURL(r *http.Request, configuredPrefix string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.Split(forwardedProto, ",")[0]
	}
	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = strings.Split(forwardedHost, ",")[0]
	}
	if host == "" {
		host = "127.0.0.1"
	}
	base := scheme + "://" + strings.TrimSpace(host)
	if prefix := publicBasePathForRequest(r, configuredPrefix); prefix != "" {
		base += prefix
	}
	return base
}

func publicBasePathForRequest(r *http.Request, configuredPrefix string) string {
	if forwardedPrefix := firstHeaderValue(r.Header.Get("X-Forwarded-Prefix")); forwardedPrefix != "" {
		if prefix, err := config.NormalizePublicBasePath(forwardedPrefix); err == nil {
			return prefix
		}
	}
	prefix, err := config.NormalizePublicBasePath(configuredPrefix)
	if err != nil {
		return ""
	}
	return prefix
}

func firstHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(value, ",")[0])
}

func cloneURL(r *http.Request) *url.URL {
	u := *r.URL
	return &u
}

func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	events, err := a.store.List(r.Context(), store.Query{Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprint(w, a.metricsText(events))
}

func (a *App) metricsText(events []event.EventEnvelope) string {
	bySource := map[event.Source]int{}
	byValidation := map[string]int{}
	failures := 0
	for _, e := range events {
		bySource[e.Source]++
		status := e.Validation.Status
		if status == "" {
			status = "unknown"
		}
		byValidation[status]++
		if status == "fail" {
			failures++
		}
	}

	var b strings.Builder
	fmt.Fprintln(&b, "# HELP dogtap_store_events Current retained event count.")
	fmt.Fprintln(&b, "# TYPE dogtap_store_events gauge")
	fmt.Fprintf(&b, "dogtap_store_events %d\n", len(events))
	fmt.Fprintln(&b, "# HELP dogtap_events_by_source Current retained event count by source.")
	fmt.Fprintln(&b, "# TYPE dogtap_events_by_source gauge")
	for source, count := range bySource {
		fmt.Fprintf(&b, "dogtap_events_by_source{source=%q} %d\n", source, count)
	}
	fmt.Fprintln(&b, "# HELP dogtap_events_by_validation Current retained event count by validation status.")
	fmt.Fprintln(&b, "# TYPE dogtap_events_by_validation gauge")
	for status, count := range byValidation {
		fmt.Fprintf(&b, "dogtap_events_by_validation{status=%q} %d\n", status, count)
	}
	fmt.Fprintln(&b, "# HELP dogtap_validation_failures Current retained validation failure count.")
	fmt.Fprintln(&b, "# TYPE dogtap_validation_failures gauge")
	fmt.Fprintf(&b, "dogtap_validation_failures %d\n", failures)

	if a.safety != nil {
		fmt.Fprintln(&b, "# HELP dogtap_intake_in_flight Current in-flight intake requests admitted by Dogtap.")
		fmt.Fprintln(&b, "# TYPE dogtap_intake_in_flight gauge")
		fmt.Fprintf(&b, "dogtap_intake_in_flight %d\n", a.safety.inFlight.Load())
		fmt.Fprintln(&b, "# HELP dogtap_intake_accepted_total Intake payloads accepted after safety controls.")
		fmt.Fprintln(&b, "# TYPE dogtap_intake_accepted_total counter")
		fmt.Fprintf(&b, "dogtap_intake_accepted_total %d\n", a.safety.accepted.Load())
		fmt.Fprintln(&b, "# HELP dogtap_intake_sample_drops_total Intake payloads dropped by sampling.")
		fmt.Fprintln(&b, "# TYPE dogtap_intake_sample_drops_total counter")
		fmt.Fprintf(&b, "dogtap_intake_sample_drops_total %d\n", a.safety.sampleDrops.Load())
		fmt.Fprintln(&b, "# HELP dogtap_intake_backpressure_drops_total Intake payloads dropped because the Dogtap queue was full.")
		fmt.Fprintln(&b, "# TYPE dogtap_intake_backpressure_drops_total counter")
		fmt.Fprintf(&b, "dogtap_intake_backpressure_drops_total %d\n", a.safety.backpressureDrops.Load())
		fmt.Fprintln(&b, "# HELP dogtap_intake_storage_drops_total Intake payloads dropped because Dogtap storage failed.")
		fmt.Fprintln(&b, "# TYPE dogtap_intake_storage_drops_total counter")
		fmt.Fprintf(&b, "dogtap_intake_storage_drops_total %d\n", a.safety.storageDrops.Load())
	}

	stats := a.forwarder.Stats()
	fmt.Fprintln(&b, "# HELP dogtap_forwarding_payloads_total Forwarding payloads handled by Dogtap.")
	fmt.Fprintln(&b, "# TYPE dogtap_forwarding_payloads_total counter")
	fmt.Fprintf(&b, "dogtap_forwarding_payloads_total %d\n", stats.Payloads)
	fmt.Fprintln(&b, "# HELP dogtap_forwarding_attempts_total Forwarding HTTP attempts made by Dogtap.")
	fmt.Fprintln(&b, "# TYPE dogtap_forwarding_attempts_total counter")
	fmt.Fprintf(&b, "dogtap_forwarding_attempts_total %d\n", stats.Attempts)
	fmt.Fprintln(&b, "# HELP dogtap_forwarding_retries_total Forwarding retries made by Dogtap.")
	fmt.Fprintln(&b, "# TYPE dogtap_forwarding_retries_total counter")
	fmt.Fprintf(&b, "dogtap_forwarding_retries_total %d\n", stats.Retries)
	fmt.Fprintln(&b, "# HELP dogtap_forwarding_successes_total Successful forwarded payloads.")
	fmt.Fprintln(&b, "# TYPE dogtap_forwarding_successes_total counter")
	fmt.Fprintf(&b, "dogtap_forwarding_successes_total %d\n", stats.Successes)
	fmt.Fprintln(&b, "# HELP dogtap_forwarding_failures_total Forwarding failures.")
	fmt.Fprintln(&b, "# TYPE dogtap_forwarding_failures_total counter")
	fmt.Fprintf(&b, "dogtap_forwarding_failures_total %d\n", stats.Failures)
	fmt.Fprintln(&b, "# HELP dogtap_forwarding_drops_total Forwarding drops after validation or retry policy.")
	fmt.Fprintln(&b, "# TYPE dogtap_forwarding_drops_total counter")
	fmt.Fprintf(&b, "dogtap_forwarding_drops_total %d\n", stats.Drops)
	return b.String()
}

func (a *App) handleCreateDebugBundle(w http.ResponseWriter, r *http.Request) {
	var req bundle.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid debug bundle filter"})
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = a.cfg.Storage.MaxEvents
	}
	events, err := a.store.List(r.Context(), store.Query{
		Source:      req.Source,
		PayloadKind: req.PayloadKind,
		Service:     req.Service,
		Env:         req.Env,
		UserID:      req.UserID,
		AccountID:   req.AccountID,
		WorkspaceID: req.WorkspaceID,
		CaseID:      req.CaseID,
		TraceID:     req.TraceID,
		SessionID:   req.SessionID,
		ViewID:      req.ViewID,
		Route:       req.Route,
		Status:      req.Status,
		Limit:       limit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Limit = limit
	writeJSON(w, http.StatusOK, bundle.New(req, events))
}

func (a *App) ingest(ctx context.Context, e event.EventEnvelope) error {
	admission, release := a.safety.admit()
	if !admission.Accepted {
		if a.dropStatus(admission.Reason) == http.StatusAccepted {
			return nil
		}
		return fmt.Errorf("dogtap intake %s", admission.Reason)
	}
	defer release()
	if sampled := a.safety.sample(); !sampled.Accepted {
		return nil
	}
	e.Validation = a.validator.Validate(e)
	if err := a.store.Add(ctx, e); err != nil {
		if a.safety != nil {
			a.safety.storageDrops.Add(1)
		}
		switch a.cfg.Mode {
		case config.ModeForward, config.ModeTee, config.ModeRedactOnly:
			return nil
		default:
			return err
		}
	}
	return nil
}

func dashboardHandler() (http.Handler, error) {
	dist, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(dist, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
