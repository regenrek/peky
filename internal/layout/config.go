package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// PaneDef defines a single pane within a layout.
type PaneDef struct {
	Title   string   `yaml:"title,omitempty"`
	Cmd     string   `yaml:"cmd,omitempty"`
	Size    string   `yaml:"size,omitempty"`    // e.g., "50%", "30"
	Split   string   `yaml:"split,omitempty"`   // "horizontal" or "vertical"
	Setup   []string `yaml:"setup,omitempty"`   // commands to run before main cmd
	Enabled string   `yaml:"enabled,omitempty"` // expression like "${VAR:-true}"
}

// LayoutSettings contains optional layout configuration.
type LayoutSettings struct {
	Width  int `yaml:"width,omitempty"`
	Height int `yaml:"height,omitempty"`
}

// LayoutConfig represents a complete layout definition.
type LayoutConfig struct {
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty"`
	Settings    LayoutSettings    `yaml:"settings,omitempty"`
	// Grid layouts (optional). If Grid is set, Panes is ignored.
	Grid     string    `yaml:"grid,omitempty"`     // e.g., "2x3"
	Command  string    `yaml:"command,omitempty"`  // run in every pane
	Commands []string  `yaml:"commands,omitempty"` // per-pane commands (row-major)
	Titles   []string  `yaml:"titles,omitempty"`   // optional per-pane titles (row-major)
	Panes    []PaneDef `yaml:"panes,omitempty"`
}

// ProjectConfig represents a project entry in the config file.
type ProjectConfig struct {
	Name    string `yaml:"name"`
	Session string `yaml:"session"`
	Path    string `yaml:"path"`
	// Layout can be a string (reference) or inline LayoutConfig
	Layout interface{}       `yaml:"layout,omitempty"`
	Vars   map[string]string `yaml:"vars,omitempty"`
}

// ToolConfig defines an external tool command.
type ToolConfig struct {
	Cmd       string `yaml:"cmd"`
	PaneTitle string `yaml:"pane_title,omitempty"`
}

// ToolsConfig groups tool definitions.
type ToolsConfig struct {
	CursorAgent ToolConfig `yaml:"cursor_agent,omitempty"`
	CodexNew    ToolConfig `yaml:"codex_new,omitempty"`
	CodexResume ToolConfig `yaml:"codex_resume,omitempty"`
}

// StatusRegexConfig holds regex patterns for dashboard status detection.
type StatusRegexConfig struct {
	Success string `yaml:"success,omitempty"`
	Error   string `yaml:"error,omitempty"`
	Running string `yaml:"running,omitempty"`
}

// AgentDetectionConfig enables agent-specific status detection.
type AgentDetectionConfig struct {
	Codex  *bool `yaml:"codex,omitempty"`
	Claude *bool `yaml:"claude,omitempty"`
}

// DashboardKeymapConfig defines dashboard key bindings.
type DashboardKeymapConfig struct {
	ProjectLeft     []string `yaml:"project_left,omitempty"`
	ProjectRight    []string `yaml:"project_right,omitempty"`
	SessionUp       []string `yaml:"session_up,omitempty"`
	SessionDown     []string `yaml:"session_down,omitempty"`
	SessionOnlyUp   []string `yaml:"session_only_up,omitempty"`
	SessionOnlyDown []string `yaml:"session_only_down,omitempty"`
	PaneNext        []string `yaml:"pane_next,omitempty"`
	PanePrev        []string `yaml:"pane_prev,omitempty"`
	Attach          []string `yaml:"attach,omitempty"`
	NewSession      []string `yaml:"new_session,omitempty"`
	TerminalFocus   []string `yaml:"terminal_focus,omitempty"`
	TogglePanes     []string `yaml:"toggle_panes,omitempty"`
	OpenProject     []string `yaml:"open_project,omitempty"`
	CommandPalette  []string `yaml:"command_palette,omitempty"`
	Refresh         []string `yaml:"refresh,omitempty"`
	EditConfig      []string `yaml:"edit_config,omitempty"`
	Kill            []string `yaml:"kill,omitempty"`
	CloseProject    []string `yaml:"close_project,omitempty"`
	Help            []string `yaml:"help,omitempty"`
	Quit            []string `yaml:"quit,omitempty"`
	Filter          []string `yaml:"filter,omitempty"`
	Scrollback      []string `yaml:"scrollback,omitempty"`
	CopyMode        []string `yaml:"copy_mode,omitempty"`
}

// HiddenProjectConfig stores a project hidden from the dashboard.
type HiddenProjectConfig struct {
	Name string `yaml:"name,omitempty"`
	Path string `yaml:"path,omitempty"`
}

// DashboardConfig configures the Peaky Panes dashboard UI.
type DashboardConfig struct {
	RefreshMS      int                   `yaml:"refresh_ms,omitempty"`
	PreviewLines   int                   `yaml:"preview_lines,omitempty"`
	PreviewCompact *bool                 `yaml:"preview_compact,omitempty"`
	IdleSeconds    int                   `yaml:"idle_seconds,omitempty"`
	StatusRegex    StatusRegexConfig     `yaml:"status_regex,omitempty"`
	PreviewMode    string                `yaml:"preview_mode,omitempty"` // grid | layout
	ProjectRoots   []string              `yaml:"project_roots,omitempty"`
	AgentDetection AgentDetectionConfig  `yaml:"agent_detection,omitempty"`
	AttachBehavior string                `yaml:"attach_behavior,omitempty"` // current | detached
	HiddenProjects []HiddenProjectConfig `yaml:"hidden_projects,omitempty"`
	Keymap         DashboardKeymapConfig `yaml:"keymap,omitempty"`
}

