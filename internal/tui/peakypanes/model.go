package peakypanes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/kregenrek/tmuxman/internal/layout"
	"github.com/kregenrek/tmuxman/internal/tmuxctl"
)

// ViewState represents the current UI view.
type ViewState int

const (
	StateHome ViewState = iota
	StateProjectPicker
	StateConfirmKill
)

// GitProject represents a project directory with .git
type GitProject struct {
	Name string
	Path string
}

func (g GitProject) Title() string       { return "📁 " + g.Name }
func (g GitProject) Description() string { return shortenPath(g.Path) }
func (g GitProject) FilterValue() string { return g.Name }

// Status describes the tmux lifecycle state of a project.
type Status int

const (
	StatusStopped Status = iota
	StatusRunning
	StatusCurrent
)

// Project represents a configured project.
type Project struct {
	Name    string
	Session string
	Path    string
	Layout  string
	Status  Status
}

// Implement list.Item interface for Project
func (p Project) Title() string {
	icon := statusIcon(p.Status)
	return fmt.Sprintf("%s %s", icon, p.Name)
}

func (p Project) Description() string {
	if p.Path != "" {
		return shortenPath(p.Path)
	}
	return "No path configured"
}

func (p Project) FilterValue() string { return p.Name }

// Config structures for YAML.
type projectConfig struct {
	Name    string `yaml:"name"`
	Session string `yaml:"session"`
	Path    string `yaml:"path"`
	Layout  string `yaml:"layout"`
}

type toolConfig struct {
	Cmd        string `yaml:"cmd"`
	WindowName string `yaml:"window_name"`
}

type toolsConfig struct {
	CursorAgent toolConfig `yaml:"cursor_agent"`
	CodexNew    toolConfig `yaml:"codex_new"`
}

type config struct {
	Tmux struct {
		Config string `yaml:"config"`
	} `yaml:"tmux"`
	Ghostty struct {
		Config string `yaml:"config"`
	} `yaml:"ghostty"`
	Projects   []projectConfig `yaml:"projects"`
	Tools      toolsConfig     `yaml:"tools"`
	LayoutDirs []string        `yaml:"layout_dirs"`
}

// Styles
var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"})

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("210")).
			Padding(1, 2)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("210"))
)

// Key bindings
type delegateKeyMap struct {
	choose key.Binding
	kill   key.Binding
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "attach/start"),
		),
		kill: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "kill session"),
		),
	}
}

func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{d.choose, d.kill}
}

func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{d.choose, d.kill}}
}

type listKeyMap struct {
	openProject key.Binding
	refresh     key.Binding
	editConfig  key.Binding
	toggleHelp  key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		openProject: key.NewBinding(
			key.WithKeys("o", "n"),
			key.WithHelp("o/n", "open project"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		editConfig: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit config"),
		),
		toggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// Model implements tea.Model for peakypanes TUI.
type Model struct {
	tmux   *tmuxctl.Client
	loader *layout.Loader

	state  ViewState
	width  int
	height int

	// List view
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	projects     []Project

	// Project picker view
	projectPicker list.Model
	gitProjects   []GitProject

	// Confirm kill dialog
	confirmProject *Project

	// Config
	configPath string
	tools      toolsConfig

	// Status
	insideTmux bool

	// Snapshot for selected project
	snapshot        tmuxctl.SessionSnapshot
	snapshotSession string
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

	// Create loader for layouts
	loader, err := layout.NewLoader()
	if err != nil {
		return nil, err
	}

	if err := loader.LoadAll(); err != nil {
		// Non-fatal, continue with empty layouts
	}

	m := &Model{
		tmux:         client,
		loader:       loader,
		configPath:   configPath,
		state:        StateHome,
		insideTmux:   os.Getenv("TMUX") != "",
		keys:         newListKeyMap(),
		delegateKeys: newDelegateKeyMap(),
	}

	// Load config and projects
	if err := m.loadConfig(); err != nil {
		// Non-fatal, continue with empty projects
	}

	// Refresh tmux session statuses
	_ = m.refreshStatuses()

	// Setup list
	m.setupList()

	// Setup project picker
	m.setupProjectPicker()

	return m, nil
}

func (m *Model) setupList() {
	delegate := list.NewDefaultDelegate()
	delegate.UpdateFunc = m.delegateUpdate
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{m.delegateKeys.choose, m.delegateKeys.kill}
	}
	delegate.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{{m.delegateKeys.choose, m.delegateKeys.kill}}
	}

	// Custom styles for the delegate
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FFFDF5")).
		BorderLeftForeground(lipgloss.Color("#7D56F4"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#B8B8B8")).
		BorderLeftForeground(lipgloss.Color("#7D56F4"))

	items := m.projectsToItems()
	l := list.New(items, delegate, 0, 0)
	l.Title = "🎩 Peaky Panes"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			m.keys.openProject,
			m.keys.refresh,
			m.keys.editConfig,
		}
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			m.keys.openProject,
			m.keys.refresh,
		}
	}

	// Set status bar info
	modeLabel := "outside tmux"
	if m.insideTmux {
		modeLabel = "inside tmux"
	}
	l.SetStatusBarItemName("session", "sessions")
	l.NewStatusMessage(statusMessageStyle.Render(fmt.Sprintf("[%s]", modeLabel)))

	m.list = l
}

