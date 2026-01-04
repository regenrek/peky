package native

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/logging"
	"github.com/regenrek/peakypanes/internal/tool"
)

// LayoutBaseSize is the normalized coordinate space for pane layouts.
const LayoutBaseSize = 1000

// PaneEventType identifies the kind of pane event.
type PaneEventType uint8

const (
	PaneEventUpdated PaneEventType = iota + 1
	PaneEventToast
	PaneEventMetaUpdated
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
	toolRegistry atomic.Value

	previewMu     sync.Mutex
	previewCache  map[string]previewState
	previewCursor int

	perfMu     sync.Mutex
	perfLogged map[string]uint8

	outputMu    sync.Mutex
	outputReady map[string]chan struct{}
	outputSeen  map[string]bool
}

// NewManager creates a new native session manager.
func NewManager() (*Manager, error) {
	mgr := &Manager{
		sessions:     make(map[string]*Session),
		panes:        make(map[string]*Pane),
		events:       make(chan PaneEvent, 128),
		previewCache: make(map[string]previewState),
	}
	reg, err := tool.DefaultRegistry()
	if err != nil {
		return nil, err
	}
	if err := mgr.SetToolRegistry(reg); err != nil {
		return nil, err
	}
	return mgr, nil
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
	var panes []*Pane
	m.mu.Lock()
	for _, session := range m.sessions {
		for _, pane := range session.Panes {
			if pane != nil && pane.output != nil {
				pane.output.disable()
			}
			if pane.window != nil {
				panes = append(panes, pane)
			}
		}
	}
	m.sessions = nil
	m.panes = nil
	m.mu.Unlock()

	if len(panes) > 0 {
		m.closePanes(panes)
	}

	m.previewMu.Lock()
	m.previewCache = nil
	m.previewCursor = 0
	m.previewMu.Unlock()

	m.outputMu.Lock()
	if m.outputReady != nil {
		for _, ch := range m.outputReady {
			close(ch)
		}
	}
	m.outputReady = nil
	m.outputSeen = nil
	m.outputMu.Unlock()

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
	m.mu.RLock()
	pane := m.panes[id]
	m.mu.RUnlock()
	if pane != nil {
		pane.SetLastActive(time.Now())
		if pane.window != nil {
			seq = pane.window.UpdateSeq()
		}
	}
	if pane != nil {
		m.logPanePerf(pane)
		if pane.window != nil && !pane.window.FirstReadAt().IsZero() {
			m.markPaneOutputReady(pane.ID)
		}
	}
	m.notify(id, seq)
}

func (m *Manager) notify(id string, seq uint64) {
	if m == nil || m.closed.Load() {
		return
	}
	m.version.Add(1)
	m.emitEvent(PaneEvent{Type: PaneEventUpdated, PaneID: id, Seq: seq})
}

func (m *Manager) notifyMeta(id string) {
	if m == nil || m.closed.Load() {
		return
	}
	m.version.Add(1)
	m.emitEvent(PaneEvent{Type: PaneEventMetaUpdated, PaneID: id})
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
		logging.LogEvery(
			context.Background(),
			"native.events.drop",
			2*time.Second,
			slog.LevelWarn,
			"native: drop pane event",
			slog.String("pane_id", event.PaneID),
			slog.Uint64("seq", event.Seq),
			slog.Int("type", int(event.Type)),
		)
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
		if pane != nil && pane.output != nil {
			pane.output.disable()
		}
		if pane.window != nil {
			_ = pane.window.Close()
		}
	}
}

func (m *Manager) nextPaneID() string {
	n := m.nextID.Add(1)
	return fmt.Sprintf("p-%d", n)
}
