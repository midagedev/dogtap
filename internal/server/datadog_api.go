package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/store"
)

const defaultDatadogSearchLimit = 10

var metricExpressionRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*:([^{}]+)(?:\{([^}]*)\})?`)

type datadogSearchRequest struct {
	Filter datadogQueryFilter `json:"filter"`
	Page   datadogPage        `json:"page"`
	Sort   string             `json:"sort"`
	Data   struct {
		Attributes struct {
			Filter datadogQueryFilter `json:"filter"`
			Page   datadogPage        `json:"page"`
			Sort   string             `json:"sort"`
		} `json:"attributes"`
	} `json:"data"`
}

type datadogQueryFilter struct {
	Query string `json:"query"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type datadogPage struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}

type datadogSearchResponse struct {
	Data  []datadogEvent `json:"data"`
	Links map[string]any `json:"links,omitempty"`
	Meta  datadogMeta    `json:"meta"`
}

type datadogMeta struct {
	Elapsed   int            `json:"elapsed"`
	Page      map[string]any `json:"page"`
	RequestID string         `json:"request_id"`
	Status    string         `json:"status"`
	Warnings  []datadogWarn  `json:"warnings,omitempty"`
}

type datadogWarn struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type datadogEvent struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes"`
}

func (a *App) handleDatadogLogsSearch(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeDatadogSearchRequest(w, r)
	if !ok {
		return
	}
	events, err := a.store.List(r.Context(), store.Query{Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	logEvents := make([]event.EventEnvelope, 0, len(events))
	for _, e := range events {
		if e.Source == event.SourceLogs || e.PayloadKind == "log" {
			logEvents = append(logEvents, e)
		}
	}
	writeJSON(w, http.StatusOK, datadogSearchResponseFor(req, filterDatadogEvents(logEvents, req, datadogKindLog), datadogKindLog))
}

func (a *App) handleDatadogRUMSearch(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeDatadogSearchRequest(w, r)
	if !ok {
		return
	}
	events, err := a.store.List(r.Context(), store.Query{Source: event.SourceRUM, Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, datadogSearchResponseFor(req, filterDatadogEvents(events, req, datadogKindRUM), datadogKindRUM))
}

func (a *App) handleDatadogSpansSearch(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeDatadogSearchRequest(w, r)
	if !ok {
		return
	}
	events, err := a.store.List(r.Context(), store.Query{Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	traceEvents := make([]event.EventEnvelope, 0, len(events))
	for _, e := range events {
		if e.Details != nil && e.Details.Trace != nil && len(e.Details.Trace.Spans) > 0 {
			traceEvents = append(traceEvents, e)
		}
	}
	writeJSON(w, http.StatusOK, datadogSearchResponseFor(req, filterDatadogEvents(traceEvents, req, datadogKindSpan), datadogKindSpan))
}

func (a *App) handleDatadogMetricQuery(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string][]string{"errors": {"query is required"}})
		return
	}
	fromSeconds, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	toSeconds, _ := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
	metricName, scope := parseMetricExpression(query)

	events, err := a.store.List(r.Context(), store.Query{PayloadKind: "metric", Limit: a.cfg.Storage.MaxEvents})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	seriesByKey := map[string]*datadogMetricSeries{}
	for _, e := range events {
		if e.Details == nil {
			continue
		}
		for _, metric := range e.Details.Metrics {
			if metricName != "" && metric.Name != metricName {
				continue
			}
			tags := metricTags(metric, e)
			if !tagsMatchScope(tags, scope) {
				continue
			}
			ts := metricTimestampMillis(metric, e)
			if fromSeconds > 0 && ts < fromSeconds*1000 {
				continue
			}
			if toSeconds > 0 && ts > toSeconds*1000 {
				continue
			}
			key := metric.Name + "|" + strings.Join(tags, ",")
			series := seriesByKey[key]
			if series == nil {
				series = &datadogMetricSeries{
					Aggr:        coalesce(metric.Aggregation, "avg"),
					DisplayName: metric.Name,
					Expression:  query,
					Metric:      metric.Name,
					QueryIndex:  0,
					Scope:       strings.Join(tags, ","),
					TagSet:      tags,
					Unit:        datadogMetricUnit(metric.Unit),
				}
				seriesByKey[key] = series
			}
			series.Pointlist = append(series.Pointlist, []any{float64(ts), metric.Value})
			series.DogtapEventIDs = appendUniqueString(series.DogtapEventIDs, e.ID)
		}
	}

	series := make([]datadogMetricSeries, 0, len(seriesByKey))
	for _, item := range seriesByKey {
		normalizeMetricSeries(item)
		series = append(series, *item)
	}
	slices.SortFunc(series, func(a, b datadogMetricSeries) int {
		return strings.Compare(a.Metric+a.Scope, b.Metric+b.Scope)
	})

	writeJSON(w, http.StatusOK, datadogMetricQueryResponse{
		Status:   "ok",
		Message:  "success",
		ResType:  "time_series",
		Query:    query,
		FromDate: fromSeconds * 1000,
		ToDate:   toSeconds * 1000,
		Series:   series,
	})
}

func decodeDatadogSearchRequest(w http.ResponseWriter, r *http.Request) (datadogSearchRequest, bool) {
	var req datadogSearchRequest
	if r.Body == nil || r.ContentLength == 0 {
		return req, true
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string][]string{"errors": {"invalid search request"}})
		return datadogSearchRequest{}, false
	}
	if req.Filter.Query == "" && req.Data.Attributes.Filter.Query != "" {
		req.Filter = req.Data.Attributes.Filter
	}
	if req.Page.Limit == 0 && req.Data.Attributes.Page.Limit != 0 {
		req.Page = req.Data.Attributes.Page
	}
	if req.Sort == "" && req.Data.Attributes.Sort != "" {
		req.Sort = req.Data.Attributes.Sort
	}
	return req, true
}

