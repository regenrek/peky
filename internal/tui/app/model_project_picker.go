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
	m.scanGitProjects()
	m.projectPicker.ResetFilter()
	m.projectPicker.SetItems(m.gitProjectsToItems())
	m.setState(StateProjectPicker)
}

func (m *Model) scanGitProjects() {
	m.gitProjects = nil

	roots := resolveProjectRoots(m.settings.ProjectRoots)
	for _, project := range workspace.ScanGitProjects(roots) {
		m.gitProjects = append(m.gitProjects, picker.ProjectItem{
			Name:        project.Name,
			Path:        project.Path,
			DisplayPath: userpath.ShortenUser(project.Path),
		})
	}
}

func (m *Model) gitProjectsToItems() []list.Item {
	items := make([]list.Item, len(m.gitProjects))
	for i, p := range m.gitProjects {
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
			m.selection.ProjectID = projectID
			m.selection.Session = sessionName
			m.selection.Pane = ""
			m.selectionVersion++
			m.rememberSelection(m.selection)
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
