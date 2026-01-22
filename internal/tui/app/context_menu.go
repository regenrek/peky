package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/views"
)

type contextMenuState struct {
	open      bool
	x         int
	y         int
	items     []contextMenuItem
	index     int
	session   string
	paneID    string
	paneIndex string
}

type contextMenuItem struct {
	ID      string
	Label   string
	Enabled bool
}

const (
	contextMenuAddLast    = "add_last"
	contextMenuSplitRight = "split_right"
	contextMenuSplitDown  = "split_down"
	contextMenuClose      = "close_pane"
	contextMenuZoom       = "zoom_pane"
	contextMenuReset      = "reset_sizes"
	contextMenuColor      = "pane_color"
)

func (m *Model) handleContextMenuMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil || m.state != StateDashboard || m.tab != TabProject {
		return nil, false
	}
	if m.hardRaw {
		return nil, false
	}
	if m.contextMenu.open {
		return m.handleContextMenuMouseOpen(msg)
	}
	return m.handleContextMenuMouseClosed(msg)
}

func (m *Model) handleContextMenuMouseOpen(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil || !m.contextMenu.open {
		return nil, false
	}
	switch msg.Action {
	case tea.MouseActionMotion:
		m.contextMenuHoverAt(msg.X, msg.Y)
		return nil, true
	case tea.MouseActionPress:
		return m.contextMenuPress(msg), true
	default:
		return nil, true
	}
}

func (m *Model) handleContextMenuMouseClosed(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil || m.contextMenu.open {
		return nil, false
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonRight {
		return nil, false
	}
	hit, ok := m.hitTestPane(msg.X, msg.Y)
	if !ok || hit.PaneID == "" {
		return nil, false
	}
	m.openContextMenu(msg.X, msg.Y, hit)
	return nil, true
}

func (m *Model) contextMenuHoverAt(x, y int) {
	if m == nil || !m.contextMenu.open {
		return
	}
	rect, _, _, ok := m.contextMenuLayout()
	if !ok || !rect.Contains(x, y) {
		return
	}
	idx := y - rect.Y
	if idx < 0 || idx >= len(m.contextMenu.items) || m.contextMenu.index == idx {
		return
	}
	m.contextMenu.index = idx
}

func (m *Model) contextMenuPress(msg tea.MouseMsg) tea.Cmd {
	if m == nil || !m.contextMenu.open {
		return nil
	}
	rect, _, _, ok := m.contextMenuLayout()
	if !ok || !rect.Contains(msg.X, msg.Y) {
		m.closeContextMenu()
		return nil
	}
	if msg.Button != tea.MouseButtonLeft {
		return nil
	}
	idx := msg.Y - rect.Y
	if idx < 0 || idx >= len(m.contextMenu.items) {
		return nil
	}
	m.contextMenu.index = idx
	return m.applyContextMenu()
}

func (m *Model) handleContextMenuKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m == nil || !m.contextMenu.open {
		return nil, false
	}
	switch msg.String() {
	case "esc":
		m.closeContextMenu()
		return nil, true
	case "up":
		m.contextMenuMove(-1)
		return nil, true
	case "down":
		m.contextMenuMove(1)
		return nil, true
	case "enter":
		return m.applyContextMenu(), true
	default:
		return nil, true
	}
}

func (m *Model) contextMenuView() views.ContextMenu {
	if m == nil || !m.contextMenu.open {
		return views.ContextMenu{}
	}
	_, viewX, viewY, ok := m.contextMenuLayout()
	if !ok {
		return views.ContextMenu{}
	}
	items := make([]views.ContextMenuItem, 0, len(m.contextMenu.items))
	for _, item := range m.contextMenu.items {
		items = append(items, views.ContextMenuItem{
			Label:   item.Label,
			Enabled: item.Enabled,
		})
	}
	return views.ContextMenu{
		Open:     true,
		X:        viewX,
		Y:        viewY,
		Items:    items,
		Selected: m.contextMenu.index,
	}
}

func (m *Model) openContextMenu(x, y int, hit mouse.PaneHit) {
	if m == nil {
		return
	}
	sessionName := strings.TrimSpace(hit.Selection.Session)
	paneIndex := strings.TrimSpace(hit.Selection.Pane)
	items := m.buildContextMenuItems(sessionName, hit.PaneID)
	m.contextMenu = contextMenuState{
		open:      true,
		x:         x,
		y:         y,
		items:     items,
		index:     firstEnabledIndex(items),
		session:   sessionName,
		paneID:    hit.PaneID,
		paneIndex: paneIndex,
	}
}

func (m *Model) closeContextMenu() {
	if m == nil {
		return
	}
	m.contextMenu = contextMenuState{}
}

func (m *Model) contextMenuLayout() (mouse.Rect, int, int, bool) {
	if m == nil || !m.contextMenu.open {
		return mouse.Rect{}, 0, 0, false
	}
	layout, ok := m.dashboardLayoutInternal("contextMenu")
	if !ok {
		return mouse.Rect{}, 0, 0, false
	}
	menuW, menuH := contextMenuSize(m.contextMenu.items)
	if menuW <= 0 || menuH <= 0 {
		return mouse.Rect{}, 0, 0, false
	}
	maxX := layout.padLeft + layout.contentWidth - menuW
	maxY := layout.padTop + layout.contentHeight - menuH
	x := clamp(m.contextMenu.x, layout.padLeft, maxX)
	y := clamp(m.contextMenu.y, layout.padTop, maxY)
	rect := mouse.Rect{X: x, Y: y, W: menuW, H: menuH}
	return rect, x - layout.padLeft, y - layout.padTop, true
}