type datadogEventKind string

const (
	datadogKindLog  datadogEventKind = "log"
	datadogKindRUM  datadogEventKind = "rum"
	datadogKindSpan datadogEventKind = "span"
)

func filterDatadogEvents(events []event.EventEnvelope, req datadogSearchRequest, kind datadogEventKind) []event.EventEnvelope {
	limit := req.Page.Limit
	if limit <= 0 {
		limit = defaultDatadogSearchLimit
	}
	if limit > len(events) && len(events) > 0 {
		limit = len(events)
	}
	out := make([]event.EventEnvelope, 0, limit)
	for _, e := range events {
		if !matchesDatadogQuery(e, req.Filter.Query, kind) {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	if req.Sort == "timestamp" {
		slices.Reverse(out)
	}
	return out
}

func datadogSearchResponseFor(req datadogSearchRequest, events []event.EventEnvelope, kind datadogEventKind) datadogSearchResponse {
	data := make([]datadogEvent, 0, len(events))
	for _, e := range events {
		switch kind {
		case datadogKindLog:
			data = append(data, datadogLogEvent(e))
		case datadogKindRUM:
			data = append(data, datadogRUMEvent(e))
		case datadogKindSpan:
			data = append(data, datadogSpanEvents(e)...)
		}
	}
	limit := req.Page.Limit
	if limit <= 0 {
		limit = defaultDatadogSearchLimit
	}
	if len(data) > limit {
		data = data[:limit]
	}
	return datadogSearchResponse{
		Data: data,
		Meta: datadogMeta{
			Elapsed:   0,
			Page:      map[string]any{},
			RequestID: "dogtap-local",
			Status:    "done",
			Warnings: []datadogWarn{{
				Code:   "dogtap_compat_subset",
				Title:  "Dogtap implements a local read-only Datadog API compatibility subset.",
				Detail: "Advanced Datadog query syntax, pagination cursors, indexes, permissions, and long-term retention are not emulated.",
			}},
		},
	}
}

func datadogLogEvent(e event.EventEnvelope) datadogEvent {
	n := e.Normalized
	message := n.ErrorMessage
	status := "info"
	var log event.LogEntry
	if e.Details != nil && len(e.Details.Logs) > 0 {
		log = e.Details.Logs[0]
		message = log.Message
		status = strings.ToLower(log.Level)
	}
	if message == "" {
		message = "dogtap log event"
	}
	return datadogEvent{
		Type: "log",
		ID:   e.ID,
		Attributes: map[string]any{
			"timestamp":  timestampString(e),
			"service":    n.Service,
			"host":       n.Host,
			"status":     status,
			"message":    message,
			"tags":       datadogTags(e),
			"attributes": datadogLogAttributes(e, log),
		},
	}
}

func datadogLogAttributes(e event.EventEnvelope, log event.LogEntry) map[string]any {
	n := e.Normalized
	attrs := map[string]any{
		"env":               coalesce(log.Env, n.Env),
		"version":           coalesce(log.Version, n.Version),
		"trace_id":          coalesce(log.TraceID, n.TraceID),
		"span_id":           coalesce(log.SpanID, n.SpanID),
		"route":             coalesce(log.Route, n.Route),
		"source":            e.Source,
		"endpoint":          e.Endpoint,
		"payload_kind":      e.PayloadKind,
		"validation":        e.Validation,
		"validation.status": e.Validation.Status,
		"dogtap_id":         e.ID,
		"dogtap.id":         e.ID,
	}
	addAttributeString(attrs, "service", coalesce(log.Service, n.Service))
	addAttributeString(attrs, "host", n.Host)
	addAttributeString(attrs, "method", coalesce(log.Method, n.Method))
	addAttributeString(attrs, "http.method", coalesce(log.Method, n.Method))
	addAttributeInt(attrs, "status_code", firstPositive(log.StatusCode, n.StatusCode))
	addAttributeInt(attrs, "http.status_code", firstPositive(log.StatusCode, n.StatusCode))
	addAttributeString(attrs, "usr.id", coalesce(log.UserID, n.UserID))
	addAttributeString(attrs, "account.id", coalesce(log.AccountID, n.AccountID))
	addAttributeString(attrs, "workspace.id", coalesce(log.WorkspaceID, n.WorkspaceID))
	addAttributeString(attrs, "case.id", coalesce(log.CaseID, n.CaseID))
	addAttributeString(attrs, "request_id", log.RequestID)
	addAttributeString(attrs, "correlation_id", log.CorrelationID)
	return attrs
}

func addAttributeString(attrs map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		attrs[key] = value
	}
}

func addAttributeInt(attrs map[string]any, key string, value int) {
	if value > 0 {
		attrs[key] = value
	}
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func datadogRUMEvent(e event.EventEnvelope) datadogEvent {
	n := e.Normalized
	rumType := e.PayloadKind
	if rumType == "" {
		rumType = "event"
	}
	return datadogEvent{
		Type: "rum",
		ID:   e.ID,
		Attributes: map[string]any{
			"timestamp": timestampString(e),
			"type":      rumType,
			"service":   n.Service,
			"tags":      datadogTags(e),
			"session": map[string]any{
				"id": n.SessionID,
			},
			"view": map[string]any{
				"id":   n.ViewID,
				"name": n.Route,
				"url":  n.Route,
			},
			"usr": map[string]any{
				"id": n.UserID,
			},
			"context": map[string]any{
				"account":   map[string]any{"id": n.AccountID},
				"workspace": map[string]any{"id": n.WorkspaceID},
				"case":      map[string]any{"id": n.CaseID},
				"trace_id":  n.TraceID,
				"route":     n.Route,
				"dogtap_id": e.ID,
			},
		},
	}
}

func datadogSpanEvents(e event.EventEnvelope) []datadogEvent {
	if e.Details == nil || e.Details.Trace == nil {
		return nil
	}
	out := make([]datadogEvent, 0, len(e.Details.Trace.Spans))
	for i, span := range e.Details.Trace.Spans {
		id := e.ID
		if span.SpanID != "" {
			id = e.ID + ":" + span.SpanID
		} else if i > 0 {
			id = e.ID + ":" + strconv.Itoa(i)
		}
		out = append(out, datadogEvent{
			Type: "span",
			ID:   id,
			Attributes: map[string]any{
				"timestamp":     timestampString(e),
				"service":       coalesce(span.Service, e.Normalized.Service),
				"name":          span.Name,
				"resource_name": span.Resource,
				"trace_id":      coalesce(span.TraceID, e.Normalized.TraceID),
				"span_id":       coalesce(span.SpanID, e.Normalized.SpanID),
				"parent_id":     coalesce(span.ParentSpanID, e.Normalized.ParentSpanID),
				"duration":      span.DurationMS,
				"error":         span.Error,
				"tags":          datadogTags(e),
				"meta": map[string]any{
					"env":        e.Normalized.Env,
					"version":    e.Normalized.Version,
					"route":      e.Normalized.Route,
					"source":     e.Source,
					"endpoint":   e.Endpoint,
					"validation": e.Validation.Status,
					"dogtap_id":  e.ID,
				},
			},
		})
	}
	return out
}

func matchesDatadogQuery(e event.EventEnvelope, query string, kind datadogEventKind) bool {
	query = strings.TrimSpace(query)
	if query == "" || query == "*" {
		return true
	}
	for _, token := range datadogQueryTokens(query) {
		token = trimDatadogQueryToken(token)
		if token == "" || strings.EqualFold(token, "AND") {
			continue
		}
		key, value, ok := strings.Cut(token, ":")
		if !ok {
			if !strings.Contains(strings.ToLower(datadogSearchText(e, kind)), strings.ToLower(datadogUnquoteQueryValue(token))) {
				return false
			}
			continue
		}
		key = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(key)), "@")
		value = datadogUnquoteQueryValue(value)
		if value == "*" || value == "" {
			continue
		}
		if !datadogFieldMatches(e, kind, key, value) {
			return false
		}
	}
	return true
}

