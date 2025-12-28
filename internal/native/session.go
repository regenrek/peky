package native

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
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
}

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

	for _, pane := range session.Panes {
		if pane.window != nil {
			_ = pane.window.Close()
		}
	}
	for _, id := range paneIDs {
		m.notify(id)
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
	if strings.TrimSpace(spec.Name) == "" {
		return nil, errors.New("native: session name is required")
	}
	if spec.Layout == nil {
		return nil, errors.New("native: layout is required")
	}
	if strings.TrimSpace(spec.Path) != "" {
		if err := validatePath(spec.Path); err != nil {
			return nil, err
		}
		spec.Path = filepath.Clean(spec.Path)
	}

	session := &Session{
		Name:       spec.Name,
		Path:       spec.Path,
		LayoutName: strings.TrimSpace(spec.LayoutName),
		CreatedAt:  time.Now(),
	}
	if session.LayoutName == "" && spec.Layout != nil {
		session.LayoutName = spec.Layout.Name
	}

	panes, err := m.buildPanes(ctx, spec)
	if err != nil {
		m.closePanes(panes)
		return nil, err
	}
	session.Panes = panes

	m.mu.Lock()
	if _, ok := m.sessions[session.Name]; ok {
		m.mu.Unlock()
		m.closePanes(panes)
		return nil, fmt.Errorf("native: session %q already exists", session.Name)
	}
	m.sessions[session.Name] = session
	for _, pane := range panes {
		m.panes[pane.ID] = pane
	}
	m.mu.Unlock()

	for _, pane := range panes {
		m.forwardUpdates(pane)
	}

	// Seed an update to render initial output.
	for _, pane := range panes {
		m.notify(pane.ID)
	}
	m.version.Add(1)

	return session, nil
}

// Snapshot returns a copy of sessions with preview lines computed.
func (m *Manager) Snapshot(previewLines int) []SessionSnapshot {
	if m == nil {
		return nil
	}
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

	out := make([]SessionSnapshot, 0, len(m.sessions))
	for _, session := range sessions {
		snap := SessionSnapshot{
			Name:       session.Name,
			Path:       session.Path,
			LayoutName: session.LayoutName,
			CreatedAt:  session.CreatedAt,
		}
		panes := append([]*Pane(nil), session.Panes...)
		sortPanesByIndex(panes)
		paneTitles := resolveSessionPaneTitles(session)
		for _, pane := range panes {
			lines := renderPreviewLines(pane.window, previewLines)
			title := strings.TrimSpace(paneTitles[pane])
			if title == "" {
				title = strings.TrimSpace(pane.Title)
			}
			if title == "" && pane.window != nil {
				title = strings.TrimSpace(pane.window.Title())
			}
			paneSnap := PaneSnapshot{
				ID:           pane.ID,
				Index:        pane.Index,
				Title:        title,
				Command:      pane.Command,
				StartCommand: pane.StartCommand,
				PID:          pane.PID,
				Active:       pane.Active,
				Left:         pane.Left,
				Top:          pane.Top,
				Width:        pane.Width,
				Height:       pane.Height,
				Dead:         pane.window != nil && pane.window.Exited(),
				DeadStatus:   pane.windowExitStatus(),
				LastActive:   pane.LastActive,
				Preview:      lines,
			}
			snap.Panes = append(snap.Panes, paneSnap)
		}
		out = append(out, snap)
	}
	return out
}

// SessionSnapshot describes a read-only view of a session.
type SessionSnapshot struct {
	Name       string
	Path       string
	LayoutName string
	Panes      []PaneSnapshot
	CreatedAt  time.Time
}

// PaneSnapshot describes a pane snapshot.
type PaneSnapshot struct {
	ID           string
	Index        string
	Title        string
	Command      string
	StartCommand string
	PID          int
	Active       bool
	Left         int
	Top          int
	Width        int
	Height       int
	Dead         bool
	DeadStatus   int
	LastActive   time.Time
	Preview      []string
}
