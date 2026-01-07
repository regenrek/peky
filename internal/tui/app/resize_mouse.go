package app

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
)

type resizeDragFlushMsg struct {
	At time.Time
}

func (m *Model) handleResizeMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil || m.state != StateDashboard || m.tab != TabProject {
		return nil, false
	}
	if m.resize.drag.active {
		return m.updateResizeDrag(msg)
	}

	switch msg.Action {
	case tea.MouseActionMotion:
		m.updateResizeHover(msg)
		return nil, false
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil, false
		}
		hit, ok := m.resizeHitTest(msg.X, msg.Y)
		if !ok {
			return nil, false
		}
		if hit.Kind == resizeHitEdge || hit.Kind == resizeHitCorner {
			return m.startResizeDrag(hit, msg), true
		}
	}
	return nil, false
}

func (m *Model) updateResizeHover(msg tea.MouseMsg) {
	hit, ok := m.resizeHitTest(msg.X, msg.Y)
	if !ok {
		m.resize.hover = resizeHoverState{}
		return
	}
	switch hit.Kind {
	case resizeHitCorner:
		m.resize.hover = resizeHoverState{corner: hit.Corner, hasCorner: true}
	case resizeHitEdge:
		m.resize.hover = resizeHoverState{edge: hit.Edge, hasEdge: true}
	default:
		m.resize.hover = resizeHoverState{}
	}
}

func (m *Model) startResizeDrag(hit resizeHit, msg tea.MouseMsg) tea.Cmd {
	session := m.selectedSession()
	if session == nil || session.Name == "" {
		return nil
	}
	geom, ok := m.resizeGeometry()
	if !ok {
		return nil
	}
	engine := m.layoutEngines[session.Name]
	if engine == nil || engine.Tree == nil {
		return nil
	}
	base := cloneLayoutEngine(engine)
	if base == nil {
		return nil
	}
	preview := cloneLayoutEngine(engine)
	if preview == nil {
		return nil
	}
	lx, ly, ok := layoutgeom.LayoutPosFromScreen(geom.Preview, msg.X, msg.Y)
	if !ok {
		return nil
	}
	snapEnabled := m.resize.snap && !msg.Alt
	m.resize.drag = resizeDragState{
		active:       true,
		session:      session.Name,
		startLayoutX: lx,
		startLayoutY: ly,
		snapEnabled:  snapEnabled,
		cursorX:      msg.X,
		cursorY:      msg.Y,
		cursorSet:    true,
	}
	m.resize.preview = resizePreviewState{
		active:  true,
		session: session.Name,
		engine:  preview,
		base:    base,
	}
	switch hit.Kind {
	case resizeHitCorner:
		m.resize.drag.corner = hit.Corner
		m.resize.drag.cornerActive = true
		m.resize.drag.edge = hit.Corner.Vertical
		m.resize.preview.corner = hit.Corner
		m.resize.preview.cornerActive = true
		if pos, ok := edgePosition(base, hit.Corner.Vertical); ok {
			m.resize.drag.baseEdgePos = pos
		}
		if pos, ok := edgePosition(base, hit.Corner.Horizontal); ok {
			m.resize.drag.baseEdgePosAlt = pos
		}
	case resizeHitEdge:
		m.resize.drag.edge = hit.Edge
		m.resize.preview.edge = hit.Edge
		if pos, ok := edgePosition(base, hit.Edge); ok {
			m.resize.drag.baseEdgePos = pos
		}
	}
	return nil
}

func (m *Model) updateResizeDrag(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil || !m.resize.drag.active {
		return nil, false
	}
	switch msg.Action {
	case tea.MouseActionMotion:
		return m.handleResizeDragMotion(msg), true
	case tea.MouseActionPress:
		if hit, ok := m.resizeHitTest(msg.X, msg.Y); !ok || (hit.Kind != resizeHitEdge && hit.Kind != resizeHitCorner) {
			return m.cancelResizeDrag(), true
		}
		return nil, true
	case tea.MouseActionRelease:
		m.updateResizeDragPreview(msg)
		return m.finishResizeDrag(), true
	default:
		return nil, true
	}
}

func (m *Model) handleResizeDragMotion(msg tea.MouseMsg) tea.Cmd {
	m.updateResizeDragPreview(msg)
	if m.resizeMouseApplyMode() == ResizeMouseApplyCommit {
		return nil
	}
	if !m.resize.drag.hasPending {
		return nil
	}

	now := time.Now()
	throttle := m.resizeMouseThrottle()
	if m.resize.drag.lastSentAt.IsZero() || now.Sub(m.resize.drag.lastSentAt) >= throttle {
		return m.flushResizeDrag(now)
	}
	if m.resize.drag.sendScheduled {
		return nil
	}
	delay := throttle - now.Sub(m.resize.drag.lastSentAt)
	if delay < 0 {
		delay = 0
	}
	m.resize.drag.sendScheduled = true
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return resizeDragFlushMsg{At: t}
	})
}

