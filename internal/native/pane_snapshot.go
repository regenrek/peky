package native

import (
	"errors"
	"fmt"
	"strings"
)

// PaneScrollbackSnapshot returns a plain-text snapshot of a pane's scrollback.
func (m *Manager) PaneScrollbackSnapshot(paneID string, rows int) (string, bool, error) {
	if m == nil {
		return "", false, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return "", false, errors.New("native: pane id is required")
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil || pane.window == nil {
		return "", false, fmt.Errorf("native: pane %q not found", paneID)
	}
	content, truncated := pane.window.SnapshotScrollback(rows)
	return content, truncated, nil
}
