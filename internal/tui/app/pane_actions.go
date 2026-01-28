package app

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type splitPaneError struct {
	msg   string
	level toastLevel
}

func (e splitPaneError) Error() string { return e.msg }

func splitPaneErr(level toastLevel, msg string) error {
	return splitPaneError{msg: msg, level: level}
}

func splitPaneErrLevel(err error) toastLevel {
	if err == nil {
		return toastInfo
	}
	if typed, ok := err.(splitPaneError); ok {
		return typed.level
	}
	return toastError
}

type splitPaneResult struct {
	sessionName string
	newIndex    string
	newPaneID   string
}

func (m *Model) addPaneSplit(vertical bool) tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	return m.addPaneSplitFor(session.Name, pane.ID, vertical)
}

func (m *Model) addPaneSplitFor(sessionName, paneID string, vertical bool) tea.Cmd {
	result, err := m.splitPaneFor(sessionName, paneID, vertical)
	if err != nil {
		level := splitPaneErrLevel(err)
		msg := err.Error()
		if level == toastError && !strings.HasPrefix(msg, "Start failed:") {
			msg = "Add pane failed: " + msg
		}
		m.setToast(msg, level)
		return nil
	}
	sel := m.selection
	sel.Session = result.sessionName
	sel.Pane = result.newIndex
	m.applySelection(sel)
	m.selectionVersion++
	m.lastSplitVertical = vertical
	m.lastSplitSet = true
	m.setToast("Added pane", toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) splitPaneFor(sessionName, paneID string, vertical bool) (splitPaneResult, error) {
	if m == nil {
		return splitPaneResult{}, splitPaneErr(toastError, "session client unavailable")
	}
	sessionName = strings.TrimSpace(sessionName)
	paneID = strings.TrimSpace(paneID)
	if sessionName == "" || paneID == "" {
		return splitPaneResult{}, splitPaneErr(toastWarning, "No pane selected")
	}
	session := m.selectedSession()
	if session == nil || session.Name != sessionName {
		session = findSessionByName(m.data.Projects, sessionName)
	}
	if session == nil {
		return splitPaneResult{}, splitPaneErr(toastWarning, "Session not found")
	}
	if session.Status == StatusStopped {
		return splitPaneResult{}, splitPaneErr(toastWarning, "Session not running")
	}
	startDir := strings.TrimSpace(session.Path)
	if startDir == "" {
		if project := m.selectedProject(); project != nil {
			startDir = strings.TrimSpace(project.Path)
		}
	}
	if startDir != "" {
		if err := validateProjectPath(startDir); err != nil {
			return splitPaneResult{}, splitPaneErr(toastError, "Start failed: "+err.Error())
		}
	}
	if m.client == nil {
		return splitPaneResult{}, splitPaneErr(toastError, "session client unavailable")
	}
	pane := findPaneByID(session.Panes, paneID)
	if pane == nil {
		return splitPaneResult{}, splitPaneErr(toastWarning, "No pane selected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	newIndex, newPaneID, err := m.client.SplitPane(ctx, session.Name, pane.Index, vertical, 0)
	if err != nil {
		return splitPaneResult{}, splitPaneErr(toastError, err.Error())
	}
	if strings.TrimSpace(newIndex) == "" {
		return splitPaneResult{}, splitPaneErr(toastError, "new pane index unavailable")
	}
	if strings.TrimSpace(newPaneID) == "" {
		return splitPaneResult{}, splitPaneErr(toastError, "new pane id unavailable")
	}
	return splitPaneResult{sessionName: session.Name, newIndex: newIndex, newPaneID: newPaneID}, nil
}

func (m *Model) swapPaneWith(target PaneSwapChoice) tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return nil
	}
	sourceSession := strings.TrimSpace(m.swapSourceSession)
	sourcePane := strings.TrimSpace(m.swapSourcePane)
	if sourceSession == "" {
		sourceSession = session.Name
	}
	if sourcePane == "" {
		if pane := m.selectedPane(); pane != nil {
			sourcePane = pane.Index
		}
	}
	if sourceSession == "" || sourcePane == "" {
		m.setToast("No pane selected", toastWarning)
		return nil
	}

	if m.client == nil {
		m.setToast("Swap pane failed: session client unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.client.SwapPanes(ctx, session.Name, sourcePane, target.PaneIndex); err != nil {
		m.setToast("Swap pane failed: "+err.Error(), toastError)
		return nil
	}
	sel := m.selection
	sel.Session = session.Name
	sel.Pane = target.PaneIndex
	m.applySelection(sel)
	m.selectionVersion++
	m.setToast("Swapped panes", toastSuccess)
	return m.requestRefreshCmd()
}

func findPaneByID(panes []PaneItem, paneID string) *PaneItem {
	if paneID == "" {
		return nil
	}
	for i := range panes {
		if panes[i].ID == paneID {
			return &panes[i]
		}
	}
	return nil
}
