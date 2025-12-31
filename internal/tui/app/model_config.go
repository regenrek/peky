package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/userpath"
	"github.com/regenrek/peakypanes/internal/workspace"
)

// ===== Project visibility in config =====

func (m *Model) hideProjectInConfig(project ProjectGroup) (bool, error) {
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	changed, err := workspace.HideProject(cfg, workspace.ProjectRef{Name: project.Name, Path: project.Path})
	if err != nil {
		return false, err
	}
	if changed {
		if err := workspace.SaveConfig(m.configPath, cfg); err != nil {
			return false, err
		}
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return changed, nil
}

func (m *Model) hideAllProjectsInConfig(projects []ProjectGroup) (int, error) {
	if len(projects) == 0 {
		return 0, nil
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return 0, fmt.Errorf("load config: %w", err)
	}
	list := make([]workspace.Project, 0, len(projects))
	for _, project := range projects {
		list = append(list, workspace.Project{
			ID:   workspace.ProjectID(project.Path, project.Name),
			Name: project.Name,
			Path: project.Path,
		})
	}
	added, err := workspace.HideAllProjects(cfg, list)
	if err != nil {
		return 0, err
	}
	if added > 0 {
		if err := workspace.SaveConfig(m.configPath, cfg); err != nil {
			return 0, err
		}
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return added, nil
}

func (m *Model) unhideProjectInConfig(entry layout.HiddenProjectConfig) (bool, error) {
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	changed, err := workspace.UnhideProject(cfg, workspace.ProjectRef{Name: entry.Name, Path: entry.Path})
	if err != nil {
		return false, err
	}
	if changed {
		if err := workspace.SaveConfig(m.configPath, cfg); err != nil {
			return false, err
		}
	}
	m.config = cfg
	m.settings.HiddenProjects = hiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	return changed, nil
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
	return m.requestRefreshCmd()
}
