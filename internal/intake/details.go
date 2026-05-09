package intake

import (
	"strconv"
	"strings"

	"github.com/midagedev/dogtap/internal/event"
)

func BuildDetails(source event.Source, payloadKind string, decoded any, normalized event.NormalizedTelemetry, contentType string, bodyBytes int) *event.TelemetryDetails {
	details := &event.TelemetryDetails{}
	if payloadKind == "replay" {
		details.Replay = replayDetail(decoded, normalized, contentType, bodyBytes)
	}
	if source == event.SourceLogs || payloadKind == "log" {
		details.Logs = logDetails(decoded, normalized)
	}
	if source == event.SourceAPM || source == event.SourceOTLP || payloadKind == "trace" {
		details.Trace = traceDetail(decoded, normalized)
	}
	if payloadKind == "metric" {
		details.Metrics = metricDetails(decoded, normalized)
	}
	if details.Replay == nil && len(details.Logs) == 0 && (details.Trace == nil || len(details.Trace.Spans) == 0) && len(details.Metrics) == 0 {
		return nil
	}
	return details
}

func replayDetail(decoded any, normalized event.NormalizedTelemetry, contentType string, bodyBytes int) *event.ReplayDetail {
	format := coalesce(findString(decoded, "replay.format", "format"), "unknown")
	recordCount := firstPositiveInt(
		findInt(decoded, "replay.recordCount"),
		findInt(decoded, "replay.frames"),
		findInt(decoded, "event.records_count"),
		findInt(decoded, "records_count"),
		replayFrameCount(findAny(decoded, "records")),
	)
	return &event.ReplayDetail{
		Format:             format,
		ContentType:        coalesce(findString(decoded, "replay.contentType"), contentType),
		Bytes:              firstPositiveInt(findInt(decoded, "replay.bytes"), bodyBytes),
		RecordCount:        recordCount,
		SegmentBytes:       firstPositiveInt(findInt(decoded, "replay.segmentBytes"), findInt(decoded, "event.compressed_segment_size")),
		SegmentContentType: findString(decoded, "replay.segmentContentType"),
		SegmentFilename:    findString(decoded, "replay.segmentFilename"),
		SessionID:          coalesce(normalized.SessionID, findString(decoded, "event.session.id", "session.id", "session_id")),
		ViewID:             coalesce(normalized.ViewID, findString(decoded, "event.view.id", "view.id", "view_id")),
		Start:              findString(decoded, "event.start", "start"),
		End:                findString(decoded, "event.end", "end"),
	}
}

func logDetails(decoded any, normalized event.NormalizedTelemetry) []event.LogEntry {
	rows := []map[string]any{}
	collectLogRows(decoded, &rows)
	if len(rows) > 0 {
		entries := make([]event.LogEntry, 0, len(rows))
		for _, row := range rows {
			entries = append(entries, logEntry(row, normalized))
		}
		return entries
	}

	items := eventItems(decoded)
	if len(items) == 0 {
		items = []any{decoded}
	}
	entries := make([]event.LogEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, logEntry(item, normalized))
	}
	return entries
}

func logEntry(row any, normalized event.NormalizedTelemetry) event.LogEntry {
	level := coalesce(findString(row, "status", "level", "severityText", "severity"), "info")
	message := coalesce(
		findString(row, "message", "msg", "body.stringValue", "body", "error.message", "error"),
		scalarString(row),
	)
	if message == "" {
		message = "log payload"
	}
	return event.LogEntry{
		Timestamp: coalesce(findString(row, "timestamp", "date", "time", "timeUnixNano", "observedTimeUnixNano"), normalized.Timestamp),
		Level:     strings.ToUpper(level),
		Message:   message,
		TraceID:   coalesce(findString(row, "trace_id", "traceId", "dd.trace_id", "trace.id"), normalized.TraceID),
	}
}

