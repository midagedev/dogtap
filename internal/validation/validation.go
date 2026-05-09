package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/redact"
)

var (
	emailPattern             = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	bearerTokenPattern       = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/\-]+=*`)
	basicTokenPattern        = regexp.MustCompile(`(?i)\bbasic\s+[a-z0-9._~+/\-]+=*`)
	secretAssignmentPattern  = regexp.MustCompile(`(?i)\b(access[_-]?token|refresh[_-]?token|id[_-]?token|api[_-]?key|authorization|password|secret)\b\s*[:=]\s*["']?[^"'\s,}]+`)
	privateKeyPattern        = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	uuidPathSegmentPattern   = regexp.MustCompile(`(?i)(^|/)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}($|/)`)
	longIDPathSegmentPattern = regexp.MustCompile(`(^|/)([0-9]{6,}|[A-Za-z0-9_-]{20,})($|/)`)
)

type Validator struct {
	cfg config.ValidationConfig
}

func New(cfg config.ValidationConfig) Validator {
	return Validator{cfg: cfg}
}

func (v Validator) Validate(e event.EventEnvelope) event.ValidationResult {
	rules := make([]event.ValidationRuleResult, 0, 8)
	if v.cfg.Required.ServiceTags {
		rules = append(rules, require("required.service", "service", e.Normalized.Service)...)
		rules = append(rules, require("required.env", "env", e.Normalized.Env)...)
	}
	if e.PayloadKind != "replay" {
		for _, field := range v.requiredFieldsFor(e.Source) {
			rules = append(rules, require("required."+string(e.Source)+"."+field, field, normalizedValue(e.Normalized, field))...)
		}
	}
	rules = append(rules, v.detectQueryStringLeakage(e)...)
	rules = append(rules, v.detectHighCardinality(e)...)
	if v.cfg.PII.Enabled {
		rules = append(rules, v.detectSensitive(e)...)
	}

	return summarize(rules)
}

func (v Validator) ValidateBatch(events []event.EventEnvelope) []event.EventEnvelope {
	out := make([]event.EventEnvelope, len(events))
	for i, e := range events {
		out[i] = e
		out[i].Validation = v.Validate(e)
	}
	for i := 1; i < len(out); i++ {
		prev := out[i-1]
		curr := out[i]
		if !sameRUMSession(prev, curr) {
			continue
		}
		extra := make([]event.ValidationRuleResult, 0, 2)
		if isLogoutSignal(prev) {
			if strings.TrimSpace(curr.Normalized.UserID) != "" {
				extra = append(extra, event.ValidationRuleResult{
					RuleID:    "context.rum.logout.user",
					Severity:  "error",
					Status:    "fail",
					Message:   "RUM user context persisted after logout",
					FieldPath: "normalized.userId",
					Evidence:  "previous consecutive RUM event cleared user context",
				})
			}
			if strings.TrimSpace(curr.Normalized.AccountID) != "" {
				extra = append(extra, event.ValidationRuleResult{
					RuleID:    "context.rum.logout.account",
					Severity:  "error",
					Status:    "fail",
					Message:   "RUM account context persisted after logout",
					FieldPath: "normalized.accountId",
					Evidence:  "previous consecutive RUM event cleared account context",
				})
			}
		}
		if targetWorkspaceID, ok := workspaceSwitchTarget(prev); ok {
			switch {
			case strings.TrimSpace(curr.Normalized.WorkspaceID) == "":
				extra = append(extra, event.ValidationRuleResult{
					RuleID:    "context.rum.workspace_switch.missing_workspace",
					Severity:  "error",
					Status:    "fail",
					Message:   "RUM workspace context is missing after workspace switch",
					FieldPath: "normalized.workspaceId",
					Evidence:  "previous consecutive RUM event declared a workspace switch target",
				})
			case curr.Normalized.WorkspaceID != targetWorkspaceID:
				extra = append(extra, event.ValidationRuleResult{
					RuleID:    "context.rum.workspace_switch.stale_workspace",
					Severity:  "error",
					Status:    "fail",
					Message:   "RUM workspace context did not match the workspace switch target",
					FieldPath: "normalized.workspaceId",
					Evidence:  "previous consecutive RUM event declared a different workspace target",
				})
			}
		}
		if len(extra) > 0 {
			out[i].Validation = mergeValidation(out[i].Validation, extra)
		}
	}
	return out
}