func datadogFieldMatches(e event.EventEnvelope, kind datadogEventKind, key string, value string) bool {
	n := e.Normalized
	candidates := []string{}
	switch key {
	case "service":
		candidates = append(candidates, n.Service)
	case "env":
		candidates = append(candidates, n.Env)
	case "version":
		candidates = append(candidates, n.Version)
	case "host":
		candidates = append(candidates, n.Host)
	case "trace_id", "trace.id", "traceid", "dd.trace_id":
		candidates = append(candidates, n.TraceID)
	case "span_id", "span.id", "spanid", "dd.span_id":
		candidates = append(candidates, n.SpanID)
	case "session.id", "session_id", "sessionid":
		candidates = append(candidates, n.SessionID)
	case "view.id", "view_id", "viewid":
		candidates = append(candidates, n.ViewID)
	case "usr.id", "user.id", "user_id", "userid":
		candidates = append(candidates, n.UserID)
	case "account.id", "account_id", "accountid":
		candidates = append(candidates, n.AccountID)
	case "workspace.id", "workspace_id", "workspaceid":
		candidates = append(candidates, n.WorkspaceID)
	case "case.id", "case_id", "caseid":
		candidates = append(candidates, n.CaseID)
	case "route", "http.route", "resource_name", "resource.name":
		candidates = append(candidates, n.Route)
	case "method", "http.method", "http.request.method":
		candidates = append(candidates, n.Method, e.Method)
	case "status_code", "http.status_code", "http.response.status_code":
		if n.StatusCode > 0 {
			candidates = append(candidates, strconv.Itoa(n.StatusCode))
		}
	case "endpoint":
		candidates = append(candidates, e.Endpoint)
	case "source":
		candidates = append(candidates, string(e.Source))
	case "type", "payload_kind":
		candidates = append(candidates, e.PayloadKind)
	case "status", "validation.status":
		candidates = append(candidates, e.Validation.Status)
		if kind == datadogKindLog && e.Details != nil {
			for _, log := range e.Details.Logs {
				candidates = append(candidates, strings.ToLower(log.Level))
			}
		}
	case "dogtap.id", "dogtap_id":
		candidates = append(candidates, e.ID)
	case "*":
		candidates = append(candidates, datadogSearchText(e, kind))
	case "request_id", "request.id":
		if kind == datadogKindLog && e.Details != nil {
			for _, log := range e.Details.Logs {
				candidates = append(candidates, log.RequestID)
			}
		}
	case "correlation_id", "correlation.id":
		if kind == datadogKindLog && e.Details != nil {
			for _, log := range e.Details.Logs {
				candidates = append(candidates, log.CorrelationID)
			}
		}
	}
	if kind == datadogKindLog && e.Details != nil {
		for _, log := range e.Details.Logs {
			candidates = append(candidates, datadogLogFieldCandidates(log, key)...)
		}
	}
	if kind == datadogKindSpan && e.Details != nil && e.Details.Trace != nil {
		for _, span := range e.Details.Trace.Spans {
			candidates = append(candidates, span.TraceID, span.SpanID, span.ParentSpanID, span.Service, span.Name, span.Resource)
		}
	}
	for _, candidate := range candidates {
		if datadogCandidateMatches(key, candidate, value) {
			return true
		}
	}
	return false
}

