package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/limits"
)

func (m *Model) openPaneColorDialog() {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	m.openPaneColorDialogFor(session.Name, pane.ID, pane.Index)
}

func (m *Model) openPaneColorDialogFor(sessionName, paneID, paneIndex string) {
	if m == nil {
		return
	}
	sessionName, paneID, paneIndex = trimPaneColorArgs(sessionName, paneID, paneIndex)
	pane := m.resolvePaneForColor(sessionName, paneID, paneIndex)
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	sessionName, paneIndex = m.resolvePaneColorSession(sessionName, paneIndex, pane)
	if !m.ensurePaneColorSessionRunning(sessionName, pane.ID) {
		return
	}
	paneIndex = normalizePaneColorIndex(paneIndex, pane)
	m.setPaneColorDialogState(sessionName, pane.ID, paneIndex, pane)
}

func (m *Model) updatePaneColor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.setState(StateDashboard)
		return m, nil
	case "1", "2", "3", "4", "5":
		value, _ := strconv.Atoi(msg.String())
		return m, m.applyPaneColor(value)
	}
	return m, nil
}

func (m *Model) applyPaneColor(value int) tea.Cmd {
	if m == nil {
		return nil
	}
	if value < limits.PaneBackgroundMin || value > limits.PaneBackgroundMax {
		m.setToast("Pane color must be 1-5", toastWarning)
		return nil
	}
	if m.client == nil {
		m.setToast("Pane color failed: session client unavailable", toastError)
		return nil
	}
	if value == m.paneColorCurrent {
		m.setState(StateDashboard)
		m.setToast("Pane color unchanged", toastInfo)
		return nil
	}
	paneID := strings.TrimSpace(m.paneColorPaneID)
	sessionName := strings.TrimSpace(m.paneColorSession)
	paneIndex := strings.TrimSpace(m.paneColorPaneIndex)
	if paneID == "" && (sessionName == "" || paneIndex == "") {
		m.setToast("No pane selected", toastWarning)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var err error
	if paneID != "" {
		err = m.client.SetPaneBackground(ctx, paneID, value)
	} else {
		err = m.client.SetPaneBackgroundByIndex(ctx, sessionName, paneIndex, value)
	}
	if err != nil {
		m.setToast("Pane color failed: "+err.Error(), toastError)
		return nil
	}
	m.paneColorCurrent = value
	m.setState(StateDashboard)
	m.setToast("Pane color set to "+strconv.Itoa(value), toastSuccess)
	return m.requestRefreshCmd()
}

func trimPaneColorArgs(sessionName, paneID, paneIndex string) (string, string, string) {
	return strings.TrimSpace(sessionName), strings.TrimSpace(paneID), strings.TrimSpace(paneIndex)
}

func (m *Model) resolvePaneForColor(sessionName, paneID, paneIndex string) *PaneItem {
	if paneID != "" {
		if pane := m.paneByID(paneID); pane != nil {
			return pane
		}
	}
	if sessionName == "" {
		return nil
	}
	session := findSessionByName(m.data.Projects, sessionName)
	if session == nil {
		return nil
	}
	if paneID != "" {
		if pane := findPaneByID(session.Panes, paneID); pane != nil {
			return pane
		}
	}
	if paneIndex != "" {
		return findPaneByIndex(session.Panes, paneIndex)
	}
	return nil
}

func (m *Model) resolvePaneColorSession(sessionName, paneIndex string, pane *PaneItem) (string, string) {
	if sessionName == "" && pane != nil {
		if sel, ok := m.selectionForPaneID(pane.ID); ok {
			sessionName = sel.Session
			if paneIndex == "" {
				paneIndex = sel.Pane
			}
		}
	}
	if sessionName == "" {
		if session := m.selectedSession(); session != nil {
			sessionName = session.Name
		}
	}
	return sessionName, paneIndex
}

func (m *Model) ensurePaneColorSessionRunning(sessionName, paneID string) bool {
	if sessionName == "" {
		return true
	}
	if m.sessionRunning(sessionName, paneID) {
		return true
	}
	m.setToast("Session not running", toastWarning)
	return false
}

func normalizePaneColorIndex(paneIndex string, pane *PaneItem) string {
	if paneIndex != "" || pane == nil {
		return paneIndex
	}
	return pane.Index
}

func (m *Model) setPaneColorDialogState(sessionName, paneID, paneIndex string, pane *PaneItem) {
	m.paneColorSession = sessionName
	m.paneColorPaneID = paneID
	m.paneColorPaneIndex = paneIndex
	m.paneColorTitle = paneColorLabel(pane)
	m.paneColorCurrent = normalizePaneBackground(pane.Background)
	m.setState(StatePaneColor)
}

func paneColorLabel(pane *PaneItem) string {
	if pane == nil {
		return ""
	}
	label := strings.TrimSpace(pane.Title)
	if label == "" {
		label = strings.TrimSpace(pane.Command)
	}
	if label == "" {
		index := strings.TrimSpace(pane.Index)
		if index == "" {
			label = "pane"
		} else {
			label = "pane " + index
		}
	}
	return label
}

func findPaneByIndex(panes []PaneItem, index string) *PaneItem {
	index = strings.TrimSpace(index)
	if index == "" {
		return nil
	}
	for i := range panes {
		if strings.TrimSpace(panes[i].Index) == index {
			return &panes[i]
		}
	}
	return nil
}
