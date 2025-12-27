package native

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/terminal"
)

const layoutBaseSize = 1000

// PaneEvent signals that a pane updated.
type PaneEvent struct {
	PaneID string
}

// SessionSpec describes a session to start.
type SessionSpec struct {
	Name       string
	Path       string
	Layout     *layout.LayoutConfig
	LayoutName string
	Env        []string
}

// Manager owns native sessions and panes.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	panes    map[string]*Pane
	events   chan PaneEvent
	nextID   atomic.Uint64
	closed   atomic.Bool
}

// Session is a native session container.
type Session struct {
	Name       string
	Path       string
	LayoutName string
	Windows    []*Window
	CreatedAt  time.Time
}

// Window groups panes for a layout window.
type Window struct {
	Index string
	Name  string
	Panes []*Pane
}

// Pane represents a running terminal pane.
type Pane struct {
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
	window       *terminal.Window
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

// Close stops all sessions and releases resources.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	if m.closed.Swap(true) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, session := range m.sessions {
		for _, window := range session.Windows {
			for _, pane := range window.Panes {
				if pane.window != nil {
					_ = pane.window.Close()
				}
			}
		}
	}
	close(m.events)
	m.sessions = nil
	m.panes = nil
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
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		names = append(names, name)
	}
	return names
}

// Window returns the terminal window for a pane ID.
func (m *Manager) Window(id string) *terminal.Window {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if pane := m.panes[id]; pane != nil {
		return pane.window
	}
	return nil
}

// PaneWindow returns the terminal window for a pane ID.
func (m *Manager) PaneWindow(paneID string) *terminal.Window {
	return m.Window(paneID)
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
	for _, window := range session.Windows {
		for _, pane := range window.Panes {
			paneIDs = append(paneIDs, pane.ID)
			delete(m.panes, pane.ID)
		}
	}
	m.mu.Unlock()

	for _, window := range session.Windows {
		for _, pane := range window.Panes {
			if pane.window != nil {
				_ = pane.window.Close()
			}
		}
	}
	for _, id := range paneIDs {
		m.notify(id)
	}
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
	return nil
}

