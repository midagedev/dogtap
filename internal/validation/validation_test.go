package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/midagedev/dogtap/internal/config"
	"github.com/midagedev/dogtap/internal/event"
	"github.com/midagedev/dogtap/internal/intake"
)

func TestValidatorFailsMissingRequiredRUMContext(t *testing.T) {
	v := New(config.Default().Validation)
	result := v.Validate(event.EventEnvelope{
		Source:     event.SourceRUM,
		Normalized: event.NormalizedTelemetry{Source: event.SourceRUM, UserID: "u1"},
	})
	if result.Status != "fail" {
		t.Fatalf("got status %q, want fail", result.Status)
	}
}

func TestValidatorDetectsSensitiveQuery(t *testing.T) {
	v := New(config.Default().Validation)
	result := v.Validate(event.EventEnvelope{
		Source: event.SourceLogs,
		Query:  map[string][]string{"access_token": []string{"secret"}},
		Normalized: event.NormalizedTelemetry{
			Source:  event.SourceLogs,
			Service: "api",
			Env:     "local",
		},
	})
	if result.Status != "fail" {
		t.Fatalf("got status %q, want fail", result.Status)
	}
	requireRule(t, result, "secret.key.query_access_token")
	for _, rule := range result.Rules {
		if strings.Contains(rule.Evidence, "secret") {
			t.Fatalf("rule %s leaked secret evidence %q", rule.RuleID, rule.Evidence)
		}
	}
}

func TestValidatorDetectsQueryStringTokenEmailAndCardinality(t *testing.T) {
	body := readFixture(t, "leaky-log.json")
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	normalized := intake.Normalize(event.SourceLogs, decoded)
	result := New(config.Default().Validation).Validate(event.EventEnvelope{
		Source: event.SourceLogs,
		Headers: map[string]string{
			"Authorization": "Bearer header-token-value",
		},
		Query: map[string][]string{
			"email":    {"operator@example.com"},
			"redirect": {"/callback?code=12345"},
		},
		RawBody:    string(body),
		Decoded:    decoded,
		Normalized: normalized,
	})

	if result.Status != "fail" {
		t.Fatalf("got status %q, want fail", result.Status)
	}
	requireRule(t, result, "leak.query_string.normalized_route")
	requireRule(t, result, "leak.query_string.query_redirect")
	requireRule(t, result, "secret.key.headers_authorization")
	requireRule(t, result, "secret.pattern.bearer.headers_authorization")
	requireRule(t, result, "pii.email.query_email")
	requireRule(t, result, "cardinality.tag.user_id")
}

func TestValidatorHighCardinalityHintsAreWarnings(t *testing.T) {
	result := New(config.Default().Validation).Validate(event.EventEnvelope{
		Source: event.SourceLogs,
		Normalized: event.NormalizedTelemetry{
			Source:  event.SourceLogs,
			Service: "api",
			Env:     "local",
			Route:   "/api/users/550e8400-e29b-41d4-a716-446655440000/profile",
			Tags: map[string]string{
				"request_id": "req-123",
			},
		},
	})

	if result.Status != "pass" {
		t.Fatalf("got status %q, want pass for warnings only", result.Status)
	}
	requireRule(t, result, "cardinality.tag.request_id")
	requireRule(t, result, "cardinality.route.dynamic_segment")
}

func TestValidatorRequiresExactCustomTag(t *testing.T) {
	cfg := config.Default().Validation
	cfg.Required.RUM = []string{"tenant.id", "workspace_id"}
	v := New(cfg)

	result := v.Validate(event.EventEnvelope{
		Source: event.SourceRUM,
		Normalized: event.NormalizedTelemetry{
			Source:      event.SourceRUM,
			Service:     "web-frontend",
			Env:         "local",
			WorkspaceID: "workspace-1",
			Tags: map[string]string{
				"unrelated": "present",
			},
		},
	})

	if result.Status != "fail" {
		t.Fatalf("got status %q, want fail", result.Status)
	}
	requireRule(t, result, "required.rum.tenant.id")

	result = v.Validate(event.EventEnvelope{
		Source: event.SourceRUM,
		Normalized: event.NormalizedTelemetry{
			Source:      event.SourceRUM,
			Service:     "web-frontend",
			Env:         "local",
			WorkspaceID: "workspace-1",
			Tags: map[string]string{
				"tenant.id": "tenant-1",
			},
		},
	})

	if result.Status != "pass" {
		t.Fatalf("got status %q, want pass: %+v", result.Status, result.Rules)
	}
}

