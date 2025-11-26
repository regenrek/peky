package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// PaneDef defines a single pane within a window.
type PaneDef struct {
	Title   string   `yaml:"title,omitempty"`
	Cmd     string   `yaml:"cmd,omitempty"`
	Size    string   `yaml:"size,omitempty"`    // e.g., "50%", "30"
	Split   string   `yaml:"split,omitempty"`   // "horizontal" or "vertical"
	Setup   []string `yaml:"setup,omitempty"`   // commands to run before main cmd
	Enabled string   `yaml:"enabled,omitempty"` // expression like "${VAR:-true}"
}

// WindowDef defines a window (tab) with its panes.
type WindowDef struct {
	Name   string    `yaml:"name"`
	Layout string    `yaml:"layout,omitempty"` // tiled, even-horizontal, even-vertical, main-horizontal, main-vertical
	Panes  []PaneDef `yaml:"panes"`
}

// LayoutSettings contains optional layout configuration.
type LayoutSettings struct {
	Width    int       `yaml:"width,omitempty"`
	Height   int       `yaml:"height,omitempty"`
	BindKeys []KeyBind `yaml:"bind_keys,omitempty"`
}

// KeyBind defines a tmux key binding.
type KeyBind struct {
	Key    string `yaml:"key"`
	Action string `yaml:"action"`
}

// LayoutConfig represents a complete layout definition.
type LayoutConfig struct {
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty"`
	Settings    LayoutSettings    `yaml:"settings,omitempty"`
	Windows     []WindowDef       `yaml:"windows"`
}

// ProjectConfig represents a project entry in the config file.
type ProjectConfig struct {
	Name    string `yaml:"name"`
	Session string `yaml:"session"`
	Path    string `yaml:"path"`
	// Layout can be a string (reference) or inline LayoutConfig
	Layout interface{} `yaml:"layout,omitempty"`
	Vars   map[string]string `yaml:"vars,omitempty"`
}

// ToolConfig defines an external tool command.
type ToolConfig struct {
	Cmd        string `yaml:"cmd"`
	WindowName string `yaml:"window_name"`
}

// ToolsConfig groups tool definitions.
type ToolsConfig struct {
	CursorAgent ToolConfig `yaml:"cursor_agent,omitempty"`
	CodexNew    ToolConfig `yaml:"codex_new,omitempty"`
	CodexResume ToolConfig `yaml:"codex_resume,omitempty"`
}

// TmuxSection holds tmux-specific config.
type TmuxSection struct {
	Config string `yaml:"config,omitempty"`
}

// GhosttySection holds ghostty-specific config.
type GhosttySection struct {
	Config string `yaml:"config,omitempty"`
}

// Config is the root configuration structure for Peaky Panes.
type Config struct {
	Tmux       TmuxSection              `yaml:"tmux,omitempty"`
	Ghostty    GhosttySection           `yaml:"ghostty,omitempty"`
	LayoutDirs []string                 `yaml:"layout_dirs,omitempty"`
	Layouts    map[string]*LayoutConfig `yaml:"layouts,omitempty"`
	Projects   []ProjectConfig          `yaml:"projects,omitempty"`
	Tools      ToolsConfig              `yaml:"tools,omitempty"`
}

// ProjectLocalConfig is the schema for .peakypanes.yml in project directories.
type ProjectLocalConfig struct {
	Session  string            `yaml:"session,omitempty"`
	Layout   *LayoutConfig     `yaml:"layout,omitempty"`
	Vars     map[string]string `yaml:"vars,omitempty"`
	Tools    ToolsConfig       `yaml:"tools,omitempty"`
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
	}

	for _, win := range layout.Windows {
		expandedWin := WindowDef{
			Name:   ExpandVars(win.Name, vars, projectPath, projectName),
			Layout: win.Layout,
		}
		for _, pane := range win.Panes {
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
			expandedWin.Panes = append(expandedWin.Panes, expandedPane)
		}
		expanded.Windows = append(expanded.Windows, expandedWin)
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
