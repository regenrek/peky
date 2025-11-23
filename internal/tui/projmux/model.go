package projmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"

	"github.com/kregenrek/tmuxman/internal/layout"
	"github.com/kregenrek/tmuxman/internal/tmuxctl"
)

// Mode describes whether projmux is running inside or outside tmux.
type Mode int

const (
	ModeOutside Mode = iota
	ModeInside
)

// Status describes the tmux lifecycle state of a project.
type Status int

const (
	StatusStopped Status = iota
	StatusRunning
	StatusCurrent
)

// ViewState represents the current UI view.
type ViewState int

const (
	StateHome ViewState = iota
	StateDetail
	StateLayout
	StateConfirmKill
)

// Project is the runtime representation of a configured project.
type Project struct {
	Name    string
	Session string
	Path    string
	Layout  string
	Status  Status
}

// YAML config structures.

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
	CodexResume toolConfig `yaml:"codex_resume"`
}

type tmuxConfigSection struct {
	Config string `yaml:"config"`
}

type ghosttyConfigSection struct {
	Config string `yaml:"config"`
}

type config struct {
	Tmux     tmuxConfigSection    `yaml:"tmux"`
	Ghostty  ghosttyConfigSection `yaml:"ghostty"`
	Projects []projectConfig      `yaml:"projects"`
	Tools    toolsConfig          `yaml:"tools"`
}

type layoutPreset struct {
	ID          string
	Title       string
	Description string
}

// Model implements tea.Model for projmux.
type Model struct {
	tmux *tmuxctl.Client

	mode  Mode
	state ViewState

	projects []Project
	cursor   int

	// Global toggle for contextual help.
	showHelp bool

	// Last error surfaced in the footer of the active view.
	errorMsg string

	// Config bookkeeping.
	configPath      string
	tools           toolsConfig
	tmuxConfig      string
	ghosttyConfig   string
	snapshotSession string
	snapshot        tmuxctl.SessionSnapshot

	// Detail view.
	detailIndex int

	// Layout builder view.
	layoutPresets     []layoutPreset
	layoutCursor      int
	layoutProjectIdx  int
	layoutReturnState ViewState

	// Confirm kill view.
	confirmKillIndex int
}

// NewModel constructs a projmux model and loads configuration and initial tmux state.
func NewModel(client *tmuxctl.Client, configPath string) (*Model, error) {
	if client == nil {
		return nil, fmt.Errorf("tmux client is required")
	}

	if configPath == "" {
		var err error
		configPath, err = defaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	if err := ensureConfigFile(configPath); err != nil {
		return nil, err
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	m := &Model{
		tmux:              client,
		mode:              detectMode(),
		state:             StateHome,
		configPath:        configPath,
		layoutPresets:     defaultLayoutPresets(),
		layoutProjectIdx:  0,
		layoutReturnState: StateHome,
		confirmKillIndex:  -1,
	}

	tmuxCfgPath := expandPath(strings.TrimSpace(cfg.Tmux.Config))
	if err := m.applyTmuxConfig(tmuxCfgPath); err != nil {
		return nil, err
	}
	m.tmuxConfig = tmuxCfgPath
	m.ghosttyConfig = expandPath(strings.TrimSpace(cfg.Ghostty.Config))
	m.tools = cfg.Tools

	for _, pc := range cfg.Projects {
		p := Project{
			Name:    pc.Name,
			Session: pc.Session,
			Path:    expandPath(pc.Path),
			Layout:  pc.Layout,
			Status:  StatusStopped,
		}
		if strings.TrimSpace(p.Name) == "" && strings.TrimSpace(p.Session) != "" {
			p.Name = p.Session
		}
		if strings.TrimSpace(p.Session) == "" && strings.TrimSpace(p.Name) != "" {
			p.Session = p.Name
		}
		m.projects = append(m.projects, p)
	}

	if err := m.refreshStatuses(); err != nil {
		m.errorMsg = err.Error()
	}

	m.loadSnapshotForIndex(m.cursor)

	return m, nil
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "projmux", "projects.yml"), nil
}

func ensureConfigFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat config %q: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir %q: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(defaultConfigYAML()), 0o644); err != nil {
		return fmt.Errorf("write default config %q: %w", path, err)
	}

	return nil
}

