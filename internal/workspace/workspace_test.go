package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestNormalizeProjectPath(t *testing.T) {
	rel := "testdata"
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("Abs() error: %v", err)
	}
	if got := NormalizeProjectPath(rel); got != abs {
		t.Fatalf("NormalizeProjectPath(%q) = %q, want %q", rel, got, abs)
	}
	if got := NormalizeProjectPath("  "); got != "" {
		t.Fatalf("NormalizeProjectPath(empty) = %q, want empty", got)
	}
	home, err := os.UserHomeDir()
	if err == nil {
		want := filepath.Join(home, "projects")
		if got := NormalizeProjectPath("~/projects"); got != want {
			t.Fatalf("NormalizeProjectPath(~) = %q, want %q", got, want)
		}
	}
}

func TestNormalizeProjectRootsAndID(t *testing.T) {
	root := t.TempDir()
	roots := []string{root, root + string(os.PathSeparator), "", "   "}
	normalized := NormalizeProjectRoots(roots)
	if len(normalized) != 1 {
		t.Fatalf("NormalizeProjectRoots() len = %d, want 1", len(normalized))
	}
	if normalized[0] != NormalizeProjectPath(root) {
		t.Fatalf("NormalizeProjectRoots() = %q, want %q", normalized[0], NormalizeProjectPath(root))
	}
	if got := ProjectID(root, "Ignored"); got != strings.ToLower(NormalizeProjectPath(root)) {
		t.Fatalf("ProjectID(path) = %q, want path id", got)
	}
	if got := ProjectID("", "Name"); got != "name" {
		t.Fatalf("ProjectID(name) = %q, want name id", got)
	}
	if got := ProjectID("", " "); got != "" {
		t.Fatalf("ProjectID(empty) = %q, want empty", got)
	}
}

func TestHiddenProjectLifecycle(t *testing.T) {
	cfg := &layout.Config{}
	if _, err := HideProject(cfg, ProjectRef{}); err == nil {
		t.Fatalf("expected error for empty project ref")
	}
	added, err := HideProject(cfg, ProjectRef{Name: "App"})
	if err != nil {
		t.Fatalf("HideProject() error: %v", err)
	}
	if !added {
		t.Fatalf("expected hidden project to be added")
	}
	added, err = HideProject(cfg, ProjectRef{Name: "App"})
	if err != nil {
		t.Fatalf("HideProject() error: %v", err)
	}
	if added {
		t.Fatalf("expected duplicate hide to return false")
	}
	path := filepath.Join(t.TempDir(), "proj")
	added, err = HideProject(cfg, ProjectRef{Path: path})
	if err != nil {
		t.Fatalf("HideProject() error: %v", err)
	}
	if !added {
		t.Fatalf("expected path hide to be added")
	}
	removed, err := UnhideProject(cfg, ProjectRef{Name: "App"})
	if err != nil {
		t.Fatalf("UnhideProject() error: %v", err)
	}
	if !removed {
		t.Fatalf("expected unhide to remove entry")
	}
	removed, err = UnhideProject(cfg, ProjectRef{ID: path})
	if err != nil {
		t.Fatalf("UnhideProject() error: %v", err)
	}
	if !removed {
		t.Fatalf("expected unhide by id to remove entry")
	}
	removed, err = UnhideProject(cfg, ProjectRef{Name: "missing"})
	if err != nil {
		t.Fatalf("UnhideProject() error: %v", err)
	}
	if removed {
		t.Fatalf("expected unhide missing to return false")
	}
}

func TestHideAllProjects(t *testing.T) {
	cfg := &layout.Config{}
	projects := []Project{
		{Name: "One", Path: filepath.Join(t.TempDir(), "one")},
		{Name: "Two", Path: filepath.Join(t.TempDir(), "two")},
	}
	added, err := HideAllProjects(cfg, projects)
	if err != nil {
		t.Fatalf("HideAllProjects() error: %v", err)
	}
	if added != 2 {
		t.Fatalf("HideAllProjects() added=%d, want 2", added)
	}
	added, err = HideAllProjects(cfg, projects)
	if err != nil {
		t.Fatalf("HideAllProjects() error: %v", err)
	}
	if added != 0 {
		t.Fatalf("HideAllProjects() added=%d, want 0", added)
	}
	if _, err := HideAllProjects(nil, projects); err == nil {
		t.Fatalf("expected error for nil config")
	}
}

