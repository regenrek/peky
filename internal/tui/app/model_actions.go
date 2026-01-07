package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/runenv"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

const daemonStopTimeout = 10 * time.Second

// ===== Session lifecycle =====

func (m *Model) attachOrStart() tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		return nil
	}
	if session.Status == StatusStopped {
		if session.Path == "" {
			m.setToast("No project path configured", toastWarning)
			return nil
		}
		return m.startProjectNative(*session, false)
	}
	return m.refreshPaneViewsCmd()
}

func (m *Model) startNewSessionWithLayout(layoutName string) tea.Cmd {
	project := m.selectedProject()
	session := m.selectedSession()
	if project == nil || session == nil {
		m.setToast("No project selected", toastWarning)
		return nil
	}
	path := session.Path
	if strings.TrimSpace(path) == "" {
		path = project.Path
	}
	if strings.TrimSpace(path) == "" {
		m.setToast("No project path configured", toastWarning)
		return nil
	}
	if err := validateProjectPath(path); err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}

	base := ""
	if session.Config != nil && strings.TrimSpace(session.Config.Session) != "" {
		base = session.Config.Session
	}
	if strings.TrimSpace(base) == "" {
		base = layout.SanitizeSessionName(project.Name)
	}
	if strings.TrimSpace(base) == "" {
		base = layout.SanitizeSessionName(session.Name)
	}
	base = layout.SanitizeSessionName(base)

	if m.client == nil {
		m.setToast("Start failed: session client unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	existing, err := m.client.SessionNames(ctx)
	if err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	newName := nextSessionName(base, existing)
	return m.startSessionNative(newName, path, layoutName, false)
}

func (m *Model) shutdownCmd() tea.Cmd {
	client := m.client
	paneViewClient := m.paneViewClient
	return func() tea.Msg {
		if client != nil {
			_ = client.Close()
		}
		if paneViewClient != nil && paneViewClient != client {
			_ = paneViewClient.Close()
		}
		return nil
	}
}

func (m *Model) requestQuit() tea.Cmd {
	if m == nil {
		return nil
	}
	runningPanes := runningPaneCount(m.data.Projects)
	switch m.settings.QuitBehavior {
	case QuitBehaviorKeep:
		return tea.Sequence(m.shutdownCmd(), tea.Quit)
	case QuitBehaviorStop:
		m.pendingQuit = quitActionStop
		m.setToast("Stopping daemon...", toastInfo)
		return m.stopDaemonCmd()
	default:
		if runningPanes == 0 {
			return tea.Sequence(m.shutdownCmd(), tea.Quit)
		}
		m.confirmQuitRunning = runningPanes
		m.setState(StateConfirmQuit)
		return nil
	}
}

func (m *Model) updateConfirmQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.confirmQuitRunning = 0
		m.setState(StateDashboard)
		return m, tea.Sequence(m.shutdownCmd(), tea.Quit)
	case "k":
		m.confirmQuitRunning = 0
		m.setState(StateDashboard)
		m.pendingQuit = quitActionStop
		m.setToast("Stopping daemon...", toastInfo)
		return m, m.stopDaemonCmd()
	case "n", "esc":
		m.confirmQuitRunning = 0
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
}

