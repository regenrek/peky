package mouse

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRectContains(t *testing.T) {
	rect := Rect{X: 2, Y: 3, W: 4, H: 5}
	if rect.Empty() {
		t.Fatalf("rect should not be empty")
	}
	if !rect.Contains(2, 3) {
		t.Fatalf("rect should contain top-left corner")
	}
	if rect.Contains(1, 3) {
		t.Fatalf("rect should not contain outside point")
	}
	if rect.Contains(6, 7) {
		t.Fatalf("rect should not contain point on max edge")
	}
}

func TestHandlerSingleClickSelectsPane(t *testing.T) {
	var h Handler
	hit := PaneHit{
		PaneID: "pane-1",
		Selection: Selection{
			ProjectID: "proj",
			Session:   "sess",
			Pane:      "1",
		},
		Outer:   Rect{X: 0, Y: 0, W: 10, H: 10},
		Content: Rect{X: 0, Y: 0, W: 10, H: 10},
	}
	selectionCalls := 0
	selectionCmdCalls := 0
	refreshCalls := 0
	forwardCalls := 0

	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, DashboardCallbacks{
		HitHeader:           func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:             func(int, int) (PaneHit, bool) { return hit, true },
		ApplySelection:      func(Selection) bool { selectionCalls++; return true },
		SelectDashboardTab:  func() bool { return false },
		SelectProjectTab:    func(string) bool { return false },
		OpenProjectPicker:   func() {},
		SelectionCmd:        func() tea.Cmd { selectionCmdCalls++; return nil },
		SelectionRefreshCmd: func() tea.Cmd { return nil },
		RefreshPaneViewsCmd: func() tea.Cmd { refreshCalls++; return nil },
		ForwardMouseEvent:   func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
	})

	if selectionCalls != 1 {
		t.Fatalf("ApplySelection calls=%d want 1", selectionCalls)
	}
	if selectionCmdCalls != 1 {
		t.Fatalf("SelectionCmd calls=%d want 1", selectionCmdCalls)
	}
	if refreshCalls != 1 {
		t.Fatalf("RefreshPaneViewsCmd calls=%d want 1", refreshCalls)
	}
	if forwardCalls != 1 {
		t.Fatalf("ForwardMouseEvent calls=%d want 1", forwardCalls)
	}
}

func TestHandlerWheelForwards(t *testing.T) {
	var h Handler
	hit := PaneHit{
		PaneID: "pane-1",
		Selection: Selection{
			ProjectID: "proj",
			Session:   "sess",
			Pane:      "1",
		},
		Outer:   Rect{X: 0, Y: 0, W: 10, H: 10},
		Content: Rect{X: 0, Y: 0, W: 10, H: 10},
	}
	forwardCalls := 0
	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp}, DashboardCallbacks{
		HitHeader:          func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:            func(int, int) (PaneHit, bool) { return hit, true },
		ApplySelection:     func(Selection) bool { return false },
		SelectDashboardTab: func() bool { return false },
		SelectProjectTab:   func(string) bool { return false },
		ForwardMouseEvent:  func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
	})
	if forwardCalls != 1 {
		t.Fatalf("ForwardMouseEvent calls=%d want 1", forwardCalls)
	}
}