func TestHiddenProjectLabels(t *testing.T) {
	entries := []layout.HiddenProjectConfig{
		{Name: "Alpha", Path: "/tmp/alpha"},
		{Name: "Beta"},
		{Path: "/tmp/gamma"},
	}
	labels := HiddenProjectLabels(entries)
	if len(labels) != 3 {
		t.Fatalf("HiddenProjectLabels() len=%d, want 3", len(labels))
	}
	if labels[0] == "" || labels[1] == "" || labels[2] == "" {
		t.Fatalf("expected non-empty labels")
	}
}

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	cfg := &layout.Config{
		Dashboard: layout.DashboardConfig{
			ProjectRoots: []string{dir},
		},
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(loaded.Dashboard.ProjectRoots) != 1 {
		t.Fatalf("LoadConfig() roots=%d, want 1", len(loaded.Dashboard.ProjectRoots))
	}
	empty, err := LoadConfig(filepath.Join(dir, "missing.yml"))
	if err != nil {
		t.Fatalf("LoadConfig(missing) error: %v", err)
	}
	if empty == nil || len(empty.Dashboard.ProjectRoots) != 0 {
		t.Fatalf("LoadConfig(missing) expected empty config")
	}
	if err := SaveConfig(path, nil); err == nil {
		t.Fatalf("expected error for nil config")
	}
}

func TestScanListWorkspaceAndFindProject(t *testing.T) {
	root := t.TempDir()
	gitA := filepath.Join(root, "projA")
	gitB := filepath.Join(root, "projB")
	mkdirGit(t, gitA)
	mkdirGit(t, gitB)
	mkdirGit(t, filepath.Join(root, "node_modules", "skip"))
	mkdirGit(t, filepath.Join(root, ".hidden", "skip"))

	configPath := filepath.Join(t.TempDir(), "config.yml")
	cfg := &layout.Config{
		Dashboard: layout.DashboardConfig{
			ProjectRoots:   []string{root},
			HiddenProjects: []layout.HiddenProjectConfig{{Name: "projA"}},
		},
		Projects: []layout.ProjectConfig{
			{Name: "ConfProj", Path: filepath.Join(root, "confproj")},
		},
	}
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}
	workspace, err := ListWorkspace(configPath)
	if err != nil {
		t.Fatalf("ListWorkspace() error: %v", err)
	}
	if len(workspace.Projects) != 3 {
		t.Fatalf("ListWorkspace() projects=%d, want 3", len(workspace.Projects))
	}
	found := false
	for _, project := range workspace.Projects {
		if project.Name == "projA" && !project.Hidden {
			t.Fatalf("expected projA to be hidden")
		}
		if project.Name == "ConfProj" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected config project to be present")
	}
	if _, err := FindProject(workspace.Projects, ProjectRef{}); err == nil {
		t.Fatalf("expected error for empty ref")
	}
	if _, err := FindProject(workspace.Projects, ProjectRef{Name: "ConfProj"}); err != nil {
		t.Fatalf("FindProject() error: %v", err)
	}
	if _, err := FindProject(workspace.Projects, ProjectRef{Path: gitA}); err != nil {
		t.Fatalf("FindProject() by path error: %v", err)
	}
}

func mkdirGit(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git error: %v", err)
	}
}

func TestScanProjectsAllowNonGit(t *testing.T) {
	root := t.TempDir()
	gitPath := filepath.Join(root, "repo")
	mkdirGit(t, gitPath)
	nonGit := filepath.Join(root, "notes")
	if err := os.MkdirAll(nonGit, 0o755); err != nil {
		t.Fatalf("mkdir non-git error: %v", err)
	}
	if got := ScanProjects([]string{root}, false); len(got) != 1 {
		t.Fatalf("ScanProjects(git-only) = %#v", got)
	}
	if got := ScanProjects([]string{root}, true); len(got) != 2 {
		t.Fatalf("ScanProjects(allow-non-git) = %#v", got)
	}
}
