package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type cursorShape int

const (
	cursorShapeUnknown cursorShape = iota
	cursorShapeText
	cursorShapePointer
)

const cursorShapeThrottle = 50 * time.Millisecond

func oscForCursorShape(shape cursorShape) string {
	switch shape {
	case cursorShapeText:
		return "\x1b]22;text\x07"
	case cursorShapePointer:
		return "\x1b]22;pointer\x07"
	default:
		return ""
	}
}

type cursorShapeFlushMsg struct {
	At time.Time
}

type oscClearMsg struct{}

func (m *Model) updateCursorShape(msg tea.MouseMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if m.state != StateDashboard {
		return nil
	}
	switch msg.Action {
	case tea.MouseActionMotion, tea.MouseActionPress, tea.MouseActionRelease:
	default:
		return nil
	}

	shape, ok := m.cursorShapeAt(msg.X, msg.Y)
	if !ok || shape == cursorShapeUnknown {
		return nil
	}
	if shape == m.cursorShape {
		return nil
	}

	now := time.Now
	if m.cursorShapeNow != nil {
		now = m.cursorShapeNow
	}

	at := now()
	if m.cursorShapeLastSentAt.IsZero() || at.Sub(m.cursorShapeLastSentAt) >= cursorShapeThrottle {
		m.cursorShape = shape
		m.cursorShapePending = cursorShapeUnknown
		m.cursorShapeLastSentAt = at
		m.cursorShapeFlushScheduled = false
		return m.emitOSC(oscForCursorShape(shape))
	}

	m.cursorShapePending = shape
	if m.cursorShapeFlushScheduled {
		return nil
	}
	after := cursorShapeThrottle - at.Sub(m.cursorShapeLastSentAt)
	if after < 0 {
		after = 0
	}
	m.cursorShapeFlushScheduled = true
	return tea.Tick(after, func(t time.Time) tea.Msg {
		return cursorShapeFlushMsg{At: t}
	})
}

func (m *Model) handleCursorShapeFlush(msg cursorShapeFlushMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	m.cursorShapeFlushScheduled = false

	if m.state != StateDashboard {
		m.cursorShapePending = cursorShapeUnknown
		return nil
	}
	pending := m.cursorShapePending
	m.cursorShapePending = cursorShapeUnknown
	if pending == cursorShapeUnknown || pending == m.cursorShape {
		return nil
	}

	at := msg.At
	if !m.cursorShapeLastSentAt.IsZero() && at.Sub(m.cursorShapeLastSentAt) < cursorShapeThrottle {
		after := cursorShapeThrottle - at.Sub(m.cursorShapeLastSentAt)
		if after < 0 {
			after = 0
		}
		m.cursorShapePending = pending
		m.cursorShapeFlushScheduled = true
		return tea.Tick(after, func(t time.Time) tea.Msg {
			return cursorShapeFlushMsg{At: t}
		})
	}

	m.cursorShape = pending
	m.cursorShapeLastSentAt = at
	return m.emitOSC(oscForCursorShape(pending))
}

func (m *Model) emitOSC(seq string) tea.Cmd {
	if m == nil {
		return nil
	}
	if seq == "" {
		return nil
	}
	m.oscPending = seq
	return tea.Tick(0, func(time.Time) tea.Msg { return oscClearMsg{} })
}

func (m *Model) cursorShapeAt(x, y int) (cursorShape, bool) {
	if m == nil || m.state != StateDashboard {
		return cursorShapeUnknown, false
	}
	if rect, ok := m.quickReplyRect(); ok && rect.Contains(x, y) {
		return cursorShapeText, true
	}
	if rect, ok := m.headerRect(); ok && rect.Contains(x, y) {
		return cursorShapePointer, true
	}
	body, ok := m.dashboardBodyRect()
	if !ok || !body.Contains(x, y) {
		return cursorShapePointer, true
	}

	if m.tab == TabDashboard {
		return cursorShapeText, true
	}
	if m.tab == TabProject {
		project := m.selectedProject()
		if project == nil || m.sidebarHidden(project) {
			return cursorShapeText, true
		}
		preview := m.projectSidebarPreviewRect(body)
		if preview.Contains(x, y) {
			return cursorShapeText, true
		}
		return cursorShapePointer, true
	}
	return cursorShapePointer, true
}
