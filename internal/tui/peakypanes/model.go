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

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/mux"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// Styles - using centralized theme for consistency
var (
	appStyle         = theme.App
	dialogStyle      = theme.Dialog
	dialogTitleStyle = theme.DialogTitle
)

// Key bindings

type dashboardKeyMap struct {
	projectLeft    key.Binding
	projectRight   key.Binding
	sessionUp      key.Binding
	sessionDown    key.Binding
	windowUp       key.Binding
	windowDown     key.Binding
	attach         key.Binding
	newSession     key.Binding
	openTerminal   key.Binding
	toggleWindows  key.Binding
	openProject    key.Binding
	commandPalette key.Binding
	refresh        key.Binding
	editConfig     key.Binding
	kill           key.Binding
	closeProject   key.Binding
	help           key.Binding
	quit           key.Binding
	filter         key.Binding
}

func newDashboardKeyMap() *dashboardKeyMap {
	return &dashboardKeyMap{
		projectLeft:    key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "project")),
		projectRight:   key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "project")),
		sessionUp:      key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "session")),
		sessionDown:    key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "session")),
		windowUp:       key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("⇧↑", "window")),
		windowDown:     key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("⇧↓", "window")),
		attach:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "attach")),
		newSession:     key.NewBinding(key.WithKeys("n", "s"), key.WithHelp("n", "new session")),
		openTerminal:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "new terminal")),
		toggleWindows:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "windows")),
		openProject:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open project")),
		commandPalette: key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("^p", "commands")),
		refresh:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		editConfig:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit config")),
		kill:           key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "kill session")),
		closeProject:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "close project")),
		help:           key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		quit:           key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		filter:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	}
}

// Model implements tea.Model for peakypanes TUI.
type Model struct {
	muxClient mux.Client
	state     ViewState
	width     int
	height    int

	configPath string
	insideMux  bool

	data      DashboardData
	selection selectionState
	settings  DashboardConfig
	config    *layout.Config

	keys *dashboardKeyMap

	expandedSessions map[string]bool

	filterInput  textinput.Model
	filterActive bool

	projectPicker  list.Model
	layoutPicker   list.Model
	commandPalette list.Model
	gitProjects    []GitProject

	confirmSession string
	confirmProject string
	confirmClose   string

	renameInput       textinput.Model
	renameSession     string
	renameWindow      string
	renameWindowIndex string

	projectRootInput textinput.Model

	toast      toastMessage
	refreshing bool

	selectionVersion uint64
	refreshInFlight  int
}

// NewModel creates a new peakypanes TUI model.
func NewModel(client mux.Client) (*Model, error) {
	if client == nil {
		return nil, fmt.Errorf("multiplexer client is required")
	}

	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	m := &Model{
		muxClient:        client,
		state:            StateDashboard,
		insideMux:        client.IsInside(),
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
	m.setupCommandPalette()

	configExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configExists = false
	}

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

	if needsProjectRootSetup(cfg, configExists) {
		m.openProjectRootSetup()
	}

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	m.beginRefresh()
	return tea.Batch(m.refreshCmd(), tickCmd(m.settings.RefreshInterval))
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (m *Model) selectionRefreshCmd() tea.Cmd {
	version := m.selectionVersion
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return selectionRefreshMsg{Version: version}
	})
}

func (m *Model) beginRefresh() {
	m.refreshInFlight++
	m.refreshing = true
}

func (m *Model) endRefresh() {
	if m.refreshInFlight > 0 {
		m.refreshInFlight--
	}
	m.refreshing = m.refreshInFlight > 0
}