func TestValidateBatchDetectsRUMLogoutContextLeak(t *testing.T) {
	v := New(config.Default().Validation)
	events := []event.EventEnvelope{
		{
			Source: event.SourceRUM,
			Decoded: map[string]any{
				"action": "clearUser",
			},
			Normalized: event.NormalizedTelemetry{
				Source:      event.SourceRUM,
				Service:     "web-frontend",
				Env:         "local",
				UserID:      "user-1",
				AccountID:   "account-1",
				WorkspaceID: "workspace-1",
				SessionID:   "session-1",
				Route:       "/logout",
			},
		},
		{
			Source: event.SourceRUM,
			Normalized: event.NormalizedTelemetry{
				Source:      event.SourceRUM,
				Service:     "web-frontend",
				Env:         "local",
				UserID:      "user-1",
				AccountID:   "account-1",
				WorkspaceID: "workspace-1",
				SessionID:   "session-1",
				Route:       "/login",
			},
		},
	}

	validated := v.ValidateBatch(events)
	if validated[1].Validation.Status != "fail" {
		t.Fatalf("got status %q, want fail", validated[1].Validation.Status)
	}
	requireRule(t, validated[1].Validation, "context.rum.logout.user")
	requireRule(t, validated[1].Validation, "context.rum.logout.account")
}

func TestValidateBatchDetectsFaroLogoutContextLeak(t *testing.T) {
	v := New(config.Default().Validation)
	events := []event.EventEnvelope{
		{
			Source: event.SourceFaro,
			Decoded: map[string]any{
				"events": []any{map[string]any{"name": "clearUser"}},
			},
			Normalized: event.NormalizedTelemetry{
				Source:      event.SourceFaro,
				Service:     "web-frontend",
				Env:         "local",
				UserID:      "user-1",
				AccountID:   "account-1",
				WorkspaceID: "workspace-1",
				SessionID:   "session-1",
				Route:       "/logout",
			},
		},
		{
			Source: event.SourceFaro,
			Normalized: event.NormalizedTelemetry{
				Source:      event.SourceFaro,
				Service:     "web-frontend",
				Env:         "local",
				UserID:      "user-1",
				AccountID:   "account-1",
				WorkspaceID: "workspace-1",
				SessionID:   "session-1",
				Route:       "/login",
			},
		},
	}

	validated := v.ValidateBatch(events)
	if validated[1].Validation.Status != "fail" {
		t.Fatalf("got status %q, want fail", validated[1].Validation.Status)
	}
	requireRule(t, validated[1].Validation, "context.rum.logout.user")
	requireRule(t, validated[1].Validation, "context.rum.logout.account")
}

func TestValidateBatchDetectsRUMWorkspaceSwitchLeak(t *testing.T) {
	v := New(config.Default().Validation)
	events := []event.EventEnvelope{
		{
			Source: event.SourceRUM,
			Decoded: map[string]any{
				"action":            "workspace.switch",
				"targetWorkspaceId": "workspace-new",
			},
			Normalized: event.NormalizedTelemetry{
				Source:      event.SourceRUM,
				Service:     "web-frontend",
				Env:         "local",
				UserID:      "user-1",
				AccountID:   "account-1",
				WorkspaceID: "workspace-old",
				SessionID:   "session-1",
				Route:       "/workspaces/switch",
				Tags: map[string]string{
					"targetWorkspaceId": "workspace-new",
				},
			},
		},
		{
			Source: event.SourceRUM,
			Normalized: event.NormalizedTelemetry{
				Source:      event.SourceRUM,
				Service:     "web-frontend",
				Env:         "local",
				UserID:      "user-1",
				AccountID:   "account-1",
				WorkspaceID: "workspace-old",
				SessionID:   "session-1",
				Route:       "/cases/case-1",
			},
		},
	}

	validated := v.ValidateBatch(events)
	if validated[1].Validation.Status != "fail" {
		t.Fatalf("got status %q, want fail", validated[1].Validation.Status)
	}
	requireRule(t, validated[1].Validation, "context.rum.workspace_switch.stale_workspace")
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "validation", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func requireRule(t *testing.T, result event.ValidationResult, ruleID string) {
	t.Helper()
	for _, rule := range result.Rules {
		if rule.RuleID == ruleID {
			return
		}
	}
	ids := make([]string, 0, len(result.Rules))
	for _, rule := range result.Rules {
		ids = append(ids, rule.RuleID)
	}
	t.Fatalf("missing rule %q in %v", ruleID, ids)
}
