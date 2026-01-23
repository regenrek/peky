package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
)

func TestHandleAuthProviderOnlyAPIKey(t *testing.T) {
	m := newTestModelLite()
	info := agent.ProviderInfo{ID: agent.ProviderOpenAI, SupportsAPIKey: true}
	out := m.handleAuthProviderOnly(info)
	if !out.Handled || out.Cmd == nil {
		t.Fatalf("expected handled outcome")
	}
}

func TestHandleAuthProviderMethodAPIKey(t *testing.T) {
	m := newTestModelLite()
	info := agent.ProviderInfo{ID: agent.ProviderOpenAI, SupportsAPIKey: true}
	out := m.handleAuthProviderMethod(info, slashCommandInput{Args: []string{"openai", "api-key"}})
	if !out.Handled || out.Cmd == nil {
		t.Fatalf("expected handled outcome")
	}
	out = m.handleAuthProviderMethod(info, slashCommandInput{Args: []string{"openai", "api-key", "sk-test"}})
	if !out.Handled || !out.ClearInput {
		t.Fatalf("expected clear input")
	}
}

func TestHandleAuthProviderMethodLogout(t *testing.T) {
	m := newTestModelLite()
	info := agent.ProviderInfo{ID: agent.ProviderOpenAI, SupportsAPIKey: true}
	out := m.handleAuthProviderMethod(info, slashCommandInput{Args: []string{"openai", "logout"}})
	if !out.Handled || !out.ClearInput {
		t.Fatalf("expected logout handled")
	}
}

func TestHandleAuthAnthropicOAuth(t *testing.T) {
	m := newTestModelLite()
	info := agent.ProviderInfo{ID: agent.ProviderAnthropic, SupportsOAuth: true}
	m.authFlow.Verifier = "verifier"
	out := m.handleAuthProviderMethod(info, slashCommandInput{Args: []string{"anthropic", "oauth", "code#state"}})
	if !out.Handled || !out.ClearInput {
		t.Fatalf("expected handled outcome")
	}
	m.authFlow.Verifier = ""
	out = m.handleAuthProviderMethod(info, slashCommandInput{Args: []string{"anthropic", "oauth", "code#state"}})
	if !out.Handled || out.ClearInput {
		t.Fatalf("expected handled warning")
	}
}

func TestHandleModelSlashCommandDisabled(t *testing.T) {
	m := newTestModelLite()
	out := m.handleModelSlashCommand("/model gpt-4o")
	if !out.Handled || !out.ClearInput {
		t.Fatalf("expected disabled model selection")
	}
}