// RenameWindow updates a window title in a session.
func (m *Manager) RenameWindow(sessionName, windowIndex, newName string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	newName = strings.TrimSpace(newName)
	if sessionName == "" || windowIndex == "" || newName == "" {
		return errors.New("native: session/window name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionName]
	if !ok {
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	for _, window := range session.Windows {
		if window.Index == windowIndex {
			window.Name = newName
			return nil
		}
	}
	return fmt.Errorf("native: window %q not found in %q", windowIndex, sessionName)
}

// RenamePane updates a pane title.
func (m *Manager) RenamePane(sessionName, windowIndex, paneIndex, newTitle string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	paneIndex = strings.TrimSpace(paneIndex)
	newTitle = strings.TrimSpace(newTitle)
	if sessionName == "" || windowIndex == "" || paneIndex == "" || newTitle == "" {
		return errors.New("native: pane name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionName]
	if !ok {
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	for _, window := range session.Windows {
		if window.Index != windowIndex {
			continue
		}
		for _, pane := range window.Panes {
			if pane.Index == paneIndex {
				pane.Title = newTitle
				if pane.window != nil {
					pane.window.SetTitle(newTitle)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
}

// ClosePane removes a pane from a window and reflows remaining panes.
func (m *Manager) ClosePane(ctx context.Context, sessionName, windowIndex, paneIndex string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || windowIndex == "" || paneIndex == "" {
		return errors.New("native: session, window, and pane are required")
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	var paneWindow *terminal.Window
	var notifyIDs []string

	m.mu.Lock()
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	window := findWindowByIndex(session, windowIndex)
	if window == nil {
		m.mu.Unlock()
		return fmt.Errorf("native: window %q not found in %q", windowIndex, sessionName)
	}
	paneIdx := -1
	var pane *Pane
	for i, candidate := range window.Panes {
		if candidate.Index == paneIndex {
			paneIdx = i
			pane = candidate
			break
		}
	}
	if pane == nil {
		m.mu.Unlock()
		return fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}

	window.Panes = append(window.Panes[:paneIdx], window.Panes[paneIdx+1:]...)
	delete(m.panes, pane.ID)
	paneWindow = pane.window

	if len(window.Panes) == 0 {
		removeWindowByIndex(session, windowIndex)
		m.mu.Unlock()
		if paneWindow != nil {
			_ = paneWindow.Close()
		}
		m.notify(pane.ID)
		return nil
	}
	if !anyPaneActive(window) {
		window.Panes[0].Active = true
	}
	retileWindow(window)
	for _, remaining := range window.Panes {
		notifyIDs = append(notifyIDs, remaining.ID)
	}
	m.mu.Unlock()

	if paneWindow != nil {
		_ = paneWindow.Close()
	}
	m.notify(pane.ID)
	for _, id := range notifyIDs {
		m.notify(id)
	}
	return nil
}

// SplitPane splits a pane inside a window and returns the new pane index.
func (m *Manager) SplitPane(ctx context.Context, sessionName, windowIndex, paneIndex string, vertical bool, percent int) (string, error) {
	if m == nil {
		return "", errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || windowIndex == "" || paneIndex == "" {
		return "", errors.New("native: session, window, and pane are required")
	}

	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.RUnlock()
		return "", fmt.Errorf("native: session %q not found", sessionName)
	}
	window := findWindowByIndex(session, windowIndex)
	if window == nil {
		m.mu.RUnlock()
		return "", fmt.Errorf("native: window %q not found in %q", windowIndex, sessionName)
	}
	target := findPaneByIndex(window, paneIndex)
	if target == nil {
		m.mu.RUnlock()
		return "", fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	newIndex := nextPaneIndex(window)
	startDir := strings.TrimSpace(session.Path)
	m.mu.RUnlock()

	if strings.TrimSpace(startDir) != "" {
		if err := validatePath(startDir); err != nil {
			return "", err
		}
	}
	pane, err := m.createPane(ctx, startDir, "", "", nil)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	session, ok = m.sessions[sessionName]
	if !ok {
		m.mu.Unlock()
		_ = pane.window.Close()
		return "", fmt.Errorf("native: session %q not found", sessionName)
	}
	window = findWindowByIndex(session, windowIndex)
	if window == nil {
		m.mu.Unlock()
		_ = pane.window.Close()
		return "", fmt.Errorf("native: window %q not found in %q", windowIndex, sessionName)
	}
	target = findPaneByIndex(window, paneIndex)
	if target == nil {
		m.mu.Unlock()
		_ = pane.window.Close()
		return "", fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	oldRect, newRect := splitRect(rectFromPane(target), vertical, percent)
	target.Left, target.Top, target.Width, target.Height = oldRect.x, oldRect.y, oldRect.w, oldRect.h

	for _, existing := range window.Panes {
		existing.Active = false
	}
	pane.Index = newIndex
	pane.Active = true
	pane.Left, pane.Top, pane.Width, pane.Height = newRect.x, newRect.y, newRect.w, newRect.h
	pane.LastActive = time.Now()
	window.Panes = append(window.Panes, pane)
	m.panes[pane.ID] = pane
	m.mu.Unlock()

	m.forwardUpdates(pane)
	m.notify(pane.ID)
	return pane.Index, nil
}

// MovePaneToNewWindow moves a pane into a new window and returns the new window and pane indexes.
func (m *Manager) MovePaneToNewWindow(ctx context.Context, sessionName, windowIndex, paneIndex string) (string, string, error) {
	if m == nil {
		return "", "", errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || windowIndex == "" || paneIndex == "" {
		return "", "", errors.New("native: session, window, and pane are required")
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		default:
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionName]
	if !ok {
		return "", "", fmt.Errorf("native: session %q not found", sessionName)
	}
	window := findWindowByIndex(session, windowIndex)
	if window == nil {
		return "", "", fmt.Errorf("native: window %q not found in %q", windowIndex, sessionName)
	}
	paneIdx := -1
	var pane *Pane
	for i, candidate := range window.Panes {
		if candidate.Index == paneIndex {
			paneIdx = i
			pane = candidate
			break
		}
	}
	if pane == nil {
		return "", "", fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}

	window.Panes = append(window.Panes[:paneIdx], window.Panes[paneIdx+1:]...)
	var notifyIDs []string
	if len(window.Panes) == 0 {
		removeWindowByIndex(session, windowIndex)
	} else {
		if !anyPaneActive(window) {
			window.Panes[0].Active = true
		}
		retileWindow(window)
		for _, remaining := range window.Panes {
			notifyIDs = append(notifyIDs, remaining.ID)
		}
	}

	windowName := strings.TrimSpace(pane.Title)
	if windowName == "" {
		windowName = "window"
	}
	newWindowIndex := nextWindowIndex(session)
	newWindow := &Window{
		Index: newWindowIndex,
		Name:  windowName,
		Panes: []*Pane{pane},
	}
	pane.Index = "0"
	pane.Active = true
	pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, layoutBaseSize, layoutBaseSize
	pane.LastActive = time.Now()
	session.Windows = append(session.Windows, newWindow)
	m.notify(pane.ID)
	for _, id := range notifyIDs {
		m.notify(id)
	}

	return newWindowIndex, pane.Index, nil
}

// SwapPanes swaps two panes within the same window.
func (m *Manager) SwapPanes(sessionName, windowIndex, paneA, paneB string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	paneA = strings.TrimSpace(paneA)
	paneB = strings.TrimSpace(paneB)
	if sessionName == "" || windowIndex == "" || paneA == "" || paneB == "" {
		return errors.New("native: session, window, and panes are required")
	}
	if paneA == paneB {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionName]
	if !ok {
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	window := findWindowByIndex(session, windowIndex)
	if window == nil {
		return fmt.Errorf("native: window %q not found in %q", windowIndex, sessionName)
	}
	first := findPaneByIndex(window, paneA)
	second := findPaneByIndex(window, paneB)
	if first == nil || second == nil {
		return fmt.Errorf("native: panes %q and %q not found in %q", paneA, paneB, sessionName)
	}

	firstRect := rectFromPane(first)
	secondRect := rectFromPane(second)
	first.Left, first.Top, first.Width, first.Height = secondRect.x, secondRect.y, secondRect.w, secondRect.h
	second.Left, second.Top, second.Width, second.Height = firstRect.x, firstRect.y, firstRect.w, firstRect.h
	first.Index, second.Index = second.Index, first.Index

	sortPanesByIndex(window)
	m.notify(first.ID)
	m.notify(second.ID)
	return nil
}

// NewWindow creates a new window with a single shell pane.
func (m *Manager) NewWindow(ctx context.Context, sessionName, windowName, path string) (*Window, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return nil, errors.New("native: session name is required")
	}
	windowName = strings.TrimSpace(windowName)
	if windowName == "" {
		windowName = "window"
	}
	if strings.TrimSpace(path) != "" {
		if err := validatePath(path); err != nil {
			return nil, err
		}
		path = filepath.Clean(path)
	}

	pane, err := m.createPane(ctx, path, windowName, "", nil)
	if err != nil {
		return nil, err
	}
	pane.Index = "0"
	pane.Active = true
	pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, layoutBaseSize, layoutBaseSize

	m.mu.Lock()
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.Unlock()
		_ = pane.window.Close()
		return nil, fmt.Errorf("native: session %q not found", sessionName)
	}
	win := &Window{
		Index: strconv.Itoa(len(session.Windows)),
		Name:  windowName,
		Panes: []*Pane{pane},
	}
	session.Windows = append(session.Windows, win)
	m.panes[pane.ID] = pane
	m.mu.Unlock()
	m.forwardUpdates(pane)
	m.notify(pane.ID)
	return win, nil
}

// SendInput writes input to a pane and updates activity timestamp.
func (m *Manager) SendInput(id string, input []byte) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	m.mu.RLock()
	pane := m.panes[id]
	m.mu.RUnlock()
	if pane == nil || pane.window == nil {
		return fmt.Errorf("native: pane %q not found", id)
	}
	if err := pane.window.SendInput(input); err != nil {
		return err
	}
	m.markActive(id)
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

	windows, panes, err := m.buildWindows(ctx, spec, session)
	if err != nil {
		m.closePanes(panes)
		return nil, err
	}
	session.Windows = windows

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

	return session, nil
}

// Snapshot returns a copy of sessions with preview lines computed.
func (m *Manager) Snapshot(previewLines int) []SessionSnapshot {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]SessionSnapshot, 0, len(m.sessions))
	for _, session := range m.sessions {
		snap := SessionSnapshot{
			Name:       session.Name,
			Path:       session.Path,
			LayoutName: session.LayoutName,
			CreatedAt:  session.CreatedAt,
		}
		for _, window := range session.Windows {
			winSnap := WindowSnapshot{
				Index: window.Index,
				Name:  window.Name,
			}
			for _, pane := range window.Panes {
				lines := renderPreviewLines(pane.window, previewLines)
				title := pane.Title
				if pane.window != nil {
					if winTitle := strings.TrimSpace(pane.window.Title()); winTitle != "" {
						title = winTitle
					}
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
				winSnap.Panes = append(winSnap.Panes, paneSnap)
			}
			snap.Windows = append(snap.Windows, winSnap)
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
	Windows    []WindowSnapshot
	CreatedAt  time.Time
}

// WindowSnapshot describes a window snapshot.
type WindowSnapshot struct {
	Index string
	Name  string
	Panes []PaneSnapshot
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

func (m *Manager) buildWindows(ctx context.Context, spec SessionSpec, session *Session) ([]*Window, []*Pane, error) {
	if spec.Layout == nil {
		return nil, nil, errors.New("native: layout is nil")
	}
	layoutCfg := spec.Layout
	if strings.TrimSpace(layoutCfg.Grid) != "" {
		windowName := strings.TrimSpace(layoutCfg.Window)
		if windowName == "" {
			windowName = strings.TrimSpace(layoutCfg.Name)
		}
		if windowName == "" {
			windowName = "grid"
		}
		win, panes, err := m.buildGridWindow(ctx, spec.Path, windowName, layoutCfg, spec.Env)
		if err != nil {
			return nil, nil, err
		}
		return []*Window{win}, panes, nil
	}
	if len(layoutCfg.Windows) == 0 {
		return nil, nil, errors.New("native: layout has no windows defined")
	}
	var windows []*Window
	var panes []*Pane
	for wi, winDef := range layoutCfg.Windows {
		windowName := strings.TrimSpace(winDef.Name)
		if windowName == "" {
			windowName = fmt.Sprintf("window-%d", wi+1)
		}
		win, winPanes, err := m.buildSplitWindow(ctx, spec.Path, windowName, winDef, spec.Env)
		if err != nil {
			return nil, panes, err
		}
		win.Index = strconv.Itoa(wi)
		windows = append(windows, win)
		panes = append(panes, winPanes...)
	}
	return windows, panes, nil
}

func (m *Manager) buildGridWindow(ctx context.Context, path, name string, layoutCfg *layout.LayoutConfig, env []string) (*Window, []*Pane, error) {
	grid, err := layout.Parse(layoutCfg.Grid)
	if err != nil {
		return nil, nil, fmt.Errorf("native: parse grid %q: %w", layoutCfg.Grid, err)
	}
	commands := layout.ResolveGridCommands(layoutCfg, grid.Panes())
	titles := layout.ResolveGridTitles(layoutCfg, grid.Panes())
	win := &Window{Index: "0", Name: name}

	cellW := layoutBaseSize / grid.Columns
	cellH := layoutBaseSize / grid.Rows
	remainderW := layoutBaseSize % grid.Columns
	remainderH := layoutBaseSize % grid.Rows

	panes := make([]*Pane, 0, grid.Panes())
	for r := 0; r < grid.Rows; r++ {
		for c := 0; c < grid.Columns; c++ {
			idx := r*grid.Columns + c
			title := ""
			if idx < len(titles) {
				title = titles[idx]
			}
			cmd := ""
			if idx < len(commands) {
				cmd = commands[idx]
			}
			left := c * cellW
			top := r * cellH
			width := cellW
			height := cellH
			if c == grid.Columns-1 {
				width += remainderW
			}
			if r == grid.Rows-1 {
				height += remainderH
			}
			pane, err := m.createPane(ctx, path, title, cmd, env)
			if err != nil {
				m.closePanes(panes)
				return nil, nil, err
			}
			pane.Index = strconv.Itoa(idx)
			pane.Left = left
			pane.Top = top
			pane.Width = width
			pane.Height = height
			if idx == 0 {
				pane.Active = true
			}
			panes = append(panes, pane)
			win.Panes = append(win.Panes, pane)
		}
	}
	return win, panes, nil
}

func (m *Manager) buildSplitWindow(ctx context.Context, path, name string, def layout.WindowDef, env []string) (*Window, []*Pane, error) {
	win := &Window{Name: name}
	if len(def.Panes) == 0 {
		pane, err := m.createPane(ctx, path, name, "", env)
		if err != nil {
			return nil, nil, err
		}
		pane.Index = "0"
		pane.Active = true
		pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, layoutBaseSize, layoutBaseSize
		win.Panes = []*Pane{pane}
		return win, []*Pane{pane}, nil
	}

	var panes []*Pane
	active := (*Pane)(nil)
	for i, paneDef := range def.Panes {
		pane, err := m.createPane(ctx, path, paneDef.Title, paneDef.Cmd, env)
		if err != nil {
			m.closePanes(panes)
			return nil, nil, err
		}
		pane.Index = strconv.Itoa(i)
		if i == 0 {
			pane.Active = true
			pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, layoutBaseSize, layoutBaseSize
			active = pane
		} else if active != nil {
			vertical := strings.EqualFold(paneDef.Split, "vertical") || strings.EqualFold(paneDef.Split, "v")
			percent := parsePercent(paneDef.Size)
			oldRect, newRect := splitRect(rectFromPane(active), vertical, percent)
			active.Left, active.Top, active.Width, active.Height = oldRect.x, oldRect.y, oldRect.w, oldRect.h
			pane.Left, pane.Top, pane.Width, pane.Height = newRect.x, newRect.y, newRect.w, newRect.h
		} else {
			pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, layoutBaseSize, layoutBaseSize
		}
		panes = append(panes, pane)
		win.Panes = append(win.Panes, pane)
	}
	return win, panes, nil
}

func (m *Manager) createPane(ctx context.Context, path, title, command string, env []string) (*Pane, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	id := m.nextPaneID()
	opts := terminal.Options{
		ID:    id,
		Title: strings.TrimSpace(title),
		Dir:   strings.TrimSpace(path),
		Env:   env,
	}
	startCommand := strings.TrimSpace(command)
	if startCommand == "" {
		opts.Command = ""
	} else {
		cmd, args, err := splitCommand(startCommand)
		if err != nil {
			return nil, fmt.Errorf("native: parse command %q: %w", startCommand, err)
		}
		opts.Command = cmd
		opts.Args = args
	}
	win, err := terminal.NewWindow(opts)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			_ = win.Close()
			return nil, ctx.Err()
		default:
		}
	}
	pane := &Pane{
		ID:           id,
		Title:        strings.TrimSpace(title),
		Command:      startCommand,
		StartCommand: startCommand,
		window:       win,
		LastActive:   time.Now(),
	}
	if win != nil && win.Exited() {
		pane.Dead = true
		pane.DeadStatus = win.ExitStatus()
	}
	if win != nil {
		pane.PID = win.PID()
	}
	return pane, nil
}

func (m *Manager) forwardUpdates(pane *Pane) {
	if pane == nil || pane.window == nil {
		return
	}
	updates := pane.window.Updates()
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

func (p *Pane) windowExitStatus() int {
	if p == nil || p.window == nil {
		return 0
	}
	return p.window.ExitStatus()
}

func renderPreviewLines(win *terminal.Window, max int) []string {
	if win == nil || max <= 0 {
		return nil
	}
	view := win.ViewANSI()
	if view == "" {
		return nil
	}
	plain := ansi.Strip(view)
	lines := strings.Split(plain, "\n")
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func splitCommand(command string) (string, []string, error) {
	parts, err := shellquote.Split(command)
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 {
		return "", nil, errors.New("empty command")
	}
	return parts[0], parts[1:], nil
}

func validatePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", clean)
	}
	return nil
}

func findWindowByIndex(session *Session, index string) *Window {
	if session == nil {
		return nil
	}
	index = strings.TrimSpace(index)
	for _, window := range session.Windows {
		if window.Index == index {
			return window
		}
	}
	return nil
}

func findPaneByIndex(window *Window, index string) *Pane {
	if window == nil {
		return nil
	}
	index = strings.TrimSpace(index)
	for _, pane := range window.Panes {
		if pane.Index == index {
			return pane
		}
	}
	return nil
}

func nextPaneIndex(window *Window) string {
	max := -1
	for _, pane := range window.Panes {
		if n, err := strconv.Atoi(strings.TrimSpace(pane.Index)); err == nil && n > max {
			max = n
		}
	}
	if max < 0 {
		return "0"
	}
	return strconv.Itoa(max + 1)
}

func nextWindowIndex(session *Session) string {
	max := -1
	for _, window := range session.Windows {
		if n, err := strconv.Atoi(strings.TrimSpace(window.Index)); err == nil && n > max {
			max = n
		}
	}
	if max < 0 {
		return "0"
	}
	return strconv.Itoa(max + 1)
}

func removeWindowByIndex(session *Session, index string) {
	if session == nil {
		return
	}
	for i, window := range session.Windows {
		if window.Index == index {
			session.Windows = append(session.Windows[:i], session.Windows[i+1:]...)
			return
		}
	}
}

func anyPaneActive(window *Window) bool {
	if window == nil {
		return false
	}
	for _, pane := range window.Panes {
		if pane.Active {
			return true
		}
	}
	return false
}

func sortPanesByIndex(window *Window) {
	if window == nil {
		return
	}
	sort.SliceStable(window.Panes, func(i, j int) bool {
		left := strings.TrimSpace(window.Panes[i].Index)
		right := strings.TrimSpace(window.Panes[j].Index)
		li, lerr := strconv.Atoi(left)
		ri, rerr := strconv.Atoi(right)
		if lerr == nil && rerr == nil {
			return li < ri
		}
		if lerr == nil {
			return true
		}
		if rerr == nil {
			return false
		}
		return left < right
	})
}

func retileWindow(window *Window) {
	if window == nil {
		return
	}
	if len(window.Panes) == 0 {
		return
	}
	sortPanesByIndex(window)

	count := len(window.Panes)
	cols := int(math.Ceil(math.Sqrt(float64(count))))
	if cols <= 0 {
		cols = 1
	}
	rows := int(math.Ceil(float64(count) / float64(cols)))
	if rows <= 0 {
		rows = 1
	}

	cellW := layoutBaseSize / cols
	cellH := layoutBaseSize / rows
	remainderW := layoutBaseSize % cols
	remainderH := layoutBaseSize % rows

	for i, pane := range window.Panes {
		row := i / cols
		col := i % cols
		left := col * cellW
		top := row * cellH
		width := cellW
		height := cellH
		if col == cols-1 {
			width += remainderW
		}
		if row == rows-1 {
			height += remainderH
		}
		pane.Left = left
		pane.Top = top
		pane.Width = width
		pane.Height = height
	}
}

type rect struct {
	x int
	y int
	w int
	h int
}

func rectFromPane(p *Pane) rect {
	if p == nil {
		return rect{x: 0, y: 0, w: layoutBaseSize, h: layoutBaseSize}
	}
	return rect{x: p.Left, y: p.Top, w: p.Width, h: p.Height}
}

func splitRect(r rect, vertical bool, percent int) (rect, rect) {
	if percent <= 0 || percent >= 100 {
		percent = 50
	}
	if vertical {
		newH := r.h * percent / 100
		if newH <= 0 || newH >= r.h {
			newH = r.h / 2
		}
		oldH := r.h - newH
		oldRect := rect{x: r.x, y: r.y, w: r.w, h: oldH}
		newRect := rect{x: r.x, y: r.y + oldH, w: r.w, h: newH}
		return oldRect, newRect
	}
	newW := r.w * percent / 100
	if newW <= 0 || newW >= r.w {
		newW = r.w / 2
	}
	oldW := r.w - newW
	oldRect := rect{x: r.x, y: r.y, w: oldW, h: r.h}
	newRect := rect{x: r.x + oldW, y: r.y, w: newW, h: r.h}
	return oldRect, newRect
}

func parsePercent(size string) int {
	size = strings.TrimSpace(size)
	if size == "" {
		return 0
	}
	size = strings.TrimSuffix(size, "%")
	pct, err := strconv.Atoi(size)
	if err != nil {
		return 0
	}
	return pct
}