func datadogLogFieldCandidates(log event.LogEntry, key string) []string {
	switch key {
	case "service":
		return []string{log.Service}
	case "env":
		return []string{log.Env}
	case "version":
		return []string{log.Version}
	case "trace_id", "trace.id", "traceid", "dd.trace_id":
		return []string{log.TraceID}
	case "span_id", "span.id", "spanid", "dd.span_id":
		return []string{log.SpanID}
	case "usr.id", "user.id", "user_id", "userid":
		return []string{log.UserID}
	case "account.id", "account_id", "accountid":
		return []string{log.AccountID}
	case "workspace.id", "workspace_id", "workspaceid":
		return []string{log.WorkspaceID}
	case "case.id", "case_id", "caseid":
		return []string{log.CaseID}
	case "route", "http.route", "resource_name", "resource.name":
		return []string{log.Route}
	case "method", "http.method", "http.request.method":
		return []string{log.Method}
	case "status_code", "http.status_code", "http.response.status_code":
		if log.StatusCode > 0 {
			return []string{strconv.Itoa(log.StatusCode)}
		}
	case "request_id", "request.id":
		return []string{log.RequestID}
	case "correlation_id", "correlation.id":
		return []string{log.CorrelationID}
	}
	return nil
}

func datadogCandidateMatches(key string, candidate string, want string) bool {
	if datadogValueMatches(candidate, want) {
		return true
	}
	if isTraceIDField(key) {
		return datadogTraceIDMatches(candidate, want)
	}
	return false
}

