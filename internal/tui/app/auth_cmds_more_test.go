package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
)

func TestAuthSetAPIKeyAndRemoveCmds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	msg, ok := authSetAPIKeyCmd(agent.ProviderOpenAI, "sk-test")().(authDoneMsg)
	if !ok || msg.Err != nil {
		t.Fatalf("msg=%#v", msg)
	}
	msg, ok = authRemoveCmd(agent.ProviderOpenAI)().(authDoneMsg)
	if !ok || msg.Err != nil {
		t.Fatalf("msg=%#v", msg)
	}
}

func TestAuthGeminiExchangeUnsupportedProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	msg, ok := authGeminiExchangeCmd(agent.ProviderGoogle, "code", "verifier")().(authDoneMsg)
	if !ok || msg.Err == nil {
		t.Fatalf("expected error, msg=%#v", msg)
	}
}

func TestHandleAuthDoneClosesDialogOnSuccess(t *testing.T) {
	m := newTestModelLite()
	m.openAuthDialog("Title", "Body", "Paste", "Footer", authDialogOAuthCode)
	if m.state != StateAuthDialog {
		t.Fatalf("state=%v", m.state)
	}
	_ = m.handleAuthDone(authDoneMsg{Provider: agent.ProviderOpenAI, Message: "ok"})
	if m.state != StateDashboard {
		t.Fatalf("state=%v want=%v", m.state, StateDashboard)
	}
}
