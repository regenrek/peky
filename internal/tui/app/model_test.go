package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/userpath"
)

// TestExpandPath tests the path expansion helper
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // Empty means check for home prefix
	}{
		{name: "empty", input: "", want: ""},
		{name: "absolute", input: "/tmp/test", want: "/tmp/test"},
		{name: "relative", input: "test/path", want: "test/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userpath.ExpandUser(tt.input)
			if tt.want != "" && got != tt.want {
				t.Errorf("ExpandUser(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestExpandPathTilde tests tilde expansion specifically
func TestExpandPathTilde(t *testing.T) {
	// Test tilde alone
	result := userpath.ExpandUser("~")
	if result == "~" {
		t.Error("ExpandUser(~) should expand to home directory")
	}

	// Test tilde with path
	result = userpath.ExpandUser("~/projects")
	if result == "~/projects" {
		t.Error("ExpandUser(~/projects) should expand to home/projects")
	}
}

// TestShortenPath tests the path shortening helper
func TestShortenPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "tmp path", input: "/tmp/test", want: "/tmp/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userpath.ShortenUser(tt.input)
			if got != tt.want {
				t.Errorf("ShortenUser(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSanitizeSessionName tests the session name sanitization
func TestSanitizeSessionName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "myproject", want: "myproject"},
		{name: "uppercase", input: "MyProject", want: "myproject"},
		{name: "spaces", input: "my project", want: "my-project"},
		{name: "underscores", input: "my_project", want: "my-project"},
		{name: "multiple dashes", input: "my--project", want: "my-project"},
		{name: "special chars", input: "my@project#123", want: "myproject123"},
		{name: "leading dash", input: "-myproject", want: "myproject"},
		{name: "trailing dash", input: "myproject-", want: "myproject"},
		{name: "empty", input: "", want: "session"},
		{name: "only special", input: "@#$%", want: "session"},
		{name: "whitespace only", input: "   ", want: "session"},
		{name: "mixed", input: " My-Project_123 ", want: "my-project-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layout.SanitizeSessionName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeSessionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestKeyBindings tests key binding creation
func TestKeyBindings(t *testing.T) {
	km, err := buildDashboardKeyMap(layout.DashboardKeymapConfig{})
	if err != nil {
		t.Fatalf("buildDashboardKeyMap() error: %v", err)
	}

	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlA}, km.projectLeft) {
		t.Error("projectLeft binding should match ctrl+a")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlD}, km.projectRight) {
		t.Error("projectRight binding should match ctrl+d")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlW}, km.sessionUp) {
		t.Error("sessionUp binding should match ctrl+w")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlS}, km.sessionDown) {
		t.Error("sessionDown binding should match ctrl+s")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyEnter}, km.attach) {
		t.Error("attach binding should match Enter key")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyTab}, km.paneNext) {
		t.Error("paneNext binding should match Tab key")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyShiftTab}, km.panePrev) {
		t.Error("panePrev binding should match Shift+Tab key")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlU}, km.togglePanes) {
		t.Error("togglePanes binding should match ctrl+u")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlG}, km.help) {
		t.Error("help binding should match ctrl+g")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlC}, km.quit) {
		t.Error("quit binding should match ctrl+c")
	}
}