func isTraceIDField(key string) bool {
	switch key {
	case "trace_id", "trace.id", "traceid", "dd.trace_id":
		return true
	default:
		return false
	}
}

func datadogValueMatches(candidate string, want string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	if strings.HasSuffix(want, "*") {
		return strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(strings.TrimSuffix(want, "*")))
	}
	return strings.EqualFold(candidate, want)
}

func datadogTraceIDMatches(candidate string, want string) bool {
	candidate = cleanTraceID(candidate)
	want = cleanTraceID(want)
	if candidate == "" || want == "" {
		return false
	}
	if strings.EqualFold(candidate, want) {
		return true
	}
	if trimLeadingTraceZeros(candidate) == trimLeadingTraceZeros(want) {
		return true
	}
	if candidateDecimal, ok := parseDecimalTraceID(candidate); ok {
		if lowHex, ok := lowTraceHex(want); ok {
			return candidateDecimal == lowHex
		}
	}
	if wantDecimal, ok := parseDecimalTraceID(want); ok {
		if lowHex, ok := lowTraceHex(candidate); ok {
			return wantDecimal == lowHex
		}
	}
	return false
}

func cleanTraceID(value string) string {
	value = strings.Trim(strings.ToLower(strings.TrimSpace(value)), `"`)
	value = strings.TrimPrefix(value, "0x")
	return value
}

