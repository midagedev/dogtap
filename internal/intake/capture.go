package intake

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/redact"
)

var nextID uint64

var ErrUnsupportedContentType = errors.New("unsupported content type")

type CaptureOptions struct {
	Source           event.Source
	AllowRawPayloads bool
	MaxBodyBytes     int64
	ForwardMode      string
}

type CaptureResult struct {
	Event           event.EventEnvelope
	ValidationEvent event.EventEnvelope
	ForwardBody     []byte
	ForwardHeader   http.Header
}

func CaptureRequest(r *http.Request, opts CaptureOptions) (CaptureResult, error) {
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = 10 << 20
	}
	raw, err := readLimited(r.Body, opts.MaxBodyBytes)
	if err != nil {
		return CaptureResult{}, err
	}
	decodedBytes, err := decodeContent(raw, r.Header.Get("Content-Encoding"))
	if err != nil {
		return CaptureResult{}, err
	}
	payloadKind := detectPayloadKind(opts.Source, r.URL.Path, r.URL.Query())
	decoded, decodeErr := decodePayload(decodedBytes, r.Header.Get("Content-Type"), opts.Source, payloadKind)
	if decodeErr != nil {
		if errors.Is(decodeErr, ErrUnsupportedContentType) {
			return CaptureResult{}, decodeErr
		}
		decoded = map[string]any{
			"decodeError": decodeErr.Error(),
			"text":        string(decodedBytes),
		}
	}

	normalized := Normalize(opts.Source, decoded)
	normalized.Source = opts.Source
	if normalized.Method == "" {
		normalized.Method = r.Method
	}
	mergeRequestTags(&normalized, r.URL.Query())

	rawBody := ""
	storedDecoded := decoded
	storedNormalized := normalized
	if opts.AllowRawPayloads {
		rawBody = string(decodedBytes)
	} else {
		storedDecoded = redact.Value(decoded)
		storedNormalized = RedactNormalized(normalized)
	}
	storedDetails := BuildDetails(opts.Source, payloadKind, storedDecoded, storedNormalized, r.Header.Get("Content-Type"), len(decodedBytes))
	validationDetails := BuildDetails(opts.Source, payloadKind, decoded, normalized, r.Header.Get("Content-Type"), len(decodedBytes))

	e := event.EventEnvelope{
		ID:               NewID(opts.Source),
		ReceivedAt:       time.Now().UTC(),
		Source:           opts.Source,
		PayloadKind:      payloadKind,
		Endpoint:         r.URL.Path,
		Method:           r.Method,
		Headers:          redact.HeaderMap(r.Header),
		Query:            redact.Query(r.URL.Query()),
		ContentType:      r.Header.Get("Content-Type"),
		ContentEncoding:  r.Header.Get("Content-Encoding"),
		BodySizeBytes:    int64(len(raw)),
		DecodedSizeBytes: int64(len(decodedBytes)),
		RawBody:          rawBody,
		Decoded:          storedDecoded,
		Details:          storedDetails,
		Normalized:       storedNormalized,
		Forwarding: event.ForwardingResult{
			Mode:      opts.ForwardMode,
			Attempted: false,
			Status:    "disabled",
		},
	}
	validationEvent := e
	validationEvent.RawBody = string(decodedBytes)
	validationEvent.Decoded = decoded
	validationEvent.Details = validationDetails
	validationEvent.Normalized = normalized
	return CaptureResult{
		Event:           e,
		ValidationEvent: validationEvent,
		ForwardBody:     append([]byte(nil), raw...),
		ForwardHeader:   r.Header.Clone(),
	}, nil
}

func detectPayloadKind(source event.Source, path string, query url.Values) string {
	switch source {
	case event.SourceRUM:
		if forwardedPath(query, "/api/v2/replay") || strings.Contains(path, "replay") {
			return "replay"
		}
		return "rum"
	case event.SourceLogs:
		return "log"
	case event.SourceAPM:
		return "trace"
	case event.SourceOTLP:
		switch {
		case strings.Contains(path, "/logs"):
			return "log"
		case strings.Contains(path, "/metrics"):
			return "metric"
		default:
			return "trace"
		}
	default:
		return string(source)
	}
}

func forwardedPath(query url.Values, want string) bool {
	for _, forwarded := range query["ddforward"] {
		parsed, err := url.Parse(forwarded)
		if err != nil {
			continue
		}
		if parsed.Path == want {
			return true
		}
	}
	return false
}