func (m *Model) updateResizeDragPreview(msg tea.MouseMsg) {
	geom, ok := m.resizeGeometry()
	if !ok {
		return
	}
	m.resize.drag.cursorX = msg.X
	m.resize.drag.cursorY = msg.Y
	m.resize.drag.cursorSet = true
	lx, ly, ok := layoutgeom.LayoutPosFromScreen(geom.Preview, msg.X, msg.Y)
	if !ok {
		return
	}
	delta := lx - m.resize.drag.startLayoutX
	altDelta := ly - m.resize.drag.startLayoutY
	if !m.resize.drag.cornerActive {
		switch m.resize.drag.edge.Edge {
		case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
			altDelta = 0
		case sessiond.ResizeEdgeUp, sessiond.ResizeEdgeDown:
			delta = altDelta
			altDelta = 0
		}
	}
	snapEnabled := m.resize.snap && !msg.Alt
	m.resize.drag.snapEnabled = snapEnabled
	if !snapEnabled {
		m.resize.drag.snapState = sessiond.SnapState{}
		m.resize.drag.snapStateAlt = sessiond.SnapState{}
	}
	appliedDelta, appliedAlt, _ := m.applyResizePreview(delta, altDelta, snapEnabled)
	m.resize.drag.pendingDelta = appliedDelta
	m.resize.drag.pendingAlt = appliedAlt
	m.resize.drag.hasPending = true
}

func (m *Model) handleResizeDragFlush(msg resizeDragFlushMsg) tea.Cmd {
	if m == nil || !m.resize.drag.active {
		return nil
	}
	if m.resizeMouseApplyMode() == ResizeMouseApplyCommit {
		return nil
	}
	if !m.resize.drag.hasPending {
		return nil
	}
	m.resize.drag.sendScheduled = false
	return m.flushResizeDrag(msg.At)
}

func (m *Model) flushResizeDrag(now time.Time) tea.Cmd {
	if m == nil || !m.resize.drag.active || !m.resize.drag.hasPending {
		return nil
	}
	if m.resizeMouseApplyMode() == ResizeMouseApplyCommit {
		return nil
	}
	delta := m.resize.drag.pendingDelta
	alt := m.resize.drag.pendingAlt
	m.resize.drag.hasPending = false
	cmd := m.sendResizeDelta(delta, alt, m.resize.drag.snapEnabled)
	m.resize.drag.lastSentAt = now
	return cmd
}

func (m *Model) finishResizeDrag() tea.Cmd {
	if m == nil || !m.resize.drag.active {
		return nil
	}
	applyMode := m.resizeMouseApplyMode()
	var cmd tea.Cmd
	if applyMode == ResizeMouseApplyCommit {
		cmd = m.sendResizeDelta(m.resize.drag.pendingDelta, m.resize.drag.pendingAlt, m.resize.drag.snapEnabled)
	} else {
		cmd = m.flushResizeDrag(time.Now())
	}
	m.commitResizePreview()
	m.resize.drag = resizeDragState{}
	m.resize.preview = resizePreviewState{}
	m.resize.invalidateCache()
	if cmd == nil {
		return m.requestRefreshCmd()
	}
	return tea.Batch(cmd, m.requestRefreshCmd())
}

func (m *Model) cancelResizeDrag() tea.Cmd {
	if m == nil || !m.resize.drag.active {
		return nil
	}
	m.resize.preview.engine = m.resize.preview.base
	applyMode := m.resizeMouseApplyMode()
	var cmd tea.Cmd
	if applyMode != ResizeMouseApplyCommit {
		m.resize.drag.pendingDelta = 0
		m.resize.drag.pendingAlt = 0
		m.resize.drag.hasPending = true
		cmd = m.flushResizeDrag(time.Now())
	}
	m.resize.drag = resizeDragState{}
	m.resize.preview = resizePreviewState{}
	m.resize.invalidateCache()
	if cmd == nil {
		return m.requestRefreshCmd()
	}
	return tea.Batch(cmd, m.requestRefreshCmd())
}

