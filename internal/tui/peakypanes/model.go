package peakypanes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/terminal"
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
	projectLeft     key.Binding
	projectRight    key.Binding
	sessionUp       key.Binding
	sessionDown     key.Binding
	sessionOnlyUp   key.Binding
	sessionOnlyDown key.Binding
	paneNext        key.Binding
	panePrev        key.Binding
	attach          key.Binding
	newSession      key.Binding
	terminalFocus   key.Binding
	toggleWindows   key.Binding
	openProject     key.Binding
	commandPalette  key.Binding
	refresh         key.Binding
	editConfig      key.Binding
	kill            key.Binding
	closeProject    key.Binding
	help            key.Binding
	quit            key.Binding
	filter          key.Binding
	scrollback      key.Binding
	copyMode        key.Binding
}

// Model implements tea.Model for peakypanes TUI.
type Model struct {
	native *native.Manager
	state  ViewState
	tab    DashboardTab
	width  int
	height int

	configPath string

	data               DashboardData
	selection          selectionState
	selectionByProject map[string]selectionState
	settings           DashboardConfig
	config             *layout.Config
	projectConfigState map[string]projectConfigState

	keys *dashboardKeyMap

	expandedSessions map[string]bool

	filterInput  textinput.Model
	filterActive bool

	quickReplyInput textinput.Model

	mouse mouseState

	projectPicker  list.Model
	layoutPicker   list.Model
	paneSwapPicker list.Model
	commandPalette list.Model
	gitProjects    []GitProject

	confirmSession     string
	confirmProject     string
	confirmClose       string
	confirmPaneSession string
	confirmPaneWindow  string
	confirmPaneIndex   string
	confirmPaneID      string
	confirmPaneTitle   string
	confirmPaneRunning bool

	renameInput       textinput.Model
	renameSession     string
	renameWindow      string
	renameWindowIndex string
	renamePane        string
	renamePaneIndex   string

	swapSourceSession string
	swapSourceWindow  string
	swapSourcePane    string
	swapSourcePaneID  string

	projectRootInput textinput.Model

	toast      toastMessage
	refreshing bool

	selectionVersion uint64
	refreshInFlight  int

	terminalFocus bool

	autoStart *AutoStartSpec
}

// NewModel creates a new peakypanes TUI model.
func NewModel() (*Model, error) {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	m := &Model{
		native:             native.NewManager(),
		state:              StateDashboard,
		tab:                TabDashboard,
		configPath:         configPath,
		expandedSessions:   make(map[string]bool),
		selectionByProject: make(map[string]selectionState),
	}

	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "filter sessions"
	m.filterInput.CharLimit = 80
	m.filterInput.Width = 28

	m.quickReplyInput = textinput.New()
	m.quickReplyInput.Placeholder = "send a quick reply…"
	m.quickReplyInput.CharLimit = 400
	m.quickReplyInput.Prompt = ""
	qrStyle := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)
	m.quickReplyInput.TextStyle = qrStyle
	m.quickReplyInput.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Background(theme.QuickReplyBg)
	m.quickReplyInput.PromptStyle = qrStyle
	m.quickReplyInput.CursorStyle = qrStyle.Copy().
		Reverse(true)
	m.quickReplyInput.Focus()

	m.setupProjectPicker()
	m.setupLayoutPicker()
	m.setupPaneSwapPicker()
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
	keys, err := buildDashboardKeyMap(cfg.Dashboard.Keymap)
	if err != nil {
		return nil, err
	}
	m.keys = keys

	if needsProjectRootSetup(cfg, configExists) {
		m.openProjectRootSetup()
	}

	return m, nil
}

// SetAutoStart queues a session to start when the TUI launches.
func (m *Model) SetAutoStart(spec AutoStartSpec) {
	m.autoStart = &spec
}