func (m *Model) setupProjectPicker() {
	// Scan for git projects
	m.scanGitProjects()

	// Create delegate for project picker
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FFFDF5")).
		BorderLeftForeground(lipgloss.Color("#25A065"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#B8B8B8")).
		BorderLeftForeground(lipgloss.Color("#25A065"))

	items := m.gitProjectsToItems()
	l := list.New(items, delegate, 0, 0)
	l.Title = "📁 Open Project"
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(lipgloss.Color("#25A065")).
		Padding(0, 1)
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

	// Walk through all directories recursively
	_ = filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip hidden directories entirely
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip node_modules, vendor, etc.
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == "__pycache__" || name == ".venv" || name == "venv" {
				return filepath.SkipDir
			}
		}

		// Check if this directory has a .git folder
		if d.IsDir() && d.Name() != ".git" {
			gitPath := filepath.Join(path, ".git")
			if _, err := os.Stat(gitPath); err == nil {
				// Get relative path from projects dir for a nicer name
				relPath, _ := filepath.Rel(projectsDir, path)
				m.gitProjects = append(m.gitProjects, GitProject{
					Name: relPath,
					Path: path,
				})
				// Don't descend into this directory's subdirectories
				// (nested git repos are handled by git submodules, not separate projects)
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

func (m *Model) projectsToItems() []list.Item {
	items := make([]list.Item, len(m.projects))
	for i, p := range m.projects {
		items[i] = p
	}
	return items
}

func (m *Model) delegateUpdate(msg tea.Msg, lm *list.Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.delegateKeys.choose):
			if item, ok := lm.SelectedItem().(Project); ok {
				if item.Status == StatusStopped {
					// Start the session
					return m.startProject(item)
				}
				// Attach to running session
				return m.attachProject(item)
			}

		case key.Matches(msg, m.delegateKeys.kill):
			if item, ok := lm.SelectedItem().(Project); ok {
				if item.Status != StatusStopped {
					m.confirmProject = &item
					m.state = StateConfirmKill
				} else {
					return lm.NewStatusMessage(statusMessageStyle.Render("Session not running"))
				}
			}
		}
	}
	return nil
}

func (m *Model) loadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config yet
		}
		return err
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	m.tools = cfg.Tools
	m.projects = nil

	for _, pc := range cfg.Projects {
		p := Project{
			Name:    pc.Name,
			Session: pc.Session,
			Path:    expandPath(pc.Path),
			Layout:  pc.Layout,
			Status:  StatusStopped,
		}
		if p.Name == "" && p.Session != "" {
			p.Name = p.Session
		}
		if p.Session == "" && p.Name != "" {
			p.Session = sanitizeSessionName(p.Name)
		}
		m.projects = append(m.projects, p)
	}

	return nil
}

