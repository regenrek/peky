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

	cases := []struct {
		name    string
		msg     tea.KeyMsg
		binding key.Binding
	}{
		{name: "projectLeft", msg: tea.KeyMsg{Type: tea.KeyCtrlQ}, binding: km.projectLeft},
		{name: "projectRight", msg: tea.KeyMsg{Type: tea.KeyCtrlE}, binding: km.projectRight},
		{name: "sessionUp", msg: tea.KeyMsg{Type: tea.KeyCtrlW}, binding: km.sessionUp},
		{name: "sessionDown", msg: tea.KeyMsg{Type: tea.KeyCtrlS}, binding: km.sessionDown},
		{name: "attach", msg: tea.KeyMsg{Type: tea.KeyEnter}, binding: km.attach},
		{name: "paneNext", msg: tea.KeyMsg{Type: tea.KeyCtrlD}, binding: km.paneNext},
		{name: "panePrev", msg: tea.KeyMsg{Type: tea.KeyCtrlA}, binding: km.panePrev},
		{name: "togglePanes", msg: tea.KeyMsg{Type: tea.KeyCtrlU}, binding: km.togglePanes},
		{name: "toggleSidebar", msg: tea.KeyMsg{Type: tea.KeyCtrlB}, binding: km.toggleSidebar},
		{name: "closeProject", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}, Alt: true}, binding: km.closeProject},
		{name: "help", msg: tea.KeyMsg{Type: tea.KeyCtrlG}, binding: km.help},
		{name: "quit", msg: tea.KeyMsg{Type: tea.KeyCtrlC}, binding: km.quit},
		{name: "resizeMode", msg: tea.KeyMsg{Type: tea.KeyCtrlR}, binding: km.resizeMode},
		{name: "refresh", msg: tea.KeyMsg{Type: tea.KeyF5}, binding: km.refresh},
	}
	for _, c := range cases {
		if !key.Matches(c.msg, c.binding) {
			t.Errorf("%s binding did not match %#v", c.name, c.msg)
		}
	}
}