func defaultConfigYAML() string {
	return `# projmux configuration
# Fill in the projects list with the tmux workspaces you want to manage.
# Each project needs a name, session, path, and optional layout preset.
tmux:
  config: ~/.config/tmux/tmux.conf

ghostty:
  config: ~/.config/ghostty/config

projects: []

tools:
  cursor_agent:
    window_name: cursor
    cmd: ""
  codex_new:
    window_name: codex
    cmd: ""
  codex_resume:
    window_name: codex-resume
    cmd: ""
`
}

func detectMode() Mode {
	if os.Getenv("TMUX") != "" || os.Getenv("TMUX_PANE") != "" {
		return ModeInside
	}
	return ModeOutside
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

func (m *Model) applyTmuxConfig(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if _, err := os.Stat(trimmed); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("tmux config %q: %w", trimmed, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.tmux.SourceFile(ctx, trimmed); err != nil {
		return fmt.Errorf("tmux source-file %q: %w", trimmed, err)
	}
	return nil
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}

func defaultLayoutPresets() []layoutPreset {
	return []layoutPreset{
		{
			ID:          "dev-3-pane",
			Title:       "dev-3-pane",
			Description: "dev window: big left, two right panes",
		},
		{
			ID:          "horizontal-2",
			Title:       "horizontal-2",
			Description: "dev window: two horizontal panes",
		},
		{
			ID:          "vertical-2",
			Title:       "vertical-2",
			Description: "dev window: two vertical panes",
		},
		{
			ID:          "logs-split",
			Title:       "logs-split",
			Description: "logs window: main pane + small bottom pane",
		},
	}
}

// refreshStatuses pulls tmux sessions and marks each project as stopped/running/current.
func (m *Model) refreshStatuses() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sessions, err := m.tmux.ListSessions(ctx)
	if err != nil {
		return err
	}

	current, err := m.tmux.CurrentSession(ctx)
	if err != nil {
		return err
	}

	running := make(map[string]struct{}, len(sessions))
	for _, s := range sessions {
		running[s] = struct{}{}
	}

	for i := range m.projects {
		p := &m.projects[i]
		p.Status = StatusStopped
		if _, ok := running[p.Session]; ok {
			if current != "" && p.Session == current {
				p.Status = StatusCurrent
			} else {
				p.Status = StatusRunning
			}
		}
	}

	m.loadSnapshotForIndex(m.cursor)
	return nil
}

func (m *Model) loadSnapshotForIndex(idx int) {
	if idx < 0 || idx >= len(m.projects) {
		m.snapshotSession = ""
		m.snapshot = tmuxctl.SessionSnapshot{}
		return
	}
	session := strings.TrimSpace(m.projects[idx].Session)
	if session == "" {
		m.snapshotSession = ""
		m.snapshot = tmuxctl.SessionSnapshot{}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	snap, err := m.tmux.SessionSnapshot(ctx, session)
	if err != nil {
		m.errorMsg = err.Error()
		return
	}
	m.snapshotSession = session
	m.snapshot = snap
}

// reloadConfig reloads config and refreshes tmux statuses.
func (m *Model) reloadConfig() error {
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return err
	}

	tmuxCfgPath := expandPath(strings.TrimSpace(cfg.Tmux.Config))
	if err := m.applyTmuxConfig(tmuxCfgPath); err != nil {
		return err
	}
	m.tmuxConfig = tmuxCfgPath
	m.ghosttyConfig = expandPath(strings.TrimSpace(cfg.Ghostty.Config))
	m.tools = cfg.Tools
	m.projects = m.projects[:0]

	for _, pc := range cfg.Projects {
		p := Project{
			Name:    pc.Name,
			Session: pc.Session,
			Path:    expandPath(pc.Path),
			Layout:  pc.Layout,
			Status:  StatusStopped,
		}
		if strings.TrimSpace(p.Name) == "" && strings.TrimSpace(p.Session) != "" {
			p.Name = p.Session
		}
		if strings.TrimSpace(p.Session) == "" && strings.TrimSpace(p.Name) != "" {
			p.Session = p.Name
		}
		m.projects = append(m.projects, p)
	}

	m.cursor = 0
	m.detailIndex = 0
	m.layoutProjectIdx = 0
	m.confirmKillIndex = -1

	return m.refreshStatuses()
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateHome:
			return m.updateHomeKey(msg)
		case StateDetail:
			return m.updateDetailKey(msg)
		case StateLayout:
			return m.updateLayoutKey(msg)
		case StateConfirmKill:
			return m.updateConfirmKey(msg)
		}
	case tea.WindowSizeMsg:
		// For now we do not store or use the size; the layout is simple.
		_ = msg
	}
	return m, nil
}

