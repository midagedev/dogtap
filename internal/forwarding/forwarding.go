package forwarding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/midagedev/dogtap/internal/event"
)

type Kind string

const (
	KindRUM    Kind = "rum"
	KindReplay Kind = "replay"
	KindLogs   Kind = "logs"
)

type Mode string

const (
	ModeDisabled   Mode = "disabled"
	ModeForward    Mode = "forward"
	ModeTee        Mode = "tee"
	ModeRedactOnly Mode = "redact-only"
)

const (
	defaultSite        = "datadoghq.com"
	defaultTimeout     = 5 * time.Second
	defaultUserAgent   = "dogtap-forwarder"
	hardMaxAttempts    = 5
	defaultMaxAttempts = 1
)

type Config struct {
	Mode          Mode
	Site          string
	APIKey        string
	TargetBaseURL string
	Retry         RetryPolicy
	Timeout       time.Duration
	UserAgent     string
	Client        *http.Client
}

type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

type Payload struct {
	Kind        Kind
	Body        []byte
	Header      http.Header
	ForwardPath string
}

type Stats struct {
	Payloads  int64
	Attempts  int64
	Retries   int64
	Successes int64
	Failures  int64
	Drops     int64
}

type Forwarder struct {
	cfg       Config
	client    *http.Client
	payloads  atomic.Int64
	attempts  atomic.Int64
	retries   atomic.Int64
	successes atomic.Int64
	failures  atomic.Int64
	drops     atomic.Int64
}

func New(cfg Config) (*Forwarder, error) {
	if cfg.Mode == "" {
		cfg.Mode = ModeDisabled
	}
	switch cfg.Mode {
	case ModeDisabled, ModeForward, ModeTee, ModeRedactOnly:
	default:
		return nil, fmt.Errorf("unsupported forwarding mode %q", cfg.Mode)
	}
	if strings.TrimSpace(cfg.Site) == "" {
		cfg.Site = defaultSite
	}
	cfg.Site = normalizeSite(cfg.Site)
	cfg.Retry.MaxAttempts = boundedAttempts(cfg.Retry.MaxAttempts)
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = defaultUserAgent
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Forwarder{cfg: cfg, client: client}, nil
}

func (f *Forwarder) Forward(ctx context.Context, payload Payload) event.ForwardingResult {
	if f.cfg.Mode == ModeDisabled || f.cfg.Mode == ModeRedactOnly {
		return event.ForwardingResult{
			Mode:      string(f.cfg.Mode),
			Attempted: false,
			Status:    "disabled",
		}
	}

	start := time.Now()
	result := event.ForwardingResult{
		Mode:      string(f.cfg.Mode),
		Attempted: true,
		Status:    "dropped",
	}
	f.payloads.Add(1)

	target, err := f.target(payload)
	if err != nil {
		return f.dropResult(result, start, "invalid_target", err.Error(), 0)
	}
	result.Target = safeTarget(target)

	if payload.Kind == KindLogs && strings.TrimSpace(f.cfg.APIKey) == "" {
		return f.dropResult(result, start, "missing_api_key", "datadog api key is required for logs forwarding", 0)
	}

	var lastStatus int
	var lastClass string
	var lastMessage string
	for attempt := 1; attempt <= f.cfg.Retry.MaxAttempts; attempt++ {
		if attempt > 1 {
			f.retries.Add(1)
			result.RetryCount = attempt - 1
			if err := sleep(ctx, f.cfg.Retry.Backoff); err != nil {
				return f.dropResult(result, start, "context_canceled", "forwarding context canceled", lastStatus)
			}
		}

		req, err := f.newRequest(ctx, target, payload)
		if err != nil {
			return f.dropResult(result, start, "build_request", "could not build forwarding request", 0)
		}

		f.attempts.Add(1)
		resp, err := f.client.Do(req)
		if err != nil {
			lastClass = "request_error"
			lastMessage = safeRequestError(ctx, err)
			if attempt < f.cfg.Retry.MaxAttempts && retryableError(ctx, err) {
				continue
			}
			return f.dropResult(result, start, lastClass, lastMessage, 0)
		}

		lastStatus = resp.StatusCode
		drainAndClose(resp.Body)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			f.successes.Add(1)
			result.Status = "success"
			result.StatusCode = resp.StatusCode
			result.DurationMS = durationMS(start)
			return result
		}

		lastClass = "upstream_status"
		lastMessage = fmt.Sprintf("upstream returned HTTP %d", resp.StatusCode)
		if attempt < f.cfg.Retry.MaxAttempts && retryableStatus(resp.StatusCode) {
			continue
		}
		return f.dropResult(result, start, lastClass, lastMessage, resp.StatusCode)
	}

	return f.dropResult(result, start, lastClass, lastMessage, lastStatus)
}

