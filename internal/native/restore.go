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
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return nil, errors.New("native: session name is required")
	}
	path := strings.TrimSpace(spec.Path)
	if path != "" {
		if err := validatePath(path); err != nil {
			return nil, err
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
	activeIndex := ""
	for _, pane := range panesSpec {
		if pane.Active {
			activeIndex = pane.Index
			break
		}
	}
	if activeIndex == "" {
		activeIndex = panesSpec[0].Index
	}
	env := append([]string(nil), spec.Env...)
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
	for _, pane := range panes {
		m.notifyPane(pane.ID)
	}
	m.version.Add(1)
	return session, nil
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
