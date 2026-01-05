package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

const offlineScrollFallbackPage = 10
const offlineScrollWheelLines = 3

func (m *Model) handleOfflineScrollInput(msg tea.KeyMsg) (tea.Cmd, bool) {
	pane := m.selectedPane()
	if pane == nil || !pane.Disconnected {
		return nil, false
	}
	if key.Matches(msg, m.keys.scrollback) {
		if m.offlineScrollActiveFor(pane.ID) {
			m.clearOfflineScroll()
		} else {
			m.toggleOfflineScroll(pane.ID)
			m.setOfflineScrollOffset(*pane, 0)
		}
		return nil, true
	}
	if !m.offlineScrollActiveFor(pane.ID) {
		return nil, false
	}
	switch msg.String() {
	case "esc", "q":
		m.clearOfflineScroll()
		return nil, true
	case "up", "k":
		m.adjustOfflineScroll(pane, 1)
		return nil, true
	case "down", "j":
		m.adjustOfflineScroll(pane, -1)
		return nil, true
	case "pgup":
		m.adjustOfflineScroll(pane, m.offlineScrollPage(pane))
		return nil, true
	case "pgdown":
		m.adjustOfflineScroll(pane, -m.offlineScrollPage(pane))
		return nil, true
	case "home", "g":
		m.setOfflineScrollOffset(*pane, m.offlineScrollMax(*pane))
		return nil, true
	case "end", "G":
		m.setOfflineScrollOffset(*pane, 0)
		return nil, true
	default:
		return nil, false
	}
}

func (m *Model) handleOfflineScrollWheel(msg tea.MouseMsg) (tea.Cmd, bool) {
	if msg.Action == tea.MouseActionMotion {
		return nil, false
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
	default:
		return nil, false
	}
	hit, ok := m.hitTestPane(msg.X, msg.Y)
	if !ok || hit.PaneID == "" || !hit.Content.Contains(msg.X, msg.Y) {
		return nil, false
	}
	pane := m.paneByID(hit.PaneID)
	if pane == nil || !pane.Disconnected {
		return nil, false
	}
	m.ensureOfflineScrollMap()
	if hit.Content.H > 0 {
		m.offlineScrollViewport[pane.ID] = hit.Content.H
	}
	if !m.offlineScrollActiveFor(pane.ID) {
		m.toggleOfflineScroll(pane.ID)
		m.setOfflineScrollOffset(*pane, 0)
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.adjustOfflineScroll(pane, offlineScrollWheelLines)
	case tea.MouseButtonWheelDown:
		m.adjustOfflineScroll(pane, -offlineScrollWheelLines)
	}
	return nil, true
}

func (m *Model) toggleOfflineScroll(paneID string) {
	if strings.TrimSpace(paneID) == "" {
		return
	}
	if m.offlineScrollActive && m.offlineScrollPane == paneID {
		m.offlineScrollActive = false
		m.offlineScrollPane = ""
		return
	}
	m.offlineScrollActive = true
	m.offlineScrollPane = paneID
	m.ensureOfflineScrollMap()
}

func (m *Model) clearOfflineScroll() {
	m.offlineScrollActive = false
	m.offlineScrollPane = ""
}

func (m *Model) offlineScrollActiveFor(paneID string) bool {
	if !m.offlineScrollActive {
		return false
	}
	return strings.TrimSpace(m.offlineScrollPane) == strings.TrimSpace(paneID)
}

func (m *Model) ensureOfflineScrollMap() {
	if m.offlineScroll == nil {
		m.offlineScroll = make(map[string]int)
	}
	if m.offlineScrollViewport == nil {
		m.offlineScrollViewport = make(map[string]int)
	}
}

func (m *Model) offlineScrollOffset(paneID string) int {
	if m == nil || strings.TrimSpace(paneID) == "" || m.offlineScroll == nil {
		return 0
	}
	return m.offlineScroll[paneID]
}

func (m *Model) setOfflineScrollOffset(pane PaneItem, offset int) {
	if strings.TrimSpace(pane.ID) == "" {
		return
	}
	m.ensureOfflineScrollMap()
	max := m.offlineScrollMax(pane)
	if offset < 0 {
		offset = 0
	}
	if offset > max {
		offset = max
	}
	m.offlineScroll[pane.ID] = offset
}

func (m *Model) adjustOfflineScroll(pane *PaneItem, delta int) {
	if pane == nil {
		return
	}
	current := m.offlineScrollOffset(pane.ID)
	m.setOfflineScrollOffset(*pane, current+delta)
}

func (m *Model) offlineScrollPage(pane *PaneItem) int {
	if pane == nil {
		return offlineScrollFallbackPage
	}
	height := m.offlineScrollViewportHeight(pane.ID)
	if height <= 0 {
		height = offlineScrollFallbackPage
	}
	if height < 1 {
		height = 1
	}
	return height
}

func (m *Model) offlineScrollViewportHeight(paneID string) int {
	if m == nil || strings.TrimSpace(paneID) == "" || m.offlineScrollViewport == nil {
		return 0
	}
	return m.offlineScrollViewport[paneID]
}

func (m *Model) offlineScrollMax(pane PaneItem) int {
	lines := len(pane.Preview)
	if lines == 0 {
		return 0
	}
	height := m.offlineScrollViewportHeight(pane.ID)
	if height <= 0 {
		height = offlineScrollFallbackPage
		if height > lines {
			height = lines
		}
	}
	max := lines - height
	if max < 0 {
		return 0
	}
	return max
}

func (m *Model) pruneOfflineScroll() {
	if m == nil || m.offlineScroll == nil {
		return
	}
	activePane := strings.TrimSpace(m.offlineScrollPane)
	keep := make(map[string]struct{})
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			for _, pane := range session.Panes {
				if pane.Disconnected {
					keep[pane.ID] = struct{}{}
				}
			}
		}
	}
	for paneID := range m.offlineScroll {
		if _, ok := keep[paneID]; !ok {
			delete(m.offlineScroll, paneID)
		}
	}
	for paneID := range m.offlineScrollViewport {
		if _, ok := keep[paneID]; !ok {
			delete(m.offlineScrollViewport, paneID)
		}
	}
	if activePane != "" {
		if _, ok := keep[activePane]; !ok {
			m.clearOfflineScroll()
		}
	}
}
