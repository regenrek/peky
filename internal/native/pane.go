package native

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
	"github.com/regenrek/peakypanes/internal/terminal"
)

// Pane represents a running terminal pane.
type Pane struct {
	ID            string
	Index         string
	Title         string
	Command       string
	StartCommand  string
	Tool          string
	PID           int
	Active        bool
	Left          int
	Top           int
	Width         int
	Height        int
	Dead          bool
	DeadStatus    int
	RestoreFailed bool
	RestoreError  string
	lastActive    atomic.Int64
	Tags          []string
	RestoreMode   sessionrestore.Mode
	window        *terminal.Window
	output        *outputLog
}

func (p *Pane) SetLastActive(t time.Time) {
	if p == nil {
		return
	}
	if t.IsZero() {
		p.lastActive.Store(0)
		return
	}
	p.lastActive.Store(t.UnixNano())
}

func (p *Pane) LastActiveAt() time.Time {
	if p == nil {
		return time.Time{}
	}
	nano := p.lastActive.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
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
			m.version.Add(1)
			return nil
		}
	}
	return fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
}

// SetPaneTool updates the recorded tool for a pane.
func (m *Manager) SetPaneTool(paneID string, tool string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return errors.New("native: pane id is required")
	}
	tool = strings.TrimSpace(tool)
	var normalized string
	if tool != "" {
		reg := m.toolRegistryRef()
		if reg == nil {
			return errors.New("native: tool registry unavailable")
		}
		normalized = reg.Normalize(tool)
		if normalized == "" {
			return fmt.Errorf("native: unknown tool %q", tool)
		}
		if !reg.Allowed(normalized) {
			return fmt.Errorf("native: tool %q is disabled", normalized)
		}
	}
	var changed bool

	m.mu.Lock()
	pane := m.panes[paneID]
	if pane == nil {
		m.mu.Unlock()
		return fmt.Errorf("native: pane %q not found", paneID)
	}
	if pane.Tool != normalized {
		pane.Tool = normalized
		changed = true
	}
	m.mu.Unlock()

	if changed {
		// notifyMeta bumps the snapshot version and emits a metadata update event.
		// Do not call notifyPane while holding Manager.mu (it takes an RLock).
		m.notifyMeta(paneID)
	}
	return nil
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
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	paneIdx, pane := findPaneByIndexAndPosition(session.Panes, paneIndex)
	if pane == nil {
		m.mu.Unlock()
		return fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	if session.Layout == nil {
		m.mu.Unlock()
		return errors.New("native: layout engine unavailable")
	}
	result, err := session.Layout.Apply(layout.CloseOp{PaneID: pane.ID})
	if err != nil {
		m.mu.Unlock()
		return err
	}
	session.Panes = append(session.Panes[:paneIdx], session.Panes[paneIdx+1:]...)
	delete(m.panes, pane.ID)
	m.dropPreviewCache(pane.ID)
	if pane.output != nil {
		pane.output.disable()
	}
	if len(session.Panes) > 0 {
		if !anyPaneActive(session.Panes) {
			session.Panes[0].Active = true
		}
		if err := applyLayoutToPanes(session); err != nil {
			m.mu.Unlock()
			return err
		}
	}
	paneWindow := pane.window
	m.mu.Unlock()

	if paneWindow != nil {
		_ = paneWindow.Close()
	}
	m.applyScrollbackBudgets()
	m.notifyPane(pane.ID)
	for _, id := range result.Affected {
		m.notifyPane(id)
	}
	return nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func findPaneByIndexAndPosition(panes []*Pane, paneIndex string) (int, *Pane) {
	for i, candidate := range panes {
		if candidate.Index == paneIndex {
			return i, candidate
		}
	}
	return -1, nil
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

	targetRestore, startDir, env, err := m.splitPanePreflight(sessionName, paneIndex)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(startDir) != "" {
		if err := validatePath(startDir); err != nil {
			return "", err
		}
	}
	pane, err := m.createPane(ctx, startDir, "", "", env)
	if err != nil {
		return "", err
	}
	pane.RestoreMode = targetRestore

	axis := layout.AxisHorizontal
	if vertical {
		axis = layout.AxisVertical
	}
	result, newIndex, err := m.splitPaneCommit(sessionName, paneIndex, pane, axis, percent)
	if err != nil {
		_ = pane.window.Close()
		return "", err
	}

	m.applyScrollbackBudgets()
	m.forwardUpdates(pane)
	m.notifyPane(pane.ID)
	for _, id := range result.Affected {
		m.notifyPane(id)
	}
	return newIndex, nil
}

func (m *Manager) splitPanePreflight(sessionName, paneIndex string) (sessionrestore.Mode, string, []string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionName]
	if !ok {
		return sessionrestore.ModeDefault, "", nil, fmt.Errorf("native: session %q not found", sessionName)
	}
	target := findPaneByIndex(session.Panes, paneIndex)
	if target == nil {
		return sessionrestore.ModeDefault, "", nil, fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	startDir := strings.TrimSpace(session.Path)
	env := append([]string(nil), session.Env...)
	return target.RestoreMode, startDir, env, nil
}

func (m *Manager) splitPaneCommit(sessionName, paneIndex string, pane *Pane, axis layout.Axis, percent int) (layout.ApplyResult, string, error) {
	if m == nil || pane == nil {
		return layout.ApplyResult{}, "", errors.New("native: split commit requires manager and pane")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionName]
	if !ok {
		return layout.ApplyResult{}, "", fmt.Errorf("native: session %q not found", sessionName)
	}
	target := findPaneByIndex(session.Panes, paneIndex)
	if target == nil {
		return layout.ApplyResult{}, "", fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}
	if session.Layout == nil {
		return layout.ApplyResult{}, "", errors.New("native: layout engine unavailable")
	}
	result, err := session.Layout.Apply(layout.SplitOp{
		PaneID:    target.ID,
		NewPaneID: pane.ID,
		Axis:      axis,
		Percent:   percent,
	})
	if err != nil {
		return layout.ApplyResult{}, "", err
	}
	for _, existing := range session.Panes {
		existing.Active = false
	}
	pane.Index = nextPaneIndex(session.Panes)
	pane.Active = true
	pane.SetLastActive(time.Now())
	session.Panes = append(session.Panes, pane)
	m.panes[pane.ID] = pane
	if err := applyLayoutToPanes(session); err != nil {
		return layout.ApplyResult{}, "", err
	}
	return result, pane.Index, nil
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
	session, ok := m.sessions[sessionName]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("native: session %q not found", sessionName)
	}
	first := findPaneByIndex(session.Panes, paneA)
	second := findPaneByIndex(session.Panes, paneB)
	if first == nil || second == nil {
		m.mu.Unlock()
		return fmt.Errorf("native: panes %q and %q not found in %q", paneA, paneB, sessionName)
	}
	if session.Layout == nil {
		m.mu.Unlock()
		return errors.New("native: layout engine unavailable")
	}
	result, err := session.Layout.Apply(layout.SwapOp{PaneA: first.ID, PaneB: second.ID})
	if err != nil {
		m.mu.Unlock()
		return err
	}
	first.Index, second.Index = second.Index, first.Index
	sortPanesByIndex(session.Panes)
	if err := applyLayoutToPanes(session); err != nil {
		m.mu.Unlock()
		return err
	}
	firstSeq := uint64(0)
	if first.window != nil {
		firstSeq = first.window.UpdateSeq()
	}
	secondSeq := uint64(0)
	if second.window != nil {
		secondSeq = second.window.UpdateSeq()
	}
	m.mu.Unlock()

	m.notify(first.ID, firstSeq)
	m.notify(second.ID, secondSeq)
	for _, id := range result.Affected {
		m.notifyPane(id)
	}
	return nil
}

// SendInput writes input to a pane and updates activity timestamp.
func (m *Manager) SendInput(ctx context.Context, id string, input []byte) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.RLock()
	pane := m.panes[id]
	m.mu.RUnlock()
	if pane == nil || pane.window == nil {
		return fmt.Errorf("native: pane %q not found", id)
	}
	if err := pane.window.SendInput(ctx, input); err != nil {
		if errors.Is(err, terminal.ErrPaneClosed) {
			m.notifyPane(id)
		}
		return err
	}
	m.markActive(id)
	return nil
}

// SendMouse forwards a mouse event to a pane and updates activity timestamp.
func (m *Manager) SendMouse(id string, event uv.MouseEvent, route terminal.MouseRoute) error {
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
	if pane.window.SendMouse(event, route) {
		m.markActive(id)
	}
	return nil
}

func (p *Pane) windowExitStatus() int {
	if p == nil || p.window == nil {
		return 0
	}
	return p.window.ExitStatus()
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