func (m *Model) View() string {
	switch m.state {
	case StateHome:
		return m.viewHome()
	case StateDetail:
		return m.viewDetail()
	case StateLayout:
		return m.viewLayout()
	case StateConfirmKill:
		return m.viewConfirmKill()
	default:
		return "projmux: unknown state\n"
	}
}

// --- Key handling helpers ---

func (m *Model) updateHomeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.loadSnapshotForIndex(m.cursor)
		}
	case "down", "j":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
			m.loadSnapshotForIndex(m.cursor)
		}
	case "left", "h":
		if m.cursor > 0 {
			m.cursor--
			m.loadSnapshotForIndex(m.cursor)
		}
	case "right", "l":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
			m.loadSnapshotForIndex(m.cursor)
		}
	case "enter":
		m.errorMsg = ""
		m.attachProjectByIndex(m.cursor)
	case "w":
		m.errorMsg = ""
		m.openGhosttyForIndex(m.cursor)
	case "d":
		if len(m.projects) > 0 {
			m.detailIndex = m.cursor
			m.state = StateDetail
			m.showHelp = false
		}
	case "L":
		if len(m.projects) > 0 {
			m.layoutProjectIdx = m.cursor
			m.layoutCursor = m.findPresetIndex(m.projects[m.layoutProjectIdx].Layout)
			m.layoutReturnState = StateHome
			m.state = StateLayout
			m.showHelp = false
		}
	case "K":
		if len(m.projects) > 0 && m.projects[m.cursor].Status != StatusStopped {
			m.confirmKillIndex = m.cursor
			m.state = StateConfirmKill
			m.showHelp = false
		}
	case "r":
		if err := m.reloadConfig(); err != nil {
			m.errorMsg = err.Error()
		} else {
			m.errorMsg = ""
		}
	case "?":
		m.showHelp = !m.showHelp
	}
	return m, nil
}

func (m *Model) updateDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "b", "esc":
		m.state = StateHome
		m.showHelp = false
	case "?":
		m.showHelp = !m.showHelp
	case "a":
		m.errorMsg = ""
		m.attachProjectByIndex(m.detailIndex)
	case "w":
		m.errorMsg = ""
		m.openGhosttyForIndex(m.detailIndex)
	case "L":
		if len(m.projects) > 0 {
			m.layoutProjectIdx = m.detailIndex
			m.layoutCursor = m.findPresetIndex(m.projects[m.layoutProjectIdx].Layout)
			m.layoutReturnState = StateDetail
			m.state = StateLayout
			m.showHelp = false
		}
	case "K":
		if len(m.projects) > 0 && m.projects[m.detailIndex].Status != StatusStopped {
			m.confirmKillIndex = m.detailIndex
			m.state = StateConfirmKill
			m.showHelp = false
		}
	case "A":
		if len(m.projects) > 0 {
			m.runToolForProject(m.projects[m.detailIndex], m.tools.CursorAgent)
		}
	case "C":
		if len(m.projects) > 0 {
			m.runToolForProject(m.projects[m.detailIndex], m.tools.CodexNew)
		}
	case "R":
		if len(m.projects) > 0 {
			m.runToolForProject(m.projects[m.detailIndex], m.tools.CodexResume)
		}
	}
	return m, nil
}

