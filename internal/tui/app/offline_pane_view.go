package app

import "strings"

func (m *Model) offlinePaneView(pane *PaneItem, width, height int) string {
	if pane == nil || width <= 0 || height <= 0 {
		return ""
	}
	if strings.TrimSpace(pane.ID) == "" {
		return ""
	}
	m.ensureOfflineScrollMap()
	if height < 1 {
		height = 1
	}
	m.offlineScrollViewport[pane.ID] = height

	lines := pane.Preview
	if len(lines) == 0 {
		return ""
	}
	offset := 0
	if m.offlineScrollActiveFor(pane.ID) {
		offset = m.offlineScrollOffset(pane.ID)
	}
	max := m.offlineScrollMax(*pane)
	if offset > max {
		offset = max
		if m.offlineScrollActiveFor(pane.ID) {
			m.offlineScroll[pane.ID] = offset
		}
	}
	start := len(lines) - height - offset
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}
