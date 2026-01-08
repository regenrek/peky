package app

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/layout"
)

func TestHandleAuthCallbackBranches(t *testing.T) {
	m := newTestModelLite()
	m.authFlow = authFlowState{Provider: agent.ProviderGoogle, Verifier: "st"}

	m.authFlow.IgnoreCallback = true
	if cmd := m.handleAuthCallback(authCallbackMsg{Provider: agent.ProviderGoogle}); cmd != nil {
		t.Fatalf("expected nil cmd when ignored")
	}
	m.authFlow.IgnoreCallback = false

	if cmd := m.handleAuthCallback(authCallbackMsg{Provider: agent.ProviderGoogle, Err: errors.New("boom")}); cmd != nil {
		t.Fatalf("expected nil cmd on error")
	}
	if m.toast.Level != toastError || !strings.Contains(m.toast.Text, "Auth failed") {
		t.Fatalf("toast=%#v", m.toast)
	}

	m.toast = toastMessage{}
	if cmd := m.handleAuthCallback(authCallbackMsg{Provider: agent.ProviderAnthropic, Code: "c", State: "st"}); cmd != nil {
		t.Fatalf("expected nil cmd on provider mismatch")
	}
	if m.toast.Text != "" {
		t.Fatalf("expected no toast on provider mismatch")
	}

	if cmd := m.handleAuthCallback(authCallbackMsg{Provider: agent.ProviderGoogle, Code: "c", State: "nope"}); cmd != nil {
		t.Fatalf("expected nil cmd on state mismatch")
	}
	if m.toast.Level != toastError || !strings.Contains(m.toast.Text, "state mismatch") {
		t.Fatalf("toast=%#v", m.toast)
	}

	m.toast = toastMessage{}
	cmd := m.handleAuthCallback(authCallbackMsg{Provider: agent.ProviderGoogle, Code: "c", State: "st"})
	if cmd == nil {
		t.Fatalf("expected cmd on success")
	}
	if m.toast.Level != toastInfo {
		t.Fatalf("toast=%#v", m.toast)
	}
}

func TestHandleAuthProviderLogoutClearsInput(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	out := m.handleAuthSlashCommand("/auth openai logout")
	if !out.Handled || !out.ClearInput {
		t.Fatalf("out=%#v", out)
	}
}
