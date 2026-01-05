package native

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
	"github.com/regenrek/peakypanes/internal/terminal"
)

// SessionSpec describes a session to start.
type SessionSpec struct {
	Name       string
	Path       string
	Layout     *layout.LayoutConfig
	LayoutName string
	Env        []string
}

// Session is a native session container.
type Session struct {
	Name       string
	Path       string
	LayoutName string
	Panes      []*Pane
	CreatedAt  time.Time
	Env        []string
}

const previewUpdateBudget = 50 * time.Millisecond

// Session returns a snapshot pointer for a session name.
func (m *Manager) Session(name string) *Session {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[name]
}

// SessionNames returns the list of session names.
func (m *Manager) SessionNames() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].CreatedAt.Equal(sessions[j].CreatedAt) {
			return sessions[i].Name < sessions[j].Name
		}
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})

	names := make([]string, 0, len(sessions))
	for _, session := range sessions {
		names = append(names, session.Name)
	}
	return names
}

// KillSession stops and removes a session.
func (m *Manager) KillSession(name string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("native: session name is required")
	}
	m.mu.Lock()
	session, ok := m.sessions[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("native: session %q not found", name)
	}
	delete(m.sessions, name)
	var paneIDs []string
	for _, pane := range session.Panes {
		paneIDs = append(paneIDs, pane.ID)
		delete(m.panes, pane.ID)
	}
	m.mu.Unlock()

	m.applyScrollbackBudgets()
	m.dropPreviewCache(paneIDs...)
	m.clearOutputWaiters(paneIDs...)
	m.closePanes(session.Panes)
	for _, id := range paneIDs {
		m.notifyPane(id)
	}
	m.version.Add(1)
	return nil
}

// RenameSession updates a session name.
func (m *Manager) RenameSession(oldName, newName string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return errors.New("native: session name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[oldName]
	if !ok {
		return fmt.Errorf("native: session %q not found", oldName)
	}
	if _, exists := m.sessions[newName]; exists {
		return fmt.Errorf("native: session %q already exists", newName)
	}
	delete(m.sessions, oldName)
	session.Name = newName
	m.sessions[newName] = session
	m.version.Add(1)
	return nil
}

// StartSession creates a new native session.
func (m *Manager) StartSession(ctx context.Context, spec SessionSpec) (*Session, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	if m.closed.Load() {
		return nil, errors.New("native: manager closed")
	}
	normalized, err := m.normalizeSessionSpec(spec)
	if err != nil {
		return nil, err
	}
	session := newSessionFromSpec(normalized)
	panes, err := m.buildPanes(ctx, normalized)
	if err != nil {
		m.closePanes(panes)
		return nil, err
	}
	session.Panes = panes
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.Debug("native: session started", slog.String("session", session.Name), slog.Int("panes", len(panes)), slog.String("layout", session.LayoutName))
	}

	if err := m.registerSession(session, panes); err != nil {
		m.closePanes(panes)
		return nil, err
	}
	m.applyScrollbackBudgets()
	m.forwardPaneUpdates(panes)
	m.seedPaneUpdates(panes)
	m.version.Add(1)

	m.dispatchLayoutSends(session, normalized.Layout)

	return session, nil
}

func (m *Manager) normalizeSessionSpec(spec SessionSpec) (SessionSpec, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return SessionSpec{}, errors.New("native: session name is required")
	}
	if spec.Layout == nil {
		return SessionSpec{}, errors.New("native: layout is required")
	}
	if strings.TrimSpace(spec.Path) != "" {
		if err := validatePath(spec.Path); err != nil {
			return SessionSpec{}, err
		}
		spec.Path = filepath.Clean(spec.Path)
	}
	return spec, nil
}

func newSessionFromSpec(spec SessionSpec) *Session {
	session := &Session{
		Name:       spec.Name,
		Path:       spec.Path,
		LayoutName: strings.TrimSpace(spec.LayoutName),
		CreatedAt:  time.Now(),
	}
	if session.LayoutName == "" && spec.Layout != nil {
		session.LayoutName = spec.Layout.Name
	}
	if len(spec.Env) > 0 {
		session.Env = append([]string(nil), spec.Env...)
	}
	return session
}

func (m *Manager) registerSession(session *Session, panes []*Pane) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[session.Name]; ok {
		return fmt.Errorf("native: session %q already exists", session.Name)
	}
	m.sessions[session.Name] = session
	for _, pane := range panes {
		m.panes[pane.ID] = pane
	}
	return nil
}

