package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

type cursorShape int

const (
	cursorShapeUnknown cursorShape = iota
	cursorShapeText
	cursorShapePointer
	cursorShapeColResize
	cursorShapeRowResize
	cursorShapeDiagNWSE
	cursorShapeDiagNESW
)

const cursorShapeThrottle = 50 * time.Millisecond

func oscForCursorShape(shape cursorShape) string {
	switch shape {
	case cursorShapeText:
		return "\x1b]22;text\x07"
	case cursorShapePointer:
		return "\x1b]22;pointer\x07"
	case cursorShapeColResize:
		return "\x1b]22;col-resize\x07\x1b]22;ew-resize\x07"
	case cursorShapeRowResize:
		return "\x1b]22;row-resize\x07\x1b]22;ns-resize\x07"
	case cursorShapeDiagNWSE:
		return "\x1b]22;nwse-resize\x07"
	case cursorShapeDiagNESW:
		return "\x1b]22;nesw-resize\x07"
	default:
		return ""
	}
}

type cursorShapeFlushMsg struct {
	At time.Time
}

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
	if m.oscEmit != nil {
		m.oscEmit(seq)
		return nil
	}
	return tea.Printf("%s", seq)
}

func (m *Model) cursorShapeAt(x, y int) (cursorShape, bool) {
	if m == nil || m.state != StateDashboard {
		return cursorShapeUnknown, false
	}
	layout, ok := m.dashboardLayoutInternal("")
	if !ok {
		return cursorShapePointer, true
	}

	header := mouse.Rect{X: layout.padLeft, Y: layout.padTop, W: layout.contentWidth, H: layout.headerHeight}
	if header.Contains(x, y) {
		return cursorShapePointer, true
	}

	bodyY := layout.padTop + layout.headerHeight + layout.headerGap
	body := mouse.Rect{X: layout.padLeft, Y: bodyY, W: layout.contentWidth, H: layout.bodyHeight}
	quickReplyY := bodyY + layout.bodyHeight
	quickReply := mouse.Rect{X: layout.padLeft, Y: quickReplyY, W: layout.contentWidth, H: layout.quickReplyHeight}
	if quickReply.Contains(x, y) {
		return cursorShapeText, true
	}
	if !body.Contains(x, y) {
		return cursorShapePointer, true
	}

	if hit, ok := m.resizeHitTest(x, y); ok {
		switch hit.Kind {
		case resizeHitEdge:
			switch hit.Edge.Edge {
			case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
				return cursorShapeColResize, true
			case sessiond.ResizeEdgeUp, sessiond.ResizeEdgeDown:
				return cursorShapeRowResize, true
			}
		case resizeHitCorner:
			diag := cornerCursorShape(hit.Corner)
			if diag != cursorShapeUnknown {
				return diag, true
			}
		}
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

func cornerCursorShape(corner resizeCornerRef) cursorShape {
	switch {
	case corner.Vertical.Edge == sessiond.ResizeEdgeLeft && corner.Horizontal.Edge == sessiond.ResizeEdgeUp:
		return cursorShapeDiagNWSE
	case corner.Vertical.Edge == sessiond.ResizeEdgeRight && corner.Horizontal.Edge == sessiond.ResizeEdgeDown:
		return cursorShapeDiagNWSE
	case corner.Vertical.Edge == sessiond.ResizeEdgeRight && corner.Horizontal.Edge == sessiond.ResizeEdgeUp:
		return cursorShapeDiagNESW
	case corner.Vertical.Edge == sessiond.ResizeEdgeLeft && corner.Horizontal.Edge == sessiond.ResizeEdgeDown:
		return cursorShapeDiagNESW
	default:
		return cursorShapeDiagNWSE
	}
}