func (m *Model) updateLayoutKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.layoutCursor > 0 {
			m.layoutCursor--
		}
	case "down", "j":
		if m.layoutCursor < len(m.layoutPresets)-1 {
			m.layoutCursor++
		}
	case "enter":
		if len(m.layoutPresets) > 0 && m.layoutProjectIdx >= 0 && m.layoutProjectIdx < len(m.projects) {
			preset := m.layoutPresets[m.layoutCursor]
			m.applyLayout(m.layoutProjectIdx, preset)
		}
	case "b", "esc":
		m.state = m.layoutReturnState
		m.showHelp = false
	case "?":
		m.showHelp = !m.showHelp
	}
	return m, nil
}

func (m *Model) updateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc", "n":
		m.state = StateHome
		m.showHelp = false
	case "y", "enter":
		m.errorMsg = ""
		m.killSessionByIndex(m.confirmKillIndex)
		m.state = StateHome
		m.showHelp = false
	case "?":
		m.showHelp = !m.showHelp
	}
	return m, nil
}

// --- Actions ---

func (m *Model) attachProjectByIndex(idx int) {
	if idx < 0 || idx >= len(m.projects) {
		return
	}
	p := m.projects[idx]
	if strings.TrimSpace(p.Session) == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.tmux.EnsureSession(ctx, tmuxctl.Options{
		Session:  p.Session,
		Layout:   layout.Grid{Rows: 1, Columns: 1},
		StartDir: p.Path,
		Attach:   true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		m.errorMsg = err.Error()
	} else {
		_ = m.refreshStatuses()
	}
}

func (m *Model) openGhosttyForIndex(idx int) {
	if idx < 0 || idx >= len(m.projects) {
		return
	}
	p := m.projects[idx]
	if strings.TrimSpace(p.Session) == "" || strings.TrimSpace(p.Path) == "" {
		return
	}
	dir := expandPath(p.Path)
	args := []string{"ghostty"}
	if cfg := strings.TrimSpace(m.ghosttyConfig); cfg != "" {
		args = append(args, "--config", cfg)
	}
	args = append(args,
		"--working-directory", dir,
		"-e", "tmux", "new-session", "-A", "-s", p.Session,
	)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		m.errorMsg = fmt.Sprintf("ghostty: %v", err)
	}
}