func (m *Model) applyResizePreview(delta, altDelta int, snapEnabled bool) (int, int, bool) {
	if m.resize.preview.base == nil {
		return 0, 0, false
	}
	preview := cloneLayoutEngine(m.resize.preview.base)
	if preview == nil {
		return 0, 0, false
	}
	snapped := false
	appliedDelta := 0
	appliedAlt := 0

	if m.resize.drag.cornerActive {
		result, err := applyResizeToEngine(preview, m.resize.drag.corner.Vertical, delta, snapEnabled, m.resize.drag.snapState)
		if err == nil {
			m.resize.drag.snapState = result.snapState
			snapped = snapped || result.snapped
		}
		resultAlt, err := applyResizeToEngine(preview, m.resize.drag.corner.Horizontal, altDelta, snapEnabled, m.resize.drag.snapStateAlt)
		if err == nil {
			m.resize.drag.snapStateAlt = resultAlt.snapState
			snapped = snapped || resultAlt.snapped
		}
		if pos, ok := edgePosition(preview, m.resize.drag.corner.Vertical); ok {
			appliedDelta = pos - m.resize.drag.baseEdgePos
		}
		if pos, ok := edgePosition(preview, m.resize.drag.corner.Horizontal); ok {
			appliedAlt = pos - m.resize.drag.baseEdgePosAlt
		}
	} else {
		result, err := applyResizeToEngine(preview, m.resize.drag.edge, delta, snapEnabled, m.resize.drag.snapState)
		if err == nil {
			m.resize.drag.snapState = result.snapState
			snapped = result.snapped
		}
		if pos, ok := edgePosition(preview, m.resize.drag.edge); ok {
			appliedDelta = pos - m.resize.drag.baseEdgePos
		}
	}
	m.resize.preview.engine = preview
	m.resize.invalidateCache()
	return appliedDelta, appliedAlt, snapped
}

func (m *Model) sendResizeDelta(targetDelta, targetAlt int, snapEnabled bool) tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	if !m.resize.drag.active {
		return nil
	}
	delta := targetDelta - m.resize.drag.lastAppliedDelta
	alt := targetAlt - m.resize.drag.lastAppliedAlt
	if delta == 0 && alt == 0 {
		return nil
	}
	m.resize.drag.lastAppliedDelta = targetDelta
	m.resize.drag.lastAppliedAlt = targetAlt

	session := m.resize.drag.session
	edge := m.resize.drag.edge
	corner := m.resize.drag.corner
	cornerActive := m.resize.drag.cornerActive
	primarySnap := m.resize.drag.snapState
	altSnap := m.resize.drag.snapStateAlt
	if !snapEnabled {
		primarySnap = sessiond.SnapState{}
		altSnap = sessiond.SnapState{}
	}
	client := m.client
	return func() tea.Msg {
		if client == nil {
			return ErrorMsg{Err: errors.New("session client unavailable"), Context: "resize pane"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if delta != 0 {
			resp, err := client.ResizePaneEdge(ctx, session, edge.PaneID, edge.Edge, delta, snapEnabled, primarySnap)
			if err != nil {
				return ErrorMsg{Err: err, Context: "resize pane"}
			}
			_ = resp
		}
		if cornerActive && alt != 0 {
			resp, err := client.ResizePaneEdge(ctx, session, corner.Horizontal.PaneID, corner.Horizontal.Edge, alt, snapEnabled, altSnap)
			if err != nil {
				return ErrorMsg{Err: err, Context: "resize pane"}
			}
			_ = resp
		}
		return nil
	}
}

func (m *Model) commitResizePreview() {
	if m == nil || !m.resize.preview.active || m.resize.preview.engine == nil {
		return
	}
	if m.layoutEngines == nil {
		m.layoutEngines = make(map[string]*layout.Engine)
	}
	m.layoutEngines[m.resize.preview.session] = m.resize.preview.engine
	m.layoutEngineVersion++
	m.resize.invalidateCache()
}

func (m *Model) resizeMouseApplyMode() string {
	if m == nil {
		return ResizeMouseApplyLive
	}
	mode := m.settings.Resize.MouseApply
	if mode == "" {
		return ResizeMouseApplyLive
	}
	return mode
}

func (m *Model) resizeMouseThrottle() time.Duration {
	if m == nil {
		return 16 * time.Millisecond
	}
	if m.settings.Resize.MouseThrottle <= 0 {
		return 16 * time.Millisecond
	}
	return m.settings.Resize.MouseThrottle
}

func edgePosition(engine *layout.Engine, edge resizeEdgeRef) (int, bool) {
	if engine == nil || engine.Tree == nil || edge.PaneID == "" {
		return 0, false
	}
	rects := engine.Tree.ViewRects()
	rect, ok := rects[edge.PaneID]
	if !ok {
		return 0, false
	}
	switch edge.Edge {
	case sessiond.ResizeEdgeLeft:
		return rect.X, true
	case sessiond.ResizeEdgeRight:
		return rect.X + rect.W, true
	case sessiond.ResizeEdgeUp:
		return rect.Y, true
	case sessiond.ResizeEdgeDown:
		return rect.Y + rect.H, true
	default:
		return 0, false
	}
}