func (m *Model) contextMenuMove(delta int) {
	if m == nil || !m.contextMenu.open || len(m.contextMenu.items) == 0 {
		return
	}
	next := m.contextMenu.index
	for i := 0; i < len(m.contextMenu.items); i++ {
		next = (next + delta + len(m.contextMenu.items)) % len(m.contextMenu.items)
		if m.contextMenu.items[next].Enabled {
			m.contextMenu.index = next
			return
		}
	}
}

func (m *Model) applyContextMenu() tea.Cmd {
	if m == nil || !m.contextMenu.open {
		return nil
	}
	item, ok := m.contextMenuSelectedItem()
	if !ok {
		m.closeContextMenu()
		return nil
	}
	if !item.Enabled {
		return nil
	}
	sessionName, paneID, paneIndex := m.contextMenuTargets()
	m.closeContextMenu()
	return m.applyContextMenuItem(item.ID, sessionName, paneID, paneIndex)
}

func (m *Model) contextMenuSelectedItem() (contextMenuItem, bool) {
	if m == nil || !m.contextMenu.open {
		return contextMenuItem{}, false
	}
	if m.contextMenu.index < 0 || m.contextMenu.index >= len(m.contextMenu.items) {
		return contextMenuItem{}, false
	}
	return m.contextMenu.items[m.contextMenu.index], true
}

func (m *Model) contextMenuTargets() (sessionName, paneID, paneIndex string) {
	if m == nil {
		return "", "", ""
	}
	sessionName = m.contextMenu.session
	paneID = m.contextMenu.paneID
	paneIndex = m.contextMenu.paneIndex
	if paneIndex != "" || paneID == "" {
		return sessionName, paneID, paneIndex
	}
	session := findSessionByName(m.data.Projects, sessionName)
	if session == nil {
		return sessionName, paneID, paneIndex
	}
	pane := findPaneByID(session.Panes, paneID)
	if pane == nil {
		return sessionName, paneID, paneIndex
	}
	return sessionName, paneID, pane.Index
}

func (m *Model) applyContextMenuItem(id, sessionName, paneID, paneIndex string) tea.Cmd {
	switch id {
	case contextMenuAddLast:
		return m.addPaneSplitFor(sessionName, paneID, m.lastSplitVertical)
	case contextMenuSplitRight:
		return m.addPaneSplitFor(sessionName, paneID, false)
	case contextMenuSplitDown:
		return m.addPaneSplitFor(sessionName, paneID, true)
	case contextMenuClose:
		return m.closePane(sessionName, paneIndex, paneID)
	case contextMenuColor:
		m.openPaneColorDialogFor(sessionName, paneID, paneIndex)
		return nil
	case contextMenuZoom:
		return m.toggleZoomPaneFor(sessionName, paneID)
	case contextMenuReset:
		return m.resetPaneSizesFor(sessionName, paneID)
	default:
		return nil
	}
}

func (m *Model) buildContextMenuItems(sessionName, paneID string) []contextMenuItem {
	items := make([]contextMenuItem, 0, 7)
	verticalLabel := "right"
	if m.lastSplitSet && m.lastSplitVertical {
		verticalLabel = "down"
	}
	canModify := m.sessionRunning(sessionName, paneID)
	items = append(items, contextMenuItem{
		ID:      contextMenuAddLast,
		Label:   "Add pane (last: " + verticalLabel + ")",
		Enabled: canModify,
	})
	items = append(items, contextMenuItem{
		ID:      contextMenuSplitRight,
		Label:   "Split right",
		Enabled: canModify,
	})
	items = append(items, contextMenuItem{
		ID:      contextMenuSplitDown,
		Label:   "Split down",
		Enabled: canModify,
	})
	items = append(items, contextMenuItem{
		ID:      contextMenuClose,
		Label:   "Close pane",
		Enabled: canModify,
	})
	items = append(items, contextMenuItem{
		ID:      contextMenuColor,
		Label:   "Set pane color",
		Enabled: canModify,
	})
	zoomLabel := "Zoom pane"
	if engine := m.layoutEngines[sessionName]; engine != nil && engine.Tree != nil {
		if engine.Tree.ZoomedPaneID == paneID && paneID != "" {
			zoomLabel = "Unzoom pane"
		}
	}
	items = append(items, contextMenuItem{
		ID:      contextMenuZoom,
		Label:   zoomLabel,
		Enabled: canModify,
	})
	items = append(items, contextMenuItem{
		ID:      contextMenuReset,
		Label:   "Reset sizes",
		Enabled: canModify,
	})
	return items
}

func (m *Model) sessionRunning(sessionName, paneID string) bool {
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return false
	}
	session := m.selectedSession()
	if session == nil || session.Name != sessionName {
		session = findSessionByName(m.data.Projects, sessionName)
	}
	if session == nil || session.Status == StatusStopped {
		return false
	}
	if paneID == "" {
		return true
	}
	if pane := findPaneByID(session.Panes, paneID); pane != nil {
		return !pane.Dead && !pane.Disconnected
	}
	return true
}

func firstEnabledIndex(items []contextMenuItem) int {
	for i, item := range items {
		if item.Enabled {
			return i
		}
	}
	return 0
}

func contextMenuSize(items []contextMenuItem) (int, int) {
	if len(items) == 0 {
		return 0, 0
	}
	maxLabel := 0
	for _, item := range items {
		if l := len(item.Label); l > maxLabel {
			maxLabel = l
		}
	}
	width := maxLabel + 4
	if width < 10 {
		width = 10
	}
	return width, len(items)
}
