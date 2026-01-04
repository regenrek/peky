package layout

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/*.yml
var embeddedLayouts embed.FS

// LayoutInfo provides metadata about an available layout.
type LayoutInfo struct {
	Name        string
	Description string
	Source      string // "builtin", "global", "project"
	Path        string // empty for builtin
}

// Loader handles loading layouts from multiple sources with proper precedence.
type Loader struct {
	globalConfigPath string
	globalLayoutsDir string
	projectDir       string

	// Cached layouts
	builtinLayouts map[string]*LayoutConfig
	globalLayouts  map[string]*LayoutConfig
	projectConfig  *ProjectLocalConfig
}

// NewLoader creates a loader with default paths.
func NewLoader() (*Loader, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	layoutsDir, err := DefaultLayoutsDir()
	if err != nil {
		return nil, err
	}
	return &Loader{
		globalConfigPath: configPath,
		globalLayoutsDir: layoutsDir,
		builtinLayouts:   make(map[string]*LayoutConfig),
		globalLayouts:    make(map[string]*LayoutConfig),
	}, nil
}

// NewLoaderWithPaths creates a loader with custom paths.
func NewLoaderWithPaths(configPath, layoutsDir, projectDir string) *Loader {
	return &Loader{
		globalConfigPath: configPath,
		globalLayoutsDir: layoutsDir,
		projectDir:       projectDir,
		builtinLayouts:   make(map[string]*LayoutConfig),
		globalLayouts:    make(map[string]*LayoutConfig),
	}
}

// SetProjectDir sets the project directory for local config detection.
func (l *Loader) SetProjectDir(dir string) {
	l.projectDir = dir
}

// LoadBuiltins loads all embedded default layouts.
func (l *Loader) LoadBuiltins() error {
	entries, err := embeddedLayouts.ReadDir("defaults")
	if err != nil {
		return fmt.Errorf("read embedded layouts: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		data, err := embeddedLayouts.ReadFile("defaults/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), err)
		}

		var layout LayoutConfig
		if err := yaml.Unmarshal(data, &layout); err != nil {
			return fmt.Errorf("parse embedded %s: %w", entry.Name(), err)
		}

		// Use filename without extension as key if name not set
		name := layout.Name
		if name == "" {
			name = strings.TrimSuffix(entry.Name(), ".yml")
			layout.Name = name
		}
		l.builtinLayouts[name] = &layout
	}

	return nil
}

// LoadGlobalLayouts loads layouts from the user's config directory.
func (l *Loader) LoadGlobalLayouts() error {
	if err := l.loadGlobalLayoutsDir(); err != nil {
		return err
	}
	if err := l.loadGlobalLayoutsFromConfig(); err != nil {
		return err
	}
	return nil
}

func (l *Loader) loadGlobalLayoutsDir() error {
	if l.globalLayoutsDir == "" {
		return nil
	}
	info, err := os.Stat(l.globalLayoutsDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(l.globalLayoutsDir)
	if err != nil {
		return fmt.Errorf("read layouts dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isLayoutFileName(name) {
			continue
		}
		path := filepath.Join(l.globalLayoutsDir, name)
		layout, err := LoadLayoutFile(path)
		if err != nil {
			// Log but don't fail on individual file errors
			continue
		}
		l.addGlobalLayout(name, layout)
	}
	return nil
}

func (l *Loader) loadGlobalLayoutsFromConfig() error {
	if l.globalConfigPath == "" {
		return nil
	}
	cfg, err := LoadConfig(l.globalConfigPath)
	if err != nil || cfg.Layouts == nil {
		return nil
	}
	for name, layout := range cfg.Layouts {
		if layout.Name == "" {
			layout.Name = name
		}
		l.globalLayouts[name] = layout
	}
	return nil
}

func (l *Loader) addGlobalLayout(filename string, layout *LayoutConfig) {
	key := layout.Name
	if key == "" {
		key = strings.TrimSuffix(strings.TrimSuffix(filename, ".yml"), ".yaml")
		layout.Name = key
	}
	l.globalLayouts[key] = layout
}

func isLayoutFileName(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}

// LoadProjectLayout loads the .peakypanes.yml from the project directory.
func (l *Loader) LoadProjectLayout() error {
	if l.projectDir == "" {
		return nil
	}

	cfg, err := LoadProjectLocal(l.projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	l.projectConfig = cfg
	return nil
}

// LoadAll loads layouts from all sources.
func (l *Loader) LoadAll() error {
	if err := l.LoadBuiltins(); err != nil {
		return err
	}
	if err := l.LoadGlobalLayouts(); err != nil {
		return err
	}
	if err := l.LoadProjectLayout(); err != nil {
		return err
	}
	return nil
}

// GetLayout retrieves a layout by name with proper precedence:
// 1. Project local (if projectDir set and has inline layout)
// 2. Global user layouts
// 3. Builtin layouts
func (l *Loader) GetLayout(name string) (*LayoutConfig, string, error) {
	// Special case: empty name with project layout
	if name == "" && l.projectConfig != nil && l.projectConfig.Layout != nil {
		return l.projectConfig.Layout, "project", nil
	}

	// Check global layouts first (user can override builtins)
	if layout, ok := l.globalLayouts[name]; ok {
		return layout, "global", nil
	}

	// Fall back to builtins
	if layout, ok := l.builtinLayouts[name]; ok {
		return layout, "builtin", nil
	}

	if name == "" {
		// Return default layout
		if layout, ok := l.builtinLayouts[DefaultLayoutName]; ok {
			return layout, "builtin", nil
		}
	}

	return nil, "", fmt.Errorf("layout %q not found", name)
}

// GetProjectLayout returns the project-local layout if available.
func (l *Loader) GetProjectLayout() *LayoutConfig {
	if l.projectConfig == nil {
		return nil
	}
	return l.projectConfig.Layout
}

// GetProjectConfig returns the project-local config if available.
func (l *Loader) GetProjectConfig() *ProjectLocalConfig {
	return l.projectConfig
}

// ListLayouts returns info about all available layouts.
func (l *Loader) ListLayouts() []LayoutInfo {
	seen := make(map[string]bool)
	var layouts []LayoutInfo

	// Project layout (highest priority)
	if l.projectConfig != nil && l.projectConfig.Layout != nil {
		name := l.projectConfig.Layout.Name
		if name == "" {
			name = "(project)"
		}
		layouts = append(layouts, LayoutInfo{
			Name:        name,
			Description: l.projectConfig.Layout.Description,
			Source:      "project",
			Path:        filepath.Join(l.projectDir, ".peakypanes.yml"),
		})
		seen[name] = true
	}

	// Global layouts
	for name, layout := range l.globalLayouts {
		if seen[name] {
			continue
		}
		layouts = append(layouts, LayoutInfo{
			Name:        name,
			Description: layout.Description,
			Source:      "global",
			Path:        filepath.Join(l.globalLayoutsDir, name+".yml"),
		})
		seen[name] = true
	}

	// Builtin layouts
	for name, layout := range l.builtinLayouts {
		if seen[name] {
			continue
		}
		layouts = append(layouts, LayoutInfo{
			Name:        name,
			Description: layout.Description,
			Source:      "builtin",
		})
		seen[name] = true
	}

	// Sort by name
	sort.Slice(layouts, func(i, j int) bool {
		return layouts[i].Name < layouts[j].Name
	})

	return layouts
}

// ExportLayout returns the YAML representation of a layout.
func (l *Loader) ExportLayout(name string) (string, error) {
	layout, _, err := l.GetLayout(name)
	if err != nil {
		return "", err
	}
	return layout.ToYAML()
}

// HasProjectConfig returns true if a .peakypanes.yml exists in the project dir.
func (l *Loader) HasProjectConfig() bool {
	if l.projectDir == "" {
		return false
	}
	path := filepath.Join(l.projectDir, ".peakypanes.yml")
	if _, err := os.Stat(path); err == nil {
		return true
	}
	path = filepath.Join(l.projectDir, ".peakypanes.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// HasGlobalConfig returns true if the global config exists.
func (l *Loader) HasGlobalConfig() bool {
	if l.globalConfigPath == "" {
		return false
	}
	_, err := os.Stat(l.globalConfigPath)
	return err == nil
}

// GetBuiltinLayout returns a builtin layout by name.
func GetBuiltinLayout(name string) (*LayoutConfig, error) {
	data, err := embeddedLayouts.ReadFile("defaults/" + name + ".yml")
	if err != nil {
		return nil, fmt.Errorf("builtin layout %q not found", name)
	}
	var layout LayoutConfig
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("parse builtin %s: %w", name, err)
	}
	if layout.Name == "" {
		layout.Name = name
	}
	return &layout, nil
}

// ListBuiltinLayouts returns names of all builtin layouts.
func ListBuiltinLayouts() ([]string, error) {
	entries, err := embeddedLayouts.ReadDir("defaults")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".yml"))
	}
	sort.Strings(names)
	return names, nil
}
