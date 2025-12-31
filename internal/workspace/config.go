package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
	"gopkg.in/yaml.v3"
)

// LoadConfig reads a config file, returning an empty config if it doesn't exist.
func LoadConfig(path string) (*layout.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &layout.Config{}, nil
		}
		return nil, err
	}
	var cfg layout.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig writes the config file, creating directories as needed.
func SaveConfig(path string, cfg *layout.Config) error {
	if cfg == nil {
		return fmt.Errorf("workspace: config is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("workspace: create config dir: %w", err)
	}
	return layout.SaveConfig(path, cfg)
}

// NormalizeHiddenProjects deduplicates hidden project entries.
func NormalizeHiddenProjects(entries []layout.HiddenProjectConfig) []layout.HiddenProjectConfig {
	if len(entries) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]layout.HiddenProjectConfig, 0, len(entries))
	for _, entry := range entries {
		entry.Name = strings.TrimSpace(entry.Name)
		entry.Path = NormalizeProjectPath(entry.Path)
		key := ProjectID(entry.Path, entry.Name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// HiddenProjectKeySet builds a lookup for hidden projects.
func HiddenProjectKeySet(entries []layout.HiddenProjectConfig) map[string]struct{} {
	if len(entries) == 0 {
		return nil
	}
	out := make(map[string]struct{})
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name)
		path := NormalizeProjectPath(entry.Path)
		if path != "" {
			out[strings.ToLower(path)] = struct{}{}
		}
		if name != "" {
			out[strings.ToLower(name)] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// HideProject updates config to hide a project.
func HideProject(cfg *layout.Config, ref ProjectRef) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("workspace: config is nil")
	}
	entry := hiddenEntryFromRef(ref)
	if entry.Name == "" && entry.Path == "" {
		return false, fmt.Errorf("workspace: invalid project ref")
	}
	existing := HiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	if existing == nil {
		existing = make(map[string]struct{})
	}
	key := ProjectID(entry.Path, entry.Name)
	if key != "" {
		if _, ok := existing[key]; ok {
			return false, nil
		}
	}
	nameKey := strings.ToLower(strings.TrimSpace(entry.Name))
	if nameKey != "" {
		if _, ok := existing[nameKey]; ok {
			return false, nil
		}
	}
	pathKey := strings.ToLower(NormalizeProjectPath(entry.Path))
	if pathKey != "" {
		if _, ok := existing[pathKey]; ok {
			return false, nil
		}
	}
	cfg.Dashboard.HiddenProjects = append(cfg.Dashboard.HiddenProjects, entry)
	cfg.Dashboard.HiddenProjects = NormalizeHiddenProjects(cfg.Dashboard.HiddenProjects)
	return true, nil
}

// UnhideProject removes hidden entries for a project.
func UnhideProject(cfg *layout.Config, ref ProjectRef) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("workspace: config is nil")
	}
	target := hiddenKeysFromRef(ref)
	if target.empty() {
		return false, fmt.Errorf("workspace: invalid project ref")
	}
	if len(cfg.Dashboard.HiddenProjects) == 0 {
		return false, nil
	}
	kept := make([]layout.HiddenProjectConfig, 0, len(cfg.Dashboard.HiddenProjects))
	removed := 0
	for _, entry := range cfg.Dashboard.HiddenProjects {
		if target.matches(entry) {
			removed++
			continue
		}
		kept = append(kept, entry)
	}
	if removed == 0 {
		return false, nil
	}
	cfg.Dashboard.HiddenProjects = NormalizeHiddenProjects(kept)
	return true, nil
}

