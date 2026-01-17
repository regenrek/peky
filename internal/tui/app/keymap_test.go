package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestBuildDashboardKeyMapOverride(t *testing.T) {
	cfg := layout.DashboardKeymapConfig{
		ProjectLeft: []string{"ctrl+h"},
		SessionUp:   []string{"ctrl+j"},
	}
	km, err := buildDashboardKeyMap(cfg)
	if err != nil {
		t.Fatalf("buildDashboardKeyMap() error: %v", err)
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlH}, km.projectLeft) {
		t.Error("projectLeft binding should match ctrl+h")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlJ}, km.sessionUp) {
		t.Error("sessionUp binding should match ctrl+j")
	}
}

func TestBuildDashboardKeyMapInvalidKey(t *testing.T) {
	cfg := layout.DashboardKeymapConfig{
		ProjectLeft: []string{"ctrl+nope"},
	}
	_, err := buildDashboardKeyMap(cfg)
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "dashboard.keymap.project_left") {
		t.Fatalf("expected field name in error, got %q", err.Error())
	}
}

func TestBuildDashboardKeyMapDuplicateKey(t *testing.T) {
	cfg := layout.DashboardKeymapConfig{
		ProjectLeft:  []string{"ctrl+a"},
		ProjectRight: []string{"ctrl+a"},
	}
	_, err := buildDashboardKeyMap(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
	if !strings.Contains(err.Error(), "already bound") {
		t.Fatalf("expected duplicate error, got %q", err.Error())
	}
}

func TestNormalizeKeyStringAndLabels(t *testing.T) {
	if got, _ := normalizeKeyString("space"); got != "space" {
		t.Fatalf("expected space normalization, got %q", got)
	}
	if got, _ := normalizeKeyString("alt+space"); got != "alt+space" {
		t.Fatalf("expected alt space normalization, got %q", got)
	}
	if _, err := normalizeKeyString(""); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if !isSingleRune("a") || isSingleRune("ab") {
		t.Fatalf("unexpected isSingleRune result")
	}
	if prettyKeyLabel("shift+tab") != "⇧tab" {
		t.Fatalf("expected pretty label for shift+tab")
	}
	binding := key.NewBinding(key.WithKeys("tab"), key.WithHelp("t", "test"))
	if keyLabel(binding) != "t" {
		t.Fatalf("expected keyLabel to use help key")
	}
}
