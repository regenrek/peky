package native

import "github.com/regenrek/peakypanes/internal/limits"

func (m *Manager) applyScrollbackBudgets() {
	if m == nil || m.closed.Load() {
		return
	}

	var panes []*Pane
	m.mu.RLock()
	for _, pane := range m.panes {
		if pane != nil && pane.window != nil {
			panes = append(panes, pane)
		}
	}
	m.mu.RUnlock()

	if len(panes) == 0 {
		return
	}

	perPane := limits.ScrollbackMaxBytesPerPane(len(panes))
	for _, pane := range panes {
		pane.window.SetScrollbackMaxBytes(perPane)
	}
}
