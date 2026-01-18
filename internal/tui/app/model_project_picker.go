package app

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/userpath"
	"github.com/regenrek/peakypanes/internal/workspace"
)

// ===== Project picker =====

func (m *Model) setupProjectPicker() {
	m.projectPicker = picker.NewProjectPicker()
}

func (m *Model) openProjectPicker() {
	m.scanProjects()
	m.projectPicker.ResetFilter()
	m.projectPicker.SetItems(m.projectPickerItems())
	m.setState(StateProjectPicker)
}

func (m *Model) scanProjects() {
	m.projectPickerProjects = nil

	roots := resolveProjectRoots(m.settings.ProjectRoots)
	allowNonGit := m.settings.ProjectRootsAllowNonGit
	seen := make(map[string]struct{})
	for _, project := range workspace.ScanProjects(roots, allowNonGit) {
		key := workspace.ProjectID(project.Path, project.Name)
		if key != "" {
			seen[key] = struct{}{}
		}
		m.projectPickerProjects = append(m.projectPickerProjects, picker.ProjectItem{
			Name:        project.Name,
			Path:        project.Path,
			DisplayPath: userpath.ShortenUser(project.Path),
			IsGit:       project.IsGit,
		})
	}
	if m.config == nil {
		return
	}
	for i := range m.config.Projects {
		name, _, path := normalizeProjectConfig(&m.config.Projects[i])
		if path == "" {
			continue
		}
		key := workspace.ProjectID(path, name)
		if key != "" {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
		}
		m.projectPickerProjects = append(m.projectPickerProjects, picker.ProjectItem{
			Name:        name,
			Path:        path,
			DisplayPath: userpath.ShortenUser(path),
			IsGit:       workspace.IsGitProjectPath(path),
		})
	}
}

func (m *Model) projectPickerItems() []list.Item {
	items := make([]list.Item, len(m.projectPickerProjects))
	for i, p := range m.projectPickerProjects {
		items[i] = p
	}
	return items
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
		if item, ok := m.projectPicker.SelectedItem().(picker.ProjectItem); ok {
			projectName := m.projectNameForPath(item.Path)
			if projectName == "" {
				projectName = item.Name
			}
			projectID := projectKey(item.Path, projectName)
			sessionName := m.projectSessionNameForPath(item.Path)
			if _, err := m.unhideProjectInConfig(layout.HiddenProjectConfig{Name: projectName, Path: item.Path}); err != nil {
				m.setToast("Unhide failed: "+err.Error(), toastError)
			}
			m.projectPicker.ResetFilter()
			m.setState(StateDashboard)
			m.rememberSelection(m.selection)
			sel := m.selection
			sel.ProjectID = projectID
			sel.Session = sessionName
			sel.Pane = ""
			m.applySelection(sel)
			m.selectionVersion++
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
