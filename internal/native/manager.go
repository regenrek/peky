package native

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/diag"
)

// LayoutBaseSize is the normalized coordinate space for pane layouts.
const LayoutBaseSize = 1000

// PaneEventType identifies the kind of pane event.
type PaneEventType uint8

const (
	PaneEventUpdated PaneEventType = iota + 1
	PaneEventToast
)

// PaneEvent signals that a pane updated or emitted a toast.
type PaneEvent struct {
	Type   PaneEventType
	PaneID string
	Seq    uint64
	Toast  string
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

	previewMu     sync.Mutex
	previewCache  map[string]previewState
	previewCursor int
}

// NewManager creates a new native session manager.
func NewManager() *Manager {
	return &Manager{
		sessions:     make(map[string]*Session),
		panes:        make(map[string]*Pane),
		events:       make(chan PaneEvent, 128),
		previewCache: make(map[string]previewState),
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

	m.previewMu.Lock()
	m.previewCache = nil
	m.previewCursor = 0
	m.previewMu.Unlock()

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
	var seq uint64
	m.mu.Lock()
	pane := m.panes[id]
	if pane != nil {
		pane.LastActive = time.Now()
		if pane.window != nil {
			seq = pane.window.UpdateSeq()
		}
	}
	m.mu.Unlock()
	m.notify(id, seq)
}

func (m *Manager) notify(id string, seq uint64) {
	if m == nil || m.closed.Load() {
		return
	}
	m.version.Add(1)
	m.emitEvent(PaneEvent{Type: PaneEventUpdated, PaneID: id, Seq: seq})
}

func (m *Manager) notifyToast(id, message string) {
	if m == nil || m.closed.Load() {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	m.emitEvent(PaneEvent{Type: PaneEventToast, PaneID: id, Toast: message})
}

func (m *Manager) emitEvent(event PaneEvent) {
	if m == nil || m.closed.Load() {
		return
	}
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()
	if m.eventsClosed {
		return
	}
	select {
	case m.events <- event:
	default:
		diag.LogEvery("native.events.drop", 2*time.Second, "native: drop pane event id=%s seq=%d type=%d", event.PaneID, event.Seq, event.Type)
	}
}

func (m *Manager) notifyPane(id string) {
	if m == nil || m.closed.Load() {
		return
	}
	var seq uint64
	m.mu.RLock()
	pane := m.panes[id]
	if pane != nil && pane.window != nil {
		seq = pane.window.UpdateSeq()
	}
	m.mu.RUnlock()
	m.notify(id, seq)
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
