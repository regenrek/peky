package views

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func TestCompactPreviewLines(t *testing.T) {
	input := []string{"", "hello", "   ", "world", "\t", "done"}
	want := []string{"hello", "world", "done"}

	got := compactPreviewLines(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compactPreviewLines() = %#v, want %#v", got, want)
	}
}

func TestOverlayCenteredBasic(t *testing.T) {
	base := "hello\nworld"
	overlay := "POP"
	out := overlayCentered(base, overlay, 12, 4)
	if !strings.Contains(out, "hello") {
		t.Fatalf("overlayCentered() missing base content: %q", out)
	}
	if !strings.Contains(out, "POP") {
		t.Fatalf("overlayCentered() missing overlay content: %q", out)
	}
}

func TestLayoutFallbackKeepsLastPreviewLine(t *testing.T) {
	pane := Pane{
		Title:  "t",
		Status: paneStatusIdle,
		Preview: []string{
			strings.Repeat("a", 40),
			strings.Repeat("b", 40),
			"hi",
		},
	}
	lines := layoutFallbackLines(pane, "")
	if len(lines) < 3 || lines[2] != "hi" {
		t.Fatalf("layoutFallbackLines() missing last preview line: %#v", lines)
	}
}

func TestViewDashboardContentSmallTerminal(t *testing.T) {
	m := Model{Width: 12, Height: 6}
	out := m.viewDashboardContent()
	if !strings.Contains(out, "Terminal too small") {
		t.Fatalf("viewDashboardContent() = %q", out)
	}
}

func TestViewQuickReplyRenders(t *testing.T) {
	input := textinput.New()
	input.SetValue("hi")
	m := Model{
		QuickReplyInput: input,
	}
	out := m.viewQuickReply(18)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("viewQuickReply() empty")
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("viewQuickReply() lines = %d, want 3", len(lines))
	}
}

func TestViewQuickReplyKeepsHeightWithSuggestions(t *testing.T) {
	input := textinput.New()
	input.SetValue("/")
	m := Model{
		QuickReplyInput: input,
		QuickReplySuggestions: []QuickReplySuggestion{
			{Text: "/kill", MatchLen: 2},
			{Text: "/rename", MatchLen: 2},
		},
	}
	out := m.viewQuickReply(40)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected quick reply height 3, got %d", len(lines))
	}
}

func TestViewQuickReplyRendersSelection(t *testing.T) {
	input := textinput.New()
	input.SetValue("hello world")
	m := Model{
		QuickReplyInput:           input,
		QuickReplySelectionActive: true,
		QuickReplySelectionStart:  0,
		QuickReplySelectionEnd:    5,
	}

	out := m.viewQuickReply(40)
	stripped := ansi.Strip(out)
	if !strings.Contains(stripped, "hello world") {
		t.Fatalf("expected content preserved, got %q", stripped)
	}
	if strings.Contains(out, "\x1b[") {
		reverse := regexp.MustCompile("\x1b\\[[0-9;]*7[0-9;]*m")
		if !reverse.MatchString(out) {
			t.Fatalf("expected reverse-video sequence in output when ANSI present")
		}
	}
}

func TestRenderQuickReplyInputSelectionTrimsAndBounds(t *testing.T) {
	base := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)

	t.Run("clamp_swap_and_trim", func(t *testing.T) {
		got := renderQuickReplyInputSelection(base, "hello\n", 10, -5)
		if ansi.Strip(got) != "hello" {
			t.Fatalf("expected selection output to preserve value, got %q", ansi.Strip(got))
		}
		if got == "" {
			t.Fatalf("expected non-empty output")
		}
	})

	t.Run("empty_after_clamp", func(t *testing.T) {
		got := renderQuickReplyInputSelection(base, "hello\n", -5, 0)
		if got != "" {
			t.Fatalf("expected empty selection when start==end after clamp, got %q", ansi.Strip(got))
		}
	})

	t.Run("empty_value", func(t *testing.T) {
		got := renderQuickReplyInputSelection(base, "", 0, 1)
		if got != "" {
			t.Fatalf("expected empty selection for empty value, got %q", ansi.Strip(got))
		}
	})
}

func TestViewDashboardQuickReplyMenuOverlay(t *testing.T) {
	input := textinput.New()
	input.SetValue("/")
	m := Model{
		Width:             60,
		Height:            16,
		QuickReplyInput:   input,
		QuickReplyEnabled: true,
		QuickReplySuggestions: []QuickReplySuggestion{
			{Text: "/kill", MatchLen: 2, Desc: "Close pane"},
			{Text: "/rename", MatchLen: 2, Desc: "Rename pane"},
		},
	}
	out := m.viewDashboardContent()
	if !strings.Contains(out, "/kill") {
		t.Fatalf("expected slash suggestion output, got %q", out)
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "/kill") && strings.Contains(line, "/rename") {
			t.Fatalf("expected slash suggestions on separate lines, got %q", line)
		}
	}
}

func TestViewPreviewFitsSmallHeight(t *testing.T) {
	m := Model{
		PreviewSession: &Session{
			Name:       "sess",
			Status:     sessionRunning,
			PaneCount:  1,
			ActivePane: "0",
			Panes: []Pane{{
				ID:     "p1",
				Index:  "0",
				Title:  "shell",
				Active: true,
				Width:  80,
				Height: 24,
			}},
		},
		SelectionPane:     "0",
		EmptyStateMessage: "empty",
	}

	width := 20
	height := 4
	out := ansi.Strip(m.viewPreview(width, height))
	lines := strings.Split(out, "\n")
	if len(lines) != height {
		t.Fatalf("viewPreview() lines = %d, want %d", len(lines), height)
	}
	if strings.Contains(out, "Pane Preview") {
		t.Fatalf("viewPreview() should not include title: %q", out)
	}
}
