package peakypanes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kregenrek/tmuxman/internal/layout"
	"github.com/kregenrek/tmuxman/internal/tmuxctl"
	"github.com/kregenrek/tmuxman/internal/tui/theme"
)

// Styles - using centralized theme for consistency
var (
	appStyle         = theme.App
	dialogStyle      = theme.Dialog
	dialogTitleStyle = theme.DialogTitle
)

// Key bindings

type dashboardKeyMap struct {
	projectLeft   key.Binding
	projectRight  key.Binding
	sessionUp     key.Binding
	sessionDown   key.Binding
	windowUp      key.Binding
	windowDown    key.Binding
	attach        key.Binding
	newSession    key.Binding
	openTerminal  key.Binding
	toggleWindows key.Binding
	openProject   key.Binding
	refresh       key.Binding
	editConfig    key.Binding
	kill          key.Binding
	closeProject  key.Binding
	help          key.Binding
	quit          key.Binding
	filter        key.Binding
}

func newDashboardKeyMap() *dashboardKeyMap {
	return &dashboardKeyMap{
		projectLeft:   key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "project")),
		projectRight:  key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "project")),
		sessionUp:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "session")),
		sessionDown:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "session")),
		windowUp:      key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("⇧↑", "window")),
		windowDown:    key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("⇧↓", "window")),
		attach:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "attach")),
		newSession:    key.NewBinding(key.WithKeys("n", "s"), key.WithHelp("n", "new session")),
		openTerminal:  key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "new terminal")),
		toggleWindows: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "windows")),
		openProject:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open project")),
		refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		editConfig:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit config")),
		kill:          key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "kill session")),
		closeProject:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "close project")),
		help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	}
}

// Model implements tea.Model for peakypanes TUI.
type Model struct {
	tmux   *tmuxctl.Client
	state  ViewState
	width  int
	height int

	configPath string
	insideTmux bool

	data      DashboardData
	selection selectionState
	settings  DashboardConfig
	config    *layout.Config

	keys *dashboardKeyMap

	expandedSessions map[string]bool

	filterInput  textinput.Model
	filterActive bool

	projectPicker list.Model
	layoutPicker  list.Model
	gitProjects   []GitProject

	confirmSession string
	confirmProject string
	confirmClose   string

	toast      toastMessage
	refreshing bool
}