func traceDetail(decoded any, normalized event.NormalizedTelemetry) *event.TraceDetail {
	rows := []map[string]any{}
	collectTraceRows(decoded, &rows)
	spans := make([]event.SpanDetail, 0, len(rows)+1)
	for _, row := range rows {
		spans = append(spans, event.SpanDetail{
			TraceID:      coalesce(findString(row, "trace_id", "traceId", "dd.trace_id", "trace.id"), normalized.TraceID),
			SpanID:       coalesce(findString(row, "span_id", "spanId", "dd.span_id", "span.id"), normalized.SpanID),
			ParentSpanID: coalesce(findString(row, "parent_id", "parentSpanId", "parent.span.id"), normalized.ParentSpanID),
			Name:         coalesce(findString(row, "name", "operationName"), normalized.Route, "span"),
			Resource:     findString(row, "resource", "resource.name", "route", "http.route"),
			Service:      coalesce(findString(row, "service", "dd.service", "service.name"), normalized.Service, "unknown-service"),
			Start:        findString(row, "start", "timestamp", "time"),
			DurationMS:   spanDurationMS(row, normalized.DurationMS),
			Error:        spanError(row),
		})
	}
	if len(spans) == 0 && (normalized.TraceID != "" || normalized.SpanID != "") {
		spans = append(spans, event.SpanDetail{
			TraceID:      normalized.TraceID,
			SpanID:       normalized.SpanID,
			ParentSpanID: normalized.ParentSpanID,
			Name:         coalesce(normalized.ErrorType, normalized.Route, string(normalized.Source)),
			Resource:     normalized.Route,
			Service:      coalesce(normalized.Service, "unknown-service"),
			DurationMS:   normalized.DurationMS,
			Error:        normalized.ErrorType != "",
		})
	}
	return &event.TraceDetail{TraceID: normalized.TraceID, Spans: spans}
}

func eventItems(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		for _, key := range []string{"logs", "exceptions", "events", "records"} {
			if nested, ok := typed[key].([]any); ok {
				return nested
			}
		}
		return []any{typed}
	default:
		return nil
	}
}

func collectTraceRows(value any, rows *[]map[string]any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectTraceRows(item, rows)
		}
	case map[string]any:
		if _, ok := typed["span_id"]; ok {
			*rows = append(*rows, typed)
		} else if _, ok := typed["spanId"]; ok {
			*rows = append(*rows, typed)
		}
		for _, item := range typed {
			collectTraceRows(item, rows)
		}
	}
}

func collectLogRows(value any, rows *[]map[string]any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectLogRows(item, rows)
		}
	case map[string]any:
		if _, ok := typed["logRecords"]; ok {
			collectLogRows(typed["logRecords"], rows)
			return
		}
		if _, ok := typed["body"]; ok {
			*rows = append(*rows, typed)
			return
		}
		for _, item := range typed {
			collectLogRows(item, rows)
		}
	}
}

func spanDurationMS(row map[string]any, fallback float64) float64 {
	duration := findFloat(row, "duration_ms", "durationMs", "duration")
	if duration == 0 {
		return fallback
	}
	if duration > 1_000_000 {
		return duration / 1_000_000
	}
	return duration
}

func spanError(row map[string]any) bool {
	value := strings.ToLower(findString(row, "error", "error.type", "error.message"))
	return value != "" && value != "0" && value != "false"
}

func metricDetails(decoded any, normalized event.NormalizedTelemetry) []event.MetricEntry {
	if entries := faroMeasurementDetails(decoded, normalized); len(entries) > 0 {
		return entries
	}
	rows := []map[string]any{}
	collectMetricRows(decoded, &rows)
	entries := make([]event.MetricEntry, 0, len(rows))
	for _, row := range rows {
		name := coalesce(findString(row, "name", "metric.name"), "metric")
		unit := findString(row, "unit")
		aggregation, points := metricDataPoints(row)
		if len(points) == 0 {
			if value, ok := metricNumber(row, "value", "asDouble", "asInt", "count", "sum"); ok {
				tags := metricTags(row, row)
				entries = append(entries, metricEntry(name, unit, aggregation, value, row, tags, normalized))
			}
			continue
		}
		for _, point := range points {
			pointRow, ok := point.(map[string]any)
			if !ok {
				continue
			}
			value, ok := metricNumber(pointRow, "asDouble", "asInt", "value", "count", "sum")
			if !ok {
				continue
			}
			tags := metricTags(row, pointRow)
			entries = append(entries, metricEntry(name, unit, aggregation, value, pointRow, tags, normalized))
		}
	}
	return entries
}

