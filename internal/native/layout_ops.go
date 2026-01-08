package native

import (
	"errors"
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
)

func (m *Manager) ResizePaneEdge(sessionName, paneID string, edge layout.ResizeEdge, delta int, snap bool, snapState layout.SnapState) (layout.ApplyResult, error) {
	op := layout.ResizeOp{PaneID: paneID, Edge: edge, Delta: delta, Snap: snap, SnapState: snapState}
	return m.applyLayoutOp(sessionName, op)
}

func (m *Manager) ResetPaneSizes(sessionName, paneID string) (layout.ApplyResult, error) {
	op := layout.ResetSizesOp{PaneID: paneID}
	return m.applyLayoutOp(sessionName, op)
}

func (m *Manager) ZoomPane(sessionName, paneID string, toggle bool) (layout.ApplyResult, error) {
	op := layout.ZoomOp{PaneID: paneID, Toggle: toggle}
	return m.applyLayoutOp(sessionName, op)
}

func (m *Manager) applyLayoutOp(sessionName string, op layout.Op) (layout.ApplyResult, error) {
	if m == nil {
		return layout.ApplyResult{}, errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return layout.ApplyResult{}, errors.New("native: session is required")
	}
	if op == nil {
		return layout.ApplyResult{}, errors.New("native: layout op is nil")
	}
	if op.Kind() == layout.OpSplit || op.Kind() == layout.OpClose {
		return layout.ApplyResult{}, errors.New("native: split/close must use dedicated methods")
	}

	m.mu.Lock()
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.Unlock()
		return layout.ApplyResult{}, fmt.Errorf("native: session %q not found", sessionName)
	}
	if session.Layout == nil {
		m.mu.Unlock()
		return layout.ApplyResult{}, errors.New("native: layout engine unavailable")
	}
	result, err := session.Layout.Apply(op)
	if err != nil {
		m.mu.Unlock()
		return layout.ApplyResult{}, err
	}
	if err := applyLayoutToPanes(session); err != nil {
		m.mu.Unlock()
		return layout.ApplyResult{}, err
	}
	m.mu.Unlock()

	for _, id := range result.Affected {
		m.notifyPane(id)
	}
	return result, nil
}
