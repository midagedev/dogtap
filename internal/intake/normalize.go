package intake

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/midagedev/dogtap/internal/event"
)

func Normalize(source event.Source, decoded any) event.NormalizedTelemetry {
	if source == event.SourceFaro {
		return normalizeFaro(decoded)
	}
	root := firstPayload(decoded)
	tags := collectTags(root)
	traceRoot := tracedPayload(decoded)
	n := event.NormalizedTelemetry{
		Source: source,
		Tags:   tags,
	}
	n.Service = coalesce(findString(root, "service", "dd.service", "service.name"), tags["service"], tags["service.name"])
	n.Env = coalesce(findString(root, "env", "dd.env", "deployment.environment", "deployment.environment.name"), tags["env"], tags["deployment.environment"], tags["deployment.environment.name"])
	n.Version = coalesce(findString(root, "version", "dd.version", "service.version"), tags["version"], tags["service.version"])
	n.Host = findString(root, "host", "hostname", "host.name")
	n.Timestamp = findString(root, "timestamp", "date", "time")
	n.TraceID = coalesce(
		findString(traceRoot, "_dd.trace_id", "trace_id", "traceId", "dd.trace_id", "trace.id"),
		tags["_dd.trace_id"],
		tags["trace_id"],
		tags["traceId"],
		tags["dd.trace_id"],
		tags["trace.id"],
	)
	n.SpanID = coalesce(
		findString(traceRoot, "_dd.span_id", "span_id", "spanId", "dd.span_id", "span.id"),
		tags["_dd.span_id"],
		tags["span_id"],
		tags["spanId"],
		tags["dd.span_id"],
		tags["span.id"],
	)
	n.ParentSpanID = coalesce(
		findString(traceRoot, "parent_id", "parentSpanId", "parent.span.id"),
		tags["parent_id"],
		tags["parentSpanId"],
		tags["parent.span.id"],
	)
	dropTagKeys(tags,
		"_dd.trace_id", "trace_id", "traceId", "dd.trace_id", "trace.id",
		"_dd.span_id", "span_id", "spanId", "dd.span_id", "span.id",
		"parent_id", "parentSpanId", "parent.span.id",
	)
	n.SessionID = findString(root, "event.session.id", "session.id", "session_id", "sessionId", "application.id")
	n.ViewID = findString(root, "event.view.id", "view.id", "view_id", "viewId")
	n.UserID = findString(root, "usr.id", "user.id", "user_id", "userId", "context.user.id")
	n.AccountID = findString(root, "account.id", "account_id", "accountId", "context.account.id")
	n.WorkspaceID = findString(root, "workspace.id", "workspace_id", "workspaceId", "context.workspace.id")
	n.CaseID = findString(root, "case.id", "case_id", "caseId", "context.case.id")
	n.Route = coalesce(
		pathFromURL(findString(traceRoot, "resource.url", "http.url")),
		findString(traceRoot, "route", "resource", "resource.name", "view.url_path", "url.path", "http.route", "http.url"),
		pathFromURL(findString(decoded, "resource.url", "http.url")),
		findString(root, "route", "resource", "resource.name", "view.url_path", "url.path", "http.route", "http.url"),
		tags["http.route"],
		tags["resource.name"],
		tags["url.path"],
	)
	n.Method = coalesce(
		findString(traceRoot, "resource.method", "method", "http.method", "http.request.method"),
		findString(decoded, "resource.method"),
		findString(root, "method", "http.method", "http.request.method"),
		tags["http.method"],
		tags["http.request.method"],
	)
	n.StatusCode = findInt(traceRoot, "resource.status_code", "status_code", "statusCode", "http.status_code", "http.response.status_code")
	if n.StatusCode == 0 {
		n.StatusCode = findInt(root, "status_code", "statusCode", "http.status_code", "http.response.status_code")
	}
	if n.StatusCode == 0 {
		n.StatusCode = findInt(decoded, "resource.status_code")
	}
	if n.StatusCode == 0 {
		n.StatusCode = firstTagInt(tags, "http.status_code", "http.response.status_code", "status_code")
	}
	n.DurationMS = findFloat(root, "duration", "duration_ms", "durationMs")
	n.ErrorType = findString(root, "error.type", "error.kind", "type")
	n.ErrorMessage = findString(root, "error.message", "message", "msg")
	if replay, ok := findAny(decoded, "records").([]any); ok && len(replay) > 0 {
		if value := findString(replay, "timestamp"); value != "" {
			n.Timestamp = coalesce(n.Timestamp, value)
		}
	}
	return n
}

func tracedPayload(root any) any {
	if payload, _, ok := bestTracedPayload(root); ok {
		return payload
	}
	return root
}

