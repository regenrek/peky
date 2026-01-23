package mouse

import tea "github.com/charmbracelet/bubbletea"

// Handler tracks mouse click state and applies dashboard mouse logic.
type Handler struct {
	dragActive bool
	dragHit    PaneHit
}

// DashboardCallbacks wires mouse handling into the app model.
type DashboardCallbacks struct {
	HitHeader           func(x, y int) (HeaderHit, bool)
	HitPane             func(x, y int) (PaneHit, bool)
	ApplySelection      func(selection Selection) bool
	SelectDashboardTab  func() bool
	SelectProjectTab    func(projectID string) bool
	OpenProjectPicker   func()
	OpenUpdateDialog    func()
	SelectionCmd        func() tea.Cmd
	SelectionRefreshCmd func() tea.Cmd
	RefreshPaneViewsCmd func() tea.Cmd
	ForwardMouseEvent   func(hit PaneHit, msg tea.MouseMsg) tea.Cmd
}

// UpdateDashboard handles mouse input while on the dashboard view.
func (h *Handler) UpdateDashboard(msg tea.MouseMsg, cb DashboardCallbacks) tea.Cmd {
	if cmd, handled := h.handleDrag(msg, cb); handled {
		return cmd
	}
	if cmd, handled := h.handleHeaderClick(msg, cb); handled {
		return cmd
	}
	if isWheelEvent(msg) {
		return h.handleWheel(msg, cb)
	}
	if cmd, handled := h.handlePanePressPassthrough(msg, cb); handled {
		return cmd
	}
	h.dragActive = false

	if !isPrimaryClick(msg) {
		return nil
	}

	hit, ok := cb.HitPane(msg.X, msg.Y)
	if !ok {
		return nil
	}
	return h.handlePaneClick(msg, cb, hit)
}

func (h *Handler) handleHeaderClick(msg tea.MouseMsg, cb DashboardCallbacks) (tea.Cmd, bool) {
	if !isPrimaryClick(msg) {
		return nil, false
	}
	hit, ok := cb.HitHeader(msg.X, msg.Y)
	if !ok {
		return nil, false
	}
	switch hit.Kind {
	case HeaderDashboard:
		if cb.SelectDashboardTab != nil && cb.SelectDashboardTab() {
			return cb.SelectionRefreshCmd(), true
		}
		return nil, true
	case HeaderProject:
		if cb.SelectProjectTab != nil && cb.SelectProjectTab(hit.ProjectID) {
			return cb.SelectionRefreshCmd(), true
		}
		return nil, true
	case HeaderNew:
		if cb.OpenProjectPicker != nil {
			cb.OpenProjectPicker()
		}
		return nil, true
	case HeaderUpdate:
		if cb.OpenUpdateDialog != nil {
			cb.OpenUpdateDialog()
		}
		return nil, true
	default:
		return nil, true
	}
}

func (h *Handler) handleDrag(msg tea.MouseMsg, cb DashboardCallbacks) (tea.Cmd, bool) {
	if !h.dragActive {
		return nil, false
	}
	switch msg.Action {
	case tea.MouseActionMotion, tea.MouseActionRelease:
		clamped := clampMouseMsg(msg, h.dragHit.Content)
		cmd := cb.ForwardMouseEvent(h.dragHit, clamped)
		if msg.Action == tea.MouseActionRelease {
			h.dragActive = false
		}
		return cmd, true
	default:
		return nil, false
	}
}

func (h *Handler) handleWheel(msg tea.MouseMsg, cb DashboardCallbacks) tea.Cmd {
	if cb.HitPane == nil || cb.ForwardMouseEvent == nil {
		return nil
	}
	hit, ok := cb.HitPane(msg.X, msg.Y)
	if !ok || hit.PaneID == "" {
		return nil
	}
	if !hit.Content.Contains(msg.X, msg.Y) {
		return nil
	}
	return cb.ForwardMouseEvent(hit, msg)
}

func (h *Handler) handlePanePressPassthrough(msg tea.MouseMsg, cb DashboardCallbacks) (tea.Cmd, bool) {
	if !isPrimaryClick(msg) {
		return nil, false
	}
	if cb.HitPane == nil || cb.ForwardMouseEvent == nil {
		return nil, false
	}
	hit, ok := cb.HitPane(msg.X, msg.Y)
	if !ok || hit.PaneID == "" {
		return nil, false
	}
	if !hit.Content.Contains(msg.X, msg.Y) {
		return nil, false
	}
	h.dragActive = true
	h.dragHit = hit
	forwardCmd := cb.ForwardMouseEvent(hit, msg)
	selectCmd := h.handlePaneClick(msg, cb, hit)
	if forwardCmd == nil {
		return selectCmd, true
	}
	if selectCmd == nil {
		return forwardCmd, true
	}
	return tea.Batch(forwardCmd, selectCmd), true
}

func (h *Handler) handlePaneClick(msg tea.MouseMsg, cb DashboardCallbacks, hit PaneHit) tea.Cmd {
	if cb.ApplySelection(hit.Selection) {
		return tea.Batch(cb.SelectionCmd(), cb.RefreshPaneViewsCmd())
	}
	return nil
}

func isPrimaryClick(msg tea.MouseMsg) bool {
	return msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft
}

func isWheelEvent(msg tea.MouseMsg) bool {
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown, tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		return true
	default:
		return false
	}
}

func clampMouseMsg(msg tea.MouseMsg, rect Rect) tea.MouseMsg {
	if rect.Empty() {
		return msg
	}
	maxX := rect.X + rect.W - 1
	maxY := rect.Y + rect.H - 1
	if maxX < rect.X || maxY < rect.Y {
		return msg
	}
	msg.X = clampInt(msg.X, rect.X, maxX)
	msg.Y = clampInt(msg.Y, rect.Y, maxY)
	return msg
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