func (m *Model) refreshStatuses() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sessions, err := m.tmux.ListSessions(ctx)
	if err != nil {
		return err
	}

	current, _ := m.tmux.CurrentSession(ctx)

	// Build a set of running sessions for quick lookup
	runningSessions := make(map[string]bool)
	for _, s := range sessions {
		runningSessions[s] = true
	}

	// Separate configured projects from dynamically discovered ones
	var configuredProjects []Project
	configuredSessions := make(map[string]bool)

	for _, p := range m.projects {
		// Keep only projects that have a Path (configured) or are still running
		if p.Path != "" {
			// This is a configured project - always keep it
			configuredProjects = append(configuredProjects, p)
			configuredSessions[p.Session] = true
		}
		// Dynamically discovered projects (no Path) will be re-added if still running
	}

	// Start fresh with configured projects
	m.projects = configuredProjects

	// Update status for configured projects
	for i := range m.projects {
		p := &m.projects[i]
		p.Status = StatusStopped
		if runningSessions[p.Session] {
			if p.Session == current {
				p.Status = StatusCurrent
			} else {
				p.Status = StatusRunning
			}
		}
	}

	// Add running sessions that are NOT in configured projects
	for _, s := range sessions {
		if !configuredSessions[s] {
			status := StatusRunning
			if s == current {
				status = StatusCurrent
			}
			m.projects = append(m.projects, Project{
				Name:    s,
				Session: s,
				Path:    "", // Unknown path for unconfigured sessions
				Layout:  "",
				Status:  status,
			})
		}
	}

	return nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		// Reserve space for logo (4 lines) at the top
		logoHeight := len(Logo) + 2
		m.list.SetSize(msg.Width-h, msg.Height-v-logoHeight)
		m.projectPicker.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case StateHome:
			return m.updateHome(msg)
		case StateProjectPicker:
			return m.updateProjectPicker(msg)
		case StateConfirmKill:
			return m.updateConfirmKill(msg)
		}
	}

	// Pass other messages to the appropriate component
	switch m.state {
	case StateHome:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case StateProjectPicker:
		var cmd tea.Cmd
		m.projectPicker, cmd = m.projectPicker.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Don't process keys while filtering
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.openProject):
		// Open project picker
		m.scanGitProjects()
		m.projectPicker.SetItems(m.gitProjectsToItems())
		m.state = StateProjectPicker
		return m, nil

	case key.Matches(msg, m.keys.refresh):
		if err := m.loadConfig(); err != nil {
			return m, m.list.NewStatusMessage(statusMessageStyle.Render("Error: " + err.Error()))
		}
		if err := m.refreshStatuses(); err != nil {
			return m, m.list.NewStatusMessage(statusMessageStyle.Render("Error: " + err.Error()))
		}
		m.list.SetItems(m.projectsToItems())
		return m, m.list.NewStatusMessage(statusMessageStyle.Render("✓ Refreshed"))

	case key.Matches(msg, m.keys.editConfig):
		return m, m.editConfig()

	case msg.String() == "q", msg.String() == "ctrl+c":
		return m, tea.Quit
	}

	// Pass to list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateProjectPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Don't process keys while filtering
	if m.projectPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.projectPicker, cmd = m.projectPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.state = StateHome
		return m, nil

	case "enter":
		// Select the project and start a session
		if item, ok := m.projectPicker.SelectedItem().(GitProject); ok {
			m.state = StateHome
			return m, m.startSessionAtPath(item.Path)
		}
		m.state = StateHome
		return m, nil

	case "q":
		// Only quit if not filtering
		m.state = StateHome
		return m, nil
	}

	var cmd tea.Cmd
	m.projectPicker, cmd = m.projectPicker.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmProject != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			session := m.confirmProject.Session
			if err := m.tmux.KillSession(ctx, session); err != nil {
				m.state = StateHome
				return m, m.list.NewStatusMessage(statusMessageStyle.Render("Error: " + err.Error()))
			}
			_ = m.refreshStatuses()
			m.list.SetItems(m.projectsToItems())
			m.confirmProject = nil
			m.state = StateHome
			return m, m.list.NewStatusMessage(statusMessageStyle.Render(fmt.Sprintf("✓ Killed session %s", session)))
		}
		m.state = StateHome
		return m, nil

	case "n", "esc":
		m.confirmProject = nil
		m.state = StateHome
		return m, nil
	}

	return m, nil
}