func RedactNormalized(n event.NormalizedTelemetry) event.NormalizedTelemetry {
	n.Service = redact.Text(n.Service)
	n.Env = redact.Text(n.Env)
	n.Version = redact.Text(n.Version)
	n.Host = redact.Text(n.Host)
	n.Timestamp = redact.Text(n.Timestamp)
	n.TraceID = redact.Text(n.TraceID)
	n.SpanID = redact.Text(n.SpanID)
	n.ParentSpanID = redact.Text(n.ParentSpanID)
	n.SessionID = redact.Text(n.SessionID)
	n.ViewID = redact.Text(n.ViewID)
	n.UserID = redact.Text(n.UserID)
	n.AccountID = redact.Text(n.AccountID)
	n.WorkspaceID = redact.Text(n.WorkspaceID)
	n.CaseID = redact.Text(n.CaseID)
	n.Route = redact.Text(n.Route)
	n.Method = redact.Text(n.Method)
	n.ErrorType = redact.Text(n.ErrorType)
	n.ErrorMessage = redact.Text(n.ErrorMessage)
	if len(n.Tags) > 0 {
		tags := make(map[string]string, len(n.Tags))
		for key, value := range n.Tags {
			if redact.IsSensitiveKey(key) {
				tags[key] = redact.Mask
			} else {
				tags[key] = redact.Text(value)
			}
		}
		n.Tags = tags
	}
	return n
}

func mergeRequestTags(n *event.NormalizedTelemetry, query url.Values) {
	tags := requestTags(query)
	if len(tags) == 0 {
		return
	}
	if n.Tags == nil {
		n.Tags = map[string]string{}
	}
	for key, value := range tags {
		if _, exists := n.Tags[key]; !exists && value != "" {
			n.Tags[key] = value
		}
	}
	n.Service = coalesce(n.Service, tags["service"], tags["service.name"])
	n.Env = coalesce(n.Env, tags["env"], tags["deployment.environment"], tags["deployment.environment.name"])
	n.Version = coalesce(n.Version, tags["version"], tags["service.version"])
}

func requestTags(query url.Values) map[string]string {
	tags := map[string]string{}
	for _, value := range query["ddtags"] {
		mergeTagString(tags, value)
	}
	for _, forwarded := range query["ddforward"] {
		parsed, err := url.Parse(forwarded)
		if err != nil {
			continue
		}
		for _, value := range parsed.Query()["ddtags"] {
			mergeTagString(tags, value)
		}
	}
	return tags
}

func mergeTagString(tags map[string]string, tagString string) {
	for _, part := range strings.Split(tagString, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), ":")
		if ok && key != "" {
			tags[key] = value
		}
	}
}

func readLimited(r io.Reader, max int64) ([]byte, error) {
	lr := io.LimitReader(r, max+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("payload exceeds max body size %d", max)
	}
	return b, nil
}

