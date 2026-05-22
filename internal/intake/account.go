package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"

	"github.com/midagedev/dogtap/internal/event"
)

// AccountHeader is the explicit, recommended way for a caller to route an
// intake request into a tenant namespace.
const AccountHeader = "X-Dogtap-Account"

const maxAccountLen = 64

// AccountFromRequest derives the Dogtap account (tenant namespace) for an
// intake request so multiple services can share one Dogtap instance without
// their telemetry mixing.
//
// Resolution order:
//  1. An explicit X-Dogtap-Account header (used verbatim, sanitized).
//  2. The Datadog client token (dd-api-key), hashed into a stable opaque id so
//     a real API key is never stored or listed in clear text.
//  3. event.DefaultAccount.
func AccountFromRequest(r *http.Request) string {
	if explicit := SanitizeAccount(r.Header.Get(AccountHeader)); explicit != "" {
		return explicit
	}
	if key := datadogAPIKey(r); key != "" {
		sum := sha256.Sum256([]byte(key))
		return "key-" + hex.EncodeToString(sum[:])[:12]
	}
	return event.DefaultAccount
}

// AccountFromValues derives an account from raw header/query carriers. It backs
// AccountFromRequest and the gRPC intake path, which only has metadata.
func AccountFromValues(explicit, apiKey string) string {
	if account := SanitizeAccount(explicit); account != "" {
		return account
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		sum := sha256.Sum256([]byte(apiKey))
		return "key-" + hex.EncodeToString(sum[:])[:12]
	}
	return event.DefaultAccount
}

func datadogAPIKey(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("DD-API-KEY")); v != "" {
		return v
	}
	query := r.URL.Query()
	if v := strings.TrimSpace(query.Get("dd-api-key")); v != "" {
		return v
	}
	// Browser RUM intake routed through a proxy carries the original query
	// string (including dd-api-key) inside the ddforward parameter.
	for _, forwarded := range query["ddforward"] {
		parsed, err := url.Parse(forwarded)
		if err != nil {
			continue
		}
		if v := strings.TrimSpace(parsed.Query().Get("dd-api-key")); v != "" {
			return v
		}
	}
	return ""
}

// SanitizeAccount keeps account ids bounded and safe to use in URLs, log lines,
// and SQL filters. Disallowed characters collapse to '-'.
func SanitizeAccount(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
		if b.Len() >= maxAccountLen {
			break
		}
	}
	return b.String()
}
