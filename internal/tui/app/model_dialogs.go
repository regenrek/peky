package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

// ===== Rename dialogs =====

func (m *Model) initRenameInput(value, placeholder string) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 80
	input.Width = 40
	input.SetValue(value)
	input.CursorEnd()
	input.Focus()
	m.renameInput = input
}

func (m *Model) openRenameSession() {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	m.renameSession = session.Name
	m.renamePane = ""
	m.renamePaneIndex = ""
	m.initRenameInput(session.Name, "new session name")
	m.setState(StateRenameSession)
}

func (m *Model) openRenamePane() {
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
	m.renameSession = session.Name
	m.renamePane = pane.Title
	m.renamePaneIndex = pane.Index
	m.initRenameInput(pane.Title, "new pane title")
	m.setState(StateRenamePane)
}

func (m *Model) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		return m, m.applyRename()
	}

	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}

func (m *Model) applyRename() tea.Cmd {
	newName := strings.TrimSpace(m.renameInput.Value())
	if err := validateSessionName(newName); err != nil {
		m.setToast(err.Error(), toastWarning)
		return nil
	}

	switch m.state {
	case StateRenameSession:
		return m.applyRenameSession(newName)
	case StateRenamePane:
		return m.applyRenamePane(newName)
	default:
		m.setState(StateDashboard)
	}
	return nil
}

func (m *Model) applyRenameSession(newName string) tea.Cmd {
	if newName == m.renameSession {
		m.setState(StateDashboard)
		m.setToast("Session name unchanged", toastInfo)
		return nil
	}
	if m.client == nil {
		m.setToast("Rename failed: session client unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := m.client.RenameSession(ctx, m.renameSession, newName); err != nil {
		m.setToast("Rename failed: "+err.Error(), toastError)
		return nil
	}
	if m.selection.Session == m.renameSession {
		m.selection.Session = newName
	}
	if m.expandedSessions[m.renameSession] {
		delete(m.expandedSessions, m.renameSession)
		m.expandedSessions[newName] = true
	}
	m.selectionVersion++
	m.rememberSelection(m.selection)
	m.setState(StateDashboard)
	m.setToast("Renamed session to "+newName, toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) applyRenamePane(newName string) tea.Cmd {
	if newName == m.renamePane {
		m.setState(StateDashboard)
		m.setToast("Pane title unchanged", toastInfo)
		return nil
	}
	session, pane, ok := m.renamePaneTarget()
	if !ok {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	if m.client == nil {
		m.setToast("Rename failed: session client unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.client.RenamePane(ctx, session, pane, newName); err != nil {
		m.setToast("Rename failed: "+err.Error(), toastError)
		return nil
	}
	m.selectionVersion++
	m.setState(StateDashboard)
	m.setToast("Renamed pane to "+newName, toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) renamePaneTarget() (string, string, bool) {
	session := strings.TrimSpace(m.renameSession)
	pane := strings.TrimSpace(m.renamePaneIndex)
	if session == "" {
		session = strings.TrimSpace(m.selection.Session)
	}
	if pane == "" {
		pane = strings.TrimSpace(m.selection.Pane)
	}
	if session == "" || pane == "" {
		return "", "", false
	}
	return session, pane, true
}

// ===== Project root setup =====

func needsProjectRootSetup(cfg *layout.Config, configExists bool) bool {
	if cfg == nil {
		return true
	}
	if !configExists {
		return true
	}
	if len(cfg.Dashboard.ProjectRoots) > 0 {
		return false
	}
	if len(cfg.Projects) > 0 {
		return false
	}
	return true
}

func (m *Model) openProjectRootSetup() {
	roots := normalizeProjectRoots(m.config.Dashboard.ProjectRoots)
	if len(roots) == 0 {
		roots = defaultProjectRoots()
	}
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "~/projects"
	input.CharLimit = 200
	input.Width = 60
	input.SetValue(strings.Join(roots, ", "))
	input.CursorEnd()
	input.Focus()
	m.projectRootInput = input
	m.setState(StateProjectRootSetup)
}

func (m *Model) updateProjectRootSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.setState(StateDashboard)
		m.setToast("Using default project roots (edit config to customize)", toastInfo)
		return m, nil
	case "enter":
		return m, m.applyProjectRootSetup()
	}

	var cmd tea.Cmd
	m.projectRootInput, cmd = m.projectRootInput.Update(msg)
	return m, cmd
}

func (m *Model) applyProjectRootSetup() tea.Cmd {
	raw := strings.TrimSpace(m.projectRootInput.Value())
	roots := parseProjectRoots(raw)
	if len(roots) == 0 {
		m.setToast("Enter at least one project root", toastWarning)
		return nil
	}

	valid, invalid := validateProjectRoots(roots)
	if len(valid) == 0 {
		m.setToast("No valid project roots found", toastError)
		return nil
	}
	if err := m.saveProjectRoots(valid); err != nil {
		m.setToast("Save failed: "+err.Error(), toastError)
		return nil
	}
	if len(invalid) > 0 {
		m.setToast("Some paths not found: "+strings.Join(invalid, ", "), toastWarning)
	} else {
		m.setToast("Saved project roots", toastSuccess)
	}
	m.setState(StateDashboard)
	return nil
}

func parseProjectRoots(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var roots []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		roots = append(roots, part)
	}
	return normalizeProjectRoots(roots)
}

func validateProjectRoots(roots []string) ([]string, []string) {
	var valid []string
	var invalid []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || info == nil || !info.IsDir() {
			invalid = append(invalid, root)
			continue
		}
		valid = append(valid, root)
	}
	return valid, invalid
}

func (m *Model) saveProjectRoots(roots []string) error {
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return err
	}
	cfg.Dashboard.ProjectRoots = roots
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		return err
	}
	if err := layout.SaveConfig(m.configPath, cfg); err != nil {
		return err
	}
	m.config = cfg
	m.settings.ProjectRoots = normalizeProjectRoots(roots)
	return nil
}