func summarize(rules []event.ValidationRuleResult) event.ValidationResult {
	status := "pass"
	blocking := 0
	warnings := 0
	for _, rule := range rules {
		if rule.Status == "fail" {
			if rule.Severity == "warning" {
				warnings++
			} else {
				blocking++
				status = "fail"
			}
		}
	}
	summary := "validation passed"
	switch {
	case blocking > 0 && warnings > 0:
		summary = fmt.Sprintf("%d validation rule(s) failed, %d warning(s)", blocking, warnings)
	case blocking > 0:
		summary = fmt.Sprintf("%d validation rule(s) failed", blocking)
	case warnings > 0:
		summary = fmt.Sprintf("%d validation warning(s)", warnings)
	}
	return event.ValidationResult{Status: status, Rules: rules, Summary: summary}
}

func mergeValidation(result event.ValidationResult, extra []event.ValidationRuleResult) event.ValidationResult {
	rules := make([]event.ValidationRuleResult, 0, len(result.Rules)+len(extra))
	rules = append(rules, result.Rules...)
	rules = append(rules, extra...)
	return summarize(rules)
}

func (v Validator) requiredFieldsFor(source event.Source) []string {
	switch source {
	case event.SourceRUM:
		return v.cfg.Required.RUM
	case event.SourceLogs:
		return v.cfg.Required.Logs
	case event.SourceAPM:
		return v.cfg.Required.APM
	case event.SourceOTLP:
		return v.cfg.Required.OTLP
	default:
		return nil
	}
}

func require(ruleID, field, value string) []event.ValidationRuleResult {
	if strings.TrimSpace(value) == "" {
		return []event.ValidationRuleResult{{
			RuleID:    ruleID,
			Severity:  "error",
			Status:    "fail",
			Message:   "required field is missing",
			FieldPath: field,
		}}
	}
	return []event.ValidationRuleResult{{
		RuleID:    ruleID,
		Severity:  "info",
		Status:    "pass",
		Message:   "required field is present",
		FieldPath: field,
	}}
}

func (v Validator) detectSensitive(e event.EventEnvelope) []event.ValidationRuleResult {
	collector := newRuleCollector()
	for _, k := range sortedKeys(e.Query) {
		values := e.Query[k]
		if redact.IsSensitiveKey(k) {
			collector.add(event.ValidationRuleResult{
				RuleID:    "secret.key." + ruleSuffix("query."+k),
				Severity:  "fatal",
				Status:    "fail",
				Message:   "sensitive query parameter detected",
				FieldPath: "query." + k,
				Evidence:  "sensitive key name",
			})
		}
		if isDatadogTransportQueryKey(k) {
			continue
		}
		for _, value := range values {
			scanSensitiveText(collector, "query."+k, value, v.cfg.PII.FailOn)
		}
	}
	for _, k := range sortedKeys(e.Headers) {
		if redact.IsSensitiveKey(k) {
			collector.add(event.ValidationRuleResult{
				RuleID:    "secret.key." + ruleSuffix("headers."+k),
				Severity:  "fatal",
				Status:    "fail",
				Message:   "sensitive header detected and masked",
				FieldPath: "headers." + k,
				Evidence:  "sensitive key name",
			})
		}
		scanSensitiveText(collector, "headers."+k, e.Headers[k], v.cfg.PII.FailOn)
	}
	scanNormalizedSensitive(collector, e.Normalized, v.cfg.PII.FailOn)
	scanDecodedSensitive(collector, e.Decoded, "decoded", v.cfg.PII.FailOn)
	if e.RawBody != "" {
		scanSensitiveText(collector, "rawBody", e.RawBody, v.cfg.PII.FailOn)
	}
	return collector.rules
}

func (v Validator) detectQueryStringLeakage(e event.EventEnvelope) []event.ValidationRuleResult {
	collector := newRuleCollector()
	scanQueryStringText(collector, "endpoint", e.Endpoint)
	for _, k := range sortedKeys(e.Headers) {
		scanQueryStringText(collector, "headers."+k, e.Headers[k])
	}
	for _, k := range sortedKeys(e.Query) {
		if isDatadogTransportQueryKey(k) {
			continue
		}
		for _, value := range e.Query[k] {
			scanQueryStringText(collector, "query."+k, value)
		}
	}
	scanNormalizedQueryStrings(collector, e.Normalized)
	scanDecodedQueryStrings(collector, e.Decoded, "decoded")
	if e.RawBody != "" {
		scanQueryStringText(collector, "rawBody", e.RawBody)
	}
	return collector.rules
}

