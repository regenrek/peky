package app

import (
	"os"

	"github.com/regenrek/peakypanes/internal/layout"
)

type projectLocalConfigCache struct {
	state  projectConfigState
	config *layout.ProjectLocalConfig
	err    error
}

func (m *Model) updateProjectLocalConfig(projectPath string, state projectConfigState) {
	projectPath = normalizeProjectPath(projectPath)
	if projectPath == "" {
		return
	}
	if m.projectLocalConfig == nil {
		m.projectLocalConfig = make(map[string]projectLocalConfigCache)
	}
	cached, ok := m.projectLocalConfig[projectPath]
	if ok && cached.state.equal(state) {
		return
	}

	var cfg *layout.ProjectLocalConfig
	var err error
	if state.exists {
		cfg, err = layout.LoadProjectLocal(projectPath)
		if err != nil && os.IsNotExist(err) {
			err = nil
			cfg = nil
		}
	}

	m.projectLocalConfig[projectPath] = projectLocalConfigCache{
		state:  state,
		config: cfg,
		err:    err,
	}
	if err != nil {
		prevErr := ""
		if ok && cached.err != nil {
			prevErr = cached.err.Error()
		}
		if err.Error() != prevErr {
			m.setToast("Project config error: "+err.Error(), toastWarning)
		}
	}
}

func (m *Model) projectLocalConfigForPath(projectPath string) *layout.ProjectLocalConfig {
	if m.projectLocalConfig == nil {
		return nil
	}
	projectPath = normalizeProjectPath(projectPath)
	if projectPath == "" {
		return nil
	}
	cached, ok := m.projectLocalConfig[projectPath]
	if !ok {
		return nil
	}
	return cached.config
}

func (m *Model) sidebarBaseHidden(project *ProjectGroup) bool {
	hidden := m.settings.SidebarHidden
	if project == nil {
		return hidden
	}
	cfg := m.projectLocalConfigForPath(project.Path)
	if cfg == nil {
		return hidden
	}
	if cfg.Dashboard.Sidebar.Hidden != nil {
		hidden = *cfg.Dashboard.Sidebar.Hidden
	}
	return hidden
}

func (m *Model) sidebarHidden(project *ProjectGroup) bool {
	base := m.sidebarBaseHidden(project)
	if project == nil {
		return base
	}
	if m.sidebarOverrides != nil {
		if hidden, ok := m.sidebarOverrides[project.ID]; ok {
			return hidden
		}
	}
	return base
}

func (m *Model) toggleSidebar() {
	project := m.selectedProject()
	if project == nil {
		return
	}
	base := m.sidebarBaseHidden(project)
	current := m.sidebarHidden(project)
	next := !current
	if next == base {
		if m.sidebarOverrides != nil {
			delete(m.sidebarOverrides, project.ID)
		}
		return
	}
	if m.sidebarOverrides == nil {
		m.sidebarOverrides = make(map[string]bool)
	}
	m.sidebarOverrides[project.ID] = next
}