func (f *Forwarder) Stats() Stats {
	return Stats{
		Payloads:  f.payloads.Load(),
		Attempts:  f.attempts.Load(),
		Retries:   f.retries.Load(),
		Successes: f.successes.Load(),
		Failures:  f.failures.Load(),
		Drops:     f.drops.Load(),
	}
}

func (f *Forwarder) target(payload Payload) (*url.URL, error) {
	raw := f.cfg.TargetBaseURL
	if raw == "" {
		var err error
		raw, err = DatadogTarget(payload.Kind, f.cfg.Site)
		if err != nil {
			return nil, err
		}
	}
	target, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse target: %w", err)
	}
	if target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("target must include scheme and host")
	}
	if target.Path == "" || target.Path == "/" {
		path, err := defaultPath(payload.Kind)
		if err != nil {
			return nil, err
		}
		target.Path = path
	}
	target.RawQuery = ""
	target.User = nil
	if payload.ForwardPath != "" {
		forwarded, err := safeForwardPath(payload.Kind, payload.ForwardPath)
		if err != nil {
			return nil, err
		}
		target.Path = forwarded.Path
		target.RawQuery = forwarded.RawQuery
	}
	return target, nil
}

func (f *Forwarder) newRequest(ctx context.Context, target *url.URL, payload Payload) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(payload.Body))
	if err != nil {
		return nil, err
	}
	copyForwardHeaders(req.Header, payload.Header)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", f.cfg.UserAgent)
	if payload.Kind == KindLogs {
		req.Header.Set("DD-API-KEY", f.cfg.APIKey)
	}
	return req, nil
}

func (f *Forwarder) dropResult(result event.ForwardingResult, start time.Time, class string, message string, statusCode int) event.ForwardingResult {
	f.failures.Add(1)
	f.drops.Add(1)
	result.Status = "dropped"
	result.StatusCode = statusCode
	result.DurationMS = durationMS(start)
	result.ErrorClass = class
	result.ErrorMessage = message
	return result
}

func DatadogTarget(kind Kind, site string) (string, error) {
	site = normalizeSite(site)
	if site == "" {
		site = defaultSite
	}
	switch kind {
	case KindRUM:
		return "https://browser-intake-" + site + "/api/v2/rum", nil
	case KindReplay:
		return "https://browser-intake-" + site + "/api/v2/replay", nil
	case KindLogs:
		return "https://http-intake.logs." + site + "/api/v2/logs", nil
	default:
		return "", fmt.Errorf("unsupported forwarding kind %q", kind)
	}
}

func defaultPath(kind Kind) (string, error) {
	switch kind {
	case KindRUM:
		return "/api/v2/rum", nil
	case KindReplay:
		return "/api/v2/replay", nil
	case KindLogs:
		return "/api/v2/logs", nil
	default:
		return "", fmt.Errorf("unsupported forwarding kind %q", kind)
	}
}

func safeForwardPath(kind Kind, raw string) (*url.URL, error) {
	forwarded, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse ddforward: %w", err)
	}
	if forwarded.Scheme != "" || forwarded.Host != "" {
		return nil, fmt.Errorf("ddforward must be a relative Datadog intake path")
	}
	wantPath, err := defaultPath(kind)
	if err != nil {
		return nil, err
	}
	if forwarded.Path != wantPath {
		return nil, fmt.Errorf("ddforward path %q does not match %q", forwarded.Path, wantPath)
	}
	forwarded.User = nil
	forwarded.Fragment = ""
	return forwarded, nil
}

func copyForwardHeaders(dst, src http.Header) {
	for _, name := range []string{
		"Content-Type",
		"Content-Encoding",
		"Accept",
		"DD-EVP-ORIGIN",
		"DD-EVP-ORIGIN-VERSION",
	} {
		if values := src.Values(name); len(values) > 0 {
			dst[name] = append([]string(nil), values...)
		}
	}
}

func boundedAttempts(configured int) int {
	if configured <= 0 {
		return defaultMaxAttempts
	}
	if configured > hardMaxAttempts {
		return hardMaxAttempts
	}
	return configured
}

func normalizeSite(site string) string {
	site = strings.TrimSpace(site)
	site = strings.TrimPrefix(site, "https://")
	site = strings.TrimPrefix(site, "http://")
	return strings.Trim(site, "/")
}

func safeTarget(target *url.URL) string {
	copy := *target
	copy.RawQuery = ""
	copy.User = nil
	return copy.String()
}

func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

func retryableError(ctx context.Context, err error) bool {
	return err != nil && ctx.Err() == nil
}

func safeRequestError(ctx context.Context, err error) string {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return "forwarding context canceled"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "forwarding context deadline exceeded"
	default:
		return "forwarding request failed"
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func drainAndClose(body io.ReadCloser) {
	defer body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4096))
}

func durationMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