func (v Validator) detectHighCardinality(e event.EventEnvelope) []event.ValidationRuleResult {
	collector := newRuleCollector()
	for _, k := range sortedKeys(e.Normalized.Tags) {
		if isHighCardinalityKey(k) {
			collector.add(event.ValidationRuleResult{
				RuleID:    "cardinality.tag." + ruleSuffix(k),
				Severity:  "warning",
				Status:    "fail",
				Message:   "high-cardinality value is attached as a tag",
				FieldPath: "normalized.tags." + k,
				Evidence:  "move identifiers into normalized context when possible",
			})
		}
	}
	if routeLooksDynamic(e.Normalized.Route) {
		collector.add(event.ValidationRuleResult{
			RuleID:    "cardinality.route.dynamic_segment",
			Severity:  "warning",
			Status:    "fail",
			Message:   "route appears to contain a dynamic identifier",
			FieldPath: "normalized.route",
			Evidence:  "prefer templated route names for telemetry",
		})
	}
	return collector.rules
}

func normalizedValue(n event.NormalizedTelemetry, field string) string {
	switch field {
	case "service":
		return n.Service
	case "env":
		return n.Env
	case "version":
		return n.Version
	case "host":
		return n.Host
	case "traceId":
		return n.TraceID
	case "trace_id":
		return n.TraceID
	case "spanId":
		return n.SpanID
	case "span_id":
		return n.SpanID
	case "sessionId":
		return n.SessionID
	case "session_id":
		return n.SessionID
	case "viewId":
		return n.ViewID
	case "view_id":
		return n.ViewID
	case "userId":
		return n.UserID
	case "usr.id", "user.id", "user_id":
		return n.UserID
	case "accountId":
		return n.AccountID
	case "account.id", "account_id":
		return n.AccountID
	case "workspaceId":
		return n.WorkspaceID
	case "workspace.id", "workspace_id":
		return n.WorkspaceID
	case "caseId":
		return n.CaseID
	case "case.id", "case_id":
		return n.CaseID
	case "route":
		return n.Route
	default:
		if value, ok := n.Tags[field]; ok {
			return value
		}
		return ""
	}
}

type ruleCollector struct {
	rules []event.ValidationRuleResult
	seen  map[string]struct{}
}

func newRuleCollector() *ruleCollector {
	return &ruleCollector{seen: map[string]struct{}{}}
}

func (c *ruleCollector) add(rule event.ValidationRuleResult) {
	key := rule.RuleID + "\x00" + rule.FieldPath
	if _, ok := c.seen[key]; ok {
		return
	}
	c.seen[key] = struct{}{}
	c.rules = append(c.rules, rule)
}

func scanSensitiveText(c *ruleCollector, fieldPath, value string, failOn []string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if emailPattern.MatchString(value) {
		c.add(event.ValidationRuleResult{
			RuleID:    "pii.email." + ruleSuffix(fieldPath),
			Severity:  "fatal",
			Status:    "fail",
			Message:   "email-like value detected",
			FieldPath: fieldPath,
			Evidence:  "email-like value",
		})
	}
	for _, match := range []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{name: "bearer", pattern: bearerTokenPattern},
		{name: "basic", pattern: basicTokenPattern},
		{name: "assignment", pattern: secretAssignmentPattern},
		{name: "private_key", pattern: privateKeyPattern},
	} {
		if match.pattern.MatchString(value) {
			c.add(event.ValidationRuleResult{
				RuleID:    "secret.pattern." + match.name + "." + ruleSuffix(fieldPath),
				Severity:  "fatal",
				Status:    "fail",
				Message:   "token or secret pattern detected",
				FieldPath: fieldPath,
				Evidence:  "matched " + match.name + " pattern",
			})
		}
	}
	for _, needle := range sortedStrings(failOn) {
		if needle == "" {
			continue
		}
		if strings.Contains(strings.ToLower(value), strings.ToLower(needle)) {
			c.add(event.ValidationRuleResult{
				RuleID:    "secret.keyword." + ruleSuffix(needle) + "." + ruleSuffix(fieldPath),
				Severity:  "fatal",
				Status:    "fail",
				Message:   "configured sensitive keyword detected",
				FieldPath: fieldPath,
				Evidence:  "matched configured sensitive keyword",
			})
		}
	}
}

