package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/layout"
)

func TestParseOAuthPaste(t *testing.T) {
	code, state := parseOAuthPaste("https://example.com/cb?code=abc&state=st")
	if code != "abc" || state != "st" {
		t.Fatalf("code=%q state=%q", code, state)
	}
	code, state = parseOAuthPaste("abc#st")
	if code != "abc" || state != "st" {
		t.Fatalf("code=%q state=%q", code, state)
	}
	code, state = parseOAuthPaste("abc")
	if code != "abc" || state != "" {
		t.Fatalf("code=%q state=%q", code, state)
	}
}

func TestAuthProviderCompletion(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyMenuIndex = 0
	m.quickReplyInput.SetValue("/auth ")
	if ok := m.applyAuthProviderCompletion(); ok {
		t.Fatalf("expected no completion")
	}
	if m.quickReplyInput.Value() != "/auth " {
		t.Fatalf("value=%q", m.quickReplyInput.Value())
	}
}

func TestAuthMethodCompletion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	m := newTestModelLite()
	m.quickReplyMenuIndex = 0
	m.quickReplyInput.SetValue("/auth google ")
	if ok := m.applyAuthMethodCompletion(); ok {
		t.Fatalf("expected no completion")
	}
	if m.quickReplyInput.Value() != "/auth google " {
		t.Fatalf("value=%q", m.quickReplyInput.Value())
	}
}

func TestModelCompletion(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{
		Agent: layout.AgentConfig{
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
	}
	m.quickReplyMenuIndex = 0
	m.quickReplyInput.SetValue("/model ")
	if ok := m.applyModelCompletion(); ok {
		t.Fatalf("expected no completion")
	}
	if m.quickReplyInput.Value() != "/model " {
		t.Fatalf("value=%q", m.quickReplyInput.Value())
	}
}

func TestAuthProviderDescAndSplitAuthCode(t *testing.T) {
	info := agent.ProviderInfo{SupportsAPIKey: true, SupportsOAuth: true}
	if got := authProviderDesc(info); got != "api key, oauth" {
		t.Fatalf("desc=%q", got)
	}
	code, state, ok := splitAuthCode("a#b")
	if !ok || code != "a" || state != "b" {
		t.Fatalf("code=%q state=%q ok=%v", code, state, ok)
	}
}
