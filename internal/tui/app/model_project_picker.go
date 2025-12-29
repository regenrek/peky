package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/userpath"
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

	for _, root := range projectRoots(m.settings.ProjectRoots) {
		m.scanGitProjectsInRoot(root)
	}
}

func projectRoots(roots []string) []string {
	if len(roots) == 0 {
		return defaultProjectRoots()
	}
	return roots
}

func (m *Model) scanGitProjectsInRoot(root string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if shouldSkipGitScanDir(d) {
			return filepath.SkipDir
		}
		if isGitProjectDir(path, d) {
			m.appendGitProject(root, path)
			return filepath.SkipDir
		}
		return nil
	})
}

func shouldSkipGitScanDir(d os.DirEntry) bool {
	if !d.IsDir() {
		return false
	}
	name := d.Name()
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "vendor", "__pycache__", ".venv", "venv":
		return true
	default:
		return false
	}
}

func isGitProjectDir(path string, d os.DirEntry) bool {
	if !d.IsDir() || d.Name() == ".git" {
		return false
	}
	gitPath := filepath.Join(path, ".git")
	_, err := os.Stat(gitPath)
	return err == nil
}

func (m *Model) appendGitProject(root, path string) {
	relPath, _ := filepath.Rel(root, path)
	m.gitProjects = append(m.gitProjects, picker.ProjectItem{
		Name:        relPath,
		Path:        path,
		DisplayPath: userpath.ShortenUser(path),
	})
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
