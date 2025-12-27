package peakypanes

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

func TestViewDashboardContentRenders(t *testing.T) {
	m := newTestModel(t)
	m.settings.RefreshInterval = 2 * time.Second
	m.settings.ShowThumbnails = true
	m.quickReplyInput.SetValue("hello")
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

	out := m.viewDashboardContent()
	if !strings.Contains(out, "Peaky Panes") {
		t.Fatalf("viewDashboardContent() missing header")
	}
	if !strings.Contains(out, "Quick Reply") {
		t.Fatalf("viewDashboardContent() missing quick reply")
	}
}

func TestRenderPanePreviewModes(t *testing.T) {
	panes := []PaneItem{
		{Index: "0", Title: "main", Left: 0, Top: 0, Width: 50, Height: 20, Active: true, Status: PaneStatusRunning},
		{Index: "1", Title: "side", Left: 50, Top: 0, Width: 50, Height: 20, Active: false, Status: PaneStatusIdle},
	}
	grid := renderPanePreview(panes, 40, 10, "grid", true, "0")
	if strings.TrimSpace(grid) == "" {
		t.Fatalf("renderPanePreview(grid) empty")
	}
	layout := renderPanePreview(panes, 40, 10, "layout", false, "1")
	if strings.TrimSpace(layout) == "" {
		t.Fatalf("renderPanePreview(layout) empty")
	}
	tiles := renderPanePreview(panes, 40, 10, "tiles", false, "")
	if strings.TrimSpace(tiles) == "" {
		t.Fatalf("renderPanePreview(tiles) empty")
	}
}

func TestRenderHelpers(t *testing.T) {
	if renderBadge(PaneStatusRunning) == "" {
		t.Fatalf("renderBadge() empty")
	}
	if line := truncateLine("hello", 3); line == "" {
		t.Fatalf("truncateLine() empty")
	}
	if line := fitLine("hello", 3); line == "" {
		t.Fatalf("fitLine() empty")
	}
	if out := padLines("x", 4, 2); out == "" {
		t.Fatalf("padLines() empty")
	}
	if out := padRight("x", 3); out != "x  " {
		t.Fatalf("padRight() = %q", out)
	}
}

func TestLayoutAndPathHelpers(t *testing.T) {
	if windowNameOrDash("") != "-" {
		t.Fatalf("windowNameOrDash(empty)")
	}
	if pathOrDash("") != "-" {
		t.Fatalf("pathOrDash(empty)")
	}
	if layoutOrDash("") != "-" {
		t.Fatalf("layoutOrDash(empty)")
	}
	m := newTestModel(t)
	if m.emptyStateMessage() == "" {
		t.Fatalf("emptyStateMessage() empty")
	}
	if out := tailLines([]string{"a", "b", "c"}, 2); len(out) != 2 {
		t.Fatalf("tailLines() = %#v", out)
	}
}

func TestViewStates(t *testing.T) {
	m := newTestModel(t)
	m.width = 80
	m.height = 24
	m.data = DashboardData{Projects: []ProjectGroup{{Name: "Proj"}}}

	m.state = StateHelp
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateHelp) empty")
	}

	m.confirmSession = "sess"
	m.confirmProject = "Proj"
	m.state = StateConfirmKill
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateConfirmKill) empty")
	}

	m.confirmClose = "Proj"
	m.state = StateConfirmCloseProject
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateConfirmCloseProject) empty")
	}

	m.renameSession = "sess"
	m.initRenameInput("sess", "new name")
	m.state = StateRenameSession
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateRenameSession) empty")
	}

	m.renameSession = "sess"
	m.renameWindowIndex = "1"
	m.renamePaneIndex = "0"
	m.initRenameInput("pane", "new name")
	m.state = StateRenamePane
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateRenamePane) empty")
	}

	m.projectRootInput.SetValue("/tmp")
	m.state = StateProjectRootSetup
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateProjectRootSetup) empty")
	}

	m.layoutPicker.SetItems([]list.Item{LayoutChoice{Label: "dev", Desc: "layout", LayoutName: "dev"}})
	m.layoutPicker.SetSize(20, 4)
	m.state = StateLayoutPicker
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateLayoutPicker) empty")
	}

	m.commandPalette.SetItems(m.commandPaletteItems())
	m.commandPalette.SetSize(20, 4)
	m.state = StateCommandPalette
	if out := m.View(); strings.TrimSpace(out) == "" {
		t.Fatalf("View(StateCommandPalette) empty")
	}
}

func TestSessionBadgeStatus(t *testing.T) {
	session := SessionItem{Status: StatusRunning}
	if status := sessionBadgeStatus(session); status != PaneStatusRunning {
		t.Fatalf("sessionBadgeStatus(running) = %v", status)
	}
	session = SessionItem{Status: StatusStopped}
	if status := sessionBadgeStatus(session); status != PaneStatusIdle {
		t.Fatalf("sessionBadgeStatus(stopped) = %v", status)
	}
	session = SessionItem{Thumbnail: PaneSummary{Line: "done", Status: PaneStatusDone}}
	if status := sessionBadgeStatus(session); status != PaneStatusDone {
		t.Fatalf("sessionBadgeStatus(thumbnail) = %v", status)
	}
}
