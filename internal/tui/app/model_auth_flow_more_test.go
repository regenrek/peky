package app

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/layout"
)

func TestAuthDialogEscClosesAndResets(t *testing.T) {
	m := newTestModelLite()
	m.openAuthDialog("Title", "Body", "Paste", "Footer", authDialogOAuthCode)
	if m.state != StateAuthDialog {
		t.Fatalf("state=%v want=%v", m.state, StateAuthDialog)
	}
	_, _ = m.updateAuthDialog(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("state=%v want=%v", m.state, StateDashboard)
	}
	if m.authDialogTitle != "" || m.authDialogBody != "" || m.authDialogFooter != "" {
		t.Fatalf("auth dialog not reset")
	}
	if !m.authFlow.IgnoreCallback {
		t.Fatalf("expected IgnoreCallback=true")
	}
}

func TestAuthDialogSubmitEmptyShowsToast(t *testing.T) {
	m := newTestModelLite()
	m.openAuthDialog("Title", "Body", "Paste", "Footer", authDialogOAuthCode)
	m.authDialogInput.SetValue("   ")
	if cmd := m.handleAuthDialogSubmit(); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if !strings.Contains(m.toast.Text, "Paste the token") {
		t.Fatalf("toast=%q", m.toast.Text)
	}
	if m.toast.Level != toastInfo {
		t.Fatalf("toast level=%v", m.toast.Level)
	}
}

func TestAuthDialogSubmitOAuthMissingVerifier(t *testing.T) {
	m := newTestModelLite()
	m.authFlow.Provider = agent.ProviderOpenAI
	m.openAuthDialog("Title", "Body", "Paste", "Footer", authDialogOAuthCode)
	m.authDialogInput.SetValue("abc")
	if cmd := m.handleAuthDialogSubmit(); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if !strings.Contains(m.toast.Text, "OAuth flow not initialized") {
		t.Fatalf("toast=%q", m.toast.Text)
	}
	if m.toast.Level != toastWarning {
		t.Fatalf("toast level=%v", m.toast.Level)
	}
}

func TestOAuthDialogBodyAndCopilotDialogBody(t *testing.T) {
	body := oauthDialogBody("https://example.com/auth", "Paste code#state", errors.New("nope"))
	if !strings.Contains(body, "Browser did not open") || !strings.Contains(body, "Paste code#state") {
		t.Fatalf("body=%q", body)
	}
	deviceBody := copilotDialogBody(agent.CopilotDeviceInfo{VerificationURI: "https://example.com", UserCode: "ABCD"}, nil)
	if !strings.Contains(deviceBody, "Enter this code") || !strings.Contains(deviceBody, "ABCD") {
		t.Fatalf("body=%q", deviceBody)
	}
}

func TestSetAgentProviderAndModel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath

	if cmd := m.setAgentProvider(agent.ProviderOpenAI); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if m.config == nil || m.config.Agent.Provider != "openai" {
		t.Fatalf("provider=%v", m.config)
	}
	if m.toast.Level != toastSuccess {
		t.Fatalf("toast level=%v", m.toast.Level)
	}

	if cmd := m.setAgentModel("   "); cmd == nil {
		t.Fatalf("expected warning cmd")
	} else if msg := cmd(); msg == nil {
		t.Fatalf("expected warning msg")
	}

	if cmd := m.setAgentModel("gpt-4o-mini"); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	loaded, err := layout.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Agent.Provider != "openai" || loaded.Agent.Model != "gpt-4o-mini" {
		t.Fatalf("cfg=%#v", loaded.Agent)
	}
}