func (m *Model) Init() tea.Cmd {
	m.beginRefresh()
	cmds := []tea.Cmd{m.refreshCmd(), tickCmd(m.settings.RefreshInterval)}
	if m.native != nil {
		cmds = append(cmds, waitNativePaneUpdate(m.native))
	}
	if m.autoStart != nil {
		cmds = append(cmds, m.startSessionNative(m.autoStart.Session, m.autoStart.Path, m.autoStart.Layout, m.autoStart.Focus))
	}
	return tea.Batch(cmds...)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func waitNativePaneUpdate(manager *native.Manager) tea.Cmd {
	if manager == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-manager.Events()
		if !ok {
			return nil
		}
		return nativePaneUpdatedMsg{PaneID: event.PaneID}
	}
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
	currentTab := m.tab
	configPath := m.configPath
	version := m.selectionVersion
	currentSettings := m.settings
	currentKeys := m.keys
	return func() tea.Msg {
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
			settings = currentSettings
		}
		keys, err := buildDashboardKeyMap(cfg.Dashboard.Keymap)
		if err != nil {
			if warning != "" {
				warning += "; "
			}
			warning += "keymap: " + err.Error()
			keys = currentKeys
		}
		result := buildDashboardData(dashboardSnapshotInput{
			Selection: selection,
			Tab:       currentTab,
			Version:   version,
			Config:    cfg,
			Settings:  settings,
			Native:    m.native,
		})
		result.Keymap = keys
		result.Warning = warning
		return dashboardSnapshotMsg{Result: result}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectPicker.SetSize(msg.Width-4, msg.Height-4)
		m.setLayoutPickerSize()
		m.setPaneSwapPickerSize()
		m.setCommandPaletteSize()
		m.setQuickReplySize()
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
	case dashboardSnapshotMsg:
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
		if msg.Result.Keymap != nil {
			m.keys = msg.Result.Keymap
		}
		if msg.Result.Version == m.selectionVersion {
			m.applySelection(msg.Result.Resolved)
		} else {
			m.applySelection(resolveSelection(m.data.Projects, m.selection))
		}
		m.syncExpandedSessions()
		if m.refreshSelectionForProjectConfig() {
			m.setToast("Project config changed: selection refreshed", toastInfo)
		}
		return m, nil
	case nativePaneUpdatedMsg:
		return m, waitNativePaneUpdate(m.native)
	case nativeSessionStartedMsg:
		if msg.Err != nil {
			m.setToast("Start failed: "+msg.Err.Error(), toastError)
			return m, nil
		}
		if msg.Name != "" {
			m.setToast("Session started: "+msg.Name, toastSuccess)
			projectName := m.projectNameForPath(msg.Path)
			if projectName != "" {
				m.selection.Project = projectName
			}
			m.selection.Session = msg.Name
			m.selection.Window = ""
			m.selection.Pane = ""
			m.selectionVersion++
			m.rememberSelection(m.selection)
		} else {
			m.setToast("Session started", toastSuccess)
		}
		m.setTerminalFocus(msg.Focus)
		m.beginRefresh()
		return m, m.refreshCmd()
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
	case tea.MouseMsg:
		if m.state == StateDashboard {
			return m.updateDashboardMouse(msg)
		}
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case StateDashboard:
			return m.updateDashboard(msg)
		case StateProjectPicker:
			return m.updateProjectPicker(msg)
		case StateLayoutPicker:
			return m.updateLayoutPicker(msg)
		case StatePaneSplitPicker:
			return m.updatePaneSplitPicker(msg)
		case StatePaneSwapPicker:
			return m.updatePaneSwapPicker(msg)
		case StateConfirmKill:
			return m.updateConfirmKill(msg)
		case StateConfirmCloseProject:
			return m.updateConfirmCloseProject(msg)
		case StateConfirmClosePane:
			return m.updateConfirmClosePane(msg)
		case StateHelp:
			return m.updateHelp(msg)
		case StateCommandPalette:
			return m.updateCommandPalette(msg)
		case StateRenameSession, StateRenameWindow, StateRenamePane:
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
	if m.state == StatePaneSwapPicker {
		var cmd tea.Cmd
		m.paneSwapPicker, cmd = m.paneSwapPicker.Update(msg)
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
			m.quickReplyInput.Focus()
			return m, nil
		case "esc":
			m.filterActive = false
			m.filterInput.SetValue("")
			m.filterInput.Blur()
			m.quickReplyInput.Focus()
			return m, nil
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	if m.supportsTerminalFocus() && m.terminalFocus {
		if msg.String() == "esc" {
			m.setTerminalFocus(false)
			return m, nil
		}
		if key.Matches(msg, m.keys.terminalFocus) {
			m.setTerminalFocus(false)
			return m, nil
		}
		// Intercept native scrollback/copy keys before PTY input.
		win := m.nativeFocusedWindow()
		if win != nil {
			res := handleNativeTerminalKey(msg, m.keys, win)
			if res.Handled {
				if res.Toast != "" {
					m.setToast(res.Toast, res.Level)
				}
				return m, res.Cmd
			}
		}
		if err := m.sendNativeKey(msg); err != nil {
			m.setToast("Input failed: "+err.Error(), toastError)
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.terminalFocus):
		if !m.supportsTerminalFocus() {
			m.setToast("Terminal focus is only available for PeakyPanes-managed sessions", toastInfo)
			return m, nil
		}
		m.setTerminalFocus(!m.terminalFocus)
		return m, nil
	case key.Matches(msg, m.keys.projectLeft):
		m.selectTab(-1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.projectRight):
		m.selectTab(1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.sessionUp):
		if m.tab == TabDashboard {
			m.selectDashboardPane(-1)
			return m, nil
		}
		m.selectSessionOrWindow(-1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.sessionDown):
		if m.tab == TabDashboard {
			m.selectDashboardPane(1)
			return m, nil
		}
		m.selectSessionOrWindow(1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.sessionOnlyUp):
		if m.tab == TabDashboard {
			m.selectDashboardPane(-1)
			return m, nil
		}
		m.selectSession(-1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.sessionOnlyDown):
		if m.tab == TabDashboard {
			m.selectDashboardPane(1)
			return m, nil
		}
		m.selectSession(1)
		return m, m.selectionRefreshCmd()
	case key.Matches(msg, m.keys.paneNext):
		if m.tab == TabDashboard {
			m.selectDashboardProject(1)
			return m, nil
		}
		return m, m.cyclePane(1)
	case key.Matches(msg, m.keys.panePrev):
		if m.tab == TabDashboard {
			m.selectDashboardProject(-1)
			return m, nil
		}
		return m, m.cyclePane(-1)
	case key.Matches(msg, m.keys.newSession):
		m.openLayoutPicker()
		return m, nil
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
		m.quickReplyInput.Blur()
		return m, nil
	case key.Matches(msg, m.keys.help):
		m.setState(StateHelp)
		return m, nil
	case key.Matches(msg, m.keys.quit):
		return m, tea.Sequence(m.shutdownCmd(), tea.Quit)
	}

	return m.updateQuickReply(msg)
}

func (m *Model) openProjectPicker() {
	m.scanGitProjects()
	m.projectPicker.ResetFilter()
	m.projectPicker.SetItems(m.gitProjectsToItems())
	m.setState(StateProjectPicker)
}

func (m *Model) updateProjectPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.projectPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.projectPicker, cmd = m.projectPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.projectPicker.ResetFilter()
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		if item, ok := m.projectPicker.SelectedItem().(GitProject); ok {
			projectName := m.projectNameForPath(item.Path)
			if projectName == "" {
				projectName = item.Name
			}
			sessionName := m.projectSessionNameForPath(item.Path)
			if _, err := m.unhideProjectInConfig(layout.HiddenProjectConfig{Name: projectName, Path: item.Path}); err != nil {
				m.setToast("Unhide failed: "+err.Error(), toastError)
			}
			m.projectPicker.ResetFilter()
			m.setState(StateDashboard)
			m.rememberSelection(m.selection)
			m.selection.Project = projectName
			m.selection.Session = sessionName
			m.selection.Window = ""
			m.selection.Pane = ""
			m.selectionVersion++
			m.rememberSelection(m.selection)
			return m, tea.Batch(m.startSessionAtPathDetached(item.Path), m.selectionRefreshCmd())
		}
		m.projectPicker.ResetFilter()
		m.setState(StateDashboard)
		return m, nil
	case "q":
		m.projectPicker.ResetFilter()
		m.setState(StateDashboard)
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
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		if item, ok := m.layoutPicker.SelectedItem().(LayoutChoice); ok {
			m.setState(StateDashboard)
			return m, m.startNewSessionWithLayout(item.LayoutName)
		}
		m.setState(StateDashboard)
		return m, nil
	case "q":
		m.setState(StateDashboard)
		return m, nil
	}

	var cmd tea.Cmd
	m.layoutPicker, cmd = m.layoutPicker.Update(msg)
	return m, cmd
}

func (m *Model) updateCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.commandPalette.ResetFilter()
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		if item, ok := m.commandPalette.SelectedItem().(CommandItem); ok {
			m.commandPalette.ResetFilter()
			m.setState(StateDashboard)
			if item.Run != nil {
				return m, item.Run(m)
			}
		}
		m.commandPalette.ResetFilter()
		m.setState(StateDashboard)
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
	m.renamePane = ""
	m.renamePaneIndex = ""
	m.initRenameInput(session.Name, "new session name")
	m.setState(StateRenameSession)
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
	m.renamePane = ""
	m.renamePaneIndex = ""
	m.initRenameInput(window.Name, "new window name")
	m.setState(StateRenameWindow)
}

