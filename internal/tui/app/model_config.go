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
	target := hiddenProjectKeysFrom(entry)
	if target.empty() {
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