func (m Model) refreshCmd() tea.Cmd {
	selection := m.selection
	configPath := m.configPath
	version := m.selectionVersion
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
		result := buildDashboardData(ctx, m.muxClient, muxSnapshotInput{Selection: selection, Version: version, Config: cfg, Settings: settings})
		result.Warning = warning
		return muxSnapshotMsg{Result: result}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectPicker.SetSize(msg.Width-4, msg.Height-4)
		m.setLayoutPickerSize()
		m.setCommandPaletteSize()
		return m, nil
	case refreshTickMsg:
		if m.refreshInFlight == 0 {
			m.beginRefresh()
			return m, tea.Batch(m.refreshCmd(), tickCmd(m.settings.RefreshInterval))
		}
		return m, tickCmd(m.settings.RefreshInterval)
	case selectionRefreshMsg:
		if msg.Version != m.selectionVersion {
			return m, nil
		}
		m.beginRefresh()
		return m, m.refreshCmd()
	case muxSnapshotMsg:
		m.endRefresh()
		if msg.Result.Err != nil {
			m.setToast("Refresh failed: "+msg.Result.Err.Error(), toastError)
			return m, nil
		}
		if msg.Result.Warning != "" {
			m.setToast("Dashboard config: "+msg.Result.Warning, toastWarning)
		}
		m.data = msg.Result.Data
		m.settings = msg.Result.Settings
		m.config = msg.Result.RawConfig
		if msg.Result.Version == m.selectionVersion {
			m.selection = msg.Result.Resolved
		} else {
			m.selection = resolveSelection(m.data.Projects, m.selection)
		}
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
	case exitAfterAttachMsg:
		return m, tea.Quit
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
		case StateCommandPalette:
			return m.updateCommandPalette(msg)
		case StateRenameSession, StateRenameWindow:
			return m.updateRename(msg)
		case StateProjectRootSetup:
			return m.updateProjectRootSetup(msg)
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
	if m.state == StateCommandPalette {
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
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
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.projectRight):
		m.selectProject(1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.sessionUp):
		m.selectSession(-1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.sessionDown):
		m.selectSession(1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.windowUp):
		m.selectWindow(-1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.windowDown):
		m.selectWindow(1)
		return m, m.selectionRefreshCmd()
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
	case key.Matches(msg, m.keys.commandPalette):
		return m, m.openCommandPalette()
	case key.Matches(msg, m.keys.refresh):
		m.setToast("Refreshing...", toastInfo)
		m.beginRefresh()
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
			m.selectionVersion++
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

func (m *Model) updateCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = StateDashboard
		return m, nil
	case "enter":
		if item, ok := m.commandPalette.SelectedItem().(CommandItem); ok {
			m.state = StateDashboard
			if item.Run != nil {
				return m, item.Run(m)
			}
		}
		m.state = StateDashboard
		return m, nil
	}

	if m.commandPalette.FilterState() == list.Filtering {
		switch msg.String() {
		case "up", "ctrl+p":
			m.commandPalette.CursorUp()
			return m, nil
		case "down", "ctrl+n":
			m.commandPalette.CursorDown()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.commandPalette, cmd = m.commandPalette.Update(msg)
	return m, cmd
}

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
	m.renameWindow = ""
	m.renameWindowIndex = ""
	m.initRenameInput(session.Name, "new session name")
	m.state = StateRenameSession
}

func (m *Model) openRenameWindow() {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	window := selectedWindow(session, m.selection.Window)
	if window == nil {
		m.setToast("No window selected", toastWarning)
		return
	}
	m.renameSession = session.Name
	m.renameWindow = window.Name
	m.renameWindowIndex = window.Index
	m.initRenameInput(window.Name, "new window name")
	m.state = StateRenameWindow
}

func (m *Model) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = StateDashboard
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
	if newName == "" {
		m.setToast("Name cannot be empty", toastWarning)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	switch m.state {
	case StateRenameSession:
		if newName == m.renameSession {
			m.state = StateDashboard
			m.setToast("Session name unchanged", toastInfo)
			return nil
		}
		if err := m.muxClient.RenameSession(ctx, m.renameSession, newName); err != nil {
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
		m.state = StateDashboard
		m.setToast("Renamed session to "+newName, toastSuccess)
		return m.refreshCmd()
	case StateRenameWindow:
		if newName == m.renameWindow {
			m.state = StateDashboard
			m.setToast("Window name unchanged", toastInfo)
			return nil
		}
		if err := m.muxClient.RenameWindow(ctx, m.renameSession, m.renameWindowIndex, newName); err != nil {
			m.setToast("Rename failed: "+err.Error(), toastError)
			return nil
		}
		m.selectionVersion++
		m.state = StateDashboard
		m.setToast("Renamed window to "+newName, toastSuccess)
		return m.refreshCmd()
	default:
		m.state = StateDashboard
	}
	return nil
}

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
	m.state = StateProjectRootSetup
}

func (m *Model) updateProjectRootSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = StateDashboard
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
	m.state = StateDashboard
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

func (m *Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmSession != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := m.muxClient.KillSession(ctx, m.confirmSession); err != nil {
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
	case StateCommandPalette:
		return m.viewCommandPalette()
	case StateRenameSession, StateRenameWindow:
		return m.viewRename()
	case StateProjectRootSetup:
		return m.viewProjectRootSetup()
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
	m.selectionVersion++
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
	m.selectionVersion++
}

func (m *Model) selectWindow(delta int) {
	session := m.selectedSession()
	if session == nil || len(session.Windows) == 0 {
		return
	}
	idx := windowIndex(session.Windows, m.selection.Window)
	idx = wrapIndex(idx+delta, len(session.Windows))
	m.selection.Window = session.Windows[idx].Index
	m.selectionVersion++
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
		args := []string{"start", "--session", session.Name, "--path", path, "--mux", m.muxClient.Type().String()}
		if session.LayoutName != "" {
			args = append(args, "--layout", session.LayoutName)
		}
		command := selfExecutable()
		if m.muxClient.Type() == mux.Tmux {
			if tmuxEnv := strings.TrimSpace(os.Getenv("TMUX")); tmuxEnv != "" {
				envArgs := []string{fmt.Sprintf("TMUX=%s", tmuxEnv), command}
				command = "/usr/bin/env"
				args = append(envArgs, args...)
			}
		}
		return m.openNewTerminal(command, args, "Session started in new terminal")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	attached, err := m.muxClient.SessionHasClients(ctx, session.Name)
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
	command, attachArgs, _ := m.muxClient.AttachCommand(target, m.insideMux)
	return m.openNewTerminal(command, attachArgs, "Opened session in new terminal")
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
	existing, err := m.muxClient.ListSessions(ctx)
	if err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	newName := nextSessionName(base, existing)

	args := []string{"start", "--session", newName, "--path", path, "--mux", m.muxClient.Type().String()}
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
	return func() tea.Msg {
		if err := m.muxClient.Attach(context.Background(), target, m.insideMux); err != nil {
			return ErrorMsg{Err: err, Context: "attach"}
		}
		if m.insideMux {
			return nil
		}
		return exitAfterAttachMsg{}
	}
}

func (m *Model) startProject(session SessionItem) tea.Cmd {
	args := []string{"start", "--session", session.Name, "--mux", m.muxClient.Type().String()}
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
	args := []string{"start", "--path", path, "--detach", "--mux", m.muxClient.Type().String()}
	return tea.ExecProcess(exec.Command(selfExecutable(), args...), func(err error) tea.Msg {
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
			if err := m.muxClient.KillSession(ctx, s.Name); err != nil {
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

	roots := m.settings.ProjectRoots
	if len(roots) == 0 {
		roots = defaultProjectRoots()
	}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
					relPath, _ := filepath.Rel(root, path)
					m.gitProjects = append(m.gitProjects, GitProject{Name: relPath, Path: path})
					return filepath.SkipDir
				}
			}
			return nil
		})
	}
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