func TestHandlerPressInPaneContent_ForwardsAndSelects(t *testing.T) {
	var h Handler
	hit := PaneHit{
		PaneID: "pane-1",
		Selection: Selection{
			ProjectID: "proj",
			Session:   "sess",
			Pane:      "1",
		},
		Outer:   Rect{X: 0, Y: 0, W: 10, H: 10},
		Content: Rect{X: 1, Y: 1, W: 8, H: 6},
	}
	selectionCalls := 0
	selectionCmdCalls := 0
	refreshCalls := 0
	forwardCalls := 0

	h.UpdateDashboard(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, DashboardCallbacks{
		HitHeader:          func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:            func(int, int) (PaneHit, bool) { return hit, true },
		ApplySelection:     func(Selection) bool { selectionCalls++; return true },
		SelectDashboardTab: func() bool { return false },
		SelectProjectTab:   func(string) bool { return false },
		SelectionCmd:       func() tea.Cmd { selectionCmdCalls++; return nil },
		RefreshPaneViewsCmd: func() tea.Cmd {
			refreshCalls++
			return nil
		},
		ForwardMouseEvent: func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
	})

	if forwardCalls != 1 {
		t.Fatalf("expected press forwarded, got %d", forwardCalls)
	}
	if selectionCalls != 1 {
		t.Fatalf("expected selection change, got %d", selectionCalls)
	}
	if selectionCmdCalls != 1 {
		t.Fatalf("expected selection cmd, got %d", selectionCmdCalls)
	}
	if refreshCalls != 1 {
		t.Fatalf("expected refresh pane views, got %d", refreshCalls)
	}
	if !h.dragActive {
		t.Fatalf("expected dragActive set after press")
	}
}

func TestHandlerDragMotionForwardsUntilRelease(t *testing.T) {
	var h Handler
	hit := PaneHit{
		PaneID: "pane-1",
		Selection: Selection{
			ProjectID: "proj",
			Session:   "sess",
			Pane:      "1",
		},
		Outer:   Rect{X: 0, Y: 0, W: 10, H: 10},
		Content: Rect{X: 0, Y: 0, W: 10, H: 10},
	}
	forwardCalls := 0
	cb := DashboardCallbacks{
		HitHeader:          func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:            func(int, int) (PaneHit, bool) { return hit, true },
		ApplySelection:     func(Selection) bool { return false },
		SelectDashboardTab: func() bool { return false },
		SelectProjectTab:   func(string) bool { return false },
		ForwardMouseEvent:  func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
	}
	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, cb)
	h.UpdateDashboard(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}, cb)
	h.UpdateDashboard(tea.MouseMsg{X: 3, Y: 3, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft}, cb)
	if forwardCalls != 3 {
		t.Fatalf("ForwardMouseEvent calls=%d want 3", forwardCalls)
	}
}

func TestHandlerHeaderClicks(t *testing.T) {
	var h Handler
	refreshCalls := 0
	openCalls := 0
	cb := DashboardCallbacks{
		HitHeader: func(int, int) (HeaderHit, bool) {
			return HeaderHit{Kind: HeaderDashboard}, true
		},
		SelectionRefreshCmd: func() tea.Cmd { refreshCalls++; return nil },
		SelectDashboardTab:  func() bool { return true },
		HitPane:             func(int, int) (PaneHit, bool) { return PaneHit{}, false },
	}
	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, cb)
	if refreshCalls != 1 {
		t.Fatalf("expected refresh command on dashboard tab")
	}

	cb.HitHeader = func(int, int) (HeaderHit, bool) {
		return HeaderHit{Kind: HeaderNew}, true
	}
	cb.OpenProjectPicker = func() { openCalls++ }
	h.UpdateDashboard(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, cb)
	if openCalls != 1 {
		t.Fatalf("expected OpenProjectPicker to be called")
	}
}

func TestHandlerReleaseClearsDragWithoutButton(t *testing.T) {
	var h Handler
	h.dragActive = true
	h.dragHit = PaneHit{
		PaneID:  "pane-1",
		Content: Rect{X: 0, Y: 0, W: 10, H: 10},
	}
	forwardCalls := 0
	cb := DashboardCallbacks{
		ForwardMouseEvent: func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
	}

	h.UpdateDashboard(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionRelease, Button: tea.MouseButtonNone}, cb)

	if forwardCalls != 1 {
		t.Fatalf("expected release forwarded, got %d", forwardCalls)
	}
	if h.dragActive {
		t.Fatalf("expected dragActive cleared on release")
	}
}