// NewModel creates a new peakypanes TUI model.
func NewModel(client *tmuxctl.Client) (*Model, error) {
	if client == nil {
		return nil, fmt.Errorf("tmux client is required")
	}

	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	m := &Model{
		tmux:             client,
		state:            StateDashboard,
		insideTmux:       os.Getenv("TMUX") != "" || os.Getenv("TMUX_PANE") != "",
		configPath:       configPath,
		keys:             newDashboardKeyMap(),
		expandedSessions: make(map[string]bool),
	}

	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "filter sessions"
	m.filterInput.CharLimit = 80
	m.filterInput.Width = 28

	m.setupProjectPicker()
	m.setupLayoutPicker()

	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	m.config = cfg
	settings, err := defaultDashboardConfig(cfg.Dashboard)
	if err != nil {
		return nil, err
	}
	m.settings = settings

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	m.refreshing = true
	return tea.Batch(m.refreshCmd(), tickCmd(m.settings.RefreshInterval))
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (m Model) refreshCmd() tea.Cmd {
	selection := m.selection
	configPath := m.configPath
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cfg, err := loadConfig(configPath)
		warning := ""
		if err != nil {
			warning = "config: " + err.Error()
			cfg = &layout.Config{}
		}
		settings, err := defaultDashboardConfig(cfg.Dashboard)
		if err != nil {
			if warning != "" {
				warning += "; "
			}
			warning += "dashboard: " + err.Error()
			settings, _ = defaultDashboardConfig(layout.DashboardConfig{})
		}
		result := buildDashboardData(ctx, m.tmux, tmuxSnapshotInput{Selection: selection, Config: cfg, Settings: settings})
		result.Warning = warning
		return tmuxSnapshotMsg{Result: result}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectPicker.SetSize(msg.Width-4, msg.Height-4)
		m.setLayoutPickerSize()
		return m, nil
	case refreshTickMsg:
		if !m.refreshing {
			m.refreshing = true
			return m, tea.Batch(m.refreshCmd(), tickCmd(m.settings.RefreshInterval))
		}
		return m, tickCmd(m.settings.RefreshInterval)
	case tmuxSnapshotMsg:
		m.refreshing = false
		if msg.Result.Err != nil {
			m.setToast("Refresh failed: "+msg.Result.Err.Error(), toastError)
			return m, nil
		}
		if msg.Result.Warning != "" {
			m.setToast("Dashboard config: "+msg.Result.Warning, toastWarning)
		}
		m.data = msg.Result.Data
		m.selection = msg.Result.Resolved
		m.settings = msg.Result.Settings
		m.config = msg.Result.RawConfig
		return m, nil
	case SuccessMsg:
		m.setToast(msg.Message, toastSuccess)
		return m, nil
	case WarningMsg:
		m.setToast(msg.Message, toastWarning)
		return m, nil
	case InfoMsg:
		m.setToast(msg.Message, toastInfo)
		return m, nil
	case ErrorMsg:
		m.setToast(msg.Error(), toastError)
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case StateDashboard:
			return m.updateDashboard(msg)
		case StateProjectPicker:
			return m.updateProjectPicker(msg)
		case StateLayoutPicker:
			return m.updateLayoutPicker(msg)
		case StateConfirmKill:
			return m.updateConfirmKill(msg)
		case StateConfirmCloseProject:
			return m.updateConfirmCloseProject(msg)
		case StateHelp:
			return m.updateHelp(msg)
		}
	}

	// Delegate to pickers when active (non-key messages)
	if m.state == StateProjectPicker {
		var cmd tea.Cmd
		m.projectPicker, cmd = m.projectPicker.Update(msg)
		return m, cmd
	}
	if m.state == StateLayoutPicker {
		var cmd tea.Cmd
		m.layoutPicker, cmd = m.layoutPicker.Update(msg)
		return m, cmd
	}

	if m.filterActive {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "enter":
			m.filterActive = false
			m.filterInput.Blur()
			return m, nil
		case "esc":
			m.filterActive = false
			m.filterInput.SetValue("")
			m.filterInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.projectLeft):
		m.selectProject(-1)
	case key.Matches(msg, m.keys.projectRight):
		m.selectProject(1)
	case key.Matches(msg, m.keys.sessionUp):
		m.selectSession(-1)
	case key.Matches(msg, m.keys.sessionDown):
		m.selectSession(1)
	case key.Matches(msg, m.keys.windowUp):
		m.selectWindow(-1)
	case key.Matches(msg, m.keys.windowDown):
		m.selectWindow(1)
	case key.Matches(msg, m.keys.attach):
		return m, m.attachOrStart()
	case key.Matches(msg, m.keys.newSession):
		m.openLayoutPicker()
		return m, nil
	case key.Matches(msg, m.keys.openTerminal):
		return m, m.openSessionInNewTerminal()
	case key.Matches(msg, m.keys.toggleWindows):
		m.toggleWindows()
	case key.Matches(msg, m.keys.openProject):
		m.openProjectPicker()
		return m, nil
	case key.Matches(msg, m.keys.refresh):
		m.setToast("Refreshing...", toastInfo)
		m.refreshing = true
		return m, m.refreshCmd()
	case key.Matches(msg, m.keys.editConfig):
		return m, m.editConfig()
	case key.Matches(msg, m.keys.kill):
		m.openKillConfirm()
		return m, nil
	case key.Matches(msg, m.keys.closeProject):
		m.openCloseProjectConfirm()
		return m, nil
	case key.Matches(msg, m.keys.filter):
		m.filterActive = true
		m.filterInput.Focus()
		return m, nil
	case key.Matches(msg, m.keys.help):
		m.state = StateHelp
		return m, nil
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m *Model) openProjectPicker() {
	m.scanGitProjects()
	m.projectPicker.SetItems(m.gitProjectsToItems())
	m.state = StateProjectPicker
}