// ZellijSection holds zellij-specific config.
type ZellijSection struct {
	Config       string `yaml:"config,omitempty"`
	LayoutDir    string `yaml:"layout_dir,omitempty"`
	BridgePlugin string `yaml:"bridge_plugin,omitempty"`
}

// GhosttySection holds ghostty-specific config.
type GhosttySection struct {
	Config string `yaml:"config,omitempty"`
}

// Config is the root configuration structure for Peaky Panes.
type Config struct {
	Zellij     ZellijSection            `yaml:"zellij,omitempty"`
	Ghostty    GhosttySection           `yaml:"ghostty,omitempty"`
	LayoutDirs []string                 `yaml:"layout_dirs,omitempty"`
	Layouts    map[string]*LayoutConfig `yaml:"layouts,omitempty"`
	Projects   []ProjectConfig          `yaml:"projects,omitempty"`
	Tools      ToolsConfig              `yaml:"tools,omitempty"`
	Dashboard  DashboardConfig          `yaml:"dashboard,omitempty"`
}

// ProjectLocalConfig is the schema for .peakypanes.yml in project directories.
type ProjectLocalConfig struct {
	Session string            `yaml:"session,omitempty"`
	Layout  *LayoutConfig     `yaml:"layout,omitempty"`
	Vars    map[string]string `yaml:"vars,omitempty"`
	Tools   ToolsConfig       `yaml:"tools,omitempty"`
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return &cfg, nil
}

// SaveConfig writes configuration to a YAML file.
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

// LoadProjectLocal reads a .peakypanes.yml from a directory.
func LoadProjectLocal(dir string) (*ProjectLocalConfig, error) {
	path := filepath.Join(dir, ".peakypanes.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try .peakypanes.yaml as fallback
			path = filepath.Join(dir, ".peakypanes.yaml")
			data, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	var cfg ProjectLocalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	// If layout is nil but we have panes or grid at top level,
	// treat the whole file as a LayoutConfig.
	if cfg.Layout == nil {
		var layout LayoutConfig
		if err := yaml.Unmarshal(data, &layout); err == nil && (len(layout.Panes) > 0 || strings.TrimSpace(layout.Grid) != "") {
			cfg.Layout = &layout
		}
	}

	return &cfg, nil
}

// LoadLayoutFile reads a standalone layout YAML file.
func LoadLayoutFile(path string) (*LayoutConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read layout %q: %w", path, err)
	}
	var layout LayoutConfig
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("parse layout %q: %w", path, err)
	}
	return &layout, nil
}

// ExpandVars replaces ${VAR}, ${VAR:-default}, and special variables in a string.
func ExpandVars(s string, vars map[string]string, projectPath, projectName string) string {
	// Add special variables
	allVars := make(map[string]string)
	for k, v := range vars {
		allVars[k] = v
	}
	allVars["PROJECT_PATH"] = projectPath
	allVars["PROJECT_NAME"] = projectName

	// Pattern: ${VAR} or ${VAR:-default}
	re := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

	result := re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		varName := parts[1]
		defaultVal := ""
		if len(parts) > 2 {
			defaultVal = parts[2]
		}

		// Check provided vars first
		if val, ok := allVars[varName]; ok && val != "" {
			return val
		}
		// Then environment
		if val := os.Getenv(varName); val != "" {
			return val
		}
		// Then default
		return defaultVal
	})

	// Also expand $HOME and ~
	if home, err := os.UserHomeDir(); err == nil {
		result = strings.ReplaceAll(result, "$HOME", home)
		if strings.HasPrefix(result, "~/") {
			result = filepath.Join(home, result[2:])
		} else if result == "~" {
			result = home
		}
	}

	return result
}

// ExpandLayoutVars expands all variables in a layout config.
func ExpandLayoutVars(layout *LayoutConfig, extraVars map[string]string, projectPath, projectName string) *LayoutConfig {
	// Merge layout vars with extra vars (extra takes precedence)
	vars := make(map[string]string)
	for k, v := range layout.Vars {
		vars[k] = v
	}
	for k, v := range extraVars {
		vars[k] = v
	}

	expanded := &LayoutConfig{
		Name:        layout.Name,
		Description: layout.Description,
		Settings:    layout.Settings,
		Vars:        vars,
		Grid:        ExpandVars(layout.Grid, vars, projectPath, projectName),
		Command:     ExpandVars(layout.Command, vars, projectPath, projectName),
	}

	for _, cmd := range layout.Commands {
		expanded.Commands = append(expanded.Commands, ExpandVars(cmd, vars, projectPath, projectName))
	}
	for _, title := range layout.Titles {
		expanded.Titles = append(expanded.Titles, ExpandVars(title, vars, projectPath, projectName))
	}

	for _, pane := range layout.Panes {
		expandedPane := PaneDef{
			Title:   ExpandVars(pane.Title, vars, projectPath, projectName),
			Cmd:     ExpandVars(pane.Cmd, vars, projectPath, projectName),
			Size:    pane.Size,
			Split:   pane.Split,
			Enabled: pane.Enabled,
		}
		for _, setup := range pane.Setup {
			expandedPane.Setup = append(expandedPane.Setup, ExpandVars(setup, vars, projectPath, projectName))
		}
		expanded.Panes = append(expanded.Panes, expandedPane)
	}

	return expanded
}

// ToYAML serializes a layout config to YAML string.
func (l *LayoutConfig) ToYAML() (string, error) {
	data, err := yaml.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DefaultConfigPath returns the default global config path.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "peakypanes", "config.yml"), nil
}

// DefaultLayoutsDir returns the default layouts directory.
func DefaultLayoutsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "peakypanes", "layouts"), nil
}