func bestTracedPayload(root any) (any, int, bool) {
	bestSize := 0
	var best any
	found := false
	consider := func(candidate any, size int, ok bool) {
		if !ok {
			return
		}
		if !found || size < bestSize {
			best = candidate
			bestSize = size
			found = true
		}
	}

	switch typed := root.(type) {
	case map[string]any:
		for _, value := range typed {
			consider(bestTracedPayload(value))
		}
		if hasTraceAndRoute(typed) {
			consider(typed, payloadSize(typed), true)
		}
	case []any:
		for _, value := range typed {
			consider(bestTracedPayload(value))
		}
	}
	return best, bestSize, found
}

func hasTraceAndRoute(root any) bool {
	if findString(root, "_dd.trace_id", "trace_id", "traceId", "dd.trace_id", "trace.id") == "" {
		return false
	}
	return coalesce(
		findString(root, "resource.url", "http.url"),
		findString(root, "route", "resource", "resource.name", "view.url_path", "url.path", "http.route", "http.url"),
	) != ""
}

func payloadSize(root any) int {
	switch typed := root.(type) {
	case map[string]any:
		size := 1
		for _, value := range typed {
			size += payloadSize(value)
		}
		return size
	case []any:
		size := 1
		for _, value := range typed {
			size += payloadSize(value)
		}
		return size
	default:
		return 1
	}
}

func normalizeFaro(decoded any) event.NormalizedTelemetry {
	tags := collectTags(decoded)
	n := event.NormalizedTelemetry{
		Source: event.SourceFaro,
		Tags:   tags,
	}
	n.Service = coalesce(findString(decoded, "meta.app.name", "app.name"), tags["service"], tags["service.name"])
	n.Env = coalesce(findString(decoded, "meta.app.environment", "app.environment"), tags["env"], tags["deployment.environment"], tags["deployment.environment.name"])
	n.Version = coalesce(findString(decoded, "meta.app.version", "app.version", "meta.app.release", "app.release"), tags["version"], tags["service.version"])
	n.Host = findString(decoded, "host", "hostname", "host.name")
	n.Timestamp = findString(decoded, "events.timestamp", "logs.timestamp", "measurements.timestamp", "exceptions.timestamp", "timestamp")
	n.TraceID = findString(decoded, "traces.resourceSpans.scopeSpans.spans.traceId", "traces.resourceSpans.scopeSpans.spans.trace_id", "traceId", "trace_id")
	n.SpanID = findString(decoded, "traces.resourceSpans.scopeSpans.spans.spanId", "traces.resourceSpans.scopeSpans.spans.span_id", "spanId", "span_id")
	n.ParentSpanID = findString(decoded, "traces.resourceSpans.scopeSpans.spans.parentSpanId", "traces.resourceSpans.scopeSpans.spans.parent_span_id", "parentSpanId", "parent_span_id")
	n.SessionID = findString(decoded, "meta.session.id", "session.id", "session_id", "sessionId")
	n.UserID = findString(decoded, "meta.user.id", "user.id", "user_id", "userId")
	n.AccountID = findString(decoded, "meta.user.attributes.accountId", "meta.user.attributes.account.id", "account.id", "accountId")
	n.WorkspaceID = findString(decoded, "meta.user.attributes.workspaceId", "meta.user.attributes.workspace.id", "workspace.id", "workspaceId")
	n.CaseID = coalesce(
		findString(decoded, "events.attributes.caseId", "events.attributes.case.id"),
		findString(decoded, "meta.user.attributes.caseId", "meta.user.attributes.case.id", "case.id", "caseId"),
	)
	n.Route = coalesce(
		findString(decoded, "events.attributes.route", "logs.context.route", "measurements.context.route", "meta.view.name", "route", "http.route", "url.path"),
		tags["http.route"],
		tags["url.path"],
		pathFromURL(tags["url.full"]),
		pathFromURL(findString(decoded, "meta.page.url", "page.url")),
	)
	n.Method = coalesce(
		findString(decoded, "method", "http.method", "http.request.method"),
		tags["http.method"],
		tags["http.request.method"],
	)
	n.StatusCode = findInt(decoded, "status_code", "statusCode", "http.status_code", "http.response.status_code")
	if n.StatusCode == 0 {
		n.StatusCode = firstTagInt(tags, "http.status_code", "http.response.status_code", "status_code")
	}
	n.DurationMS = coalesceFaroDuration(decoded)
	n.ErrorType = findString(decoded, "exceptions.type", "exceptions.value", "logs.level")
	n.ErrorMessage = findString(decoded, "exceptions.message", "logs.message", "events.name")
	return n
}

func pathFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Path == "" {
		return raw
	}
	return parsed.Path
}

func coalesceFaroDuration(decoded any) float64 {
	for _, path := range []string{
		"measurements.values.duration",
		"measurements.values.duration_ms",
		"measurements.values.lcp",
		"measurements.values.fcp",
		"measurements.values.ttfb",
	} {
		if value := findFloat(decoded, path); value > 0 {
			return value
		}
	}
	return 0
}

