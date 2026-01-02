package native

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/agenttool"
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
		if detected := agenttool.Normalize(tool); detected != "" {
			normalized = string(detected)
		} else {
			return fmt.Errorf("native: unknown tool %q", tool)
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

	paneWindow, paneID, notifyIDs, lastPane, err := m.removePaneFromSession(sessionName, paneIndex)
	if err != nil {
		return err
	}
	if paneWindow != nil {
		_ = paneWindow.Close()
	}
	m.notifyPane(paneID)
	if lastPane {
		return nil
	}
	for _, id := range notifyIDs {
		m.notifyPane(id)
	}
	m.version.Add(1)
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

func (m *Manager) removePaneFromSession(sessionName, paneIndex string) (*terminal.Window, string, []string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionName]
	if !ok {
		return nil, "", nil, false, fmt.Errorf("native: session %q not found", sessionName)
	}
	paneIdx, pane := findPaneByIndexAndPosition(session.Panes, paneIndex)
	if pane == nil {
		return nil, "", nil, false, fmt.Errorf("native: pane %q not found in %q", paneIndex, sessionName)
	}

	session.Panes = append(session.Panes[:paneIdx], session.Panes[paneIdx+1:]...)
	delete(m.panes, pane.ID)
	m.dropPreviewCache(pane.ID)
	if pane.output != nil {
		pane.output.disable()
	}
	paneWindow := pane.window

	if len(session.Panes) == 0 {
		return paneWindow, pane.ID, nil, true, nil
	}
	if !anyPaneActive(session.Panes) {
		session.Panes[0].Active = true
	}
	retilePanes(session.Panes)
	notifyIDs := make([]string, 0, len(session.Panes))
	for _, remaining := range session.Panes {
		notifyIDs = append(notifyIDs, remaining.ID)
	}
	return paneWindow, pane.ID, notifyIDs, false, nil
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
	env := append([]string(nil), session.Env...)
	m.mu.RUnlock()

	if strings.TrimSpace(startDir) != "" {
		if err := validatePath(startDir); err != nil {
			return "", err
		}
	}
	pane, err := m.createPane(ctx, startDir, "", "", env)
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
	pane.SetLastActive(time.Now())
	session.Panes = append(session.Panes, pane)
	m.panes[pane.ID] = pane
	m.mu.Unlock()

	m.forwardUpdates(pane)
	m.notifyPane(pane.ID)
	m.version.Add(1)
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

	firstRect := rectFromPane(first)
	secondRect := rectFromPane(second)
	first.Left, first.Top, first.Width, first.Height = secondRect.x, secondRect.y, secondRect.w, secondRect.h
	second.Left, second.Top, second.Width, second.Height = firstRect.x, firstRect.y, firstRect.w, firstRect.h
	first.Index, second.Index = second.Index, first.Index

	sortPanesByIndex(session.Panes)
	firstSeq := uint64(0)
	if first.window != nil {
		firstSeq = first.window.UpdateSeq()
	}
	secondSeq := uint64(0)
	if second.window != nil {
		secondSeq = second.window.UpdateSeq()
	}
	m.version.Add(1)
	m.mu.Unlock()

	m.notify(first.ID, firstSeq)
	m.notify(second.ID, secondSeq)
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
		if errors.Is(err, terminal.ErrPaneClosed) {
			m.notifyPane(id)
		}
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

	cellW := LayoutBaseSize / cols
	cellH := LayoutBaseSize / rows
	remainderW := LayoutBaseSize % cols
	remainderH := LayoutBaseSize % rows

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
		return rect{x: 0, y: 0, w: LayoutBaseSize, h: LayoutBaseSize}
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
