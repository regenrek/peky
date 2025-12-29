package mouse

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const doubleClickThreshold = 350 * time.Millisecond

// Handler tracks mouse click state and applies dashboard mouse logic.
type Handler struct {
	lastClickAt     time.Time
	lastClickPaneID string
	lastClickButton tea.MouseButton
}

// DashboardCallbacks wires mouse handling into the app model.
type DashboardCallbacks struct {
	HitHeader             func(x, y int) (HeaderHit, bool)
	HitPane               func(x, y int) (PaneHit, bool)
	HitIsSelected         func(hit PaneHit) bool
	ApplySelection        func(selection Selection) bool
	SelectDashboardTab    func() bool
	SelectProjectTab      func(projectID string) bool
	OpenProjectPicker     func()
	SetTerminalFocus      func(focus bool)
	TerminalFocus         func() bool
	SupportsTerminalFocus func() bool
	SelectionCmd          func() tea.Cmd
	SelectionRefreshCmd   func() tea.Cmd
	RefreshPaneViewsCmd   func() tea.Cmd
	ForwardMouseEvent     func(hit PaneHit, msg tea.MouseMsg) tea.Cmd
	FocusUnavailable      func()
}

// UpdateDashboard handles mouse input while on the dashboard view.
func (h *Handler) UpdateDashboard(msg tea.MouseMsg, cb DashboardCallbacks) tea.Cmd {
	if cmd, handled := h.handleHeaderClick(msg, cb); handled {
		return cmd
	}

	hit, ok := cb.HitPane(msg.X, msg.Y)
	if cb.TerminalFocus != nil && cb.TerminalFocus() {
		return h.handleTerminalFocusMouse(msg, cb, hit, ok)
	}

	if !isPrimaryClick(msg) || !ok {
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
	if cb.TerminalFocus != nil && cb.TerminalFocus() {
		cb.SetTerminalFocus(false)
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
	default:
		return nil, true
	}
}

func (h *Handler) handleTerminalFocusMouse(msg tea.MouseMsg, cb DashboardCallbacks, hit PaneHit, ok bool) tea.Cmd {
	if ok && cb.HitIsSelected(hit) && hit.Content.Contains(msg.X, msg.Y) {
		return cb.ForwardMouseEvent(hit, msg)
	}
	if isPrimaryClick(msg) && ok && !cb.HitIsSelected(hit) {
		cb.SetTerminalFocus(false)
		changed := cb.ApplySelection(hit.Selection)
		h.recordClick(hit, msg)
		if changed {
			return tea.Batch(cb.SelectionCmd(), cb.RefreshPaneViewsCmd())
		}
		return cb.RefreshPaneViewsCmd()
	}
	return nil
}

func (h *Handler) handlePaneClick(msg tea.MouseMsg, cb DashboardCallbacks, hit PaneHit) tea.Cmd {
	if h.isDoubleClick(hit, msg) {
		h.clearLastClick()
		changed := cb.ApplySelection(hit.Selection)
		if cb.SupportsTerminalFocus != nil && !cb.SupportsTerminalFocus() {
			if cb.FocusUnavailable != nil {
				cb.FocusUnavailable()
			}
			if changed {
				return cb.SelectionCmd()
			}
			return nil
		}
		cb.SetTerminalFocus(true)
		cmds := []tea.Cmd{cb.RefreshPaneViewsCmd()}
		if changed {
			cmds = append(cmds, cb.SelectionCmd())
		}
		return tea.Batch(cmds...)
	}
	h.recordClick(hit, msg)
	if cb.ApplySelection(hit.Selection) {
		return tea.Batch(cb.SelectionCmd(), cb.RefreshPaneViewsCmd())
	}
	return nil
}

func isPrimaryClick(msg tea.MouseMsg) bool {
	return msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft
}

func (h *Handler) recordClick(hit PaneHit, msg tea.MouseMsg) {
	h.lastClickAt = time.Now()
	h.lastClickPaneID = hit.PaneID
	h.lastClickButton = msg.Button
}

func (h *Handler) clearLastClick() {
	h.lastClickAt = time.Time{}
	h.lastClickPaneID = ""
	h.lastClickButton = tea.MouseButtonNone
}

func (h *Handler) isDoubleClick(hit PaneHit, msg tea.MouseMsg) bool {
	if hit.PaneID == "" {
		return false
	}
	if h.lastClickPaneID != hit.PaneID {
		return false
	}
	if h.lastClickButton != msg.Button {
		return false
	}
	if time.Since(h.lastClickAt) > doubleClickThreshold {
		return false
	}
	return true
}