func scanQueryStringText(c *ruleCollector, fieldPath, value string) {
	if !hasQueryString(value) {
		return
	}
	c.add(event.ValidationRuleResult{
		RuleID:    "leak.query_string." + ruleSuffix(fieldPath),
		Severity:  "error",
		Status:    "fail",
		Message:   "query string detected in telemetry value",
		FieldPath: fieldPath,
		Evidence:  "query string present",
	})
}

func scanNormalizedSensitive(c *ruleCollector, n event.NormalizedTelemetry, failOn []string) {
	for _, field := range normalizedStringFields(n) {
		scanSensitiveText(c, field.path, field.value, failOn)
	}
	for _, k := range sortedKeys(n.Tags) {
		fieldPath := "normalized.tags." + k
		if redact.IsSensitiveKey(k) {
			c.add(event.ValidationRuleResult{
				RuleID:    "secret.key." + ruleSuffix(fieldPath),
				Severity:  "fatal",
				Status:    "fail",
				Message:   "sensitive tag key detected",
				FieldPath: fieldPath,
				Evidence:  "sensitive key name",
			})
		}
		scanSensitiveText(c, fieldPath, n.Tags[k], failOn)
	}
}

func scanNormalizedQueryStrings(c *ruleCollector, n event.NormalizedTelemetry) {
	for _, field := range normalizedStringFields(n) {
		scanQueryStringText(c, field.path, field.value)
	}
	for _, k := range sortedKeys(n.Tags) {
		scanQueryStringText(c, "normalized.tags."+k, n.Tags[k])
	}
}

func scanDecodedSensitive(c *ruleCollector, v any, path string, failOn []string) {
	switch typed := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(typed) {
			fieldPath := joinFieldPath(path, k)
			if redact.IsSensitiveKey(k) {
				c.add(event.ValidationRuleResult{
					RuleID:    "secret.key." + ruleSuffix(fieldPath),
					Severity:  "fatal",
					Status:    "fail",
					Message:   "sensitive payload key detected",
					FieldPath: fieldPath,
					Evidence:  "sensitive key name",
				})
			}
			scanDecodedSensitive(c, typed[k], fieldPath, failOn)
		}
	case []any:
		for i, value := range typed {
			scanDecodedSensitive(c, value, fmt.Sprintf("%s[%d]", path, i), failOn)
		}
	case string:
		scanSensitiveText(c, path, typed, failOn)
	}
}

func scanDecodedQueryStrings(c *ruleCollector, v any, path string) {
	switch typed := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(typed) {
			scanDecodedQueryStrings(c, typed[k], joinFieldPath(path, k))
		}
	case []any:
		for i, value := range typed {
			scanDecodedQueryStrings(c, value, fmt.Sprintf("%s[%d]", path, i))
		}
	case string:
		scanQueryStringText(c, path, typed)
	}
}

func normalizedStringFields(n event.NormalizedTelemetry) []struct {
	path  string
	value string
} {
	return []struct {
		path  string
		value string
	}{
		{path: "normalized.service", value: n.Service},
		{path: "normalized.env", value: n.Env},
		{path: "normalized.version", value: n.Version},
		{path: "normalized.host", value: n.Host},
		{path: "normalized.timestamp", value: n.Timestamp},
		{path: "normalized.traceId", value: n.TraceID},
		{path: "normalized.spanId", value: n.SpanID},
		{path: "normalized.parentSpanId", value: n.ParentSpanID},
		{path: "normalized.sessionId", value: n.SessionID},
		{path: "normalized.viewId", value: n.ViewID},
		{path: "normalized.userId", value: n.UserID},
		{path: "normalized.accountId", value: n.AccountID},
		{path: "normalized.workspaceId", value: n.WorkspaceID},
		{path: "normalized.caseId", value: n.CaseID},
		{path: "normalized.route", value: n.Route},
		{path: "normalized.method", value: n.Method},
		{path: "normalized.errorType", value: n.ErrorType},
		{path: "normalized.errorMessage", value: n.ErrorMessage},
	}
}

func hasQueryString(value string) bool {
	idx := strings.Index(value, "?")
	if idx < 0 {
		return false
	}
	tail := value[idx+1:]
	if stop := strings.IndexAny(tail, " \t\r\n\"'<>"); stop >= 0 {
		tail = tail[:stop]
	}
	return strings.Contains(tail, "=")
}

func routeLooksDynamic(route string) bool {
	if route == "" || strings.Contains(route, "{") || strings.Contains(route, ":") {
		return false
	}
	path, _, _ := strings.Cut(route, "?")
	return uuidPathSegmentPattern.MatchString(path) || longIDPathSegmentPattern.MatchString(path)
}

