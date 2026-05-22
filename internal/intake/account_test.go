package intake

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/midagedev/dogtap/internal/event"
)

func TestAccountFromRequestExplicitHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/rum", nil)
	req.Header.Set(AccountHeader, "checkout-suite")
	if got := AccountFromRequest(req); got != "checkout-suite" {
		t.Fatalf("account = %q, want checkout-suite", got)
	}
}

func TestAccountFromRequestHeaderWinsOverAPIKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/rum", nil)
	req.Header.Set(AccountHeader, "set-a")
	req.Header.Set("DD-API-KEY", "token-xyz")
	if got := AccountFromRequest(req); got != "set-a" {
		t.Fatalf("account = %q, want set-a", got)
	}
}

func TestAccountFromRequestHashesAPIKey(t *testing.T) {
	header := httptest.NewRequest(http.MethodPost, "/api/v2/logs", nil)
	header.Header.Set("DD-API-KEY", "token-xyz")

	query := httptest.NewRequest(http.MethodPost, "/rum?dd-api-key=token-xyz", nil)

	hdr := AccountFromRequest(header)
	qry := AccountFromRequest(query)
	if hdr != qry {
		t.Fatalf("header-derived %q != query-derived %q", hdr, qry)
	}
	if !strings.HasPrefix(hdr, "key-") {
		t.Fatalf("hashed account = %q, want key- prefix", hdr)
	}
	if hdr == "token-xyz" {
		t.Fatalf("raw api key leaked into account id")
	}
}

func TestAccountFromRequestDistinctKeysDistinctAccounts(t *testing.T) {
	a := httptest.NewRequest(http.MethodPost, "/api/v2/logs", nil)
	a.Header.Set("DD-API-KEY", "token-a")
	b := httptest.NewRequest(http.MethodPost, "/api/v2/logs", nil)
	b.Header.Set("DD-API-KEY", "token-b")
	if AccountFromRequest(a) == AccountFromRequest(b) {
		t.Fatal("different client tokens collapsed into the same account")
	}
}

func TestAccountFromRequestDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/rum", nil)
	if got := AccountFromRequest(req); got != event.DefaultAccount {
		t.Fatalf("account = %q, want %q", got, event.DefaultAccount)
	}
}

func TestAccountFromRequestProxyForwardedAPIKey(t *testing.T) {
	// Browser RUM routed through a proxy carries dd-api-key inside ddforward.
	req := httptest.NewRequest(http.MethodPost,
		"/datadog-intake-proxy?ddforward=%2Fapi%2Fv2%2Frum%3Fdd-api-key%3Dtoken-xyz", nil)
	plain := httptest.NewRequest(http.MethodPost, "/rum?dd-api-key=token-xyz", nil)
	if AccountFromRequest(req) != AccountFromRequest(plain) {
		t.Fatal("ddforward-embedded dd-api-key not resolved to the same account")
	}
}

func TestSanitizeAccount(t *testing.T) {
	cases := map[string]string{
		"  set 1 ":               "set-1",
		"a/b":                    "a-b",
		"ok-_.name":              "ok-_.name",
		"":                       "",
		"   ":                    "",
		strings.Repeat("x", 100): strings.Repeat("x", maxAccountLen),
	}
	for in, want := range cases {
		if got := SanitizeAccount(in); got != want {
			t.Errorf("SanitizeAccount(%q) = %q, want %q", in, got, want)
		}
	}
}