func faroMeasurementDetails(decoded any, normalized event.NormalizedTelemetry) []event.MetricEntry {
	measurements, ok := findAny(decoded, "measurements").([]any)
	if !ok || len(measurements) == 0 {
		return nil
	}
	entries := []event.MetricEntry{}
	for _, measurement := range measurements {
		row, ok := measurement.(map[string]any)
		if !ok {
			continue
		}
		metricType := coalesce(findString(row, "type", "name"), "faro.measurement")
		values, ok := findAny(row, "values").(map[string]any)
		if !ok || len(values) == 0 {
			if value, ok := metricNumber(row, "value"); ok {
				entries = append(entries, faroMetricEntry(metricType, value, row, normalized))
			}
			continue
		}
		for name, rawValue := range values {
			value, ok := numericValue(rawValue)
			if !ok {
				continue
			}
			metricName := metricType
			if name != "duration" && name != "value" {
				metricName = metricType + "." + name
			}
			entries = append(entries, faroMetricEntry(metricName, value, row, normalized))
		}
	}
	return entries
}

func faroMetricEntry(name string, value float64, row map[string]any, normalized event.NormalizedTelemetry) event.MetricEntry {
	return event.MetricEntry{
		Name:      name,
		Service:   normalized.Service,
		Value:     value,
		Route:     coalesce(findString(row, "context.route"), normalized.Route),
		Timestamp: coalesce(findString(row, "timestamp"), normalized.Timestamp),
	}
}

func collectMetricRows(value any, rows *[]map[string]any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectMetricRows(item, rows)
		}
	case map[string]any:
		if _, hasName := typed["name"]; hasName {
			if hasMetricData(typed) {
				*rows = append(*rows, typed)
			}
		}
		for _, item := range typed {
			collectMetricRows(item, rows)
		}
	}
}

func hasMetricData(row map[string]any) bool {
	for _, key := range []string{"gauge", "sum", "histogram", "summary", "dataPoints", "data_points", "value", "asDouble", "asInt"} {
		if _, ok := row[key]; ok {
			return true
		}
	}
	return false
}

func metricDataPoints(row map[string]any) (string, []any) {
	for _, kind := range []string{"gauge", "sum", "histogram", "summary"} {
		if metric, ok := row[kind].(map[string]any); ok {
			if points, ok := findAny(metric, "dataPoints").([]any); ok {
				return kind, points
			}
			if points, ok := findAny(metric, "data_points").([]any); ok {
				return kind, points
			}
		}
	}
	if points, ok := findAny(row, "dataPoints").([]any); ok {
		return "", points
	}
	if points, ok := findAny(row, "data_points").([]any); ok {
		return "", points
	}
	return "", nil
}

func metricEntry(name, unit, aggregation string, value float64, point map[string]any, tags map[string]string, normalized event.NormalizedTelemetry) event.MetricEntry {
	return event.MetricEntry{
		Name:        name,
		Service:     coalesce(tags["service"], tags["service.name"], normalized.Service),
		Unit:        unit,
		Value:       value,
		Aggregation: aggregation,
		Route:       coalesce(tags["http.route"], tags["route"], tags["resource.name"], normalized.Route),
		Timestamp:   coalesce(findString(point, "timestamp", "time", "timeUnixNano", "time_unix_nano"), normalized.Timestamp),
	}
}

func metricTags(metric map[string]any, point map[string]any) map[string]string {
	tags := map[string]string{}
	collectAttributePairs(metric, tags)
	collectAttributePairs(point, tags)
	return tags
}

func metricNumber(root any, paths ...string) (float64, bool) {
	for _, path := range paths {
		if value, ok := numericValue(findAny(root, path)); ok {
			return value, true
		}
	}
	return 0, false
}

func numericValue(raw any) (float64, bool) {
	switch value := raw.(type) {
	case int:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint8:
		return float64(value), true
	case uint16:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
