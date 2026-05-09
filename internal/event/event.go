package event

import "time"

type Source string

const (
	SourceRUM     Source = "rum"
	SourceAPM     Source = "apm"
	SourceLogs    Source = "logs"
	SourceOTLP    Source = "otlp"
	SourceFaro    Source = "faro"
	SourceUnknown Source = "unknown"
)

type EventEnvelope struct {
	ID               string              `json:"id"`
	ReceivedAt       time.Time           `json:"receivedAt"`
	Source           Source              `json:"source"`
	PayloadKind      string              `json:"payloadKind,omitempty"`
	Endpoint         string              `json:"endpoint"`
	Method           string              `json:"method"`
	Headers          map[string]string   `json:"headers"`
	Query            map[string][]string `json:"query,omitempty"`
	ContentType      string              `json:"contentType,omitempty"`
	ContentEncoding  string              `json:"contentEncoding,omitempty"`
	BodySizeBytes    int64               `json:"bodySizeBytes"`
	DecodedSizeBytes int64               `json:"decodedSizeBytes"`
	RawBody          string              `json:"rawBody,omitempty"`
	Decoded          any                 `json:"decoded,omitempty"`
	Details          *TelemetryDetails   `json:"details,omitempty"`
	Normalized       NormalizedTelemetry `json:"normalized"`
	Validation       ValidationResult    `json:"validation"`
	Forwarding       ForwardingResult    `json:"forwarding"`
}

type TelemetryDetails struct {
	Replay  *ReplayDetail `json:"replay,omitempty"`
	Logs    []LogEntry    `json:"logs,omitempty"`
	Trace   *TraceDetail  `json:"trace,omitempty"`
	Metrics []MetricEntry `json:"metrics,omitempty"`
}

type ReplayDetail struct {
	Format             string `json:"format,omitempty"`
	ContentType        string `json:"contentType,omitempty"`
	Bytes              int    `json:"bytes,omitempty"`
	RecordCount        int    `json:"recordCount,omitempty"`
	SegmentBytes       int    `json:"segmentBytes,omitempty"`
	SegmentContentType string `json:"segmentContentType,omitempty"`
	SegmentFilename    string `json:"segmentFilename,omitempty"`
	SessionID          string `json:"sessionId,omitempty"`
	ViewID             string `json:"viewId,omitempty"`
	Start              string `json:"start,omitempty"`
	End                string `json:"end,omitempty"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp,omitempty"`
	Level     string `json:"level,omitempty"`
	Message   string `json:"message,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
}

type TraceDetail struct {
	TraceID string       `json:"traceId,omitempty"`
	Spans   []SpanDetail `json:"spans,omitempty"`
}

type MetricEntry struct {
	Name        string  `json:"name,omitempty"`
	Service     string  `json:"service,omitempty"`
	Unit        string  `json:"unit,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Aggregation string  `json:"aggregation,omitempty"`
	Route       string  `json:"route,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
}

type SpanDetail struct {
	TraceID       string  `json:"traceId,omitempty"`
	SpanID        string  `json:"spanId,omitempty"`
	ParentSpanID  string  `json:"parentSpanId,omitempty"`
	Name          string  `json:"name,omitempty"`
	Resource      string  `json:"resource,omitempty"`
	Service       string  `json:"service,omitempty"`
	Start         string  `json:"start,omitempty"`
	DurationMS    float64 `json:"durationMs,omitempty"`
	Error         bool    `json:"error,omitempty"`
	NormalizedRef string  `json:"normalizedRef,omitempty"`
}

type NormalizedTelemetry struct {
	Service      string            `json:"service,omitempty"`
	Env          string            `json:"env,omitempty"`
	Version      string            `json:"version,omitempty"`
	Host         string            `json:"host,omitempty"`
	Source       Source            `json:"source"`
	Timestamp    string            `json:"timestamp,omitempty"`
	TraceID      string            `json:"traceId,omitempty"`
	SpanID       string            `json:"spanId,omitempty"`
	ParentSpanID string            `json:"parentSpanId,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
	ViewID       string            `json:"viewId,omitempty"`
	UserID       string            `json:"userId,omitempty"`
	AccountID    string            `json:"accountId,omitempty"`
	WorkspaceID  string            `json:"workspaceId,omitempty"`
	CaseID       string            `json:"caseId,omitempty"`
	Route        string            `json:"route,omitempty"`
	Method       string            `json:"method,omitempty"`
	StatusCode   int               `json:"statusCode,omitempty"`
	DurationMS   float64           `json:"durationMs,omitempty"`
	ErrorType    string            `json:"errorType,omitempty"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

type ValidationResult struct {
	Status  string                 `json:"status"`
	Rules   []ValidationRuleResult `json:"rules"`
	Summary string                 `json:"summary,omitempty"`
}

type ValidationRuleResult struct {
	RuleID    string `json:"ruleId"`
	Severity  string `json:"severity"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	FieldPath string `json:"fieldPath,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
}

type ForwardingResult struct {
	Mode         string `json:"mode"`
	Attempted    bool   `json:"attempted"`
	Target       string `json:"target,omitempty"`
	Status       string `json:"status"`
	StatusCode   int    `json:"statusCode,omitempty"`
	DurationMS   int64  `json:"durationMs,omitempty"`
	RetryCount   int    `json:"retryCount,omitempty"`
	ErrorClass   string `json:"errorClass,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
