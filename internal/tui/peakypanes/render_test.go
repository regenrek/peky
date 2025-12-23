package peakypanes

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
	pane := PaneItem{
		Title:  "t",
		Status: PaneStatusIdle,
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
	m, _ := newTestModel(t, nil)
	m.width = 12
	m.height = 6
	out := m.viewDashboardContent()
	if !strings.Contains(out, "Terminal too small") {
		t.Fatalf("viewDashboardContent() = %q", out)
	}
}

func TestViewQuickReplyClampsInputWidth(t *testing.T) {
	m, _ := newTestModel(t, nil)
	out := m.viewQuickReply(18)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("viewQuickReply() empty")
	}
	contentWidth := 18 - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	maxWidth := contentWidth - 6
	if maxWidth < 0 {
		maxWidth = 0
	}
	if m.quickReplyInput.Width > maxWidth {
		t.Fatalf("quickReplyInput.Width = %d, want <= %d", m.quickReplyInput.Width, maxWidth)
	}
}

func TestViewPreviewFitsSmallHeight(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.settings.RefreshInterval = 2 * time.Second
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			WindowCount:  1,
			ActiveWindow: "1",
			Windows: []WindowItem{{
				Index:  "1",
				Name:   "main",
				Active: true,
				Panes: []PaneItem{{
					Index:  "0",
					Title:  "shell",
					Active: true,
					Width:  80,
					Height: 24,
				}},
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess", Window: "1", Pane: "0"}

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