func decodeContent(raw []byte, encoding string) ([]byte, error) {
	if !strings.Contains(strings.ToLower(encoding), "gzip") {
		return raw, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode gzip: %w", err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	return decoded, nil
}

func decodePayload(body []byte, contentType string, source event.Source, payloadKind string) (any, error) {
	mediaType := contentType
	params := map[string]string{}
	if parsed, parsedParams, err := mime.ParseMediaType(contentType); err == nil {
		mediaType = parsed
		params = parsedParams
	}
	if mediaType == "" {
		mediaType = "application/json"
	}
	if source == event.SourceRUM {
		if payloadKind == "replay" {
			return decodeReplayPayload(body, contentType, mediaType, params), nil
		}
		if decoded, ok := decodeJSONBatch(body); ok {
			return decoded, nil
		}
		if isTextContent(mediaType) {
			return map[string]any{"message": string(body)}, nil
		}
		return nil, fmt.Errorf("%w %q for %s", ErrUnsupportedContentType, contentType, source)
	}

	if source == event.SourceLogs {
		if strings.Contains(mediaType, "json") {
			var decoded any
			if err := json.Unmarshal(body, &decoded); err == nil {
				return decoded, nil
			}
		}
		if isTextContent(mediaType) || mediaType == "application/logplex-1" {
			return map[string]any{"message": string(body)}, nil
		}
		return nil, fmt.Errorf("%w %q for %s", ErrUnsupportedContentType, contentType, source)
	}

	if source == event.SourceAPM {
		if strings.Contains(mediaType, "msgpack") || mediaType == "application/octet-stream" {
			var decoded any
			if err := msgpack.Unmarshal(body, &decoded); err == nil {
				return normalizeMapKeys(decoded), nil
			}
		}
		if strings.Contains(mediaType, "json") {
			var decoded any
			if err := json.Unmarshal(body, &decoded); err == nil {
				return decoded, nil
			}
		}
		return nil, fmt.Errorf("%w %q for %s", ErrUnsupportedContentType, contentType, source)
	}

	if source == event.SourceOTLP {
		if strings.Contains(mediaType, "json") {
			var decoded any
			if err := json.Unmarshal(body, &decoded); err == nil {
				return decoded, nil
			}
		}
		if strings.Contains(mediaType, "protobuf") || mediaType == "application/octet-stream" {
			return map[string]any{"bytes": len(body), "contentType": contentType}, nil
		}
		return nil, fmt.Errorf("%w %q for %s", ErrUnsupportedContentType, contentType, source)
	}

	if strings.Contains(mediaType, "json") {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err == nil {
			return decoded, nil
		}
	}
	if source == event.SourceAPM || strings.Contains(mediaType, "msgpack") {
		var decoded any
		if err := msgpack.Unmarshal(body, &decoded); err == nil {
			return normalizeMapKeys(decoded), nil
		}
	}
	if isTextContent(mediaType) || mediaType == "application/logplex-1" {
		return map[string]any{"message": string(body)}, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err == nil {
		return decoded, nil
	}
	return map[string]any{"bytes": len(body), "contentType": contentType}, nil
}

func decodeReplayPayload(body []byte, contentType string, mediaType string, params map[string]string) any {
	if strings.HasPrefix(mediaType, "multipart/") {
		return decodeReplayMultipart(body, contentType, params)
	}
	if decoded, ok := decodeJSONBatch(body); ok {
		return map[string]any{
			"replay": map[string]any{
				"format":      "json",
				"contentType": contentType,
				"bytes":       len(body),
				"frames":      replayFrameCount(decoded),
			},
			"records": decoded,
		}
	}
	if isTextContent(mediaType) {
		return map[string]any{
			"replay": map[string]any{
				"format":      "text",
				"contentType": contentType,
				"bytes":       len(body),
			},
			"text": string(body),
		}
	}
	return map[string]any{
		"replay": map[string]any{
			"format":      "binary",
			"contentType": contentType,
			"bytes":       len(body),
		},
	}
}

func decodeReplayMultipart(body []byte, contentType string, params map[string]string) any {
	summary := map[string]any{
		"format":      "multipart",
		"contentType": contentType,
		"bytes":       len(body),
	}
	decoded := map[string]any{"replay": summary}
	boundary := params["boundary"]
	if boundary == "" {
		summary["decodeError"] = "missing multipart boundary"
		return decoded
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			summary["decodeError"] = err.Error()
			break
		}
		partBytes, err := io.ReadAll(part)
		if err != nil {
			summary["decodeError"] = err.Error()
			break
		}
		switch part.FormName() {
		case "event":
			var metadata any
			if err := json.Unmarshal(partBytes, &metadata); err != nil {
				decoded["event"] = map[string]any{"decodeError": err.Error(), "bytes": len(partBytes)}
			} else {
				metadata = normalizeMapKeys(metadata)
				decoded["event"] = metadata
				summary["recordCount"] = firstPositiveInt(findInt(metadata, "records_count"), findInt(metadata, "recordsCount"))
			}
		case "segment":
			summary["segmentBytes"] = len(partBytes)
			summary["segmentContentType"] = part.Header.Get("Content-Type")
			if filename := part.FileName(); filename != "" {
				summary["segmentFilename"] = filename
			}
			if records, encoding, ok := decodeReplaySegment(partBytes); ok {
				decoded["records"] = records
				summary["segmentEncoding"] = encoding
				summary["frames"] = replayFrameCount(records)
				if summary["recordCount"] == nil || summary["recordCount"] == 0 {
					summary["recordCount"] = replayFrameCount(records)
				}
			}
		default:
			if name := part.FormName(); name != "" {
				decoded[name] = map[string]any{"bytes": len(partBytes), "contentType": part.Header.Get("Content-Type")}
			}
		}
	}
	return decoded
}

func decodeReplaySegment(body []byte) (any, string, bool) {
	if decoded, ok := decodeJSONBatch(body); ok {
		return decoded, "identity", true
	}
	attempts := []struct {
		name string
		fn   func([]byte) ([]byte, error)
	}{
		{name: "gzip", fn: decodeGzipBytes},
		{name: "zlib", fn: decodeZlibBytes},
		{name: "deflate", fn: decodeDeflateBytes},
	}
	for _, attempt := range attempts {
		inflated, err := attempt.fn(body)
		if err != nil {
			continue
		}
		if decoded, ok := decodeJSONBatch(inflated); ok {
			return decoded, attempt.name, true
		}
	}
	return nil, "", false
}

func decodeGzipBytes(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func decodeZlibBytes(body []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func decodeDeflateBytes(body []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(body))
	defer reader.Close()
	return io.ReadAll(reader)
}

func replayFrameCount(decoded any) int {
	switch typed := decoded.(type) {
	case []any:
		return len(typed)
	case map[string]any:
		for _, key := range []string{"records", "events", "segments"} {
			if frames, ok := typed[key].([]any); ok {
				return len(frames)
			}
		}
	}
	return 0
}

func isTextContent(mediaType string) bool {
	return strings.HasPrefix(mediaType, "text/")
}

func decodeJSONBatch(body []byte) (any, bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err == nil {
		return decoded, true
	}
	lines := bytes.Split(trimmed, []byte{'\n'})
	events := make([]any, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var item any
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, false
		}
		events = append(events, item)
	}
	if len(events) == 0 {
		return nil, false
	}
	return events, true
}

func normalizeMapKeys(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			out[k] = normalizeMapKeys(value)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			out[fmt.Sprint(k)] = normalizeMapKeys(value)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, value := range typed {
			out[i] = normalizeMapKeys(value)
		}
		return out
	default:
		return typed
	}
}

func NewID(source event.Source) string {
	id := atomic.AddUint64(&nextID, 1)
	return string(source) + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36) + "-" + strconv.FormatUint(id, 36)
}