func (m *Model) updateProjectPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.projectPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.projectPicker, cmd = m.projectPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.state = StateDashboard
		return m, nil
	case "enter":
		if item, ok := m.projectPicker.SelectedItem().(GitProject); ok {
			m.state = StateDashboard
			m.selection.Project = item.Name
			m.selection.Session = ""
			m.selection.Window = ""
			return m, m.startSessionAtPathDetached(item.Path)
		}
		m.state = StateDashboard
		return m, nil
	case "q":
		m.state = StateDashboard
		return m, nil
	}

	var cmd tea.Cmd
	m.projectPicker, cmd = m.projectPicker.Update(msg)
	return m, cmd
}

func (m *Model) updateLayoutPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.layoutPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.layoutPicker, cmd = m.layoutPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.state = StateDashboard
		return m, nil
	case "enter":
		if item, ok := m.layoutPicker.SelectedItem().(LayoutChoice); ok {
			m.state = StateDashboard
			return m, m.startNewSessionWithLayout(item.LayoutName)
		}
		m.state = StateDashboard
		return m, nil
	case "q":
		m.state = StateDashboard
		return m, nil
	}

	var cmd tea.Cmd
	m.layoutPicker, cmd = m.layoutPicker.Update(msg)
	return m, cmd
}

func (m *Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmSession != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := m.tmux.KillSession(ctx, m.confirmSession); err != nil {
				m.setToast("Kill failed: "+err.Error(), toastError)
				m.state = StateDashboard
				return m, nil
			}
			m.setToast("Killed session "+m.confirmSession, toastSuccess)
			m.confirmSession = ""
			m.confirmProject = ""
			m.state = StateDashboard
			return m, m.refreshCmd()
		}
		m.state = StateDashboard
		return m, nil
	case "n", "esc":
		m.confirmSession = ""
		m.confirmProject = ""
		m.state = StateDashboard
		return m, nil
	}
	return m, nil
}

func (m *Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		m.state = StateDashboard
		return m, nil
	}
	return m, nil
}

func (m *Model) View() string {
	switch m.state {
	case StateDashboard:
		return m.viewDashboard()
	case StateProjectPicker:
		return appStyle.Render(m.projectPicker.View())
	case StateLayoutPicker:
		return m.viewLayoutPicker()
	case StateConfirmKill:
		return m.viewConfirmKill()
	case StateConfirmCloseProject:
		return m.viewConfirmCloseProject()
	case StateHelp:
		return m.viewHelp()
	default:
		return m.viewDashboard()
	}
}

// ===== Selection helpers =====

func (m *Model) selectProject(delta int) {
	if len(m.data.Projects) == 0 {
		return
	}
	idx := m.projectIndex(m.selection.Project)
	idx = wrapIndex(idx+delta, len(m.data.Projects))
	m.selection.Project = m.data.Projects[idx].Name
	if len(m.data.Projects[idx].Sessions) > 0 {
		m.selection.Session = m.data.Projects[idx].Sessions[0].Name
		m.selection.Window = m.data.Projects[idx].Sessions[0].ActiveWindow
	}
}

func (m *Model) selectSession(delta int) {
	project := m.selectedProject()
	if project == nil || len(project.Sessions) == 0 {
		return
	}
	filtered := m.filteredSessions(project.Sessions)
	if len(filtered) == 0 {
		return
	}
	idx := sessionIndex(filtered, m.selection.Session)
	idx = wrapIndex(idx+delta, len(filtered))
	m.selection.Session = filtered[idx].Name
	m.selection.Window = filtered[idx].ActiveWindow
}