// ===== Command palette =====

func (m *Model) setupCommandPalette() {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.Secondary)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "⌘ Command Palette"
	l.Styles.Title = theme.TitleAlt
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("command", "commands")
	m.commandPalette = l
}

func (m *Model) openCommandPalette() tea.Cmd {
	m.setCommandPaletteSize()
	m.commandPalette.SetFilterState(list.Filtering)
	cmd := m.commandPalette.SetItems(m.commandPaletteItems())
	m.state = StateCommandPalette
	return cmd
}

func (m *Model) setCommandPaletteSize() {
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
	desiredW := clamp(availableW, 46, 100)
	desiredH := clamp(availableH, 12, 26)
	listW := desiredW - hFrame
	listH := desiredH - vFrame
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	if listH < 6 {
		listH = clamp(m.height-vFrame, 6, m.height)
	}
	m.commandPalette.SetSize(listW, listH)
}

func (m *Model) commandPaletteItems() []list.Item {
	items := []CommandItem{
		{Label: "Project: Open project picker", Desc: "Scan project roots and create session", Run: func(m *Model) tea.Cmd {
			m.openProjectPicker()
			return nil
		}},
		{Label: "Project: Set project roots", Desc: "Choose folders to scan for projects", Run: func(m *Model) tea.Cmd {
			m.openProjectRootSetup()
			return nil
		}},
		{Label: "Project: Close project", Desc: "Kill all running sessions in project", Run: func(m *Model) tea.Cmd {
			m.openCloseProjectConfirm()
			return nil
		}},
		{Label: "Session: Attach / start", Desc: "Attach to running session or start if stopped", Run: func(m *Model) tea.Cmd {
			return m.attachOrStart()
		}},
		{Label: "Session: New session", Desc: "Pick a layout and create a new session", Run: func(m *Model) tea.Cmd {
			m.openLayoutPicker()
			return nil
		}},
		{Label: "Session: Open in new terminal", Desc: "Open session in Ghostty window", Run: func(m *Model) tea.Cmd {
			return m.openSessionInNewTerminal()
		}},
		{Label: "Session: Kill session", Desc: "Kill the selected session", Run: func(m *Model) tea.Cmd {
			m.openKillConfirm()
			return nil
		}},
		{Label: "Session: Rename session", Desc: "Rename the selected session", Run: func(m *Model) tea.Cmd {
			m.openRenameSession()
			return nil
		}},
		{Label: "Window: Toggle window list", Desc: "Expand/collapse window list", Run: func(m *Model) tea.Cmd {
			m.toggleWindows()
			return nil
		}},
		{Label: "Window: Rename window", Desc: "Rename the selected window", Run: func(m *Model) tea.Cmd {
			m.openRenameWindow()
			return nil
		}},
		{Label: "Other: Refresh", Desc: "Refresh dashboard data", Run: func(m *Model) tea.Cmd {
			m.setToast("Refreshing...", toastInfo)
			m.refreshing = true
			return m.refreshCmd()
		}},
		{Label: "Other: Edit config", Desc: "Open config in $EDITOR", Run: func(m *Model) tea.Cmd {
			return m.editConfig()
		}},
		{Label: "Other: Filter sessions", Desc: "Filter session list", Run: func(m *Model) tea.Cmd {
			m.filterActive = true
			m.filterInput.Focus()
			return nil
		}},
		{Label: "Other: Help", Desc: "Show help", Run: func(m *Model) tea.Cmd {
			m.state = StateHelp
			return nil
		}},
		{Label: "Other: Quit", Desc: "Exit PeakyPanes", Run: func(m *Model) tea.Cmd {
			return tea.Quit
		}},
	}
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
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
