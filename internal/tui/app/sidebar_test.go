package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSidebarHiddenOverrides(t *testing.T) {
	m := newTestModelLite()
	m.tab = TabProject

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".peky.yml"), []byte("dashboard:\n  sidebar:\n    hidden: true\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	project := &m.data.Projects[0]
	project.Path = dir
	project.ID = projectKey(dir, project.Name)
	m.selection.ProjectID = project.ID
	m.selection.Session = project.Sessions[0].Name

	projectPath := normalizeProjectPath(dir)
	m.updateProjectLocalConfig(projectPath, projectConfigStateForPath(projectPath))

	if !m.sidebarHidden(project) {
		t.Fatalf("expected sidebar hidden from project config")
	}

	m.toggleSidebar()
	if m.sidebarHidden(project) {
		t.Fatalf("expected sidebar toggled visible")
	}

	m.toggleSidebar()
	if !m.sidebarHidden(project) {
		t.Fatalf("expected sidebar toggled back to hidden")
	}
}
