package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

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
	if e.Details != nil && len(e.Details.Logs) > 0 {
		message = e.Details.Logs[0].Message
		status = strings.ToLower(e.Details.Logs[0].Level)
	}
	if message == "" {
		message = "dogtap log event"
	}
	return datadogEvent{
		Type: "log",
		ID:   e.ID,
		Attributes: map[string]any{
			"timestamp": timestampString(e),
			"service":   n.Service,
			"host":      n.Host,
			"status":    status,
			"message":   message,
			"tags":      datadogTags(e),
			"attributes": map[string]any{
				"env":        n.Env,
				"version":    n.Version,
				"trace_id":   n.TraceID,
				"span_id":    n.SpanID,
				"route":      n.Route,
				"source":     e.Source,
				"endpoint":   e.Endpoint,
				"validation": e.Validation,
				"dogtap_id":  e.ID,
			},
		},
	}
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
	for _, token := range strings.Fields(query) {
		token = strings.Trim(token, "()")
		if token == "" || strings.EqualFold(token, "AND") {
			continue
		}
		key, value, ok := strings.Cut(token, ":")
		if !ok {
			if !strings.Contains(strings.ToLower(datadogSearchText(e, kind)), strings.ToLower(strings.Trim(token, `"`))) {
				return false
			}
			continue
		}
		key = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(key)), "@")
		value = strings.Trim(strings.TrimSpace(value), `"`)
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
	case "source":
		candidates = append(candidates, string(e.Source))
	case "type":
		candidates = append(candidates, e.PayloadKind)
	case "status":
		candidates = append(candidates, e.Validation.Status)
		if kind == datadogKindLog && e.Details != nil {
			for _, log := range e.Details.Logs {
				candidates = append(candidates, strings.ToLower(log.Level))
			}
		}
	}
	if kind == datadogKindSpan && e.Details != nil && e.Details.Trace != nil {
		for _, span := range e.Details.Trace.Spans {
			candidates = append(candidates, span.TraceID, span.SpanID, span.ParentSpanID, span.Service, span.Name, span.Resource)
		}
	}
	for _, candidate := range candidates {
		if datadogValueMatches(candidate, value) {
			return true
		}
	}
	return false
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

func datadogSearchText(e event.EventEnvelope, kind datadogEventKind) string {
	parts := []string{e.ID, string(e.Source), e.PayloadKind, e.Endpoint, e.Normalized.Service, e.Normalized.Env, e.Normalized.Route, e.Normalized.TraceID, e.Normalized.SessionID, e.Normalized.UserID, e.Normalized.ErrorMessage}
	if kind == datadogKindLog && e.Details != nil {
		for _, log := range e.Details.Logs {
			parts = append(parts, log.Message, log.Level, log.TraceID)
		}
	}
	if kind == datadogKindSpan && e.Details != nil && e.Details.Trace != nil {
		for _, span := range e.Details.Trace.Spans {
			parts = append(parts, span.Name, span.Resource, span.Service, span.TraceID, span.SpanID)
		}
	}
	return strings.Join(parts, " ")
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
	Aggr        string   `json:"aggr,omitempty"`
	DisplayName string   `json:"display_name"`
	End         int64    `json:"end,omitempty"`
	Expression  string   `json:"expression"`
	Interval    int64    `json:"interval,omitempty"`
	Length      int      `json:"length"`
	Metric      string   `json:"metric"`
	Pointlist   [][]any  `json:"pointlist"`
	QueryIndex  int      `json:"query_index"`
	Scope       string   `json:"scope,omitempty"`
	Start       int64    `json:"start,omitempty"`
	TagSet      []string `json:"tag_set,omitempty"`
	Unit        []ddUnit `json:"unit,omitempty"`
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
	for _, raw := range strings.Split(matches[2], ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "*" {
			continue
		}
		key, value, ok := strings.Cut(raw, ":")
		if ok {
			scope[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return strings.TrimSpace(matches[1]), scope
}

func metricTags(metric event.MetricEntry, e event.EventEnvelope) []string {
	tags := []string{}
	add := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			tags = append(tags, key+":"+value)
		}
	}
	add("service", coalesce(metric.Service, e.Normalized.Service))
	add("env", e.Normalized.Env)
	add("version", e.Normalized.Version)
	add("route", coalesce(metric.Route, e.Normalized.Route))
	slices.Sort(tags)
	return tags
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