func (m *Model) stopDaemonCmd() tea.Cmd {
	client := m.client
	version := ""
	if client != nil {
		version = client.Version()
	}
	return func() tea.Msg {
		if version == "" {
			return daemonStopMsg{Err: errors.New("daemon version unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), daemonStopTimeout)
		defer cancel()
		if err := sessiond.StopDaemon(ctx, version); err != nil {
			return daemonStopMsg{Err: err}
		}
		return daemonStopMsg{}
	}
}

func (m *Model) handleDaemonStop(msg daemonStopMsg) tea.Cmd {
	if m.pendingQuit != quitActionStop {
		if msg.Err != nil {
			m.setToast("Stop daemon failed: "+msg.Err.Error(), toastError)
		}
		return nil
	}
	m.pendingQuit = quitActionNone
	if msg.Err != nil {
		m.setToast("Stop daemon failed: "+msg.Err.Error(), toastError)
		return nil
	}
	return tea.Sequence(m.shutdownCmd(), tea.Quit)
}

func (m *Model) startProjectNative(session SessionItem, focus bool) tea.Cmd {
	path := strings.TrimSpace(session.Path)
	if path == "" {
		if project := m.selectedProject(); project != nil {
			path = strings.TrimSpace(project.Path)
		}
	}
	if path == "" {
		m.setToast("No project path configured", toastWarning)
		return nil
	}
	if err := validateProjectPath(path); err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	return m.startSessionNative(session.Name, path, session.LayoutName, focus)
}

func (m *Model) startSessionNative(sessionName, path, layoutName string, focus bool) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return sessionStartedMsg{Path: path, Err: errors.New("session client unavailable"), Focus: focus}
		}
		ctx, cancel := context.WithTimeout(context.Background(), runenv.StartSessionTimeout())
		defer cancel()
		resp, err := m.client.StartSession(ctx, sessiond.StartSessionRequest{
			Name:       sessionName,
			Path:       path,
			LayoutName: layoutName,
		})
		return sessionStartedMsg{Name: resp.Name, Path: resp.Path, Err: err, Focus: focus}
	}
}

func (m *Model) startSessionAtPathDetached(path string) tea.Cmd {
	if err := validateProjectPath(path); err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	if m.client == nil {
		m.setToast("Start failed: session client unavailable", toastError)
		return nil
	}
	return m.startSessionNative("", path, "", false)
}

func (m *Model) editConfig() tea.Cmd {
	configPath, err := m.requireConfigPath()
	if err != nil {
		m.setToast("Edit config failed: "+err.Error(), toastError)
		return nil
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return tea.ExecProcess(exec.Command(editor, configPath), func(error) tea.Msg { return nil })
}

// ===== Kill/Close confirmations =====

func (m *Model) openKillConfirm() {
	session := m.selectedSession()
	if session == nil || session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	m.confirmSession = session.Name
	if project := m.selectedProject(); project != nil {
		m.confirmProject = project.Name
	} else {
		m.confirmProject = ""
	}
	m.setState(StateConfirmKill)
}

func (m *Model) openCloseProjectConfirm() {
	project := m.selectedProject()
	if project == nil {
		m.setToast("No project selected", toastWarning)
		return
	}
	m.confirmClose = project.Name
	m.confirmCloseID = project.ID
	m.setState(StateConfirmCloseProject)
}

func (m *Model) openCloseAllProjectsConfirm() {
	if len(m.data.Projects) == 0 {
		m.setToast("No projects to close", toastInfo)
		return
	}
	m.setState(StateConfirmCloseAllProjects)
}

func (m *Model) openClosePaneConfirm() tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return nil
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	running := !pane.Dead && !pane.Disconnected
	if !running {
		m.setState(StateDashboard)
		m.setToast("Closing pane...", toastInfo)
		return m.closePane(session.Name, pane.Index, pane.ID)
	}
	title := strings.TrimSpace(pane.Title)
	if title == "" {
		title = strings.TrimSpace(pane.Command)
	}
	if title == "" {
		title = fmt.Sprintf("pane %s", pane.Index)
	}
	m.confirmPaneSession = session.Name
	m.confirmPaneIndex = pane.Index
	m.confirmPaneID = pane.ID
	m.confirmPaneTitle = title
	m.confirmPaneRunning = running
	m.setState(StateConfirmClosePane)
	return nil
}

func (m *Model) updateConfirmCloseProject(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, m.applyCloseProject()
	case "k":
		return m, m.killProjectSessions()
	case "n", "esc":
		m.confirmClose = ""
		m.confirmCloseID = ""
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
}

func (m *Model) updateConfirmCloseAllProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, m.applyCloseAllProjects()
	case "k":
		return m, m.killAllProjectSessions()
	case "n", "esc":
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
}

func (m *Model) updateConfirmClosePane(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, m.applyClosePane()
	case "n", "esc":
		m.resetConfirmPane()
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
}

func (m *Model) resetConfirmPane() {
	m.confirmPaneSession = ""
	m.confirmPaneIndex = ""
	m.confirmPaneID = ""
	m.confirmPaneTitle = ""
	m.confirmPaneRunning = false
}

func (m *Model) applyCloseProject() tea.Cmd {
	name := strings.TrimSpace(m.confirmClose)
	projectID := strings.TrimSpace(m.confirmCloseID)
	m.confirmClose = ""
	m.confirmCloseID = ""
	m.setState(StateDashboard)
	if projectID == "" {
		return nil
	}
	project := findProjectByID(m.data.Projects, projectID)
	if project == nil {
		m.setToast("Project not found", toastWarning)
		return nil
	}
	if name == "" {
		name = project.Name
	}
	hidden, err := m.hideProjectInConfig(*project)
	if err != nil {
		m.setToast("Close failed: "+err.Error(), toastError)
		return nil
	}
	if !hidden {
		m.setToast("Project already hidden", toastInfo)
		return nil
	}
	m.setToast("Closed project "+name, toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) applyCloseAllProjects() tea.Cmd {
	m.setState(StateDashboard)
	if len(m.data.Projects) == 0 {
		m.setToast("No projects to close", toastInfo)
		return nil
	}
	hidden, err := m.hideAllProjectsInConfig(m.data.Projects)
	if err != nil {
		m.setToast("Close failed: "+err.Error(), toastError)
		return nil
	}
	if hidden == 0 {
		m.setToast("Projects already hidden", toastInfo)
		return nil
	}
	m.applySelection(selectionState{})
	m.selectionVersion++
	m.setToast("Closed all projects", toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) applyClosePane() tea.Cmd {
	session := strings.TrimSpace(m.confirmPaneSession)
	pane := strings.TrimSpace(m.confirmPaneIndex)
	paneID := strings.TrimSpace(m.confirmPaneID)
	m.resetConfirmPane()
	m.setState(StateDashboard)
	if session == "" || pane == "" {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	return m.closePane(session, pane, paneID)
}

func (m *Model) closePane(sessionName, paneIndex, paneID string) tea.Cmd {
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || paneIndex == "" {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	session := m.selectedSession()
	if session == nil || session.Name != sessionName {
		if session = findSessionByName(m.data.Projects, sessionName); session == nil {
			m.setToast("Session not found", toastWarning)
			return nil
		}
	}

	if m.client == nil {
		m.setToast("Close pane failed: session client unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.client.ClosePane(ctx, sessionName, paneIndex); err != nil {
		m.setToast("Close pane failed: "+err.Error(), toastError)
		return nil
	}
	sel := m.selection
	sel.Pane = ""
	m.applySelection(sel)
	m.selectionVersion++
	m.setToast("Closed pane", toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) killProjectSessions() tea.Cmd {
	name := strings.TrimSpace(m.confirmClose)
	projectID := strings.TrimSpace(m.confirmCloseID)
	m.confirmClose = ""
	m.confirmCloseID = ""
	m.setState(StateDashboard)
	if projectID == "" {
		return nil
	}
	project := findProjectByID(m.data.Projects, projectID)
	if project == nil {
		m.setToast("Project not found", toastWarning)
		return nil
	}
	if name == "" {
		name = project.Name
	}
	var running []SessionItem
	for _, s := range project.Sessions {
		if s.Status != StatusStopped {
			running = append(running, s)
		}
	}
	if len(running) == 0 {
		m.setToast("No running sessions to kill", toastInfo)
		return nil
	}
	var failed []string
	for _, s := range running {
		if m.client == nil {
			failed = append(failed, s.Name)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := m.client.KillSession(ctx, s.Name); err != nil {
			failed = append(failed, s.Name)
		}
		cancel()
	}
	if len(failed) > 0 {
		m.setToast("Kill failed: "+strings.Join(failed, ", "), toastError)
		return m.requestRefreshCmd()
	}
	m.setToast("Killed sessions for "+name, toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) killAllProjectSessions() tea.Cmd {
	m.setState(StateDashboard)
	if len(m.data.Projects) == 0 {
		m.setToast("No running sessions to kill", toastInfo)
		return nil
	}
	var running []SessionItem
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			if session.Status != StatusStopped {
				running = append(running, session)
			}
		}
	}
	if len(running) == 0 {
		m.setToast("No running sessions to kill", toastInfo)
		return nil
	}
	var failed []string
	for _, s := range running {
		if m.client == nil {
			failed = append(failed, s.Name)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := m.client.KillSession(ctx, s.Name); err != nil {
			failed = append(failed, s.Name)
		}
		cancel()
	}
	if len(failed) > 0 {
		m.setToast("Kill failed: "+strings.Join(failed, ", "), toastError)
		return m.requestRefreshCmd()
	}
	m.setToast("Killed all running sessions", toastSuccess)
	return m.requestRefreshCmd()
}

func runningPaneCount(projects []ProjectGroup) int {
	count := 0
	for _, project := range projects {
		for _, session := range project.Sessions {
			for _, pane := range session.Panes {
				if !pane.Dead && !pane.Disconnected {
					count++
				}
			}
		}
	}
	return count
}

func (m *Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmSession != "" {
			if m.client == nil {
				m.setToast("Kill failed: session client unavailable", toastError)
				m.setState(StateDashboard)
				return m, nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := m.client.KillSession(ctx, m.confirmSession); err != nil {
				m.setToast("Kill failed: "+err.Error(), toastError)
				m.setState(StateDashboard)
				return m, nil
			}
			m.setToast("Killed session "+m.confirmSession, toastSuccess)
			m.confirmSession = ""
			m.confirmProject = ""
			m.setState(StateDashboard)
			return m, m.requestRefreshCmd()
		}
		m.setState(StateDashboard)
		return m, nil
	case "n", "esc":
		m.confirmSession = ""
		m.confirmProject = ""
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
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
	sessionName = strings.TrimSpace(sessionName)
	paneID = strings.TrimSpace(paneID)
	if sessionName == "" || paneID == "" {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	session := m.selectedSession()
	if session == nil || session.Name != sessionName {
		session = findSessionByName(m.data.Projects, sessionName)
	}
	if session == nil {
		m.setToast("Session not found", toastWarning)
		return nil
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return nil
	}
	startDir := strings.TrimSpace(session.Path)
	if startDir == "" {
		if project := m.selectedProject(); project != nil {
			startDir = strings.TrimSpace(project.Path)
		}
	}
	if startDir != "" {
		if err := validateProjectPath(startDir); err != nil {
			m.setToast("Start failed: "+err.Error(), toastError)
			return nil
		}
	}

	if m.client == nil {
		m.setToast("Add pane failed: session client unavailable", toastError)
		return nil
	}
	pane := findPaneByID(session.Panes, paneID)
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	newIndex, err := m.client.SplitPane(ctx, session.Name, pane.Index, vertical, 0)
	if err != nil {
		m.setToast("Add pane failed: "+err.Error(), toastError)
		return nil
	}
	sel := m.selection
	sel.Session = session.Name
	sel.Pane = newIndex
	m.applySelection(sel)
	m.selectionVersion++
	m.lastSplitVertical = vertical
	m.lastSplitSet = true
	m.setToast("Added pane", toastSuccess)
	return m.requestRefreshCmd()
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
