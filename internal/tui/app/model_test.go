package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
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
		msg     tuiinput.KeyMsg
		binding key.Binding
	}{
		{name: "projectLeft", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'a', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.projectLeft},
		{name: "projectRight", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'd', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.projectRight},
		{name: "sessionUp", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'w', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.sessionUp},
		{name: "sessionDown", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 's', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.sessionDown},
		{name: "attach", msg: tuiinput.KeyMsg{Key: uv.Key{Code: uv.KeyEnter}}, binding: km.attach},
		{name: "paneNext", msg: tuiinput.KeyMsg{Key: uv.Key{Code: uv.KeyRight, Mod: uv.ModCtrl | uv.ModShift}}, binding: km.paneNext},
		{name: "panePrev", msg: tuiinput.KeyMsg{Key: uv.Key{Code: uv.KeyLeft, Mod: uv.ModCtrl | uv.ModShift}}, binding: km.panePrev},
		{name: "toggleLastPane", msg: tuiinput.KeyMsg{Key: uv.Key{Code: uv.KeySpace, Mod: uv.ModCtrl | uv.ModShift}}, binding: km.toggleLastPane},
		{name: "togglePanes", msg: tuiinput.KeyMsg{Key: uv.Key{Code: ']', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.togglePanes},
		{name: "toggleSidebar", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'b', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.toggleSidebar},
		{name: "closeProject", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'c', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.closeProject},
		{name: "help", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'g', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.help},
		{name: "quit", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'q', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.quit},
		{name: "resizeMode", msg: tuiinput.KeyMsg{Key: uv.Key{Code: 'r', Mod: uv.ModCtrl | uv.ModShift}}, binding: km.resizeMode},
		{name: "refresh", msg: tuiinput.KeyMsg{Key: uv.Key{Code: uv.KeyF5}}, binding: km.refresh},
	}
	for _, c := range cases {
		if !matchesBinding(c.msg, c.binding) {
			t.Errorf("%s binding did not match %#v", c.name, c.msg)
		}
	}
}
