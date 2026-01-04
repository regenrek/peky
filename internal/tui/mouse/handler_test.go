package mouse

import (
	"testing"
	"time"

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

	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, DashboardCallbacks{
		HitHeader:             func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:               func(int, int) (PaneHit, bool) { return hit, true },
		HitIsSelected:         func(PaneHit) bool { return false },
		ApplySelection:        func(Selection) bool { selectionCalls++; return true },
		SelectDashboardTab:    func() bool { return false },
		SelectProjectTab:      func(string) bool { return false },
		OpenProjectPicker:     func() {},
		SetTerminalFocus:      func(bool) {},
		TerminalFocus:         func() bool { return false },
		SupportsTerminalFocus: func() bool { return true },
		SelectionCmd:          func() tea.Cmd { selectionCmdCalls++; return nil },
		SelectionRefreshCmd:   func() tea.Cmd { return nil },
		RefreshPaneViewsCmd:   func() tea.Cmd { refreshCalls++; return nil },
		ForwardMouseEvent:     func(PaneHit, tea.MouseMsg) tea.Cmd { t.Fatalf("unexpected forward"); return nil },
		FocusUnavailable:      func() { t.Fatalf("unexpected focus unavailable") },
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
}

func TestHandlerDoubleClickEntersFocus(t *testing.T) {
	var h Handler
	hit := PaneHit{
		PaneID: "pane-2",
		Selection: Selection{
			ProjectID: "proj",
			Session:   "sess",
			Pane:      "2",
		},
		Outer:   Rect{X: 0, Y: 0, W: 10, H: 10},
		Content: Rect{X: 0, Y: 0, W: 10, H: 10},
	}
	selectionCalls := 0
	refreshCalls := 0
	selectionCmdCalls := 0
	focusCalls := 0
	terminalFocus := false

	cb := DashboardCallbacks{
		HitHeader:          func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:            func(int, int) (PaneHit, bool) { return hit, true },
		HitIsSelected:      func(PaneHit) bool { return false },
		ApplySelection:     func(Selection) bool { selectionCalls++; return true },
		SelectDashboardTab: func() bool { return false },
		SelectProjectTab:   func(string) bool { return false },
		OpenProjectPicker:  func() {},
		SetTerminalFocus: func(focus bool) {
			focusCalls++
			terminalFocus = focus
		},
		TerminalFocus:         func() bool { return terminalFocus },
		SupportsTerminalFocus: func() bool { return true },
		SelectionCmd:          func() tea.Cmd { selectionCmdCalls++; return nil },
		SelectionRefreshCmd:   func() tea.Cmd { return nil },
		RefreshPaneViewsCmd:   func() tea.Cmd { refreshCalls++; return nil },
		ForwardMouseEvent:     func(PaneHit, tea.MouseMsg) tea.Cmd { t.Fatalf("unexpected forward"); return nil },
		FocusUnavailable:      func() { t.Fatalf("unexpected focus unavailable") },
	}

	msg := tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	h.UpdateDashboard(msg, cb)
	h.lastClickAt = h.lastClickAt.Add(-doubleClickThreshold / 2)
	h.UpdateDashboard(msg, cb)

	if focusCalls == 0 || !terminalFocus {
		t.Fatalf("expected terminal focus to be set on double click")
	}
	if selectionCalls < 2 {
		t.Fatalf("ApplySelection calls=%d want >=2", selectionCalls)
	}
	if refreshCalls == 0 || selectionCmdCalls == 0 {
		t.Fatalf("expected refresh and selection commands on double click")
	}
}

func TestHandlerFocusUnavailable(t *testing.T) {
	var h Handler
	hit := PaneHit{
		PaneID: "pane-3",
		Selection: Selection{
			ProjectID: "proj",
			Session:   "sess",
			Pane:      "3",
		},
		Outer:   Rect{X: 0, Y: 0, W: 10, H: 10},
		Content: Rect{X: 0, Y: 0, W: 10, H: 10},
	}
	focusCalls := 0
	toastCalls := 0

	cb := DashboardCallbacks{
		HitHeader:          func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:            func(int, int) (PaneHit, bool) { return hit, true },
		HitIsSelected:      func(PaneHit) bool { return false },
		ApplySelection:     func(Selection) bool { return true },
		SelectDashboardTab: func() bool { return false },
		SelectProjectTab:   func(string) bool { return false },
		OpenProjectPicker:  func() {},
		SetTerminalFocus: func(bool) {
			focusCalls++
		},
		TerminalFocus:         func() bool { return false },
		SupportsTerminalFocus: func() bool { return false },
		SelectionCmd:          func() tea.Cmd { return nil },
		SelectionRefreshCmd:   func() tea.Cmd { return nil },
		RefreshPaneViewsCmd:   func() tea.Cmd { return nil },
		ForwardMouseEvent:     func(PaneHit, tea.MouseMsg) tea.Cmd { return nil },
		FocusUnavailable:      func() { toastCalls++ },
	}

	msg := tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	h.UpdateDashboard(msg, cb)
	h.lastClickAt = h.lastClickAt.Add(-doubleClickThreshold / 2)
	h.UpdateDashboard(msg, cb)

	if toastCalls != 1 {
		t.Fatalf("FocusUnavailable calls=%d want 1", toastCalls)
	}
	if focusCalls != 0 {
		t.Fatalf("SetTerminalFocus calls=%d want 0", focusCalls)
	}
}

func TestHandlerHeaderClicks(t *testing.T) {
	var h Handler
	refreshCalls := 0
	openCalls := 0
	setFocusCalls := 0

	cb := DashboardCallbacks{
		HitHeader: func(int, int) (HeaderHit, bool) {
			return HeaderHit{Kind: HeaderDashboard}, true
		},
		SelectionRefreshCmd: func() tea.Cmd { refreshCalls++; return nil },
		SelectDashboardTab:  func() bool { return true },
		SetTerminalFocus:    func(bool) { setFocusCalls++ },
		TerminalFocus:       func() bool { return true },
		HitPane:             func(int, int) (PaneHit, bool) { return PaneHit{}, false },
	}

	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, cb)
	if refreshCalls != 1 {
		t.Fatalf("expected refresh command on dashboard tab")
	}
	if setFocusCalls != 1 {
		t.Fatalf("expected terminal focus cleared on header click")
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

func TestHandlerTerminalFocusMouse(t *testing.T) {
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
	selectionCalls := 0
	focusCalls := 0

	cb := DashboardCallbacks{
		HitHeader:             func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:               func(int, int) (PaneHit, bool) { return hit, true },
		HitIsSelected:         func(PaneHit) bool { return true },
		ForwardMouseEvent:     func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
		ApplySelection:        func(Selection) bool { selectionCalls++; return true },
		SetTerminalFocus:      func(bool) { focusCalls++ },
		TerminalFocus:         func() bool { return true },
		SupportsTerminalFocus: func() bool { return true },
		SelectionCmd:          func() tea.Cmd { return nil },
		RefreshPaneViewsCmd:   func() tea.Cmd { return nil },
	}

	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, cb)
	if forwardCalls != 1 {
		t.Fatalf("expected forward mouse event")
	}

	cb.HitIsSelected = func(PaneHit) bool { return false }
	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, cb)
	if selectionCalls == 0 || focusCalls == 0 {
		t.Fatalf("expected selection change and focus clear when switching panes")
	}
}