func (m *Manager) forwardPaneUpdates(panes []*Pane) {
	for _, pane := range panes {
		m.forwardUpdates(pane)
	}
}

func (m *Manager) seedPaneUpdates(panes []*Pane) {
	// Seed an update to render initial output.
	for _, pane := range panes {
		m.notifyPane(pane.ID)
	}
}

// Snapshot returns a copy of sessions with preview lines computed.
func (m *Manager) Snapshot(ctx context.Context, previewLines int) []SessionSnapshot {
	if m == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if previewLines < 0 {
		previewLines = 0
	}

	out, paneRefs, paneIDs := m.snapshotSessions()
	if previewLines <= 0 || len(paneRefs) == 0 {
		return out
	}

	states, cursor := m.snapshotPreviewStates(paneIDs)
	needsUpdate := previewNeedsUpdate(paneRefs, states, previewLines)
	updates, nextCursor := m.collectPreviewUpdates(ctx, previewLines, paneRefs, states, needsUpdate, cursor)
	m.applyPreviewUpdates(updates, nextCursor)
	applyPreviewLines(out, paneRefs, states, previewLines)

	return out
}

type panePreviewRef struct {
	id         string
	window     *terminal.Window
	lastActive time.Time
	seq        uint64
	sessionIdx int
	paneIdx    int
}

func (m *Manager) snapshotSessions() ([]SessionSnapshot, []panePreviewRef, []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].CreatedAt.Equal(sessions[j].CreatedAt) {
			return sessions[i].Name < sessions[j].Name
		}
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})

	out := make([]SessionSnapshot, len(sessions))
	paneRefs := make([]panePreviewRef, 0, len(sessions)*2)
	paneIDs := make([]string, 0, len(sessions)*2)
	for si, session := range sessions {
		out[si] = SessionSnapshot{
			Name:       session.Name,
			Path:       session.Path,
			LayoutName: session.LayoutName,
			CreatedAt:  session.CreatedAt,
			Env:        append([]string(nil), session.Env...),
		}
		panes := append([]*Pane(nil), session.Panes...)
		sortPanesByIndex(panes)
		paneTitles := resolveSessionPaneTitles(session)
		out[si].Panes = make([]PaneSnapshot, len(panes))
		for pi, pane := range panes {
			title := paneSnapshotTitle(pane, paneTitles)
			cwd := ""
			bytesIn := uint64(0)
			bytesOut := uint64(0)
			if pane.window != nil {
				cwd = pane.window.Cwd()
				bytesIn = pane.window.BytesIn()
				bytesOut = pane.window.BytesOut()
			}
			out[si].Panes[pi] = PaneSnapshot{
				ID:            pane.ID,
				Index:         pane.Index,
				Title:         title,
				Command:       pane.Command,
				StartCommand:  pane.StartCommand,
				Tool:          pane.Tool,
				PID:           pane.PID,
				Active:        pane.Active,
				Left:          pane.Left,
				Top:           pane.Top,
				Width:         pane.Width,
				Height:        pane.Height,
				Dead:          pane.window != nil && pane.window.Dead(),
				DeadStatus:    pane.windowExitStatus(),
				LastActive:    pane.LastActiveAt(),
				RestoreFailed: pane.RestoreFailed,
				RestoreError:  pane.RestoreError,
				RestoreMode:   pane.RestoreMode,
				Cwd:           cwd,
				Tags:          append([]string(nil), pane.Tags...),
				BytesIn:       bytesIn,
				BytesOut:      bytesOut,
			}
			seq := uint64(0)
			if pane.window != nil {
				seq = pane.window.UpdateSeq()
			}
			paneRefs = append(paneRefs, panePreviewRef{
				id:         pane.ID,
				window:     pane.window,
				lastActive: pane.LastActiveAt(),
				seq:        seq,
				sessionIdx: si,
				paneIdx:    pi,
			})
			paneIDs = append(paneIDs, pane.ID)
		}
	}
	return out, paneRefs, paneIDs
}

func paneSnapshotTitle(pane *Pane, paneTitles map[*Pane]string) string {
	title := strings.TrimSpace(paneTitles[pane])
	if title == "" {
		title = strings.TrimSpace(pane.Title)
	}
	if title == "" && pane.window != nil {
		title = strings.TrimSpace(pane.window.Title())
	}
	return title
}

