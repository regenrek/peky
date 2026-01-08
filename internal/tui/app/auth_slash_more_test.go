package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/layout"
)

func TestHandleAuthSlashCommandUnknownProviderHandled(t *testing.T) {
	m := newTestModelLite()
	out := m.handleAuthSlashCommand("/auth nope")
	if !out.Handled || out.Cmd == nil {
		t.Fatalf("expected handled warning cmd")
	}
	msg := out.Cmd()
	warn, ok := msg.(WarningMsg)
	if !ok || !strings.Contains(warn.Message, "Unknown provider") {
		t.Fatalf("msg=%#v", msg)
	}
}

func TestHandleAuthSlashCommandProviderOnlyPrefillsAPIKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	m.quickReplyInput.SetValue("/auth openai ")

	out := m.handleAuthSlashCommand(m.quickReplyInput.Value())
	if !out.Handled {
		t.Fatalf("expected handled")
	}
	if !strings.HasPrefix(m.quickReplyInput.Value(), "/auth openai api-key ") {
		t.Fatalf("value=%q", m.quickReplyInput.Value())
	}
}

func TestHandleAuthAPIKeyRequiresKey(t *testing.T) {
	m := newTestModelLite()
	out := m.handleAuthAPIKey(agent.ProviderInfo{ID: agent.ProviderOpenAI, SupportsAPIKey: true}, []string{"openai", "api-key", " "})
	if !out.Handled || out.Cmd == nil {
		t.Fatalf("expected handled warning")
	}
	msg := out.Cmd()
	warn, ok := msg.(WarningMsg)
	if !ok || warn.Message != "API key required" {
		t.Fatalf("msg=%#v", msg)
	}
}

func TestHandleAnthropicOAuthValidatesCodeState(t *testing.T) {
	m := newTestModelLite()
	out := m.handleAuthOAuth(agent.ProviderInfo{ID: agent.ProviderAnthropic, SupportsOAuth: true}, []string{"anthropic", "oauth", "abc"})
	if !out.Handled || out.Cmd == nil {
		t.Fatalf("expected handled warning cmd")
	}
	msg := out.Cmd()
	warn, ok := msg.(WarningMsg)
	if !ok || !strings.Contains(warn.Message, "code#state") {
		t.Fatalf("msg=%#v", msg)
	}
}