func (m *Model) selectWindow(delta int) {
	session := m.selectedSession()
	if session == nil || len(session.Windows) == 0 {
		return
	}
	idx := windowIndex(session.Windows, m.selection.Window)
	idx = wrapIndex(idx+delta, len(session.Windows))
	m.selection.Window = session.Windows[idx].Index
}

func (m *Model) toggleWindows() {
	session := m.selectedSession()
	if session == nil {
		return
	}
	current := m.expandedSessions[session.Name]
	m.expandedSessions[session.Name] = !current
}

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
		return m.startProject(*session)
	}
	return m.attachSession(*session)
}

func (m *Model) openSessionInNewTerminal() tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		return nil
	}
	tmuxEnv := strings.TrimSpace(os.Getenv("TMUX"))
	tmuxSocket := tmuxSocketFromEnv(tmuxEnv)

	if session.Status == StatusStopped {
		project := m.selectedProject()
		path := session.Path
		if strings.TrimSpace(path) == "" && project != nil {
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
		args := []string{"start", "--session", session.Name, "--path", path}
		if session.LayoutName != "" {
			args = append(args, "--layout", session.LayoutName)
		}
		command := selfExecutable()
		if tmuxEnv != "" {
			envArgs := []string{fmt.Sprintf("TMUX=%s", tmuxEnv), command}
			command = "/usr/bin/env"
			args = append(envArgs, args...)
		}
		return m.openNewTerminal(command, args, "Session started in new terminal")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	attached, err := m.tmux.SessionHasClients(ctx, session.Name)
	if err != nil {
		m.setToast("Check clients failed: "+err.Error(), toastError)
		return nil
	}
	if attached {
		return m.focusTerminalApp("Session already open")
	}

	target := session.Name
	if m.selection.Window != "" {
		target = fmt.Sprintf("%s:%s", session.Name, m.selection.Window)
	}
	tmuxPath := m.tmux.Binary()
	if tmuxPath == "" {
		tmuxPath = "tmux"
	}
	attachArgs := []string{}
	if tmuxSocket != "" {
		attachArgs = append(attachArgs, "-S", tmuxSocket)
	}
	attachArgs = append(attachArgs, "attach-session", "-t", target)
	return m.openNewTerminal(tmuxPath, attachArgs, "Opened session in new terminal")
}

func (m *Model) openNewTerminal(command string, args []string, successMsg string) tea.Cmd {
	cmd := m.newTerminalCommand(command, args)
	if cmd == nil {
		return NewWarningCmd("No terminal command configured")
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return WarningMsg{Message: "Open terminal failed: " + err.Error()}
		}
		if successMsg == "" {
			return nil
		}
		return SuccessMsg{Message: successMsg}
	})
}

func (m *Model) focusTerminalApp(message string) tea.Cmd {
	cmd := m.focusTerminalCommand()
	if cmd == nil {
		if message != "" {
			return NewInfoCmd(message)
		}
		return nil
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return WarningMsg{Message: "Focus terminal failed: " + err.Error()}
		}
		if message != "" {
			return InfoMsg{Message: message}
		}
		return nil
	})
}

func (m *Model) newTerminalCommand(command string, args []string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		openArgs := append([]string{"-na", "Ghostty.app", "--args", "-e", command}, args...)
		return exec.Command("open", openArgs...)
	default:
		openArgs := append([]string{"+new-window", "-e", command}, args...)
		return exec.Command("ghostty", openArgs...)
	}
}

func (m *Model) focusTerminalCommand() *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-a", "Ghostty")
	default:
		return nil
	}
}