// ===== Help view =====

func (m *Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		m.setState(StateDashboard)
		return m, nil
	case key.Matches(msg, m.keys.help):
		m.setState(StateDashboard)
		return m, nil
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	}
	return m, nil
}

// ===== Quick reply =====

func (m *Model) setQuickReplySize() {
	if m.width <= 0 {
		return
	}
	hFrame, _ := appStyle.GetFrameSize()
	contentWidth := m.width - hFrame
	if contentWidth <= 0 {
		contentWidth = m.width
	}
	barWidth := clamp(contentWidth-6, 30, 90)
	labelWidth := len("Quick Reply: ")
	inputWidth := barWidth - labelWidth - 2
	if inputWidth < 10 {
		inputWidth = 10
	}
	m.quickReplyInput.Width = inputWidth
}

func (m *Model) openQuickReply() tea.Cmd {
	if m.selectedPane() == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	m.setTerminalFocus(false)
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()
	m.resetQuickReplyHistory()
	return m.refreshPaneViewsCmd()
}

func (m *Model) updateQuickReply(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.paneNext):
		return m, m.cyclePane(1)
	case key.Matches(msg, m.keys.panePrev):
		return m, m.cyclePane(-1)
	}
	if m.quickReplyHistoryActive() && shouldExitQuickReplyHistory(msg) {
		m.resetQuickReplyHistory()
	}
	switch msg.String() {
	case "up":
		if m.moveQuickReplyHistory(-1) {
			return m, nil
		}
	case "down":
		if m.moveQuickReplyHistory(1) {
			return m, nil
		}
	}
	switch msg.String() {
	case "enter":
		text := strings.TrimSpace(m.quickReplyInput.Value())
		if text == "" {
			return m, m.attachOrStart()
		}
		outcome := m.handleQuickReplyCommand(text)
		if outcome.Handled {
			if outcome.RecordPrompt {
				m.rememberQuickReply(text)
			}
			if outcome.ClearInput {
				m.quickReplyInput.SetValue("")
				m.quickReplyInput.CursorEnd()
				m.resetQuickReplyHistory()
			}
			return m, outcome.Cmd
		}
		m.rememberQuickReply(text)
		m.resetQuickReplyHistory()
		return m, m.sendQuickReply()
	case "esc":
		m.quickReplyInput.SetValue("")
		m.quickReplyInput.CursorEnd()
		m.resetQuickReplyHistory()
		return m, nil
	}
	var cmd tea.Cmd
	m.quickReplyInput, cmd = m.quickReplyInput.Update(msg)
	return m, cmd
}

func (m *Model) sendQuickReply() tea.Cmd {
	text := strings.TrimSpace(m.quickReplyInput.Value())
	if text == "" {
		return NewInfoCmd("Nothing to send")
	}
	m.quickReplyInput.SetValue("")
	pane := m.selectedPane()
	if pane == nil || strings.TrimSpace(pane.ID) == "" {
		return NewWarningCmd("No pane selected")
	}
	paneID := strings.TrimSpace(pane.ID)
	if m.isPaneInputDisabled(paneID) {
		return nil
	}
	if pane.Dead {
		return func() tea.Msg {
			return newPaneClosedMsg(paneID, nil)
		}
	}
	payload := quickReplyTextBytes(*pane, text)
	label := strings.TrimSpace(pane.Title)
	if label == "" {
		label = fmt.Sprintf("pane %s", pane.Index)
	}
	return func() tea.Msg {
		if m.client == nil {
			return ErrorMsg{Err: errors.New("session client unavailable"), Context: "send to pane"}
		}
		if m.isPaneInputDisabled(paneID) {
			return nil
		}
		if pane := m.paneByID(paneID); pane == nil || pane.Dead {
			return newPaneClosedMsg(paneID, nil)
		}
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if err := m.client.SendInput(ctx, paneID, payload); err != nil {
			if isPaneClosedError(err) {
				return newPaneClosedMsg(paneID, err)
			}
			return ErrorMsg{Err: err, Context: "send to pane"}
		}
		if err := m.client.SendInput(ctx, paneID, []byte{'\r'}); err != nil {
			if isPaneClosedError(err) {
				return newPaneClosedMsg(paneID, err)
			}
			return ErrorMsg{Err: err, Context: "send to pane"}
		}
		if label != "" {
			return SuccessMsg{Message: "Sent to " + label}
		}
		return SuccessMsg{Message: "Sent"}
	}
}
