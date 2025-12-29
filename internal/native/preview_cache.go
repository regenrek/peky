package native

import "time"

type previewState struct {
	lines    []string
	sourceAt time.Time
}

func (m *Manager) snapshotPreviewStates(ids []string) (map[string]previewState, int) {
	if m == nil {
		return nil, 0
	}
	states := make(map[string]previewState, len(ids))
	m.previewMu.Lock()
	cursor := m.previewCursor
	for _, id := range ids {
		if state, ok := m.previewCache[id]; ok {
			states[id] = state
		}
	}
	m.previewMu.Unlock()
	return states, cursor
}

func (m *Manager) applyPreviewUpdates(updates map[string]previewState, cursor int) {
	if m == nil {
		return
	}
	if len(updates) == 0 && cursor < 0 {
		return
	}
	m.previewMu.Lock()
	if m.previewCache == nil {
		m.previewCache = make(map[string]previewState)
	}
	for id, state := range updates {
		m.previewCache[id] = state
	}
	if cursor >= 0 {
		m.previewCursor = cursor
	}
	m.previewMu.Unlock()
}

func (m *Manager) dropPreviewCache(ids ...string) {
	if m == nil {
		return
	}
	m.previewMu.Lock()
	if m.previewCache != nil {
		for _, id := range ids {
			delete(m.previewCache, id)
		}
	}
	m.previewMu.Unlock()
}
