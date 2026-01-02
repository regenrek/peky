package native

import "time"

func (m *Manager) markPaneOutputReady(id string) {
	if m == nil || id == "" {
		return
	}
	m.outputMu.Lock()
	if m.outputSeen == nil {
		m.outputSeen = make(map[string]bool)
	}
	if m.outputSeen[id] {
		m.outputMu.Unlock()
		return
	}
	m.outputSeen[id] = true
	if m.outputReady != nil {
		if ch, ok := m.outputReady[id]; ok {
			close(ch)
			delete(m.outputReady, id)
		}
	}
	m.outputMu.Unlock()
	m.notifyPane(id)
}

func (m *Manager) paneOutputWaiter(id string) (<-chan struct{}, bool) {
	if m == nil || id == "" {
		return nil, false
	}
	m.outputMu.Lock()
	defer m.outputMu.Unlock()
	if m.outputSeen != nil && m.outputSeen[id] {
		return nil, true
	}
	if m.outputReady == nil {
		m.outputReady = make(map[string]chan struct{})
	}
	ch := m.outputReady[id]
	if ch == nil {
		ch = make(chan struct{})
		m.outputReady[id] = ch
	}
	return ch, false
}

func (m *Manager) waitForPaneOutput(id string, timeout time.Duration) bool {
	if m == nil || id == "" || m.closed.Load() {
		return false
	}
	ch, ready := m.paneOutputWaiter(id)
	if ready {
		return true
	}
	if timeout <= 0 {
		<-ch
		return true
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ch:
		return true
	case <-timer.C:
		return false
	}
}

func (m *Manager) clearOutputWaiters(ids ...string) {
	if m == nil || len(ids) == 0 {
		return
	}
	m.outputMu.Lock()
	defer m.outputMu.Unlock()
	for _, id := range ids {
		if id == "" {
			continue
		}
		if m.outputReady != nil {
			if ch, ok := m.outputReady[id]; ok {
				close(ch)
				delete(m.outputReady, id)
			}
		}
		if m.outputSeen != nil {
			delete(m.outputSeen, id)
		}
	}
}