func tmuxSocketFromEnv(env string) string {
	if env == "" {
		return ""
	}
	parts := strings.SplitN(env, ",", 2)
	return strings.TrimSpace(parts[0])
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
		base = sanitizeSessionName(project.Name)
	}
	if strings.TrimSpace(base) == "" {
		base = sanitizeSessionName(session.Name)
	}
	base = sanitizeSessionName(base)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	existing, err := m.tmux.ListSessions(ctx)
	if err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	newName := nextSessionName(base, existing)

	args := []string{"start", "--session", newName, "--path", path}
	if strings.TrimSpace(layoutName) != "" {
		args = append(args, "--layout", layoutName)
	}
	return tea.ExecProcess(exec.Command(selfExecutable(), args...), func(err error) tea.Msg {
		if err != nil {
			return WarningMsg{Message: "Start failed: " + err.Error()}
		}
		return SuccessMsg{Message: "Session started: " + newName}
	})
}

func (m *Model) attachSession(session SessionItem) tea.Cmd {
	target := session.Name
	if m.selection.Window != "" {
		target = fmt.Sprintf("%s:%s", session.Name, m.selection.Window)
	}
	if m.insideTmux {
		return tea.ExecProcess(exec.Command("tmux", "switch-client", "-t", target), func(error) tea.Msg { return nil })
	}
	return tea.ExecProcess(exec.Command("tmux", "attach-session", "-t", target), func(error) tea.Msg { return nil })
}

func (m *Model) startProject(session SessionItem) tea.Cmd {
	args := []string{"start", "--session", session.Name}
	if session.Path != "" {
		if err := validateProjectPath(session.Path); err != nil {
			m.setToast("Start failed: "+err.Error(), toastError)
			return nil
		}
		args = append(args, "--path", session.Path)
	}
	if session.LayoutName != "" {
		args = append(args, "--layout", session.LayoutName)
	}
	return tea.ExecProcess(exec.Command(selfExecutable(), args...), func(err error) tea.Msg {
		if err != nil {
			return WarningMsg{Message: "Start failed: " + err.Error()}
		}
		return SuccessMsg{Message: "Session started"}
	})
}

func (m *Model) startSessionAtPathDetached(path string) tea.Cmd {
	if err := validateProjectPath(path); err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	return tea.ExecProcess(exec.Command(selfExecutable(), "start", "--path", path, "--detach"), func(err error) tea.Msg {
		if err != nil {
			return WarningMsg{Message: "Start failed: " + err.Error()}
		}
		return SuccessMsg{Message: "Session started (detached)"}
	})
}

func (m *Model) editConfig() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return tea.ExecProcess(exec.Command(editor, m.configPath), func(error) tea.Msg { return nil })
}

func (m *Model) openKillConfirm() {
	session := m.selectedSession()
	if session == nil || session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	m.confirmSession = session.Name
	m.confirmProject = m.selection.Project
	m.state = StateConfirmKill
}

func (m *Model) openCloseProjectConfirm() {
	project := m.selectedProject()
	if project == nil {
		m.setToast("No project selected", toastWarning)
		return
	}
	m.confirmClose = project.Name
	m.state = StateConfirmCloseProject
}

func (m *Model) updateConfirmCloseProject(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		name := strings.TrimSpace(m.confirmClose)
		if name == "" {
			m.state = StateDashboard
			return m, nil
		}
		project := findProject(m.data.Projects, name)
		if project == nil {
			m.setToast("Project not found", toastWarning)
			m.confirmClose = ""
			m.state = StateDashboard
			return m, nil
		}
		var running []SessionItem
		for _, s := range project.Sessions {
			if s.Status != StatusStopped {
				running = append(running, s)
			}
		}
		if len(running) == 0 {
			m.setToast("No running sessions to close", toastInfo)
			m.confirmClose = ""
			m.state = StateDashboard
			return m, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var failed []string
		for _, s := range running {
			if err := m.tmux.KillSession(ctx, s.Name); err != nil {
				failed = append(failed, s.Name)
			}
		}
		m.confirmClose = ""
		m.state = StateDashboard
		if len(failed) > 0 {
			m.setToast("Close failed: "+strings.Join(failed, ", "), toastError)
			return m, m.refreshCmd()
		}
		m.setToast("Closed project "+name, toastSuccess)
		return m, m.refreshCmd()
	case "n", "esc":
		m.confirmClose = ""
		m.state = StateDashboard
		return m, nil
	}
	return m, nil
}

