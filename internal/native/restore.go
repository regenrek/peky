package native

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// PaneRestoreSpec describes a pane to restore.
type PaneRestoreSpec struct {
	Index        string
	Title        string
	Command      string
	StartCommand string
	Active       bool
	Left         int
	Top          int
	Width        int
	Height       int

	RestoreFailed bool
	RestoreError  string
}

// SessionRestoreSpec describes a session to restore.
type SessionRestoreSpec struct {
	Name       string
	Path       string
	LayoutName string
	CreatedAt  time.Time
	Env        []string
	Panes      []PaneRestoreSpec
}

// RestoreSession recreates a session from persisted state.
func (m *Manager) RestoreSession(ctx context.Context, spec SessionRestoreSpec) (*Session, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	if m.closed.Load() {
		return nil, errors.New("native: manager closed")
	}
	name, path, panesSpec, env, err := normalizeRestoreSpec(spec)
	if err != nil {
		return nil, err
	}
	activeIndex := activePaneIndex(panesSpec)
	panes, err := m.restorePanes(ctx, path, env, panesSpec, activeIndex)
	if err != nil {
		return nil, err
	}
	session := buildRestoredSession(spec, name, path, env, panes)
	if err := m.commitRestoredSession(session, panes); err != nil {
		return nil, err
	}
	m.finishRestoredSession(panes)
	return session, nil
}

func normalizeRestoreSpec(spec SessionRestoreSpec) (string, string, []PaneRestoreSpec, []string, error) {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return "", "", nil, nil, errors.New("native: session name is required")
	}
	path := strings.TrimSpace(spec.Path)
	if path != "" {
		if err := validatePath(path); err != nil {
			return "", "", nil, nil, err
		}
		path = filepath.Clean(path)
	}
	panesSpec := spec.Panes
	if len(panesSpec) == 0 {
		panesSpec = []PaneRestoreSpec{{
			Index:         "0",
			Active:        true,
			RestoreFailed: true,
			RestoreError:  "no panes to restore",
		}}
	}
	env := append([]string(nil), spec.Env...)
	return name, path, panesSpec, env, nil
}

func activePaneIndex(panesSpec []PaneRestoreSpec) string {
	for _, pane := range panesSpec {
		if pane.Active {
			return pane.Index
		}
	}
	if len(panesSpec) == 0 {
		return ""
	}
	return panesSpec[0].Index
}

func (m *Manager) restorePanes(ctx context.Context, path string, env []string, panesSpec []PaneRestoreSpec, activeIndex string) ([]*Pane, error) {
	panes := make([]*Pane, 0, len(panesSpec))
	for _, paneSpec := range panesSpec {
		active := paneSpec.Index == activeIndex
		pane, err := m.restorePane(ctx, path, env, paneSpec, active)
		if err != nil {
			m.closePanes(panes)
			return nil, err
		}
		panes = append(panes, pane)
	}
	return panes, nil
}

func buildRestoredSession(spec SessionRestoreSpec, name, path string, env []string, panes []*Pane) *Session {
	session := &Session{
		Name:       name,
		Path:       path,
		LayoutName: strings.TrimSpace(spec.LayoutName),
		CreatedAt:  spec.CreatedAt,
		Env:        env,
		Panes:      panes,
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	return session
}

func (m *Manager) commitRestoredSession(session *Session, panes []*Pane) error {
	m.mu.Lock()
	if _, ok := m.sessions[session.Name]; ok {
		m.mu.Unlock()
		m.closePanes(panes)
		return fmt.Errorf("native: session %q already exists", session.Name)
	}
	m.sessions[session.Name] = session
	for _, pane := range panes {
		m.panes[pane.ID] = pane
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) finishRestoredSession(panes []*Pane) {
	for _, pane := range panes {
		m.forwardUpdates(pane)
	}
	for _, pane := range panes {
		m.notifyPane(pane.ID)
	}
	m.version.Add(1)
}

func (m *Manager) restorePane(ctx context.Context, path string, env []string, spec PaneRestoreSpec, active bool) (*Pane, error) {
	intended := strings.TrimSpace(spec.Command)
	startCmd := strings.TrimSpace(spec.StartCommand)
	if startCmd == "" {
		startCmd = intended
	}
	if intended == "" {
		intended = startCmd
	}
	title := strings.TrimSpace(spec.Title)
	pane, err := m.createPane(ctx, path, title, startCmd, env)
	if err != nil {
		fallback, fallbackErr := m.createPane(ctx, path, title, "", env)
		if fallbackErr != nil {
			return nil, fmt.Errorf("native: restore pane fallback: %w", fallbackErr)
		}
		fallback.RestoreFailed = true
		fallback.RestoreError = err.Error()
		fallback.Command = intended
		fallback.StartCommand = ""
		pane = fallback
	} else {
		pane.Command = intended
		pane.StartCommand = startCmd
	}
	pane.Index = strings.TrimSpace(spec.Index)
	pane.Active = active
	pane.Left = clampRestoreValue(spec.Left, 0, LayoutBaseSize)
	pane.Top = clampRestoreValue(spec.Top, 0, LayoutBaseSize)
	pane.Width = clampRestoreValue(spec.Width, 1, LayoutBaseSize)
	pane.Height = clampRestoreValue(spec.Height, 1, LayoutBaseSize)
	if active {
		pane.LastActive = time.Now()
	}
	return pane, nil
}

func clampRestoreValue(value, min, max int) int {
	if value < min {
		return min
	}
	if max > 0 && value > max {
		return max
	}
	return value
}