// HideAllProjects hides all provided projects.
func HideAllProjects(cfg *layout.Config, projects []Project) (int, error) {
	if cfg == nil {
		return 0, fmt.Errorf("workspace: config is nil")
	}
	if len(projects) == 0 {
		return 0, nil
	}
	existing := HiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	if existing == nil {
		existing = make(map[string]struct{})
	}
	added := 0
	for _, project := range projects {
		if appendHiddenProject(cfg, project, existing) {
			added++
		}
	}
	if added == 0 {
		return 0, nil
	}
	cfg.Dashboard.HiddenProjects = NormalizeHiddenProjects(cfg.Dashboard.HiddenProjects)
	return added, nil
}

func appendHiddenProject(cfg *layout.Config, project Project, existing map[string]struct{}) bool {
	key := ProjectID(project.Path, project.Name)
	if key == "" {
		return false
	}
	if _, ok := existing[key]; ok {
		return false
	}
	name := strings.TrimSpace(project.Name)
	nameKey := strings.ToLower(name)
	path := NormalizeProjectPath(project.Path)
	pathKey := strings.ToLower(path)
	entry := layout.HiddenProjectConfig{Name: name, Path: path}
	cfg.Dashboard.HiddenProjects = append(cfg.Dashboard.HiddenProjects, entry)
	if nameKey != "" {
		existing[nameKey] = struct{}{}
	}
	if pathKey != "" {
		existing[pathKey] = struct{}{}
	}
	return true
}

type hiddenKeys struct {
	pathKey string
	nameKey string
}

func hiddenKeysFromRef(ref ProjectRef) hiddenKeys {
	path := NormalizeProjectPath(ref.Path)
	name := strings.TrimSpace(ref.Name)
	if path == "" && name == "" && ref.ID != "" {
		if looksLikePath(ref.ID) {
			path = NormalizeProjectPath(ref.ID)
		} else {
			name = ref.ID
		}
	}
	return hiddenKeys{
		pathKey: strings.ToLower(path),
		nameKey: strings.ToLower(strings.TrimSpace(name)),
	}
}

func hiddenEntryFromRef(ref ProjectRef) layout.HiddenProjectConfig {
	path := NormalizeProjectPath(ref.Path)
	name := strings.TrimSpace(ref.Name)
	if path == "" && name == "" && ref.ID != "" {
		if looksLikePath(ref.ID) {
			path = NormalizeProjectPath(ref.ID)
		} else {
			name = ref.ID
		}
	}
	return layout.HiddenProjectConfig{Name: name, Path: path}
}

func (h hiddenKeys) empty() bool {
	return h.pathKey == "" && h.nameKey == ""
}

func (h hiddenKeys) matches(entry layout.HiddenProjectConfig) bool {
	other := hiddenKeys{
		pathKey: strings.ToLower(NormalizeProjectPath(entry.Path)),
		nameKey: strings.ToLower(strings.TrimSpace(entry.Name)),
	}
	if h.pathKey != "" && other.pathKey != "" {
		return h.pathKey == other.pathKey
	}
	if h.nameKey == "" || other.nameKey == "" {
		return false
	}
	if other.pathKey == "" || h.pathKey == "" {
		return h.nameKey == other.nameKey
	}
	return false
}

func looksLikePath(value string) bool {
	return strings.Contains(value, string(os.PathSeparator)) || strings.HasPrefix(value, ".") || strings.HasPrefix(value, "/")
}

// HiddenProjectLabels returns sorted labels for hidden projects.
func HiddenProjectLabels(entries []layout.HiddenProjectConfig) []string {
	labels := make([]string, 0, len(entries))
	for _, entry := range NormalizeHiddenProjects(entries) {
		labels = append(labels, hiddenProjectLabel(entry))
	}
	sort.Strings(labels)
	return labels
}

func hiddenProjectLabel(entry layout.HiddenProjectConfig) string {
	name := strings.TrimSpace(entry.Name)
	path := strings.TrimSpace(entry.Path)
	switch {
	case name != "" && path != "":
		return fmt.Sprintf("%s (%s)", name, path)
	case name != "":
		return name
	case path != "":
		return path
	default:
		return "unknown project"
	}
}
