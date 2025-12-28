package views

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/x/ansi"
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

func TestRenderPaneTileKeepsLastPreviewLine(t *testing.T) {
	pane := Pane{
		Title:  "t",
		Status: paneStatusIdle,
		Preview: []string{
			strings.Repeat("a", 40),
			strings.Repeat("b", 40),
			"hi",
		},
	}

	colors := tileBorderColors{
		top:    borderColorFor(borderLevelDefault),
		right:  borderColorFor(borderLevelDefault),
		bottom: borderColorFor(borderLevelDefault),
		left:   borderColorFor(borderLevelDefault),
	}
	out := renderPaneTile(pane, 12, 6, false, false, tileBorders{
		top:    true,
		right:  true,
		bottom: true,
		left:   true,
		colors: colors,
	})
	if !strings.Contains(out, "hi") {
		t.Fatalf("renderPaneTile() missing last preview line: %q", out)
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
		Keys:            KeyHints{TerminalFocus: "ctrl+t"},
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

func TestViewPreviewFitsSmallHeight(t *testing.T) {
	m := Model{
		PreviewMode: "grid",
		PreviewSession: &Session{
			Name:       "sess",
			Status:     sessionRunning,
			PaneCount:  1,
			ActivePane: "0",
			Panes: []Pane{{
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