func (m *Model) runToolForProject(p Project, tool toolConfig) {
	if strings.TrimSpace(tool.Cmd) == "" {
		m.errorMsg = "tool command not configured"
		return
	}
	if strings.TrimSpace(p.Session) == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dir := expandPath(p.Path)
	_, err := m.tmux.EnsureSession(ctx, tmuxctl.Options{
		Session:  p.Session,
		Layout:   layout.Grid{Rows: 1, Columns: 1},
		StartDir: dir,
		Attach:   false,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		m.errorMsg = err.Error()
		return
	}

	if err := m.tmux.NewWindow(ctx, p.Session, tool.WindowName, dir, tool.Cmd); err != nil {
		m.errorMsg = err.Error()
		return
	}

	_ = m.refreshStatuses()
}

func (m *Model) killSessionByIndex(idx int) {
	if idx < 0 || idx >= len(m.projects) {
		return
	}
	p := m.projects[idx]
	if strings.TrimSpace(p.Session) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.tmux.KillSession(ctx, p.Session); err != nil {
		m.errorMsg = err.Error()
	} else {
		_ = m.refreshStatuses()
	}
}

func (m *Model) applyLayout(projectIndex int, preset layoutPreset) {
	if projectIndex < 0 || projectIndex >= len(m.projects) {
		return
	}
	p := m.projects[projectIndex]
	if strings.TrimSpace(p.Session) == "" {
		return
	}
	dir := expandPath(p.Path)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure the session exists but do not attach.
	_, err := m.tmux.EnsureSession(ctx, tmuxctl.Options{
		Session:  p.Session,
		Layout:   layout.Grid{Rows: 1, Columns: 1},
		StartDir: dir,
		Attach:   false,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		m.errorMsg = err.Error()
		return
	}

	switch preset.ID {
	case "dev-3-pane":
		m.applyDev3Pane(ctx, p.Session, dir)
	case "horizontal-2":
		m.applyHorizontal2(ctx, p.Session, dir)
	case "vertical-2":
		m.applyVertical2(ctx, p.Session, dir)
	case "logs-split":
		m.applyLogsSplit(ctx, p.Session, dir)
	default:
		m.errorMsg = fmt.Sprintf("unknown layout preset %q", preset.ID)
		return
	}

	m.projects[projectIndex].Layout = preset.ID
}

// dev-3-pane: window "dev" with editor big left, two stacked panes on the right.
func (m *Model) applyDev3Pane(ctx context.Context, session, dir string) {
	window := "dev"
	_ = m.tmux.KillWindow(ctx, session, window)
	if err := m.tmux.NewWindow(ctx, session, window, dir, ""); err != nil {
		m.errorMsg = err.Error()
		return
	}
	target := fmt.Sprintf("%s:%s", session, window)
	// Split left/right.
	if err := m.tmux.SplitWindow(ctx, target, dir, false, 0); err != nil {
		m.errorMsg = err.Error()
		return
	}
	// Split the right-hand pane into top/bottom.
	rightTarget := fmt.Sprintf("%s:%s.1", session, window)
	if err := m.tmux.SplitWindow(ctx, rightTarget, dir, true, 0); err != nil {
		m.errorMsg = err.Error()
		return
	}
}

// horizontal-2: window "dev" with two horizontal (top/bottom) panes.
func (m *Model) applyHorizontal2(ctx context.Context, session, dir string) {
	window := "dev"
	_ = m.tmux.KillWindow(ctx, session, window)
	if err := m.tmux.NewWindow(ctx, session, window, dir, ""); err != nil {
		m.errorMsg = err.Error()
		return
	}
	target := fmt.Sprintf("%s:%s", session, window)
	if err := m.tmux.SplitWindow(ctx, target, dir, true, 0); err != nil {
		m.errorMsg = err.Error()
		return
	}
}

// vertical-2: window "dev" with two vertical (left/right) panes.
func (m *Model) applyVertical2(ctx context.Context, session, dir string) {
	window := "dev"
	_ = m.tmux.KillWindow(ctx, session, window)
	if err := m.tmux.NewWindow(ctx, session, window, dir, ""); err != nil {
		m.errorMsg = err.Error()
		return
	}
	target := fmt.Sprintf("%s:%s", session, window)
	if err := m.tmux.SplitWindow(ctx, target, dir, false, 0); err != nil {
		m.errorMsg = err.Error()
		return
	}
}

// logs-split: window "logs" with main pane and small bottom pane.
func (m *Model) applyLogsSplit(ctx context.Context, session, dir string) {
	window := "logs"
	_ = m.tmux.KillWindow(ctx, session, window)
	if err := m.tmux.NewWindow(ctx, session, window, dir, ""); err != nil {
		m.errorMsg = err.Error()
		return
	}
	target := fmt.Sprintf("%s:%s", session, window)
	if err := m.tmux.SplitWindow(ctx, target, dir, true, 25); err != nil {
		m.errorMsg = err.Error()
		return
	}
}

// --- Small helpers for view rendering ---

func (m *Model) modeLabel() string {
	if m.mode == ModeInside {
		return "inside tmux"
	}
	return "outside tmux"
}

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

func statusTag(s Status) string {
	switch s {
	case StatusCurrent:
		return "[current]"
	case StatusRunning:
		return "[running]"
	case StatusStopped:
		return "[stopped]"
	default:
		return ""
	}
}

func shortenPath(p string) string {
	if p == "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		rel := strings.TrimPrefix(p, home)
		if rel == "" {
			return "~"
		}
		rel = strings.TrimPrefix(rel, string(os.PathSeparator))
		return filepath.Join("~", rel)
	}
	return p
}

func (m *Model) findPresetIndex(layoutID string) int {
	if layoutID == "" {
		return 0
	}
	for i, p := range m.layoutPresets {
		if p.ID == layoutID {
			return i
		}
	}
	return 0
}

// --- Views ---

func (m *Model) viewHome() string {
	var b strings.Builder
	fmt.Fprintf(&b, "projmux  [%s]\n\n", m.modeLabel())
	b.WriteString(m.renderNavBar())
	b.WriteString("\n")
	b.WriteString(m.renderSnapshotView())

	if m.errorMsg != "" {
		fmt.Fprintf(&b, "\nError: %s\n", m.errorMsg)
	}

	if m.showHelp {
		b.WriteString("\nHome keys:\n")
		b.WriteString("  ←/h, →/l  cycle projects\n")
		b.WriteString("  ↑/k, ↓/j  move list selection\n")
		b.WriteString("  enter     attach/switch to project session\n")
		b.WriteString("  d         project detail view\n")
		b.WriteString("  L         layout builder for project\n")
		b.WriteString("  w         open Ghostty window\n")
		b.WriteString("  r         reload config and tmux sessions\n")
		b.WriteString("  K         kill tmux session (with confirm)\n")
		b.WriteString("  ?         toggle this help\n")
		b.WriteString("  q         quit\n")
	} else {
		b.WriteString("\n←/→ cycle  ↑/↓ move  enter attach  d details  L layout  w ghostty  r reload  K kill  q quit  ? help\n")
	}

	return b.String()
}

func (m *Model) renderNavBar() string {
	if len(m.projects) == 0 {
		return "No projects configured."
	}
	var parts []string
	for i, p := range m.projects {
		label := fmt.Sprintf("%s %s", statusIcon(p.Status), p.Name)
		if i == m.cursor {
			label = fmt.Sprintf("[%s]", label)
		} else {
			label = fmt.Sprintf(" %s ", label)
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
}

func (m *Model) renderSnapshotView() string {
	if len(m.projects) == 0 {
		return ""
	}
	selected := m.projects[m.cursor]
	session := strings.TrimSpace(selected.Session)
	if session == "" {
		return "No session configured for selected project."
	}
	if session != m.snapshotSession {
		return fmt.Sprintf("Snapshot unavailable for %s.", session)
	}
	if len(m.snapshot.Windows) == 0 {
		return fmt.Sprintf("Session %s has no windows yet.", session)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Session: %s (%s)\n", session, shortenPath(selected.Path))
	for _, win := range m.snapshot.Windows {
		indicator := " "
		if win.Active {
			indicator = "◆"
		}
		fmt.Fprintf(&b, " %s Window %s (%s)\n", indicator, win.Index, win.Name)
		for _, pane := range win.Panes {
			paneIndicator := " "
			if pane.Active {
				paneIndicator = "→"
			}
			fmt.Fprintf(&b, "    %s pane %s  %s\n", paneIndicator, pane.Index, pane.Title)
		}
	}
	return b.String()
}

func (m *Model) viewDetail() string {
	if len(m.projects) == 0 {
		return "No projects configured. Press b to go back.\n"
	}
	if m.detailIndex < 0 || m.detailIndex >= len(m.projects) {
		m.detailIndex = 0
	}
	p := m.projects[m.detailIndex]

	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s  %s\n\n", p.Name, statusTag(p.Status), shortenPath(p.Path))

	b.WriteString("Session:\n")
	fmt.Fprintf(&b, "  a attach / switch to %s\n", p.Session)
	b.WriteString("  w open Ghostty window\n")
	b.WriteString("  L open layout builder\n")
	b.WriteString("  K kill session\n\n")

	b.WriteString("Tools:\n")
	b.WriteString("  A run cursor-agent\n")
	b.WriteString("  C run codex\n")
	b.WriteString("  R run codex resume\n\n")

	b.WriteString("Other:\n")
	b.WriteString("  b back to Home\n")
	b.WriteString("  q quit\n")

	if m.errorMsg != "" {
		fmt.Fprintf(&b, "\nError: %s\n", m.errorMsg)
	}

	if m.showHelp {
		b.WriteString("\nDetail keys:\n")
		b.WriteString("  a         attach / switch\n")
		b.WriteString("  w         Ghostty window\n")
		b.WriteString("  L         layout builder\n")
		b.WriteString("  K         kill session\n")
		b.WriteString("  A/C/R     cursor-agent, codex, codex resume\n")
		b.WriteString("  b         back to Home\n")
		b.WriteString("  ?         toggle this help\n")
		b.WriteString("  q         quit\n")
	} else {
		b.WriteString("\nKeys: a attach  w ghostty  L layout  K kill  A cursor  C codex  R resume  b back  q quit  ? help\n")
	}

	return b.String()
}

func (m *Model) viewLayout() string {
	if len(m.projects) == 0 || len(m.layoutPresets) == 0 {
		return "No layouts or projects configured.\n"
	}
	if m.layoutProjectIdx < 0 || m.layoutProjectIdx >= len(m.projects) {
		m.layoutProjectIdx = 0
	}
	p := m.projects[m.layoutProjectIdx]

	var b strings.Builder
	fmt.Fprintf(&b, "Layout presets for %s (%s)\n\n", p.Name, p.Session)

	for i, preset := range m.layoutPresets {
		cursor := " "
		if i == m.layoutCursor {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %-14s - %s\n", cursor, preset.Title, preset.Description)
	}

	if m.errorMsg != "" {
		fmt.Fprintf(&b, "\nError: %s\n", m.errorMsg)
	}

	if m.showHelp {
		b.WriteString("\nLayout keys:\n")
		b.WriteString("  ↑/k, ↓/j  move selection\n")
		b.WriteString("  enter     apply selected layout\n")
		b.WriteString("  b/esc     back\n")
		b.WriteString("  ?         toggle this help\n")
		b.WriteString("  q         quit\n")
	} else {
		b.WriteString("\n↑/↓ move  enter apply  b back  ? help  q quit\n")
	}

	return b.String()
}

func (m *Model) viewConfirmKill() string {
	var b strings.Builder
	b.WriteString("Kill tmux session?\n\n")
	if m.confirmKillIndex >= 0 && m.confirmKillIndex < len(m.projects) {
		p := m.projects[m.confirmKillIndex]
		fmt.Fprintf(&b, "Session: %s (%s)\n", p.Session, p.Name)
	}
	if m.errorMsg != "" {
		fmt.Fprintf(&b, "\nError: %s\n", m.errorMsg)
	}
	if m.showHelp {
		b.WriteString("\nConfirm keys:\n")
		b.WriteString("  y/enter   kill session\n")
		b.WriteString("  n/esc/q   cancel and return to Home\n")
		b.WriteString("  ?         toggle this help\n")
	} else {
		b.WriteString("\nConfirm: y yes  n no  (? help)\n")
	}
	return b.String()
}