func firstPayload(v any) any {
	switch typed := v.(type) {
	case []any:
		if len(typed) == 0 {
			return v
		}
		return firstPayload(typed[0])
	case map[string]any:
		if events, ok := typed["events"]; ok {
			return firstPayload(events)
		}
		if series, ok := typed["traces"]; ok {
			return firstPayload(series)
		}
		return typed
	default:
		return v
	}
}

func collectTags(v any) map[string]string {
	tags := map[string]string{}
	if tagString := findString(v, "ddtags", "tags"); strings.Contains(tagString, ":") {
		for _, part := range strings.Split(tagString, ",") {
			key, value, ok := strings.Cut(strings.TrimSpace(part), ":")
			if ok && key != "" {
				tags[key] = value
			}
		}
	}
	if tagMap, ok := findAny(v, "tags").(map[string]any); ok {
		for k, value := range tagMap {
			tags[k] = scalarString(value)
		}
	}
	if meta, ok := findAny(v, "meta").(map[string]any); ok {
		for k, value := range meta {
			if strings.HasPrefix(k, "_dd.") {
				continue
			}
			if _, exists := tags[k]; !exists {
				tags[k] = scalarString(value)
			}
		}
	}
	collectAttributePairs(v, tags)
	return tags
}

func collectAttributePairs(v any, tags map[string]string) {
	switch typed := v.(type) {
	case map[string]any:
		key, hasKey := typed["key"].(string)
		if hasKey {
			if value := attributeValue(typed["value"]); value != "" {
				tags[key] = value
			}
		}
		for _, value := range typed {
			collectAttributePairs(value, tags)
		}
	case []any:
		for _, item := range typed {
			collectAttributePairs(item, tags)
		}
	}
}

func dropTagKeys(tags map[string]string, keys ...string) {
	for _, key := range keys {
		delete(tags, key)
	}
}

func attributeValue(v any) string {
	switch typed := v.(type) {
	case map[string]any:
		for _, key := range []string{"stringValue", "intValue", "doubleValue", "boolValue"} {
			if value := scalarString(typed[key]); value != "" {
				return value
			}
		}
		return ""
	default:
		return scalarString(v)
	}
}

func findString(root any, paths ...string) string {
	for _, path := range paths {
		if value := scalarString(findAny(root, path)); value != "" {
			return value
		}
	}
	return ""
}

func findInt(root any, paths ...string) int {
	for _, path := range paths {
		switch value := findAny(root, path).(type) {
		case int:
			return value
		case int8:
			return int(value)
		case int16:
			return int(value)
		case int32:
			return int(value)
		case int64:
			return int(value)
		case uint64:
			return int(value)
		case float64:
			return int(value)
		case string:
			parsed, err := strconv.Atoi(value)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func findFloat(root any, paths ...string) float64 {
	for _, path := range paths {
		switch value := findAny(root, path).(type) {
		case float64:
			return value
		case float32:
			return float64(value)
		case int:
			return float64(value)
		case int64:
			return float64(value)
		case string:
			parsed, err := strconv.ParseFloat(value, 64)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func firstTagInt(tags map[string]string, keys ...string) int {
	for _, key := range keys {
		if parsed, err := strconv.Atoi(tags[key]); err == nil {
			return parsed
		}
	}
	return 0
}

func findAny(root any, path string) any {
	if root == nil || path == "" {
		return nil
	}
	if value, ok := findPath(root, strings.Split(path, ".")); ok {
		return value
	}
	return deepFind(root, normalizeKey(path))
}

func findPath(root any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return root, true
	}
	switch typed := root.(type) {
	case map[string]any:
		if value, ok := typed[parts[0]]; ok {
			return findPath(value, parts[1:])
		}
		joined := strings.Join(parts, ".")
		if value, ok := typed[joined]; ok {
			return value, true
		}
	case []any:
		for _, item := range typed {
			if value, ok := findPath(item, parts); ok {
				return value, true
			}
		}
	}
	return nil, false
}

func deepFind(root any, normalizedKey string) any {
	switch typed := root.(type) {
	case map[string]any:
		for k, value := range typed {
			if normalizeKey(k) == normalizedKey {
				return value
			}
		}
		for _, value := range typed {
			if found := deepFind(value, normalizedKey); found != nil {
				return found
			}
		}
	case []any:
		for _, value := range typed {
			if found := deepFind(value, normalizedKey); found != nil {
				return found
			}
		}
	}
	return nil
}

func normalizeKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, ".", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	return key
}

func scalarString(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.Itoa(int(typed))
	case int16:
		return strconv.Itoa(int(typed))
	case int32:
		return strconv.Itoa(int(typed))
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
