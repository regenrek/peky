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

	uv "github.com/charmbracelet/ultraviolet"
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
	mu           sync.RWMutex
	eventsMu     sync.Mutex
	sessions     map[string]*Session
	panes        map[string]*Pane
	events       chan PaneEvent
	eventsClosed bool
	nextID       atomic.Uint64
	closed       atomic.Bool
}

// Session is a native session container.
type Session struct {
	Name       string
	Path       string
	LayoutName string
	Panes      []*Pane
	CreatedAt  time.Time
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

// RenamePane updates a pane title.
func (m *Manager) RenamePane(sessionName, paneIndex, newTitle string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	newTitle = strings.TrimSpace(newTitle)
	if sessionName == "" || paneIndex == "" || newTitle == "" {
		return errors.New("native: pane name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionName]
	if !ok {
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	for _, pane := range session.Panes {
		if pane.Index == paneIndex {
			pane.Title = newTitle
			if pane.window != nil {
				pane.window.SetTitle(newTitle)
			}
			return nil
		}
	}
	return fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
}

// ClosePane removes a pane from a session and reflows remaining panes.
func (m *Manager) ClosePane(ctx context.Context, sessionName, paneIndex string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || paneIndex == "" {
		return errors.New("native: session and pane are required")
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
	paneIdx := -1
	var pane *Pane
	for i, candidate := range session.Panes {
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

	session.Panes = append(session.Panes[:paneIdx], session.Panes[paneIdx+1:]...)
	delete(m.panes, pane.ID)
	paneWindow = pane.window

	if len(session.Panes) == 0 {
		m.mu.Unlock()
		if paneWindow != nil {
			_ = paneWindow.Close()
		}
		m.notify(pane.ID)
		return nil
	}
	if !anyPaneActive(session.Panes) {
		session.Panes[0].Active = true
	}
	retilePanes(session.Panes)
	for _, remaining := range session.Panes {
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

// SplitPane splits a pane inside a session and returns the new pane index.
func (m *Manager) SplitPane(ctx context.Context, sessionName, paneIndex string, vertical bool, percent int) (string, error) {
	if m == nil {
		return "", errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || paneIndex == "" {
		return "", errors.New("native: session and pane are required")
	}

	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.RUnlock()
		return "", fmt.Errorf("native: session %q not found", sessionName)
	}
	target := findPaneByIndex(session.Panes, paneIndex)
	if target == nil {
		m.mu.RUnlock()
		return "", fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	newIndex := nextPaneIndex(session.Panes)
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
	target = findPaneByIndex(session.Panes, paneIndex)
	if target == nil {
		m.mu.Unlock()
		_ = pane.window.Close()
		return "", fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	oldRect, newRect := splitRect(rectFromPane(target), vertical, percent)
	target.Left, target.Top, target.Width, target.Height = oldRect.x, oldRect.y, oldRect.w, oldRect.h

	for _, existing := range session.Panes {
		existing.Active = false
	}
	pane.Index = newIndex
	pane.Active = true
	pane.Left, pane.Top, pane.Width, pane.Height = newRect.x, newRect.y, newRect.w, newRect.h
	pane.LastActive = time.Now()
	session.Panes = append(session.Panes, pane)
	m.panes[pane.ID] = pane
	m.mu.Unlock()

	m.forwardUpdates(pane)
	m.notify(pane.ID)
	return pane.Index, nil
}

// SwapPanes swaps two panes within the same session.
func (m *Manager) SwapPanes(sessionName, paneA, paneB string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	sessionName = strings.TrimSpace(sessionName)
	paneA = strings.TrimSpace(paneA)
	paneB = strings.TrimSpace(paneB)
	if sessionName == "" || paneA == "" || paneB == "" {
		return errors.New("native: session and panes are required")
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
	first := findPaneByIndex(session.Panes, paneA)
	second := findPaneByIndex(session.Panes, paneB)
	if first == nil || second == nil {
		return fmt.Errorf("native: panes %q and %q not found in %q", paneA, paneB, sessionName)
	}

	firstRect := rectFromPane(first)
	secondRect := rectFromPane(second)
	first.Left, first.Top, first.Width, first.Height = secondRect.x, secondRect.y, secondRect.w, secondRect.h
	second.Left, second.Top, second.Width, second.Height = firstRect.x, firstRect.y, firstRect.w, firstRect.h
	first.Index, second.Index = second.Index, first.Index

	sortPanesByIndex(session.Panes)
	m.notify(first.ID)
	m.notify(second.ID)
	return nil
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

// SendMouse forwards a mouse event to a pane and updates activity timestamp.
func (m *Manager) SendMouse(id string, event uv.MouseEvent) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	m.mu.RLock()
	pane := m.panes[id]
	m.mu.RUnlock()
	if pane == nil || pane.window == nil {
		return fmt.Errorf("native: pane %q not found", id)
	}
	if event == nil {
		return nil
	}
	if pane.window.SendMouse(event) {
		m.markActive(id)
	}
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
		paneTitles := resolveSessionPaneTitles(session)
		for _, pane := range session.Panes {
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

func (m *Manager) buildPanes(ctx context.Context, spec SessionSpec) ([]*Pane, error) {
	if spec.Layout == nil {
		return nil, errors.New("native: layout is nil")
	}
	layoutCfg := spec.Layout
	if strings.TrimSpace(layoutCfg.Grid) != "" {
		return m.buildGridPanes(ctx, spec.Path, layoutCfg, spec.Env)
	}
	if len(layoutCfg.Panes) == 0 {
		return nil, errors.New("native: layout has no panes defined")
	}
	return m.buildSplitPanes(ctx, spec.Path, layoutCfg.Panes, spec.Env)
}

func (m *Manager) buildGridPanes(ctx context.Context, path string, layoutCfg *layout.LayoutConfig, env []string) ([]*Pane, error) {
	grid, err := layout.Parse(layoutCfg.Grid)
	if err != nil {
		return nil, fmt.Errorf("native: parse grid %q: %w", layoutCfg.Grid, err)
	}
	commands := layout.ResolveGridCommands(layoutCfg, grid.Panes())
	titles := layout.ResolveGridTitles(layoutCfg, grid.Panes())

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
				return nil, err
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
		}
	}
	return panes, nil
}

func (m *Manager) buildSplitPanes(ctx context.Context, path string, defs []layout.PaneDef, env []string) ([]*Pane, error) {
	var panes []*Pane
	active := (*Pane)(nil)
	for i, paneDef := range defs {
		pane, err := m.createPane(ctx, path, paneDef.Title, paneDef.Cmd, env)
		if err != nil {
			m.closePanes(panes)
			return nil, err
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
	}
	return panes, nil
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

func findPaneByIndex(panes []*Pane, index string) *Pane {
	index = strings.TrimSpace(index)
	for _, pane := range panes {
		if pane.Index == index {
			return pane
		}
	}
	return nil
}

func nextPaneIndex(panes []*Pane) string {
	max := -1
	for _, pane := range panes {
		if n, err := strconv.Atoi(strings.TrimSpace(pane.Index)); err == nil && n > max {
			max = n
		}
	}
	if max < 0 {
		return "0"
	}
	return strconv.Itoa(max + 1)
}

func anyPaneActive(panes []*Pane) bool {
	for _, pane := range panes {
		if pane.Active {
			return true
		}
	}
	return false
}

func sortPanesByIndex(panes []*Pane) {
	sort.SliceStable(panes, func(i, j int) bool {
		left := strings.TrimSpace(panes[i].Index)
		right := strings.TrimSpace(panes[j].Index)
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

func retilePanes(panes []*Pane) {
	if len(panes) == 0 {
		return
	}
	sortPanesByIndex(panes)

	count := len(panes)
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

	for i, pane := range panes {
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
