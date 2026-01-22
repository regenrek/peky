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
	sessionName = strings.TrimSpace(sessionName)
	paneID = strings.TrimSpace(paneID)
	paneIndex = strings.TrimSpace(paneIndex)

	var pane *PaneItem
	if paneID != "" {
		pane = m.paneByID(paneID)
	}
	if pane == nil && sessionName != "" {
		session := findSessionByName(m.data.Projects, sessionName)
		if session != nil {
			if paneID != "" {
				pane = findPaneByID(session.Panes, paneID)
			}
			if pane == nil && paneIndex != "" {
				pane = findPaneByIndex(session.Panes, paneIndex)
			}
		}
	}
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	if sessionName == "" {
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
	if sessionName != "" && !m.sessionRunning(sessionName, pane.ID) {
		m.setToast("Session not running", toastWarning)
		return
	}
	if paneIndex == "" {
		paneIndex = pane.Index
	}
	m.paneColorSession = sessionName
	m.paneColorPaneID = pane.ID
	m.paneColorPaneIndex = paneIndex
	m.paneColorTitle = paneColorLabel(pane)
	m.paneColorCurrent = normalizePaneBackground(pane.Background)
	m.setState(StatePaneColor)
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