func isHighCardinalityKey(key string) bool {
	switch normalizeIdentifier(key) {
	case "accountid", "caseid", "correlationid", "email", "requestid", "sessionid", "spanid", "traceid", "userid", "usrid", "workspaceid":
		return true
	default:
		return false
	}
}

func isDatadogTransportQueryKey(key string) bool {
	switch strings.ToLower(key) {
	case "ddforward", "ddtags", "ddsource", "dd-api-key", "dd-evp-origin", "dd-evp-origin-version", "dd-request-id", "batch_time":
		return true
	default:
		return false
	}
}

func sameRUMSession(prev, curr event.EventEnvelope) bool {
	if prev.Source != event.SourceRUM || curr.Source != event.SourceRUM {
		return false
	}
	prevSession := strings.TrimSpace(prev.Normalized.SessionID)
	currSession := strings.TrimSpace(curr.Normalized.SessionID)
	return prevSession != "" && prevSession == currSession
}

func isLogoutSignal(e event.EventEnvelope) bool {
	text := compactEventText(e)
	for _, needle := range []string{"logout", "signout", "clearuser", "clearaccount"} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func workspaceSwitchTarget(e event.EventEnvelope) (string, bool) {
	text := compactEventText(e)
	switchSignal := false
	for _, needle := range []string{"workspaceswitch", "switchworkspace", "workspacechanged", "changeworkspace"} {
		if strings.Contains(text, needle) {
			switchSignal = true
			break
		}
	}
	if !switchSignal {
		return "", false
	}
	for _, key := range []string{
		"targetWorkspaceId",
		"target_workspace_id",
		"target.workspace.id",
		"workspace.target.id",
		"toWorkspaceId",
		"to_workspace_id",
		"newWorkspaceId",
		"new_workspace_id",
		"nextWorkspaceId",
		"next_workspace_id",
	} {
		if value := strings.TrimSpace(e.Normalized.Tags[key]); value != "" {
			return value, true
		}
		if value := strings.TrimSpace(findDecodedString(e.Decoded, key)); value != "" {
			return value, true
		}
	}
	return "", false
}

func findDecodedString(v any, key string) string {
	target := normalizeIdentifier(key)
	switch typed := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(typed) {
			if normalizeIdentifier(k) == target {
				return scalarString(typed[k])
			}
		}
		for _, k := range sortedKeys(typed) {
			if value := findDecodedString(typed[k], key); value != "" {
				return value
			}
		}
	case []any:
		for _, value := range typed {
			if found := findDecodedString(value, key); found != "" {
				return found
			}
		}
	}
	return ""
}

func compactEventText(e event.EventEnvelope) string {
	parts := []string{
		e.Endpoint,
		e.RawBody,
		e.Normalized.Route,
		e.Normalized.ErrorType,
		e.Normalized.ErrorMessage,
	}
	for _, k := range sortedKeys(e.Normalized.Tags) {
		parts = append(parts, k, e.Normalized.Tags[k])
	}
	if e.Decoded != nil {
		if b, err := json.Marshal(e.Decoded); err == nil {
			parts = append(parts, string(b))
		}
	}
	return normalizeIdentifier(strings.Join(parts, " "))
}

func joinFieldPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func ruleSuffix(value string) string {
	suffix := strings.Trim(normalizeIdentifierWithSeparator(value), "_")
	if suffix == "" {
		return "value"
	}
	return suffix
}

func normalizeIdentifier(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeIdentifierWithSeparator(value string) string {
	var b strings.Builder
	lastSeparator := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSeparator = false
			continue
		}
		if !lastSeparator {
			b.WriteRune('_')
			lastSeparator = true
		}
	}
	return b.String()
}

func scalarString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case int:
		return fmt.Sprint(typed)
	case int8:
		return fmt.Sprint(typed)
	case int16:
		return fmt.Sprint(typed)
	case int32:
		return fmt.Sprint(typed)
	case int64:
		return fmt.Sprint(typed)
	case uint:
		return fmt.Sprint(typed)
	case uint8:
		return fmt.Sprint(typed)
	case uint16:
		return fmt.Sprint(typed)
	case uint32:
		return fmt.Sprint(typed)
	case uint64:
		return fmt.Sprint(typed)
	case float32:
		return fmt.Sprint(typed)
	case float64:
		return fmt.Sprint(typed)
	case bool:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}