func (m *Model) openNewWindow() tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
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

	if m.native == nil {
		m.setToast("New window failed: native manager unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := m.native.NewWindow(ctx, session.Name, "", startDir); err != nil {
		m.setToast("New window failed: "+err.Error(), toastError)
		return nil
	}
	m.selectionVersion++
	m.setToast("Opened new window", toastSuccess)
	return m.refreshCmd()
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
	window := selectedWindow(session, m.selection.Window)
	if window == nil {
		m.setToast("No window selected", toastWarning)
		return
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	m.renameSession = session.Name
	m.renameWindow = window.Name
	m.renameWindowIndex = window.Index
	m.renamePane = pane.Title
	m.renamePaneIndex = pane.Index
	m.initRenameInput(pane.Title, "new pane title")
	m.setState(StateRenamePane)
}

func (m *Model) addPaneSplit(vertical bool) tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return nil
	}
	if m.selectedPane() == nil {
		m.setToast("No pane selected", toastWarning)
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

	if m.native == nil {
		m.setToast("Add pane failed: native manager unavailable", toastError)
		return nil
	}
	window := selectedWindow(session, m.selection.Window)
	if window == nil {
		m.setToast("No window selected", toastWarning)
		return nil
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	newIndex, err := m.native.SplitPane(ctx, session.Name, window.Index, pane.Index, vertical, 0)
	if err != nil {
		m.setToast("Add pane failed: "+err.Error(), toastError)
		return nil
	}
	m.selection.Session = session.Name
	m.selection.Window = window.Index
	m.selection.Pane = newIndex
	m.selectionVersion++
	m.rememberSelection(m.selection)
	m.setToast("Added pane", toastSuccess)
	return m.refreshCmd()
}

func (m *Model) movePaneToNewWindow() tea.Cmd {
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
	window := selectedWindow(session, m.selection.Window)
	if window == nil {
		m.setToast("No window selected", toastWarning)
		return nil
	}

	if m.native == nil {
		m.setToast("Move pane failed: native manager unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	newWindow, newPane, err := m.native.MovePaneToNewWindow(ctx, session.Name, window.Index, pane.Index)
	if err != nil {
		m.setToast("Move pane failed: "+err.Error(), toastError)
		return nil
	}
	m.selection.Session = session.Name
	m.selection.Window = newWindow
	m.selection.Pane = newPane
	m.selectionVersion++
	m.rememberSelection(m.selection)
	m.setToast("Moved pane to new window", toastSuccess)
	return m.refreshCmd()
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
	sourceWindow := strings.TrimSpace(m.swapSourceWindow)
	sourcePane := strings.TrimSpace(m.swapSourcePane)
	if sourceSession == "" {
		sourceSession = session.Name
	}
	if sourceWindow == "" {
		sourceWindow = strings.TrimSpace(m.selection.Window)
	}
	if sourcePane == "" {
		if pane := m.selectedPane(); pane != nil {
			sourcePane = pane.Index
		}
	}
	if sourceSession == "" || sourceWindow == "" || sourcePane == "" {
		m.setToast("No pane selected", toastWarning)
		return nil
	}

	if m.native == nil {
		m.setToast("Swap pane failed: native manager unavailable", toastError)
		return nil
	}
	if err := m.native.SwapPanes(session.Name, sourceWindow, sourcePane, target.PaneIndex); err != nil {
		m.setToast("Swap pane failed: "+err.Error(), toastError)
		return nil
	}
	m.selection.Session = session.Name
	m.selection.Window = sourceWindow
	m.selection.Pane = target.PaneIndex
	m.selectionVersion++
	m.rememberSelection(m.selection)
	m.setToast("Swapped panes", toastSuccess)
	return m.refreshCmd()
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
		if newName == m.renameSession {
			m.setState(StateDashboard)
			m.setToast("Session name unchanged", toastInfo)
			return nil
		}
		if m.native == nil {
			m.setToast("Rename failed: native manager unavailable", toastError)
			return nil
		}
		if err := m.native.RenameSession(m.renameSession, newName); err != nil {
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
		return m.refreshCmd()
	case StateRenameWindow:
		if newName == m.renameWindow {
			m.setState(StateDashboard)
			m.setToast("Window name unchanged", toastInfo)
			return nil
		}
		if m.native == nil {
			m.setToast("Rename failed: native manager unavailable", toastError)
			return nil
		}
		if err := m.native.RenameWindow(m.renameSession, m.renameWindowIndex, newName); err != nil {
			m.setToast("Rename failed: "+err.Error(), toastError)
			return nil
		}
		m.selectionVersion++
		m.rememberSelection(m.selection)
		m.setState(StateDashboard)
		m.setToast("Renamed window to "+newName, toastSuccess)
		return m.refreshCmd()
	case StateRenamePane:
		if newName == m.renamePane {
			m.setState(StateDashboard)
			m.setToast("Pane title unchanged", toastInfo)
			return nil
		}
		session := strings.TrimSpace(m.renameSession)
		window := strings.TrimSpace(m.renameWindowIndex)
		pane := strings.TrimSpace(m.renamePaneIndex)
		if session == "" {
			session = strings.TrimSpace(m.selection.Session)
		}
		if window == "" {
			window = strings.TrimSpace(m.selection.Window)
		}
		if pane == "" {
			pane = strings.TrimSpace(m.selection.Pane)
		}
		if session == "" || window == "" || pane == "" {
			m.setToast("No pane selected", toastWarning)
			return nil
		}
		if m.native == nil {
			m.setToast("Rename failed: native manager unavailable", toastError)
			return nil
		}
		if err := m.native.RenamePane(session, window, pane, newName); err != nil {
			m.setToast("Rename failed: "+err.Error(), toastError)
			return nil
		}
		m.selectionVersion++
		m.setState(StateDashboard)
		m.setToast("Renamed pane to "+newName, toastSuccess)
		return m.refreshCmd()
	default:
		m.setState(StateDashboard)
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

func (m *Model) hideProjectInConfig(project ProjectGroup) (bool, error) {
	key := normalizeProjectKey(project.Path, project.Name)
	if key == "" {
		return false, fmt.Errorf("invalid project key")
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	existing := hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	if _, ok := existing[key]; ok {
		return false, nil
	}
	nameKey := strings.ToLower(strings.TrimSpace(project.Name))
	if nameKey != "" {
		if _, ok := existing[nameKey]; ok {
			return false, nil
		}
	}
	pathKey := strings.ToLower(normalizeProjectPath(project.Path))
	if pathKey != "" {
		if _, ok := existing[pathKey]; ok {
			return false, nil
		}
	}
	entry := layout.HiddenProjectConfig{
		Name: strings.TrimSpace(project.Name),
		Path: normalizeProjectPath(project.Path),
	}
	cfg.Dashboard.HiddenProjects = append(cfg.Dashboard.HiddenProjects, entry)
	cfg.Dashboard.HiddenProjects = normalizeHiddenProjects(cfg.Dashboard.HiddenProjects)
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := layout.SaveConfig(m.configPath, cfg); err != nil {
		return false, err
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return true, nil
}

func (m *Model) unhideProjectInConfig(entry layout.HiddenProjectConfig) (bool, error) {
	targetPathKey := strings.ToLower(normalizeProjectPath(entry.Path))
	targetNameKey := strings.ToLower(strings.TrimSpace(entry.Name))
	if targetPathKey == "" && targetNameKey == "" {
		return false, nil
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	if len(cfg.Dashboard.HiddenProjects) == 0 {
		return false, nil
	}
	kept := make([]layout.HiddenProjectConfig, 0, len(cfg.Dashboard.HiddenProjects))
	removed := 0
	for _, existing := range cfg.Dashboard.HiddenProjects {
		existingPathKey := strings.ToLower(normalizeProjectPath(existing.Path))
		existingNameKey := strings.ToLower(strings.TrimSpace(existing.Name))
		match := false
		if targetPathKey != "" && existingPathKey != "" && existingPathKey == targetPathKey {
			match = true
		}
		if !match && existingPathKey == "" && targetNameKey != "" && existingNameKey == targetNameKey {
			match = true
		}
		if !match && targetPathKey == "" && targetNameKey != "" && existingNameKey == targetNameKey {
			match = true
		}
		if match {
			removed++
			continue
		}
		kept = append(kept, existing)
	}
	if removed == 0 {
		return false, nil
	}
	cfg.Dashboard.HiddenProjects = normalizeHiddenProjects(kept)
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := layout.SaveConfig(m.configPath, cfg); err != nil {
		return false, err
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return true, nil
}

func (m *Model) hiddenProjectEntries() []layout.HiddenProjectConfig {
	if m.config == nil {
		return nil
	}
	entries := normalizeHiddenProjects(m.config.Dashboard.HiddenProjects)
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return hiddenProjectLabel(entries[i]) < hiddenProjectLabel(entries[j])
	})
	return entries
}

func hiddenProjectLabel(entry layout.HiddenProjectConfig) string {
	name := strings.TrimSpace(entry.Name)
	path := strings.TrimSpace(entry.Path)
	if name != "" && path != "" {
		return fmt.Sprintf("%s (%s)", name, shortenPath(path))
	}
	if name != "" {
		return name
	}
	if path != "" {
		return shortenPath(path)
	}
	return "unknown project"
}

func (m *Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmSession != "" {
			if m.native == nil {
				m.setToast("Kill failed: native manager unavailable", toastError)
				m.setState(StateDashboard)
				return m, nil
			}
			if err := m.native.KillSession(m.confirmSession); err != nil {
				m.setToast("Kill failed: "+err.Error(), toastError)
				m.setState(StateDashboard)
				return m, nil
			}
			m.setToast("Killed session "+m.confirmSession, toastSuccess)
			m.confirmSession = ""
			m.confirmProject = ""
			m.setState(StateDashboard)
			return m, m.refreshCmd()
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

func (m *Model) View() string {
	switch m.state {
	case StateDashboard:
		return m.viewDashboard()
	case StateProjectPicker:
		return appStyle.Render(m.projectPicker.View())
	case StateLayoutPicker:
		return m.viewLayoutPicker()
	case StatePaneSplitPicker:
		return m.viewPaneSplitPicker()
	case StatePaneSwapPicker:
		return m.viewPaneSwapPicker()
	case StateConfirmKill:
		return m.viewConfirmKill()
	case StateConfirmCloseProject:
		return m.viewConfirmCloseProject()
	case StateConfirmClosePane:
		return m.viewConfirmClosePane()
	case StateHelp:
		return m.viewHelp()
	case StateCommandPalette:
		return m.viewCommandPalette()
	case StateRenameSession, StateRenameWindow, StateRenamePane:
		return m.viewRename()
	case StateProjectRootSetup:
		return m.viewProjectRootSetup()
	default:
		return m.viewDashboard()
	}
}

func (m *Model) setState(state ViewState) {
	m.state = state
	if state == StateDashboard && !m.filterActive && !m.terminalFocus {
		m.quickReplyInput.Focus()
	} else {
		m.quickReplyInput.Blur()
	}
}

func (m *Model) setTerminalFocus(enabled bool) {
	if m.terminalFocus == enabled {
		return
	}
	m.terminalFocus = enabled
	if enabled {
		if m.filterActive {
			m.filterActive = false
			m.filterInput.Blur()
		}
		m.quickReplyInput.Blur()
		return
	}
	if m.state == StateDashboard && !m.filterActive {
		m.quickReplyInput.Focus()
	}
}

func (m *Model) supportsTerminalFocus() bool {
	if m.native == nil {
		return false
	}
	pane := m.selectedPane()
	if pane == nil {
		return false
	}
	if strings.TrimSpace(pane.ID) == "" {
		return false
	}
	return true
}

func (m *Model) syncExpandedSessions() {
	if m.expandedSessions == nil {
		m.expandedSessions = make(map[string]bool)
	}
	current := make(map[string]struct{})
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			current[session.Name] = struct{}{}
			if _, ok := m.expandedSessions[session.Name]; !ok {
				m.expandedSessions[session.Name] = true
			}
		}
	}
	for name := range m.expandedSessions {
		if _, ok := current[name]; !ok {
			delete(m.expandedSessions, name)
		}
	}
}

// ===== Selection helpers =====

func (m *Model) rememberSelection(sel selectionState) {
	if sel.Project == "" {
		return
	}
	if m.selectionByProject == nil {
		m.selectionByProject = make(map[string]selectionState)
	}
	m.selectionByProject[sel.Project] = sel
}

func (m *Model) selectionForProject(project string) selectionState {
	if project == "" {
		return selectionState{}
	}
	if m.selectionByProject != nil {
		if sel, ok := m.selectionByProject[project]; ok {
			sel.Project = project
			return sel
		}
	}
	return selectionState{Project: project}
}

func (m *Model) applySelection(sel selectionState) {
	m.selection = sel
	m.rememberSelection(sel)
	if m.terminalFocus && !m.supportsTerminalFocus() {
		m.setTerminalFocus(false)
	}
}

func (m *Model) refreshSelectionForProjectConfig() bool {
	project := m.selectedProject()
	if project == nil {
		return false
	}
	projectPath := normalizeProjectPath(project.Path)
	if projectPath == "" {
		return false
	}
	if m.projectConfigState == nil {
		m.projectConfigState = make(map[string]projectConfigState)
	}
	state := projectConfigStateForPath(projectPath)
	prev, ok := m.projectConfigState[projectPath]
	m.projectConfigState[projectPath] = state
	if !ok || prev.equal(state) {
		return false
	}

	delete(m.selectionByProject, project.Name)
	desired := selectionState{Project: project.Name}
	var resolved selectionState
	if m.tab == TabDashboard {
		resolved = resolveDashboardSelection(m.data.Projects, desired)
		if resolved.Project == "" {
			resolved = resolveSelection(m.data.Projects, desired)
		}
	} else {
		resolved = resolveSelection(m.data.Projects, desired)
	}
	if resolved.Project == "" {
		return false
	}
	m.applySelection(resolved)
	m.selectionVersion++
	return true
}

func (m *Model) selectTab(delta int) {
	total := len(m.data.Projects) + 1
	if total <= 1 {
		m.tab = TabDashboard
		return
	}

	if m.tab == TabDashboard {
		m.tab = TabProject
		projectName := m.data.Projects[0].Name
		if delta < 0 {
			projectName = m.data.Projects[len(m.data.Projects)-1].Name
		}
		resolved := resolveSelection(m.data.Projects, m.selectionForProject(projectName))
		m.applySelection(resolved)
		m.selectionVersion++
		return
	}

	idx, ok := m.projectIndexFor(m.selection.Project)
	if !ok {
		idx = 0
	}
	current := idx + 1
	next := wrapIndex(current+delta, total)
	if next == 0 {
		m.tab = TabDashboard
		m.selectionVersion++
		return
	}
	projectName := m.data.Projects[next-1].Name
	resolved := resolveSelection(m.data.Projects, m.selectionForProject(projectName))
	m.applySelection(resolved)
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
	m.selection.Window = ""
	m.selection.Pane = ""
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

func (m *Model) selectSessionOrWindow(delta int) {
	project := m.selectedProject()
	if project == nil || len(project.Sessions) == 0 {
		return
	}
	filtered := m.filteredSessions(project.Sessions)
	if len(filtered) == 0 {
		return
	}
	type entry struct {
		session string
		window  string
	}
	items := make([]entry, 0, len(filtered))
	for _, session := range filtered {
		items = append(items, entry{session: session.Name})
		if !m.sessionExpanded(session.Name) {
			continue
		}
		for _, window := range session.Windows {
			items = append(items, entry{session: session.Name, window: window.Index})
		}
	}
	if len(items) == 0 {
		return
	}

	current := -1
	if m.selection.Window != "" {
		for i, item := range items {
			if item.session == m.selection.Session && item.window == m.selection.Window {
				current = i
				break
			}
		}
	}
	if current == -1 {
		for i, item := range items {
			if item.session == m.selection.Session && item.window == "" {
				current = i
				break
			}
		}
	}
	if current == -1 {
		current = 0
	}

	next := items[wrapIndex(current+delta, len(items))]
	m.selection.Session = next.session
	m.selection.Window = next.window
	m.selection.Pane = ""
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

func (m *Model) selectDashboardPane(delta int) {
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return
	}
	filtered := m.filteredDashboardColumns(columns)
	if len(filtered) == 0 {
		return
	}
	projectIndex := m.dashboardProjectIndex(filtered)
	if projectIndex < 0 || projectIndex >= len(filtered) {
		projectIndex = 0
	}
	column := filtered[projectIndex]
	if len(column.Panes) == 0 {
		return
	}
	idx := dashboardPaneIndex(column.Panes, m.selection)
	if idx < 0 {
		idx = 0
	}
	idx = wrapIndex(idx+delta, len(column.Panes))
	pane := column.Panes[idx]
	m.selection.Project = column.ProjectName
	m.selection.Session = pane.SessionName
	m.selection.Window = pane.WindowIndex
	m.selection.Pane = pane.Pane.Index
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

func (m *Model) selectDashboardProject(delta int) {
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return
	}
	filtered := m.filteredDashboardColumns(columns)
	if len(filtered) == 0 {
		return
	}
	idx := m.dashboardProjectIndex(filtered)
	if idx < 0 {
		idx = 0
	}
	idx = wrapIndex(idx+delta, len(filtered))
	target := filtered[idx]
	desired := m.selectionForProject(target.ProjectName)
	resolved := resolveDashboardSelectionFromColumns(filtered, desired)
	if resolved.Project == "" {
		resolved.Project = target.ProjectName
	}
	m.applySelection(resolved)
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
	m.selection.Pane = ""
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

func (m *Model) selectPane(delta int) {
	session := m.selectedSession()
	if session == nil {
		return
	}
	type paneRef struct {
		windowIndex string
		paneIndex   string
	}
	var panes []paneRef
	for _, window := range session.Windows {
		if len(window.Panes) == 0 {
			continue
		}
		for _, pane := range window.Panes {
			panes = append(panes, paneRef{windowIndex: window.Index, paneIndex: pane.Index})
		}
	}
	if len(panes) == 0 {
		return
	}

	currentWindow := strings.TrimSpace(m.selection.Window)
	if currentWindow == "" {
		currentWindow = session.ActiveWindow
	}
	currentPane := strings.TrimSpace(m.selection.Pane)
	if currentPane == "" {
		if window := selectedWindow(session, currentWindow); window != nil && len(window.Panes) > 0 {
			if active := activePaneIndex(window.Panes); active != "" {
				currentPane = active
			} else {
				currentPane = window.Panes[0].Index
			}
		}
	}

	idx := -1
	for i, ref := range panes {
		if ref.windowIndex == currentWindow && ref.paneIndex == currentPane {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 0
	}
	idx = wrapIndex(idx+delta, len(panes))
	next := panes[idx]
	m.selection.Window = next.windowIndex
	m.selection.Pane = next.paneIndex
	m.rememberSelection(m.selection)
}

func (m *Model) toggleWindows() {
	session := m.selectedSession()
	if session == nil {
		return
	}
	current := m.expandedSessions[session.Name]
	m.expandedSessions[session.Name] = !current
}

func (m *Model) cyclePane(delta int) tea.Cmd {
	prevWindow := m.selection.Window
	prevPane := m.selection.Pane
	m.selectPane(delta)
	changed := m.selection.Window != prevWindow || m.selection.Pane != prevPane
	if !changed {
		return nil
	}
	m.selectionVersion++
	if m.selection.Window != prevWindow {
		return m.selectionRefreshCmd()
	}
	return nil
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
		focus := m.settings.AttachBehavior != AttachBehaviorDetached
		return m.startProjectNative(*session, focus)
	}
	m.setTerminalFocus(true)
	return nil
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

	if m.native == nil {
		m.setToast("Start failed: native manager unavailable", toastError)
		return nil
	}
	existing := m.native.SessionNames()
	newName := nextSessionName(base, existing)
	focus := m.settings.AttachBehavior != AttachBehaviorDetached
	return m.startSessionNative(newName, path, layoutName, focus)
}

func (m *Model) shutdownCmd() tea.Cmd {
	nativeMgr := m.native
	return func() tea.Msg {
		if nativeMgr != nil {
			nativeMgr.Close()
		}
		return nil
	}
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
		if m.native == nil {
			return nativeSessionStartedMsg{Path: path, Err: errors.New("native manager unavailable"), Focus: focus}
		}
		loader, err := layout.NewLoader()
		if err != nil {
			return nativeSessionStartedMsg{Path: path, Err: err, Focus: focus}
		}
		loader.SetProjectDir(path)
		if err := loader.LoadAll(); err != nil {
			return nativeSessionStartedMsg{Path: path, Err: err, Focus: focus}
		}

		sessionName = layout.ResolveSessionName(path, sessionName, loader.GetProjectConfig())
		if strings.TrimSpace(sessionName) == "" {
			return nativeSessionStartedMsg{Path: path, Err: errors.New("session name is required"), Focus: focus}
		}

		var selectedLayout *layout.LayoutConfig
		if layoutName != "" {
			selectedLayout, _, err = loader.GetLayout(layoutName)
			if err != nil {
				return nativeSessionStartedMsg{Path: path, Err: err, Focus: focus}
			}
		} else if loader.HasProjectConfig() {
			selectedLayout = loader.GetProjectLayout()
			if selectedLayout == nil {
				selectedLayout, _, _ = loader.GetLayout("dev-3")
			}
		} else {
			selectedLayout, _, _ = loader.GetLayout("dev-3")
		}
		if selectedLayout == nil {
			return nativeSessionStartedMsg{Path: path, Err: errors.New("no layout found"), Focus: focus}
		}

		projectName := filepath.Base(path)
		var projectVars map[string]string
		if loader.GetProjectConfig() != nil {
			projectVars = loader.GetProjectConfig().Vars
		}
		expanded := layout.ExpandLayoutVars(selectedLayout, projectVars, path, projectName)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = m.native.StartSession(ctx, native.SessionSpec{
			Name:       sessionName,
			Path:       path,
			Layout:     expanded,
			LayoutName: selectedLayout.Name,
		})
		return nativeSessionStartedMsg{Name: sessionName, Path: path, Err: err, Focus: focus}
	}
}

func (m *Model) startSessionAtPathDetached(path string) tea.Cmd {
	if err := validateProjectPath(path); err != nil {
		m.setToast("Start failed: "+err.Error(), toastError)
		return nil
	}
	if m.native == nil {
		m.setToast("Start failed: native manager unavailable", toastError)
		return nil
	}
	return m.startSessionNative("", path, "", false)
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
	m.setState(StateConfirmKill)
}

func (m *Model) openCloseProjectConfirm() {
	project := m.selectedProject()
	if project == nil {
		m.setToast("No project selected", toastWarning)
		return
	}
	m.confirmClose = project.Name
	m.setState(StateConfirmCloseProject)
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
	window := selectedWindow(session, m.selection.Window)
	if window == nil {
		m.setToast("No window selected", toastWarning)
		return nil
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	running := !pane.Dead
	if !running {
		m.setState(StateDashboard)
		m.setToast("Closing pane...", toastInfo)
		return m.closePane(session.Name, window.Index, pane.Index, pane.ID)
	}
	title := strings.TrimSpace(pane.Title)
	if title == "" {
		title = strings.TrimSpace(pane.Command)
	}
	if title == "" {
		title = fmt.Sprintf("pane %s", pane.Index)
	}
	m.confirmPaneSession = session.Name
	m.confirmPaneWindow = window.Index
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
	m.confirmPaneWindow = ""
	m.confirmPaneIndex = ""
	m.confirmPaneID = ""
	m.confirmPaneTitle = ""
	m.confirmPaneRunning = false
}

func (m *Model) applyCloseProject() tea.Cmd {
	name := strings.TrimSpace(m.confirmClose)
	m.confirmClose = ""
	m.setState(StateDashboard)
	if name == "" {
		return nil
	}
	project := findProject(m.data.Projects, name)
	if project == nil {
		m.setToast("Project not found", toastWarning)
		return nil
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
	return m.refreshCmd()
}

func (m *Model) applyClosePane() tea.Cmd {
	session := strings.TrimSpace(m.confirmPaneSession)
	window := strings.TrimSpace(m.confirmPaneWindow)
	pane := strings.TrimSpace(m.confirmPaneIndex)
	paneID := strings.TrimSpace(m.confirmPaneID)
	m.resetConfirmPane()
	m.setState(StateDashboard)
	if session == "" || window == "" || pane == "" {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	return m.closePane(session, window, pane, paneID)
}

func (m *Model) closePane(sessionName, windowIndex, paneIndex, paneID string) tea.Cmd {
	sessionName = strings.TrimSpace(sessionName)
	windowIndex = strings.TrimSpace(windowIndex)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || windowIndex == "" || paneIndex == "" {
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

	if m.native == nil {
		m.setToast("Close pane failed: native manager unavailable", toastError)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.native.ClosePane(ctx, sessionName, windowIndex, paneIndex); err != nil {
		m.setToast("Close pane failed: "+err.Error(), toastError)
		return nil
	}
	m.selection.Pane = ""
	if m.selection.Window == windowIndex {
		m.selection.Window = ""
	}
	m.selectionVersion++
	m.rememberSelection(m.selection)
	m.setToast("Closed pane", toastSuccess)
	return m.refreshCmd()
}

func (m *Model) killProjectSessions() tea.Cmd {
	name := strings.TrimSpace(m.confirmClose)
	m.confirmClose = ""
	m.setState(StateDashboard)
	if name == "" {
		return nil
	}
	project := findProject(m.data.Projects, name)
	if project == nil {
		m.setToast("Project not found", toastWarning)
		return nil
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
		if m.native == nil {
			failed = append(failed, s.Name)
			continue
		}
		if err := m.native.KillSession(s.Name); err != nil {
			failed = append(failed, s.Name)
		}
	}
	if len(failed) > 0 {
		m.setToast("Kill failed: "+strings.Join(failed, ", "), toastError)
		return m.refreshCmd()
	}
	m.setToast("Killed sessions for "+name, toastSuccess)
	return m.refreshCmd()
}

// ===== Project picker =====

func (m *Model) setupProjectPicker() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.AccentAlt)

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

func (m *Model) projectNameForPath(path string) string {
	path = normalizeProjectPath(path)
	if path == "" {
		return ""
	}
	if m.config != nil {
		for i := range m.config.Projects {
			name, _, cfgPath := normalizeProjectConfig(&m.config.Projects[i])
			if cfgPath != "" && cfgPath == path {
				return name
			}
		}
	}
	return filepath.Base(path)
}

func (m *Model) projectSessionNameForPath(path string) string {
	cfg, err := layout.LoadProjectLocal(path)
	if err != nil && !os.IsNotExist(err) {
		m.setToast("Project config error: "+err.Error(), toastWarning)
	}
	if err != nil {
		return layout.ResolveSessionName(path, "", nil)
	}
	return layout.ResolveSessionName(path, "", cfg)
}

// ===== Layout picker =====

func (m *Model) setupLayoutPicker() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.AccentAlt)

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
	m.setState(StateLayoutPicker)
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

// ===== Pane split picker =====

func (m *Model) openPaneSplitPicker() {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	if m.selectedPane() == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	m.setState(StatePaneSplitPicker)
}

func (m *Model) updatePaneSplitPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "r":
		m.setState(StateDashboard)
		return m, m.addPaneSplit(false)
	case "d":
		m.setState(StateDashboard)
		return m, m.addPaneSplit(true)
	}
	return m, nil
}

// ===== Pane swap picker =====

func (m *Model) setupPaneSwapPicker() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.AccentAlt)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "🔁 Swap Pane"
	l.Styles.Title = theme.TitleAlt
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("pane", "panes")
	m.paneSwapPicker = l
}

func (m *Model) openPaneSwapPicker() {
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
	if window == nil || len(window.Panes) == 0 {
		m.setToast("No window selected", toastWarning)
		return
	}
	source := m.selectedPane()
	if source == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	if len(window.Panes) < 2 {
		m.setToast("Not enough panes to swap", toastInfo)
		return
	}
	var items []list.Item
	for _, pane := range window.Panes {
		if pane.Index == source.Index {
			continue
		}
		title := strings.TrimSpace(pane.Title)
		if title == "" {
			title = strings.TrimSpace(pane.Command)
		}
		if title == "" {
			title = fmt.Sprintf("pane %s", pane.Index)
		}
		label := fmt.Sprintf("pane %s — %s", pane.Index, title)
		desc := strings.TrimSpace(pane.Command)
		if desc == "" {
			desc = "swap target"
		}
		items = append(items, PaneSwapChoice{
			Label:       label,
			Desc:        desc,
			WindowIndex: window.Index,
			PaneIndex:   pane.Index,
		})
	}
	if len(items) == 0 {
		m.setToast("No pane selected", toastWarning)
		return
	}
	m.swapSourceSession = session.Name
	m.swapSourceWindow = window.Index
	m.swapSourcePane = source.Index
	m.swapSourcePaneID = source.ID
	m.paneSwapPicker.SetItems(items)
	m.setPaneSwapPickerSize()
	m.setState(StatePaneSwapPicker)
}

func (m *Model) setPaneSwapPickerSize() {
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
	desiredH := clamp(availableH, 12, 24)
	listW := desiredW - hFrame
	listH := desiredH - vFrame
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	if listH < 6 {
		listH = clamp(m.height-vFrame, 6, m.height)
	}
	m.paneSwapPicker.SetSize(listW, listH)
}

func (m *Model) updatePaneSwapPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.paneSwapPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.paneSwapPicker, cmd = m.paneSwapPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		if item, ok := m.paneSwapPicker.SelectedItem().(PaneSwapChoice); ok {
			m.setState(StateDashboard)
			return m, m.swapPaneWith(item)
		}
		m.setState(StateDashboard)
		return m, nil
	}

	var cmd tea.Cmd
	m.paneSwapPicker, cmd = m.paneSwapPicker.Update(msg)
	return m, cmd
}

// ===== Command palette =====

func (m *Model) setupCommandPalette() {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt)

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
	m.commandPalette.ResetFilter()
	m.commandPalette.SetFilterState(list.Filtering)
	cmd := m.commandPalette.SetItems(m.commandPaletteItems())
	m.setState(StateCommandPalette)
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
		{Label: "Project: Close project", Desc: "Hide project from tabs (sessions stay running)", Run: func(m *Model) tea.Cmd {
			m.openCloseProjectConfirm()
			return nil
		}},
	}
	for _, entry := range m.hiddenProjectEntries() {
		entry := entry
		label := hiddenProjectLabel(entry)
		items = append(items, CommandItem{
			Label: "Project: Reopen " + label,
			Desc:  "Show hidden project",
			Run: func(m *Model) tea.Cmd {
				return m.reopenHiddenProject(entry)
			},
		})
	}
	items = append(items, []CommandItem{
		{Label: "Session: Attach / start", Desc: "Attach to running session or start if stopped", Run: func(m *Model) tea.Cmd {
			return m.attachOrStart()
		}},
		{Label: "Session: New session", Desc: "Pick a layout and create a new session", Run: func(m *Model) tea.Cmd {
			m.openLayoutPicker()
			return nil
		}},
		{Label: "Session: Kill session", Desc: "Kill the selected session", Run: func(m *Model) tea.Cmd {
			m.openKillConfirm()
			return nil
		}},
		{Label: "Pane: Add pane", Desc: "Split the selected pane", Run: func(m *Model) tea.Cmd {
			m.openPaneSplitPicker()
			return nil
		}},
		{Label: "Pane: Move to new window", Desc: "Move the selected pane into a new window", Run: func(m *Model) tea.Cmd {
			return m.movePaneToNewWindow()
		}},
		{Label: "Pane: Swap pane", Desc: "Swap the selected pane with another", Run: func(m *Model) tea.Cmd {
			m.openPaneSwapPicker()
			return nil
		}},
		{Label: "Pane: Close pane", Desc: "Close the selected pane", Run: func(m *Model) tea.Cmd {
			return m.openClosePaneConfirm()
		}},
		{Label: "Pane: Quick reply", Desc: "Send a short follow-up to the selected pane", Run: func(m *Model) tea.Cmd {
			return m.openQuickReply()
		}},
		{Label: "Pane: Rename pane", Desc: "Rename the selected pane title", Run: func(m *Model) tea.Cmd {
			m.openRenamePane()
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
		{Label: "Window: New window", Desc: "Create a new window in the selected session", Run: func(m *Model) tea.Cmd {
			return m.openNewWindow()
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
			m.quickReplyInput.Blur()
			return nil
		}},
		{Label: "Other: Help", Desc: "Show help", Run: func(m *Model) tea.Cmd {
			m.setState(StateHelp)
			return nil
		}},
		{Label: "Other: Quit", Desc: "Exit PeakyPanes", Run: func(m *Model) tea.Cmd {
			return tea.Quit
		}},
	}...)
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func (m *Model) reopenHiddenProject(entry layout.HiddenProjectConfig) tea.Cmd {
	label := hiddenProjectLabel(entry)
	changed, err := m.unhideProjectInConfig(entry)
	if err != nil {
		m.setToast("Reopen failed: "+err.Error(), toastError)
		return nil
	}
	if !changed {
		m.setToast("Project already visible", toastInfo)
		return nil
	}
	m.setToast("Reopened project "+label, toastSuccess)
	return m.refreshCmd()
}

func (m *Model) openQuickReply() tea.Cmd {
	if m.selectedPane() == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	m.setTerminalFocus(false)
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()
	return nil
}

func (m *Model) updateQuickReply(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.paneNext):
		return m, m.cyclePane(1)
	case key.Matches(msg, m.keys.panePrev):
		return m, m.cyclePane(-1)
	}
	switch msg.String() {
	case "enter":
		if strings.TrimSpace(m.quickReplyInput.Value()) == "" {
			return m, m.attachOrStart()
		}
		return m, m.sendQuickReply()
	case "esc":
		m.quickReplyInput.SetValue("")
		m.quickReplyInput.CursorEnd()
		return m, nil
	}
	var cmd tea.Cmd
	m.quickReplyInput, cmd = m.quickReplyInput.Update(msg)
	return m, cmd
}

// nativeFocusedWindow returns the terminal.Window for the selected native pane.
func (m *Model) nativeFocusedWindow() *terminal.Window {
	if m.native == nil {
		return nil
	}
	p := m.selectedPane()
	if p == nil || strings.TrimSpace(p.ID) == "" {
		return nil
	}
	return m.native.PaneWindow(p.ID)
}

func (m *Model) sendNativeKey(msg tea.KeyMsg) error {
	if m.native == nil {
		return errors.New("native manager unavailable")
	}
	pane := m.selectedPane()
	if pane == nil || strings.TrimSpace(pane.ID) == "" {
		return errors.New("no pane selected")
	}
	payload := encodeKeyMsg(msg)
	if len(payload) == 0 {
		return nil
	}
	return m.native.SendInput(pane.ID, payload)
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
	label := strings.TrimSpace(pane.Title)
	if label == "" {
		label = fmt.Sprintf("pane %s", pane.Index)
	}
	return func() tea.Msg {
		if m.native == nil {
			return ErrorMsg{Err: errors.New("native manager unavailable"), Context: "send to pane"}
		}
		if err := m.native.SendInput(pane.ID, []byte(text)); err != nil {
			return ErrorMsg{Err: err, Context: "send to pane"}
		}
		if err := m.native.SendInput(pane.ID, []byte{'\r'}); err != nil {
			return ErrorMsg{Err: err, Context: "send to pane"}
		}
		if label != "" {
			return SuccessMsg{Message: "Sent to " + label}
		}
		return SuccessMsg{Message: "Sent"}
	}
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

func (m *Model) projectIndexFor(name string) (int, bool) {
	for i := range m.data.Projects {
		if m.data.Projects[i].Name == name {
			return i, true
		}
	}
	return -1, false
}

func (m *Model) dashboardProjectIndex(columns []DashboardProjectColumn) int {
	if len(columns) == 0 {
		return -1
	}
	if m.selection.Project != "" {
		for i, column := range columns {
			if column.ProjectName == m.selection.Project {
				return i
			}
		}
	}
	if m.selection.Session != "" {
		for i, column := range columns {
			for _, pane := range column.Panes {
				if pane.SessionName == m.selection.Session {
					return i
				}
			}
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

func paneIndex(panes []PaneItem, idx string) int {
	for i := range panes {
		if panes[i].Index == idx {
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
	if m.tab == TabDashboard {
		if project, _ := findProjectForSession(m.data.Projects, m.selection.Session); project != nil {
			return project
		}
	}
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
	if m.tab == TabDashboard {
		if _, session := findProjectForSession(m.data.Projects, m.selection.Session); session != nil {
			return session
		}
		for i := range m.data.Projects {
			if len(m.data.Projects[i].Sessions) > 0 {
				return &m.data.Projects[i].Sessions[0]
			}
		}
		return nil
	}
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

func (m *Model) selectedPane() *PaneItem {
	session := m.selectedSession()
	if session == nil {
		return nil
	}
	window := selectedWindow(session, m.selection.Window)
	if window == nil || len(window.Panes) == 0 {
		return nil
	}
	if m.selection.Pane != "" {
		for i := range window.Panes {
			if window.Panes[i].Index == m.selection.Pane {
				return &window.Panes[i]
			}
		}
	}
	for i := range window.Panes {
		if window.Panes[i].Active {
			return &window.Panes[i]
		}
	}
	return &window.Panes[0]
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

func (m *Model) filteredDashboardColumns(columns []DashboardProjectColumn) []DashboardProjectColumn {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return columns
	}
	filter = strings.ToLower(filter)
	out := make([]DashboardProjectColumn, 0, len(columns))
	for _, column := range columns {
		next := DashboardProjectColumn{
			ProjectName: column.ProjectName,
			ProjectPath: column.ProjectPath,
		}
		for _, pane := range column.Panes {
			if strings.Contains(strings.ToLower(pane.ProjectName), filter) ||
				strings.Contains(strings.ToLower(pane.ProjectPath), filter) ||
				strings.Contains(strings.ToLower(pane.SessionName), filter) ||
				strings.Contains(strings.ToLower(pane.WindowName), filter) ||
				strings.Contains(strings.ToLower(pane.WindowIndex), filter) ||
				strings.Contains(strings.ToLower(pane.Pane.Title), filter) ||
				strings.Contains(strings.ToLower(pane.Pane.Command), filter) {
				next.Panes = append(next.Panes, pane)
			}
		}
		out = append(out, next)
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