func TestHandlerWheelForwardsWithoutTerminalFocus(t *testing.T) {
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
	forwardCalls := 0
	selectionCalls := 0
	focusCalls := 0

	cb := DashboardCallbacks{
		HitHeader:             func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:               func(int, int) (PaneHit, bool) { return hit, true },
		HitIsSelected:         func(PaneHit) bool { return false },
		ApplySelection:        func(Selection) bool { selectionCalls++; return true },
		SetTerminalFocus:      func(bool) { focusCalls++ },
		TerminalFocus:         func() bool { return false },
		SupportsTerminalFocus: func() bool { return true },
		SelectionCmd:          func() tea.Cmd { return nil },
		RefreshPaneViewsCmd:   func() tea.Cmd { return nil },
		ForwardMouseEvent:     func(PaneHit, tea.MouseMsg) tea.Cmd { forwardCalls++; return nil },
	}

	h.UpdateDashboard(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp}, cb)

	if forwardCalls != 1 {
		t.Fatalf("expected wheel to be forwarded, got %d", forwardCalls)
	}
	if selectionCalls != 0 {
		t.Fatalf("expected no selection changes, got %d", selectionCalls)
	}
	if focusCalls != 0 {
		t.Fatalf("expected no focus changes, got %d", focusCalls)
	}
}

func TestHandlerDoubleClickDetection(t *testing.T) {
	var h Handler
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	h.lastClickAt = time.Now()
	h.lastClickPaneID = "pane-1"
	h.lastClickButton = tea.MouseButtonLeft
	if !h.isDoubleClick(PaneHit{PaneID: "pane-1"}, msg) {
		t.Fatalf("expected double click")
	}

	h.lastClickAt = time.Now().Add(-doubleClickThreshold * 2)
	if h.isDoubleClick(PaneHit{PaneID: "pane-1"}, msg) {
		t.Fatalf("expected double click to expire")
	}
	if h.isDoubleClick(PaneHit{PaneID: "other"}, msg) {
		t.Fatalf("expected different pane to be false")
	}
}

func TestHandlerMotionDoesNotHitTestWhenNotInTerminalFocus(t *testing.T) {
	var h Handler
	h.UpdateDashboard(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}, DashboardCallbacks{
		HitHeader:     func(int, int) (HeaderHit, bool) { return HeaderHit{}, false },
		HitPane:       func(int, int) (PaneHit, bool) { t.Fatalf("unexpected HitPane call"); return PaneHit{}, false },
		TerminalFocus: func() bool { return false },
	})
}
