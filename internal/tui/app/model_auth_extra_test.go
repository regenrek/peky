package app

import (
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
)

func TestParseSlashCommandInput(t *testing.T) {
	input, ok := parseSlashCommandInput("/auth openai ")
	if !ok || input.Command != "auth" || len(input.Args) != 1 || !input.TrailingSpace {
		t.Fatalf("input=%#v ok=%v", input, ok)
	}
	if _, ok := parseSlashCommandInput("nope"); ok {
		t.Fatalf("expected false for non-slash")
	}
}

func TestParseOAuthPasteExtra(t *testing.T) {
	code, state := parseOAuthPaste("https://example.com/callback?code=abc&state=xyz")
	if code != "abc" || state != "xyz" {
		t.Fatalf("code=%q state=%q", code, state)
	}
	code, state = parseOAuthPaste("abc#xyz")
	if code != "abc" || state != "xyz" {
		t.Fatalf("code=%q state=%q", code, state)
	}
	code, state = parseOAuthPaste("token")
	if code != "token" || state != "" {
		t.Fatalf("code=%q state=%q", code, state)
	}
}

func TestAuthProviderDescAndSuggestion(t *testing.T) {
	desc := authProviderDesc(agent.ProviderInfo{SupportsAPIKey: true, SupportsOAuth: true})
	if desc != "api key, oauth" {
		t.Fatalf("desc=%q", desc)
	}
	s := authMethodSuggestion("oauth", "Login via OAuth", "o")
	if s.Text == "" || s.MatchLen != 1 {
		t.Fatalf("suggestion=%#v", s)
	}
	if s := authMethodSuggestion("oauth", "Login", "x"); s.Text != "" {
		t.Fatalf("expected filtered suggestion")
	}
}

func TestModelMenuIncludesCurrent(t *testing.T) {
	m := newTestModelLite()
	m.config.Agent.Provider = string(agent.ProviderOpenAI)
	m.config.Agent.Model = "custom-model"
	menu := m.modelMenu("")
	foundCurrent := false
	for _, sug := range menu.suggestions {
		if strings.TrimSpace(sug.Value) == "custom-model" {
			foundCurrent = true
			break
		}
	}
	if !foundCurrent {
		t.Fatalf("expected current model in suggestions")
	}
}

func TestSplitAuthCodeAndContains(t *testing.T) {
	code, state, ok := splitAuthCode("abc#xyz")
	if !ok || code != "abc" || state != "xyz" {
		t.Fatalf("code=%q state=%q ok=%v", code, state, ok)
	}
	if stringSliceContains([]string{"a", "b"}, "c") {
		t.Fatalf("expected missing value")
	}
}