func previewNeedsUpdate(refs []panePreviewRef, states map[string]previewState, previewLines int) []bool {
	needsUpdate := make([]bool, len(refs))
	for i, ref := range refs {
		if ref.window == nil {
			continue
		}
		state, ok := states[ref.id]
		if !ok || len(state.lines) < previewLines || state.sourceSeq < ref.seq || state.dirty {
			needsUpdate[i] = true
		}
	}
	return needsUpdate
}

func (m *Manager) collectPreviewUpdates(
	ctx context.Context,
	previewLines int,
	refs []panePreviewRef,
	states map[string]previewState,
	needsUpdate []bool,
	cursor int,
) (map[string]previewState, int) {
	start := cursor
	if start < 0 || start >= len(refs) {
		start = 0
	}
	updates := make(map[string]previewState)
	lastProcessed := -1
	stopper := newPreviewUpdateStopper(ctx)
	for i := 0; i < len(refs); i++ {
		idx := (start + i) % len(refs)
		if !needsUpdate[idx] {
			continue
		}
		if stopper.shouldStop() {
			break
		}
		ref := refs[idx]
		if ref.window == nil {
			continue
		}
		lines, ready := renderPreviewLines(ref.window, previewLines)
		if len(lines) == 0 {
			lastProcessed = idx
			continue
		}
		state := previewState{lines: append([]string(nil), lines...), sourceSeq: ref.seq, dirty: !ready}
		states[ref.id] = state
		updates[ref.id] = state
		lastProcessed = idx
	}

	nextCursor := -1
	if lastProcessed >= 0 {
		nextCursor = lastProcessed + 1
		if nextCursor >= len(refs) {
			nextCursor = 0
		}
	}
	return updates, nextCursor
}

type previewUpdateStopper struct {
	ctx            context.Context
	deadline       time.Time
	hasDeadline    bool
	budgetDeadline time.Time
}

func newPreviewUpdateStopper(ctx context.Context) previewUpdateStopper {
	deadline, hasDeadline := ctx.Deadline()
	budgetDeadline := time.Time{}
	if previewUpdateBudget > 0 {
		budgetDeadline = time.Now().Add(previewUpdateBudget)
		if hasDeadline && deadline.Before(budgetDeadline) {
			budgetDeadline = deadline
		}
	} else if hasDeadline {
		budgetDeadline = deadline
	}
	return previewUpdateStopper{
		ctx:            ctx,
		deadline:       deadline,
		hasDeadline:    hasDeadline,
		budgetDeadline: budgetDeadline,
	}
}

func (s previewUpdateStopper) shouldStop() bool {
	if s.ctx.Err() != nil {
		return true
	}
	if s.hasDeadline && time.Now().After(s.deadline) {
		return true
	}
	if !s.budgetDeadline.IsZero() && time.Now().After(s.budgetDeadline) {
		return true
	}
	return false
}

func applyPreviewLines(out []SessionSnapshot, refs []panePreviewRef, states map[string]previewState, previewLines int) {
	for _, ref := range refs {
		state, ok := states[ref.id]
		if !ok || len(state.lines) == 0 {
			continue
		}
		lines := state.lines
		if len(lines) > previewLines {
			lines = lines[len(lines)-previewLines:]
		}
		out[ref.sessionIdx].Panes[ref.paneIdx].Preview = append([]string(nil), lines...)
	}
}

// SessionSnapshot describes a read-only view of a session.
type SessionSnapshot struct {
	Name       string
	Path       string
	LayoutName string
	Panes      []PaneSnapshot
	CreatedAt  time.Time
	Env        []string
}

// PaneSnapshot describes a pane snapshot.
type PaneSnapshot struct {
	ID            string
	Index         string
	Title         string
	Command       string
	StartCommand  string
	Cwd           string
	Tool          string
	PID           int
	Active        bool
	Left          int
	Top           int
	Width         int
	Height        int
	Dead          bool
	DeadStatus    int
	LastActive    time.Time
	Preview       []string
	RestoreFailed bool
	RestoreError  string
	RestoreMode   sessionrestore.Mode
	Disconnected  bool
	SnapshotAt    time.Time
	Tags          []string
	BytesIn       uint64
	BytesOut      uint64
}
