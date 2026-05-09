package redact

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const Mask = "***REDACTED***"

var (
	emailPattern            = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	bearerTokenPattern      = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/\-]+=*`)
	basicTokenPattern       = regexp.MustCompile(`(?i)\bbasic\s+[a-z0-9._~+/\-]+=*`)
	secretAssignmentPattern = regexp.MustCompile(`(?i)\b(access[_-]?token|refresh[_-]?token|id[_-]?token|api[_-]?key|authorization|password|secret)\b\s*[:=]\s*["']?[^"'\s,}&]+`)
	privateKeyPattern       = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
)

var sensitiveKeys = []string{
	"authorization",
	"cookie",
	"set-cookie",
	"api_key",
	"apikey",
	"dd-api-key",
	"dd-application-key",
	"access_token",
	"refresh_token",
	"id_token",
	"token",
	"password",
	"secret",
}

func HeaderMap(headers map[string][]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, values := range headers {
		joined := strings.Join(values, ",")
		if IsSensitiveKey(k) {
			out[k] = Mask
			continue
		}
		out[k] = Text(joined)
	}
	return out
}

func Query(q map[string][]string) map[string][]string {
	out := make(map[string][]string, len(q))
	for k, values := range q {
		masked := make([]string, len(values))
		for i, value := range values {
			if IsSensitiveKey(k) {
				masked[i] = Mask
			} else {
				masked[i] = Text(value)
			}
		}
		out[k] = masked
	}
	return out
}

func Value(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			if IsSensitiveKey(k) {
				out[k] = Mask
				continue
			}
			out[k] = Value(value)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, value := range typed {
			out[i] = Value(value)
		}
		return out
	case string:
		return Text(typed)
	default:
		return typed
	}
}

func JSONText(raw []byte) string {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Text(string(raw))
	}
	redacted := Value(decoded)
	b, err := json.MarshalIndent(redacted, "", "  ")
	if err != nil {
		return Text(fmt.Sprint(redacted))
	}
	return string(b)
}

func Text(s string) string {
	s = emailPattern.ReplaceAllString(s, Mask)
	s = bearerTokenPattern.ReplaceAllString(s, "Bearer "+Mask)
	s = basicTokenPattern.ReplaceAllString(s, "Basic "+Mask)
	s = secretAssignmentPattern.ReplaceAllStringFunc(s, func(match string) string {
		if key, _, ok := strings.Cut(match, ":"); ok {
			return strings.TrimSpace(key) + ": " + Mask
		}
		if key, _, ok := strings.Cut(match, "="); ok {
			return strings.TrimSpace(key) + "=" + Mask
		}
		return Mask
	})
	s = privateKeyPattern.ReplaceAllString(s, Mask)
	return s
}

func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, sensitive := range sensitiveKeys {
		if normalized == sensitive || strings.Contains(normalized, sensitive) {
			return true
		}
	}
	return false
}