// ===== Project picker =====

func (m *Model) setupProjectPicker() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.Secondary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.Secondary)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "📁 Open Project"
	l.Styles.Title = theme.TitleAlt
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("project", "projects")
	m.projectPicker = l
}

func (m *Model) scanGitProjects() {
	m.gitProjects = nil

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	projectsDir := filepath.Join(home, "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return
	}

	_ = filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == "__pycache__" || name == ".venv" || name == "venv" {
				return filepath.SkipDir
			}
		}
		if d.IsDir() && d.Name() != ".git" {
			gitPath := filepath.Join(path, ".git")
			if _, err := os.Stat(gitPath); err == nil {
				relPath, _ := filepath.Rel(projectsDir, path)
				m.gitProjects = append(m.gitProjects, GitProject{Name: relPath, Path: path})
				return filepath.SkipDir
			}
		}
		return nil
	})
}

func (m *Model) gitProjectsToItems() []list.Item {
	items := make([]list.Item, len(m.gitProjects))
	for i, p := range m.gitProjects {
		items[i] = p
	}
	return items
}

// ===== Layout picker =====

func (m *Model) setupLayoutPicker() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.Secondary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.Secondary)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "🧩 New Session Layout"
	l.Styles.Title = theme.TitleAlt
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("layout", "layouts")
	m.layoutPicker = l
}

func (m *Model) openLayoutPicker() {
	project := m.selectedProject()
	session := m.selectedSession()
	if project == nil || session == nil {
		m.setToast("No project selected", toastWarning)
		return
	}
	path := session.Path
	if strings.TrimSpace(path) == "" {
		path = project.Path
	}
	if strings.TrimSpace(path) == "" {
		m.setToast("No project path configured", toastWarning)
		return
	}

	choices, err := m.loadLayoutChoices(path)
	if err != nil {
		m.setToast("Layouts: "+err.Error(), toastError)
		return
	}
	m.layoutPicker.SetItems(layoutChoicesToItems(choices))
	m.setLayoutPickerSize()
	m.state = StateLayoutPicker
}

func (m *Model) setLayoutPickerSize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	hFrame, vFrame := dialogStyle.GetFrameSize()
	availableW := m.width - 6
	availableH := m.height - 4
	if availableW < 30 {
		availableW = m.width
	}
	if availableH < 10 {
		availableH = m.height
	}
	desiredW := clamp(availableW, 40, 90)
	desiredH := clamp(availableH, 12, 26)
	listW := desiredW - hFrame
	listH := desiredH - vFrame
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	if listH < 6 {
		listH = clamp(m.height-vFrame, 6, m.height)
	}
	m.layoutPicker.SetSize(listW, listH)
}

