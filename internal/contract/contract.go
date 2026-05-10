package contract

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/midagedev/dogtap/internal/event"
	"gopkg.in/yaml.v3"
)

type Definition struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Checks      []CheckDefinition `json:"checks" yaml:"checks"`
}

type CheckDefinition struct {
	ID          string   `json:"id" yaml:"id"`
	Type        string   `json:"type" yaml:"type"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
	PayloadKind string   `json:"payloadKind,omitempty" yaml:"payloadKind,omitempty"`
	Service     string   `json:"service,omitempty" yaml:"service,omitempty"`
	Route       string   `json:"route,omitempty" yaml:"route,omitempty"`
	RouteRegex  string   `json:"routeRegex,omitempty" yaml:"routeRegex,omitempty"`
	Metric      string   `json:"metric,omitempty" yaml:"metric,omitempty"`
	Pattern     string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Fields      []string `json:"fields,omitempty" yaml:"fields,omitempty"`
	From        Selector `json:"from,omitempty" yaml:"from,omitempty"`
	To          Selector `json:"to,omitempty" yaml:"to,omitempty"`
	Hint        string   `json:"hint,omitempty" yaml:"hint,omitempty"`
}

type Selector struct {
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
	PayloadKind string   `json:"payloadKind,omitempty" yaml:"payloadKind,omitempty"`
	Service     string   `json:"service,omitempty" yaml:"service,omitempty"`
	Route       string   `json:"route,omitempty" yaml:"route,omitempty"`
	RouteRegex  string   `json:"routeRegex,omitempty" yaml:"routeRegex,omitempty"`
	Fields      []string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

type Result struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Status      string        `json:"status"`
	Summary     Summary       `json:"summary"`
	Checks      []CheckResult `json:"checks"`
}

type Summary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type CheckResult struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	Matched     int      `json:"matched,omitempty"`
	EventIDs    []string `json:"eventIds,omitempty"`
	TraceIDs    []string `json:"traceIds,omitempty"`
	Description string   `json:"description,omitempty"`
	Hint        string   `json:"hint,omitempty"`
}

func LoadFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read contract %s: %w", path, err)
	}
	var def Definition
	switch strings.ToLower(strings.TrimSpace(fileExt(path))) {
	case ".json":
		err = json.Unmarshal(body, &def)
	default:
		err = yaml.Unmarshal(body, &def)
	}
	if err != nil {
		return Definition{}, fmt.Errorf("parse contract %s: %w", path, err)
	}
	return Normalize(def), nil
}

func Normalize(def Definition) Definition {
	def.Name = strings.TrimSpace(def.Name)
	def.Description = strings.TrimSpace(def.Description)
	for index := range def.Checks {
		check := &def.Checks[index]
		check.ID = strings.TrimSpace(check.ID)
		check.Type = strings.TrimSpace(check.Type)
		check.Description = strings.TrimSpace(check.Description)
		check.Source = strings.TrimSpace(check.Source)
		check.PayloadKind = strings.TrimSpace(check.PayloadKind)
		check.Service = strings.TrimSpace(check.Service)
		check.Route = strings.TrimSpace(check.Route)
		check.RouteRegex = strings.TrimSpace(check.RouteRegex)
		check.Metric = strings.TrimSpace(check.Metric)
		check.Pattern = strings.TrimSpace(check.Pattern)
		check.Hint = strings.TrimSpace(check.Hint)
		check.Fields = normalizeList(check.Fields)
		check.From = normalizeSelector(check.From)
		check.To = normalizeSelector(check.To)
	}
	return def
}

func Evaluate(def Definition, events []event.EventEnvelope) Result {
	def = Normalize(def)
	result := Result{
		Name:        def.Name,
		Description: def.Description,
		Status:      "pass",
		Checks:      make([]CheckResult, 0, len(def.Checks)),
	}
	for _, check := range def.Checks {
		checkResult := evaluateCheck(check, events)
		result.Checks = append(result.Checks, checkResult)
		if checkResult.Status == "fail" {
			result.Status = "fail"
		}
	}
	result.Summary.Total = len(result.Checks)
	for _, check := range result.Checks {
		if check.Status == "pass" {
			result.Summary.Passed++
		} else {
			result.Summary.Failed++
		}
	}
	return result
}

func EvaluateAll(defs []Definition, events []event.EventEnvelope) []Result {
	results := make([]Result, 0, len(defs))
	for _, def := range defs {
		if strings.TrimSpace(def.Name) == "" && len(def.Checks) == 0 {
			continue
		}
		results = append(results, Evaluate(def, events))
	}
	return results
}

func FrontendBackendReadiness() Definition {
	return Definition{
		Name:        "frontend-backend-readiness",
		Description: "Checks whether one frontend/backend workflow has inspectable RUM, replay, logs, traces, metrics, and no obvious sensitive values.",
		Labels:      map[string]string{"scope": "built-in", "audience": "dashboard"},
		Checks: []CheckDefinition{
			{
				ID:          "browser-session-context",
				Type:        "event",
				Description: "Browser telemetry has a stable session and non-PII user context.",
				Source:      "rum",
				Fields:      []string{"sessionId", "userId"},
				Hint:        "Check Browser RUM initialization, session sampling, and user context setters.",
			},
			{
				ID:          "session-replay-payload",
				Type:        "event",
				Description: "A Session Replay payload reached Dogtap.",
				Source:      "rum",
				PayloadKind: "replay",
				Hint:        "Check session replay enablement, replay sample rate, and proxy routing to /api/v2/replay.",
			},
			{
				ID:          "backend-log",
				Type:        "log-message",
				Description: "A backend log entry reached Dogtap.",
				Pattern:     ".+",
				Hint:        "Send logs through Datadog logs HTTP, OTLP logs, or a log-forwarder bridge; Dogtap does not tail containers by itself.",
			},
			{
				ID:          "backend-trace",
				Type:        "event",
				Description: "A backend trace payload reached Dogtap.",
				PayloadKind: "trace",
				Hint:        "Check DD_TRACE_AGENT_URL, DD_AGENT_HOST/DD_TRACE_AGENT_PORT, or OTLP trace exporter settings.",
			},
			{
				ID:          "workflow-metric",
				Type:        "metric",
				Description: "At least one workflow or runtime metric reached Dogtap.",
				Pattern:     ".+",
				Hint:        "Check OTLP metrics exporter enablement, endpoint, and export interval.",
			},
			{
				ID:          "no-obvious-sensitive-values",
				Type:        "no-sensitive-values",
				Description: "Visible normalized fields and logs do not contain email, bearer token, or JWT values.",
				Hint:        "Review RUM context, log messages, tags, headers, and query strings before forwarding telemetry.",
			},
		},
	}
}

func DefaultDashboardContracts() []Definition {
	return []Definition{FrontendBackendReadiness()}
}

func evaluateCheck(check CheckDefinition, events []event.EventEnvelope) CheckResult {
	if check.ID == "" {
		check.ID = check.Type
	}
	switch check.Type {
	case "event":
		return evaluateEventCheck(check, events)
	case "log-message":
		return evaluateLogMessageCheck(check, events)
	case "metric":
		return evaluateMetricCheck(check, events)
	case "trace-correlation":
		return evaluateTraceCorrelationCheck(check, events)
	case "no-sensitive-values":
		return evaluateNoSensitiveValuesCheck(check, events)
	default:
		return CheckResult{
			ID:          check.ID,
			Type:        check.Type,
			Status:      "fail",
			Message:     fmt.Sprintf("Unsupported contract check type %q.", check.Type),
			Description: check.Description,
			Hint:        "Update the contract check type or upgrade Dogtap.",
		}
	}
}

func evaluateEventCheck(check CheckDefinition, events []event.EventEnvelope) CheckResult {
	selector := Selector{
		Source:      check.Source,
		PayloadKind: check.PayloadKind,
		Service:     check.Service,
		Route:       check.Route,
		RouteRegex:  check.RouteRegex,
		Fields:      check.Fields,
	}
	matches := matchingEvents(events, selector)
	return countResult(check, len(matches), eventIDs(matches), nil, fmt.Sprintf("Observed %d matching event(s).", len(matches)))
}

func evaluateLogMessageCheck(check CheckDefinition, events []event.EventEnvelope) CheckResult {
	pattern, err := compilePattern(check.Pattern)
	if err != nil {
		return invalidPatternResult(check, err)
	}
	selector := Selector{
		Source:      check.Source,
		PayloadKind: check.PayloadKind,
		Service:     check.Service,
		Route:       check.Route,
		RouteRegex:  check.RouteRegex,
		Fields:      check.Fields,
	}
	matches := []event.EventEnvelope{}
	for _, e := range events {
		if !matchesSelector(e, selector) || e.Details == nil {
			continue
		}
		for _, log := range e.Details.Logs {
			if pattern.MatchString(log.Message) {
				matches = append(matches, e)
				break
			}
		}
	}
	return countResult(check, len(matches), eventIDs(matches), nil, fmt.Sprintf("Observed %d matching log event(s).", len(matches)))
}

func evaluateMetricCheck(check CheckDefinition, events []event.EventEnvelope) CheckResult {
	pattern, err := compilePattern(coalesce(check.Pattern, regexp.QuoteMeta(check.Metric)))
	if err != nil {
		return invalidPatternResult(check, err)
	}
	selector := Selector{
		Source:      check.Source,
		PayloadKind: check.PayloadKind,
		Service:     check.Service,
		Route:       check.Route,
		RouteRegex:  check.RouteRegex,
		Fields:      check.Fields,
	}
	matches := []event.EventEnvelope{}
	for _, e := range events {
		if !matchesSelector(e, selector) || e.Details == nil {
			continue
		}
		for _, metric := range e.Details.Metrics {
			if pattern.MatchString(metric.Name) {
				matches = append(matches, e)
				break
			}
		}
	}
	return countResult(check, len(matches), eventIDs(matches), nil, fmt.Sprintf("Observed %d matching metric event(s).", len(matches)))
}

func evaluateTraceCorrelationCheck(check CheckDefinition, events []event.EventEnvelope) CheckResult {
	fromEvents := matchingEvents(events, check.From)
	toEvents := matchingEvents(events, check.To)
	fromTraceIDs := map[string]bool{}
	for _, e := range fromEvents {
		if traceID := canonicalTraceID(e.Normalized.TraceID); traceID != "" {
			fromTraceIDs[traceID] = true
		}
		if e.Details != nil && e.Details.Trace != nil {
			for _, span := range e.Details.Trace.Spans {
				if traceID := canonicalTraceID(span.TraceID); traceID != "" {
					fromTraceIDs[traceID] = true
				}
			}
		}
	}
	matchedEvents := []event.EventEnvelope{}
	matchedTraces := map[string]bool{}
	for _, e := range toEvents {
		for _, traceID := range eventTraceIDs(e) {
			if fromTraceIDs[traceID] {
				matchedEvents = append(matchedEvents, e)
				matchedTraces[traceID] = true
				break
			}
		}
	}
	return countResult(check, len(matchedEvents), eventIDs(matchedEvents), sortedSet(matchedTraces), fmt.Sprintf("Observed %d correlated trace event(s).", len(matchedEvents)))
}

func evaluateNoSensitiveValuesCheck(check CheckDefinition, events []event.EventEnvelope) CheckResult {
	leaks := []string{}
	for _, e := range events {
		for _, value := range visibleValues(e) {
			if containsSensitiveValue(value) {
				leaks = append(leaks, e.ID)
				break
			}
		}
	}
	leaks = normalizeList(leaks)
	if len(leaks) == 0 {
		return CheckResult{
			ID:          check.ID,
			Type:        check.Type,
			Status:      "pass",
			Message:     "No obvious email, bearer token, or JWT values were visible in normalized fields or log messages.",
			Description: check.Description,
		}
	}
	return CheckResult{
		ID:          check.ID,
		Type:        check.Type,
		Status:      "fail",
		Message:     fmt.Sprintf("Found %d event(s) with obvious sensitive values.", len(leaks)),
		EventIDs:    firstN(leaks, 5),
		Matched:     len(leaks),
		Description: check.Description,
		Hint:        coalesce(check.Hint, "Review RUM context, log messages, tags, headers, and query strings before forwarding telemetry."),
	}
}

func matchingEvents(events []event.EventEnvelope, selector Selector) []event.EventEnvelope {
	matches := make([]event.EventEnvelope, 0, len(events))
	for _, e := range events {
		if matchesSelector(e, selector) {
			matches = append(matches, e)
		}
	}
	return matches
}

func matchesSelector(e event.EventEnvelope, selector Selector) bool {
	if selector.Source != "" && string(e.Source) != selector.Source {
		return false
	}
	if selector.PayloadKind != "" && e.PayloadKind != selector.PayloadKind {
		return false
	}
	if selector.Service != "" && e.Normalized.Service != selector.Service {
		return false
	}
	if selector.Route != "" && e.Normalized.Route != selector.Route {
		return false
	}
	if selector.RouteRegex != "" {
		pattern, err := regexp.Compile(selector.RouteRegex)
		if err != nil || !pattern.MatchString(e.Normalized.Route) {
			return false
		}
	}
	for _, field := range selector.Fields {
		if fieldValue(e, field) == "" {
			return false
		}
	}
	return true
}

func fieldValue(e event.EventEnvelope, field string) string {
	switch strings.TrimSpace(field) {
	case "service":
		return e.Normalized.Service
	case "env":
		return e.Normalized.Env
	case "version":
		return e.Normalized.Version
	case "host":
		return e.Normalized.Host
	case "traceId":
		return e.Normalized.TraceID
	case "spanId":
		return e.Normalized.SpanID
	case "parentSpanId":
		return e.Normalized.ParentSpanID
	case "sessionId":
		return e.Normalized.SessionID
	case "viewId":
		return e.Normalized.ViewID
	case "userId":
		return e.Normalized.UserID
	case "accountId":
		return e.Normalized.AccountID
	case "workspaceId":
		return e.Normalized.WorkspaceID
	case "caseId":
		return e.Normalized.CaseID
	case "route":
		return e.Normalized.Route
	case "method":
		return e.Normalized.Method
	case "statusCode":
		if e.Normalized.StatusCode == 0 {
			return ""
		}
		return fmt.Sprint(e.Normalized.StatusCode)
	default:
		return ""
	}
}

func countResult(check CheckDefinition, count int, eventIDs []string, traceIDs []string, passMessage string) CheckResult {
	result := CheckResult{
		ID:          check.ID,
		Type:        check.Type,
		Description: check.Description,
		Matched:     count,
		EventIDs:    firstN(eventIDs, 5),
		TraceIDs:    firstN(traceIDs, 5),
	}
	if count > 0 {
		result.Status = "pass"
		result.Message = passMessage
		return result
	}
	result.Status = "fail"
	result.Message = "Expected workflow telemetry was not observed."
	result.Hint = coalesce(check.Hint, defaultHint(check))
	return result
}

func invalidPatternResult(check CheckDefinition, err error) CheckResult {
	return CheckResult{
		ID:          check.ID,
		Type:        check.Type,
		Status:      "fail",
		Message:     "Invalid contract regex pattern.",
		Description: check.Description,
		Hint:        err.Error(),
	}
}

func defaultHint(check CheckDefinition) string {
	switch check.Type {
	case "event":
		return "Check the expected source, payload kind, service, route, and required context fields in events.json."
	case "log-message":
		return "Check that backend logs are routed to Dogtap and include the expected structured message or route."
	case "metric":
		return "Check OTLP metrics exporter configuration, endpoint, and export interval."
	case "trace-correlation":
		return "Check trace propagation between browser resources and backend spans/logs."
	default:
		return "Inspect events.json and debug-bundle.json for nearby workflow telemetry."
	}
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if strings.TrimSpace(pattern) == "" {
		pattern = ".+"
	}
	return regexp.Compile(pattern)
}

func eventIDs(events []event.EventEnvelope) []string {
	ids := make([]string, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	return normalizeList(ids)
}

func eventTraceIDs(e event.EventEnvelope) []string {
	traceIDs := map[string]bool{}
	if traceID := canonicalTraceID(e.Normalized.TraceID); traceID != "" {
		traceIDs[traceID] = true
	}
	if e.Details != nil && e.Details.Trace != nil {
		for _, span := range e.Details.Trace.Spans {
			if traceID := canonicalTraceID(span.TraceID); traceID != "" {
				traceIDs[traceID] = true
			}
		}
	}
	return sortedSet(traceIDs)
}

func visibleValues(e event.EventEnvelope) []string {
	values := []string{
		e.Endpoint,
		e.Normalized.Service,
		e.Normalized.Env,
		e.Normalized.Version,
		e.Normalized.Host,
		e.Normalized.TraceID,
		e.Normalized.SpanID,
		e.Normalized.SessionID,
		e.Normalized.ViewID,
		e.Normalized.UserID,
		e.Normalized.AccountID,
		e.Normalized.WorkspaceID,
		e.Normalized.CaseID,
		e.Normalized.Route,
		e.Normalized.ErrorType,
		e.Normalized.ErrorMessage,
	}
	for key, value := range e.Headers {
		values = append(values, key, value)
	}
	for key, valuesForKey := range e.Query {
		values = append(values, key)
		values = append(values, valuesForKey...)
	}
	for key, value := range e.Normalized.Tags {
		values = append(values, key, value)
	}
	if e.Details != nil {
		for _, log := range e.Details.Logs {
			values = append(values, log.Message, log.TraceID)
		}
	}
	return values
}

func containsSensitiveValue(value string) bool {
	return emailPattern.MatchString(value) || bearerPattern.MatchString(value) || jwtPattern.MatchString(value)
}

var (
	emailPattern  = regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}`)
	bearerPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{12,}\b`)
	jwtPattern    = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

func canonicalTraceID(value string) string {
	raw := strings.TrimSpace(value)
	lower := strings.ToLower(raw)
	if raw == "" || raw == "0" {
		return ""
	}
	if isDecimal(raw) {
		parsed, ok := new(big.Int).SetString(raw, 10)
		if !ok {
			return ""
		}
		return leftPad(parsed.Text(16), 32)
	}
	if isHex(lower) {
		return leftPad(lower, 32)
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		switch len(decoded) {
		case 16:
			return hex.EncodeToString(decoded)
		case 8:
			return leftPad(hex.EncodeToString(decoded), 32)
		}
	}
	return ""
}

func isDecimal(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func isHex(value string) bool {
	if value == "" || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func leftPad(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return strings.Repeat("0", width-len(value)) + value
}

func normalizeSelector(selector Selector) Selector {
	selector.Source = strings.TrimSpace(selector.Source)
	selector.PayloadKind = strings.TrimSpace(selector.PayloadKind)
	selector.Service = strings.TrimSpace(selector.Service)
	selector.Route = strings.TrimSpace(selector.Route)
	selector.RouteRegex = strings.TrimSpace(selector.RouteRegex)
	selector.Fields = normalizeList(selector.Fields)
	return selector
}

func normalizeList(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstN(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func fileExt(path string) string {
	index := strings.LastIndex(path, ".")
	if index < 0 {
		return ""
	}
	return path[index:]
}
