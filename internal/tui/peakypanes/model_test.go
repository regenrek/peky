package peakypanes

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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
			got := expandPath(tt.input)
			if tt.want != "" && got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestExpandPathTilde tests tilde expansion specifically
func TestExpandPathTilde(t *testing.T) {
	// Test tilde alone
	result := expandPath("~")
	if result == "~" {
		t.Error("expandPath(~) should expand to home directory")
	}

	// Test tilde with path
	result = expandPath("~/projects")
	if result == "~/projects" {
		t.Error("expandPath(~/projects) should expand to home/projects")
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
			got := shortenPath(tt.input)
			if got != tt.want {
				t.Errorf("shortenPath(%q) = %q, want %q", tt.input, got, tt.want)
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
			got := sanitizeSessionName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSessionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestGitProjectListItem tests the GitProject type's list.Item implementation
func TestGitProjectListItem(t *testing.T) {
	gp := GitProject{
		Name: "my-repo",
		Path: "/home/user/projects/my-repo",
	}

	// Test Title
	title := gp.Title()
	if title != "📁 my-repo" {
		t.Errorf("GitProject.Title() = %q, want %q", title, "📁 my-repo")
	}

	// Test FilterValue
	filter := gp.FilterValue()
	if filter != "my-repo" {
		t.Errorf("GitProject.FilterValue() = %q, want %q", filter, "my-repo")
	}
}

// TestKeyBindings tests key binding creation
func TestKeyBindings(t *testing.T) {
	km := newDashboardKeyMap()
	if km == nil {
		t.Fatal("newDashboardKeyMap() returned nil")
	}

	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, km.projectLeft) {
		t.Error("projectLeft binding should match a")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, km.projectRight) {
		t.Error("projectRight binding should match d")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}, km.sessionUp) {
		t.Error("sessionUp binding should match w")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, km.sessionDown) {
		t.Error("sessionDown binding should match s")
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
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlG}, km.help) {
		t.Error("help binding should match ctrl+g")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlC}, km.quit) {
		t.Error("quit binding should match ctrl+c")
	}
}