func (m Model) attachProject(p Project) tea.Cmd {
	session := p.Session

	// If inside tmux, use switch-client; otherwise use attach
	if m.insideTmux {
		return tea.ExecProcess(
			exec.Command("tmux", "switch-client", "-t", session),
			func(err error) tea.Msg {
				return nil
			},
		)
	}

	return tea.ExecProcess(
		exec.Command("tmux", "attach-session", "-t", session),
		func(err error) tea.Msg {
			return nil
		},
	)
}

func (m Model) startProject(p Project) tea.Cmd {
	// Start session using peakypanes start
	args := []string{"start", "--session", p.Session}
	if p.Path != "" {
		args = append(args, "--path", p.Path)
	}
	if p.Layout != "" {
		args = append(args, "--layout", p.Layout)
	}

	return tea.ExecProcess(
		exec.Command("peakypanes", args...),
		func(err error) tea.Msg {
			return nil
		},
	)
}

func (m Model) startSessionAtPath(path string) tea.Cmd {
	return tea.ExecProcess(
		exec.Command("peakypanes", "start", "--path", path),
		func(err error) tea.Msg {
			return nil
		},
	)
}

func (m Model) editConfig() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	return tea.ExecProcess(
		exec.Command(editor, m.configPath),
		func(err error) tea.Msg {
			return nil
		},
	)
}

func (m Model) View() string {
	switch m.state {
	case StateHome:
		return m.viewHome()
	case StateProjectPicker:
		return appStyle.Render(m.projectPicker.View())
	case StateConfirmKill:
		return m.viewConfirmKill()
	default:
		return m.viewHome()
	}
}

func (m Model) viewHome() string {
	var s strings.Builder

	// Logo at the top
	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFF99")).
		Bold(true)
	for _, line := range Logo {
		s.WriteString(logoStyle.Render(line))
		s.WriteString("\n")
	}
	s.WriteString("\n")

	// List view
	s.WriteString(m.list.View())

	return appStyle.Render(s.String())
}

func (m Model) viewConfirmKill() string {
	// Render list view dimmed in background
	listView := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(m.list.View())

	// Dialog content
	var dialogContent strings.Builder

	dialogContent.WriteString(dialogTitleStyle.Render("⚠️  Kill Session?"))
	dialogContent.WriteString("\n\n")

	if m.confirmProject != nil {
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

		dialogContent.WriteString(labelStyle.Render("Session: "))
		dialogContent.WriteString(valueStyle.Render(m.confirmProject.Session))
		dialogContent.WriteString("\n")
		dialogContent.WriteString(labelStyle.Render("Project: "))
		dialogContent.WriteString(valueStyle.Render(m.confirmProject.Name))
		dialogContent.WriteString("\n\n")
	}

	noteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	dialogContent.WriteString(noteStyle.Render("Kill the session: Notice this won't delete your project"))
	dialogContent.WriteString("\n\n")

	choiceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	dialogContent.WriteString(choiceStyle.Render("y"))
	dialogContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(" confirm • "))
	dialogContent.WriteString(choiceStyle.Render("n"))
	dialogContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())

	// Combine list and dialog
	return appStyle.Render(listView + "\n\n" + dialog)
}

// Helper functions

func statusIcon(s Status) string {
	switch s {
	case StatusCurrent:
		return "◆"
	case StatusRunning:
		return "●"
	case StatusStopped:
		return "○"
	default:
		return "?"
	}
}

func expandPath(p string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			if strings.HasPrefix(p, "~/") {
				return filepath.Join(home, p[2:])
			}
		}
	}
	return p
}

func shortenPath(p string) string {
	if p == "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func sanitizeSessionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "session"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "session"
	}
	return result
}