func (m *Model) loadLayoutChoices(projectPath string) ([]LayoutChoice, error) {
	loader, err := layout.NewLoader()
	if err != nil {
		return nil, err
	}
	loader.SetProjectDir(projectPath)
	if err := loader.LoadAll(); err != nil {
		return nil, err
	}

	choices := []LayoutChoice{{
		Label:      "auto (project/default)",
		Desc:       "Use .peakypanes.yml or dev-3",
		LayoutName: "",
	}}

	layouts := loader.ListLayouts()
	seen := map[string]bool{}
	for _, info := range layouts {
		name := strings.TrimSpace(info.Name)
		if info.Source == "project" && (name == "" || name == "(project)") {
			continue
		}
		if name == "" {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true

		label := fmt.Sprintf("%s [%s]", name, info.Source)
		desc := strings.TrimSpace(info.Description)
		if desc == "" {
			if cfg, _, err := loader.GetLayout(name); err == nil {
				desc = layoutSummary(cfg)
			}
		}
		if desc == "" {
			desc = "layout"
		}
		choices = append(choices, LayoutChoice{
			Label:      label,
			Desc:       desc,
			LayoutName: name,
		})
	}
	return choices, nil
}

func layoutChoicesToItems(choices []LayoutChoice) []list.Item {
	items := make([]list.Item, len(choices))
	for i, c := range choices {
		items[i] = c
	}
	return items
}

// ===== Helpers =====

func (m *Model) projectIndex(name string) int {
	for i := range m.data.Projects {
		if m.data.Projects[i].Name == name {
			return i
		}
	}
	return 0
}

func sessionIndex(sessions []SessionItem, name string) int {
	for i := range sessions {
		if sessions[i].Name == name {
			return i
		}
	}
	return 0
}

func windowIndex(windows []WindowItem, idx string) int {
	for i := range windows {
		if windows[i].Index == idx {
			return i
		}
	}
	return 0
}

func wrapIndex(idx, total int) int {
	if total <= 0 {
		return 0
	}
	if idx < 0 {
		return total - 1
	}
	if idx >= total {
		return 0
	}
	return idx
}

func (m *Model) selectedProject() *ProjectGroup {
	for i := range m.data.Projects {
		if m.data.Projects[i].Name == m.selection.Project {
			return &m.data.Projects[i]
		}
	}
	if len(m.data.Projects) > 0 {
		return &m.data.Projects[0]
	}
	return nil
}

func (m *Model) selectedSession() *SessionItem {
	project := m.selectedProject()
	if project == nil {
		return nil
	}
	for i := range project.Sessions {
		if project.Sessions[i].Name == m.selection.Session {
			return &project.Sessions[i]
		}
	}
	if len(project.Sessions) > 0 {
		return &project.Sessions[0]
	}
	return nil
}

func (m *Model) filteredSessions(sessions []SessionItem) []SessionItem {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return sessions
	}
	filter = strings.ToLower(filter)
	var out []SessionItem
	for _, s := range sessions {
		if strings.Contains(strings.ToLower(s.Name), filter) || strings.Contains(strings.ToLower(s.Path), filter) {
			out = append(out, s)
		}
	}
	return out
}

func layoutSummary(cfg *layout.LayoutConfig) string {
	if cfg == nil {
		return ""
	}
	if strings.TrimSpace(cfg.Grid) != "" {
		if grid, err := layout.Parse(cfg.Grid); err == nil {
			return fmt.Sprintf("%d panes • %s grid", grid.Panes(), grid)
		}
		return fmt.Sprintf("grid %s", cfg.Grid)
	}
	windows := len(cfg.Windows)
	panes := 0
	for _, w := range cfg.Windows {
		panes += len(w.Panes)
	}
	if windows == 0 && panes == 0 {
		return ""
	}
	if windows == 1 {
		return fmt.Sprintf("%d panes • 1 window", panes)
	}
	return fmt.Sprintf("%d panes • %d windows", panes, windows)
}

// ===== Toasts =====

type toastLevel int

const (
	toastInfo toastLevel = iota
	toastSuccess
	toastWarning
	toastError
)

type toastMessage struct {
	Text  string
	Level toastLevel
	Until time.Time
}

func (m *Model) setToast(text string, level toastLevel) {
	m.toast = toastMessage{Text: text, Level: level, Until: time.Now().Add(3 * time.Second)}
}

func (m *Model) toastText() string {
	if m.toast.Text == "" || time.Now().After(m.toast.Until) {
		return ""
	}
	switch m.toast.Level {
	case toastSuccess:
		return theme.StatusMessage.Render(m.toast.Text)
	case toastWarning:
		return theme.StatusWarning.Render(m.toast.Text)
	case toastError:
		return theme.StatusError.Render(m.toast.Text)
	default:
		return theme.StatusMessage.Render(m.toast.Text)
	}
}
