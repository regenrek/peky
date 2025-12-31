package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/userpath"
)

// ===== Project visibility in config =====

func (m *Model) hideProjectInConfig(project ProjectGroup) (bool, error) {
	key := normalizeProjectKey(project.Path, project.Name)
	if key == "" {
		return false, fmt.Errorf("invalid project key")
	}
	configPath, err := m.requireConfigPath()
	if err != nil {
		return false, err
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
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := layout.SaveConfig(configPath, cfg); err != nil {
		return false, err
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return true, nil
}

func (m *Model) hideAllProjectsInConfig(projects []ProjectGroup) (int, error) {
	if len(projects) == 0 {
		return 0, nil
	}
	configPath, err := m.requireConfigPath()
	if err != nil {
		return 0, err
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return 0, fmt.Errorf("load config: %w", err)
	}
	existing := hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
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
	cfg.Dashboard.HiddenProjects = normalizeHiddenProjects(cfg.Dashboard.HiddenProjects)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return 0, fmt.Errorf("create config dir: %w", err)
	}
	if err := layout.SaveConfig(configPath, cfg); err != nil {
		return 0, err
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return added, nil
}

func appendHiddenProject(cfg *layout.Config, project ProjectGroup, existing map[string]struct{}) bool {
	key := normalizeProjectKey(project.Path, project.Name)
	if key == "" {
		return false
	}
	if _, ok := existing[key]; ok {
		return false
	}
	name := strings.TrimSpace(project.Name)
	nameKey := strings.ToLower(name)
	if nameKey != "" {
		if _, ok := existing[nameKey]; ok {
			return false
		}
	}
	path := normalizeProjectPath(project.Path)
	pathKey := strings.ToLower(path)
	if pathKey != "" {
		if _, ok := existing[pathKey]; ok {
			return false
		}
	}
	entry := layout.HiddenProjectConfig{
		Name: name,
		Path: path,
	}
	cfg.Dashboard.HiddenProjects = append(cfg.Dashboard.HiddenProjects, entry)
	if nameKey != "" {
		existing[nameKey] = struct{}{}
	}
	if pathKey != "" {
		existing[pathKey] = struct{}{}
	}
	return true
}

func (m *Model) unhideProjectInConfig(entry layout.HiddenProjectConfig) (bool, error) {
	target := hiddenProjectKeysFrom(entry)
	if target.empty() {
		return false, nil
	}
	configPath, err := m.requireConfigPath()
	if err != nil {
		return false, err
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
		if target.matches(existing) {
			removed++
			continue
		}
		kept = append(kept, existing)
	}
	if removed == 0 {
		return false, nil
	}
	cfg.Dashboard.HiddenProjects = normalizeHiddenProjects(kept)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := layout.SaveConfig(configPath, cfg); err != nil {
		return false, err
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return true, nil
}

type hiddenProjectKeys struct {
	pathKey string
	nameKey string
}

func hiddenProjectKeysFrom(entry layout.HiddenProjectConfig) hiddenProjectKeys {
	return hiddenProjectKeys{
		pathKey: strings.ToLower(normalizeProjectPath(entry.Path)),
		nameKey: strings.ToLower(strings.TrimSpace(entry.Name)),
	}
}

func (keys hiddenProjectKeys) empty() bool {
	return keys.pathKey == "" && keys.nameKey == ""
}

func (keys hiddenProjectKeys) matches(entry layout.HiddenProjectConfig) bool {
	existing := hiddenProjectKeysFrom(entry)
	if keys.pathKey != "" && existing.pathKey != "" {
		return existing.pathKey == keys.pathKey
	}
	if keys.nameKey == "" || existing.nameKey == "" {
		return false
	}
	if existing.pathKey == "" || keys.pathKey == "" {
		return existing.nameKey == keys.nameKey
	}
	return false
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
		return fmt.Sprintf("%s (%s)", name, userpath.ShortenUser(path))
	}
	if name != "" {
		return name
	}
	if path != "" {
		return userpath.ShortenUser(path)
	}
	return "unknown project"
}

func (m *Model) requireConfigPath() (string, error) {
	path := strings.TrimSpace(m.configPath)
	if path == "" {
		return "", fmt.Errorf("config path unavailable (fresh-config mode)")
	}
	return path, nil
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
	return m.requestRefreshCmd()
}