func trimLeadingTraceZeros(value string) string {
	trimmed := strings.TrimLeft(value, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

func parseDecimalTraceID(value string) (uint64, bool) {
	if value == "" {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil
}

func lowTraceHex(value string) (uint64, bool) {
	if value == "" {
		return 0, false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return 0, false
		}
	}
	if len(value) > 16 {
		value = value[len(value)-16:]
	}
	parsed, err := strconv.ParseUint(value, 16, 64)
	return parsed, err == nil
}

func datadogSearchText(e event.EventEnvelope, kind datadogEventKind) string {
	parts := []string{e.ID, string(e.Source), e.PayloadKind, e.Endpoint, e.Normalized.Service, e.Normalized.Env, e.Normalized.Route, e.Normalized.TraceID, e.Normalized.SessionID, e.Normalized.UserID, e.Normalized.ErrorMessage}
	if kind == datadogKindLog && e.Details != nil {
		for _, log := range e.Details.Logs {
			parts = append(parts, log.Message, log.Level, log.TraceID, log.SpanID, log.Route, log.Method, log.RequestID, log.CorrelationID)
		}
	}
	if kind == datadogKindSpan && e.Details != nil && e.Details.Trace != nil {
		for _, span := range e.Details.Trace.Spans {
			parts = append(parts, span.Name, span.Resource, span.Service, span.TraceID, span.SpanID)
		}
	}
	return strings.Join(parts, " ")
}

func datadogQueryTokens(query string) []string {
	tokens := []string{}
	var current strings.Builder
	inQuote := false
	escaped := false
	flush := func() {
		token := strings.TrimSpace(current.String())
		if token != "" {
			tokens = append(tokens, token)
		}
		current.Reset()
	}

	for _, r := range query {
		if escaped {
			current.WriteRune('\\')
			current.WriteRune(r)
			escaped = false
			continue
		}
		switch {
		case r == '\\':
			escaped = true
		case r == '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case unicode.IsSpace(r) && !inQuote:
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	flush()
	return tokens
}

func trimDatadogQueryToken(token string) string {
	token = strings.TrimSpace(token)
	for strings.HasPrefix(token, "(") && !strings.HasPrefix(token, `"`) {
		token = strings.TrimSpace(strings.TrimPrefix(token, "("))
	}
	for strings.HasSuffix(token, ")") && datadogTrailingParenIsOutsideQuote(token) {
		token = strings.TrimSpace(strings.TrimSuffix(token, ")"))
	}
	return token
}

func datadogTrailingParenIsOutsideQuote(token string) bool {
	inQuote := false
	escaped := false
	runes := []rune(token)
	for i, r := range runes {
		if i == len(runes)-1 {
			break
		}
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inQuote = !inQuote
		}
	}
	return !inQuote
}

func datadogUnquoteQueryValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return datadogUnescapeQueryValue(value)
}

func datadogUnescapeQueryValue(value string) string {
	var out strings.Builder
	escaped := false
	for _, r := range value {
		if escaped {
			out.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		out.WriteRune(r)
	}
	if escaped {
		out.WriteRune('\\')
	}
	return out.String()
}

func datadogTags(e event.EventEnvelope) []string {
	n := e.Normalized
	tags := []string{}
	addTag := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			tags = append(tags, key+":"+value)
		}
	}
	addTag("service", n.Service)
	addTag("env", n.Env)
	addTag("version", n.Version)
	addTag("host", n.Host)
	addTag("source", string(e.Source))
	addTag("route", n.Route)
	addTag("trace_id", n.TraceID)
	addTag("span_id", n.SpanID)
	addTag("session_id", n.SessionID)
	addTag("user_id", n.UserID)
	addTag("account_id", n.AccountID)
	addTag("workspace_id", n.WorkspaceID)
	addTag("case_id", n.CaseID)
	slices.Sort(tags)
	return tags
}

func timestampString(e event.EventEnvelope) string {
	if e.Normalized.Timestamp != "" {
		return e.Normalized.Timestamp
	}
	if !e.ReceivedAt.IsZero() {
		return e.ReceivedAt.UTC().Format(time.RFC3339Nano)
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

type datadogMetricQueryResponse struct {
	Status   string                `json:"status"`
	Message  string                `json:"message"`
	ResType  string                `json:"res_type"`
	Query    string                `json:"query"`
	FromDate int64                 `json:"from_date,omitempty"`
	ToDate   int64                 `json:"to_date,omitempty"`
	Series   []datadogMetricSeries `json:"series"`
}

type datadogMetricSeries struct {
	Aggr           string   `json:"aggr,omitempty"`
	DisplayName    string   `json:"display_name"`
	DogtapEventIDs []string `json:"dogtap_event_ids,omitempty"`
	End            int64    `json:"end,omitempty"`
	Expression     string   `json:"expression"`
	Interval       int64    `json:"interval,omitempty"`
	Length         int      `json:"length"`
	Metric         string   `json:"metric"`
	Pointlist      [][]any  `json:"pointlist"`
	QueryIndex     int      `json:"query_index"`
	Scope          string   `json:"scope,omitempty"`
	Start          int64    `json:"start,omitempty"`
	TagSet         []string `json:"tag_set,omitempty"`
	Unit           []ddUnit `json:"unit,omitempty"`
}

type ddUnit struct {
	Family    string `json:"family,omitempty"`
	Name      string `json:"name,omitempty"`
	Plural    string `json:"plural,omitempty"`
	ScaleFact int    `json:"scale_factor,omitempty"`
	ShortName string `json:"short_name,omitempty"`
}

func parseMetricExpression(query string) (string, map[string]string) {
	matches := metricExpressionRE.FindStringSubmatch(strings.TrimSpace(query))
	if len(matches) == 0 {
		return strings.TrimSpace(query), nil
	}
	scope := map[string]string{}
	for _, raw := range splitDatadogCommaList(matches[2]) {
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "*" {
			continue
		}
		key, value, ok := strings.Cut(raw, ":")
		if ok {
			scope[strings.TrimSpace(key)] = datadogUnquoteQueryValue(value)
		}
	}
	return strings.TrimSpace(matches[1]), scope
}

func splitDatadogCommaList(value string) []string {
	parts := []string{}
	var current strings.Builder
	inQuote := false
	escaped := false
	flush := func() {
		parts = append(parts, current.String())
		current.Reset()
	}
	for _, r := range value {
		if escaped {
			current.WriteRune('\\')
			current.WriteRune(r)
			escaped = false
			continue
		}
		switch {
		case r == '\\':
			escaped = true
		case r == '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case r == ',' && !inQuote:
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	flush()
	return parts
}

func metricTags(metric event.MetricEntry, e event.EventEnvelope) []string {
	seen := map[string]string{}
	add := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			seen[key] = value
		}
	}
	for key, value := range metric.Tags {
		add(key, value)
	}
	add("service", coalesce(metric.Service, e.Normalized.Service))
	add("service.name", coalesce(metric.Tags["service.name"], metric.Service, e.Normalized.Service))
	add("env", e.Normalized.Env)
	add("deployment.environment", coalesce(metric.Tags["deployment.environment"], e.Normalized.Env))
	add("version", e.Normalized.Version)
	add("service.version", coalesce(metric.Tags["service.version"], e.Normalized.Version))
	route := coalesce(metric.Route, metric.Tags["http.route"], metric.Tags["route"], e.Normalized.Route)
	add("route", route)
	add("http.route", route)
	add("resource.name", coalesce(metric.Tags["resource.name"], route))
	method := coalesce(metric.Tags["http.request.method"], metric.Tags["http.method"], metric.Tags["method"], e.Normalized.Method)
	add("method", method)
	add("http.method", method)
	add("http.request.method", method)
	status := coalesce(metric.Tags["http.response.status_code"], metric.Tags["http.status_code"], metric.Tags["status_code"])
	if status == "" && e.Normalized.StatusCode > 0 {
		status = strconv.Itoa(e.Normalized.StatusCode)
	}
	add("status_code", status)
	add("http.status_code", status)
	add("http.response.status_code", status)
	tags := make([]string, 0, len(seen))
	for key, value := range seen {
		tags = append(tags, key+":"+value)
	}
	slices.Sort(tags)
	return tags
}

func appendUniqueString(values []string, next string) []string {
	if next == "" {
		return values
	}
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func tagsMatchScope(tags []string, scope map[string]string) bool {
	if len(scope) == 0 {
		return true
	}
	available := map[string]string{}
	for _, tag := range tags {
		key, value, ok := strings.Cut(tag, ":")
		if ok {
			available[key] = value
		}
	}
	for key, want := range scope {
		if want == "*" {
			continue
		}
		if !datadogValueMatches(available[key], want) {
			return false
		}
	}
	return true
}

func metricTimestampMillis(metric event.MetricEntry, e event.EventEnvelope) int64 {
	for _, raw := range []string{metric.Timestamp, e.Normalized.Timestamp} {
		if ts := parseTimestampMillis(raw); ts > 0 {
			return ts
		}
	}
	if !e.ReceivedAt.IsZero() {
		return e.ReceivedAt.UnixMilli()
	}
	return time.Now().UTC().UnixMilli()
}

// Datadog-compatible metric responses use millisecond timestamps. Dogtap can
// receive OTLP JSON nanoseconds, Unix milliseconds/seconds, or normalized
// RFC3339 timestamps, so this parser keeps the conversion deliberately narrow
// and returns 0 when the unit is ambiguous.
func parseTimestampMillis(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
		switch {
		case parsed > 1_000_000_000_000_000:
			return parsed / 1_000_000
		case parsed > 1_000_000_000_000:
			return parsed
		case parsed > 1_000_000_000:
			return parsed * 1000
		default:
			return 0
		}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.UnixMilli()
	}
	return 0
}

func normalizeMetricSeries(series *datadogMetricSeries) {
	slices.SortFunc(series.Pointlist, func(a, b []any) int {
		return comparePointTimestamp(a, b)
	})
	series.Length = len(series.Pointlist)
	if len(series.Pointlist) == 0 {
		return
	}
	start, _ := pointTimestamp(series.Pointlist[0])
	end, _ := pointTimestamp(series.Pointlist[len(series.Pointlist)-1])
	series.Start = start
	series.End = end
	if len(series.Pointlist) > 1 {
		next, _ := pointTimestamp(series.Pointlist[1])
		series.Interval = next - start
	}
}

func comparePointTimestamp(a, b []any) int {
	left, _ := pointTimestamp(a)
	right, _ := pointTimestamp(b)
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func pointTimestamp(point []any) (int64, bool) {
	if len(point) == 0 {
		return 0, false
	}
	switch value := point[0].(type) {
	case int64:
		return value, true
	case float64:
		return int64(value), true
	case int:
		return int64(value), true
	default:
		return 0, false
	}
}

func datadogMetricUnit(unit string) []ddUnit {
	if unit == "" {
		return nil
	}
	return []ddUnit{{
		Family:    "custom",
		Name:      unit,
		Plural:    unit,
		ScaleFact: 1,
		ShortName: unit,
	}}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
