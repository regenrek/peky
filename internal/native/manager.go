package native

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const layoutBaseSize = 1000

// PaneEvent signals that a pane updated.
type PaneEvent struct {
	PaneID string
}

// Manager owns native sessions and panes.
type Manager struct {
	mu           sync.RWMutex
	eventsMu     sync.Mutex
	sessions     map[string]*Session
	panes        map[string]*Pane
	events       chan PaneEvent
	eventsClosed bool
	nextID       atomic.Uint64
	version      atomic.Uint64
	closed       atomic.Bool
}

// NewManager creates a new native session manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		panes:    make(map[string]*Pane),
		events:   make(chan PaneEvent, 128),
	}
}

// Events returns pane update events.
func (m *Manager) Events() <-chan PaneEvent {
	if m == nil {
		return nil
	}
	return m.events
}

// Version returns a monotonically increasing version for snapshot changes.
func (m *Manager) Version() uint64 {
	if m == nil {
		return 0
	}
	return m.version.Load()
}

// Close stops all sessions and releases resources.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	if m.closed.Swap(true) {
		return
	}
	m.mu.Lock()
	for _, session := range m.sessions {
		for _, pane := range session.Panes {
			if pane.window != nil {
				_ = pane.window.Close()
			}
		}
	}
	m.sessions = nil
	m.panes = nil
	m.mu.Unlock()

	m.eventsMu.Lock()
	if !m.eventsClosed {
		m.eventsClosed = true
		close(m.events)
	}
	m.eventsMu.Unlock()
}

func (m *Manager) forwardUpdates(pane *Pane) {
	if pane == nil || pane.window == nil {
		return
	}
	updates := pane.window.Updates()
	if updates == nil {
		return
	}
	go func(id string) {
		for range updates {
			m.markActive(id)
		}
	}(pane.ID)
}

func (m *Manager) markActive(id string) {
	if m == nil || m.closed.Load() {
		return
	}
	m.mu.Lock()
	pane := m.panes[id]
	if pane != nil {
		pane.LastActive = time.Now()
	}
	m.mu.Unlock()
	m.notify(id)
}

func (m *Manager) notify(id string) {
	if m == nil || m.closed.Load() {
		return
	}
	m.version.Add(1)
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()
	if m.eventsClosed {
		return
	}
	select {
	case m.events <- PaneEvent{PaneID: id}:
	default:
	}
}

func (m *Manager) closePanes(panes []*Pane) {
	for _, pane := range panes {
		if pane.window != nil {
			_ = pane.window.Close()
		}
	}
}

func (m *Manager) nextPaneID() string {
	n := m.nextID.Add(1)
	return fmt.Sprintf("p-%d", n)
}
