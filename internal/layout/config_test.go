package layout

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExpandVars(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ENV_ONLY", "from-env")

	vars := map[string]string{
		"FOO":          "bar",
		"PROJECT_NAME": "ignored",
	}

	got := ExpandVars("${FOO}-${ENV_ONLY}-${MISSING:-def}-${PROJECT_PATH}-${PROJECT_NAME}", vars, "/work/app", "myapp")
	want := "bar-from-env-def-/work/app-myapp"
	if got != want {
		t.Fatalf("ExpandVars() = %q, want %q", got, want)
	}

	if got := ExpandVars("$HOME", vars, "/work/app", "myapp"); got != tmpHome {
		t.Fatalf("ExpandVars($HOME) = %q, want %q", got, tmpHome)
	}

	if got := ExpandVars("~", vars, "/work/app", "myapp"); got != tmpHome {
		t.Fatalf("ExpandVars(~) = %q, want %q", got, tmpHome)
	}

	if got := ExpandVars("~/projects", vars, "/work/app", "myapp"); got != filepath.Join(tmpHome, "projects") {
		t.Fatalf("ExpandVars(~/projects) = %q, want %q", got, filepath.Join(tmpHome, "projects"))
	}
}

func TestExpandLayoutVars(t *testing.T) {
	layout := &LayoutConfig{
		Name: "demo",
		Vars: map[string]string{
			"FOO": "one",
			"BAR": "two",
		},
		Grid:    "${FOO}",
		Command: "${EXTRA}",
		Commands: []string{
			"${FOO}",
			"${BAR}",
		},
		Titles: []string{"${PROJECT_PATH}"},
		Panes: []PaneDef{
			{Title: "${BAR}", Cmd: "${EXTRA}", Setup: []string{"${FOO}"}},
		},
	}

	extra := map[string]string{
		"FOO":   "override",
		"EXTRA": "extra",
	}

	expanded := ExpandLayoutVars(layout, extra, "/work/app", "myapp")
	if expanded.Vars["FOO"] != "override" {
		t.Fatalf("expanded.Vars[FOO] = %q", expanded.Vars["FOO"])
	}
	if expanded.Vars["BAR"] != "two" {
		t.Fatalf("expanded.Vars[BAR] = %q", expanded.Vars["BAR"])
	}
	if expanded.Vars["EXTRA"] != "extra" {
		t.Fatalf("expanded.Vars[EXTRA] = %q", expanded.Vars["EXTRA"])
	}
	if expanded.Grid != "override" {
		t.Fatalf("expanded.Grid = %q", expanded.Grid)
	}
	if expanded.Command != "extra" {
		t.Fatalf("expanded.Command = %q", expanded.Command)
	}
	if !reflect.DeepEqual(expanded.Commands, []string{"override", "two"}) {
		t.Fatalf("expanded.Commands = %#v", expanded.Commands)
	}
	if !reflect.DeepEqual(expanded.Titles, []string{"/work/app"}) {
		t.Fatalf("expanded.Titles = %#v", expanded.Titles)
	}
	if len(expanded.Panes) != 1 || expanded.Panes[0].Title != "two" {
		t.Fatalf("expanded.Panes = %#v", expanded.Panes)
	}
	if expanded.Panes[0].Cmd != "extra" {
		t.Fatalf("expanded.Panes[0].Cmd = %q", expanded.Panes[0].Cmd)
	}
	if !reflect.DeepEqual(expanded.Panes[0].Setup, []string{"override"}) {
		t.Fatalf("expanded.Panes[0].Setup = %#v", expanded.Panes[0].Setup)
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yml")

	cfg := &Config{
		Layouts: map[string]*LayoutConfig{
			"demo": {Name: "demo", Grid: "2x2"},
		},
		Projects: []ProjectConfig{
			{Name: "app", Session: "app", Path: "/tmp/app", Layout: "demo"},
		},
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if loaded.Layouts["demo"].Grid != "2x2" {
		t.Fatalf("loaded layout grid = %q", loaded.Layouts["demo"].Grid)
	}
	if len(loaded.Projects) != 1 || loaded.Projects[0].Name != "app" {
		t.Fatalf("loaded projects = %#v", loaded.Projects)
	}
}

func TestLoadProjectLocalInlineLayout(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".peakypanes.yml")
	if err := os.WriteFile(path, []byte("grid: 2x2\ncommands: [\"echo hi\"]\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := LoadProjectLocal(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectLocal() error: %v", err)
	}
	if cfg.Layout == nil || strings.TrimSpace(cfg.Layout.Grid) != "2x2" {
		t.Fatalf("LoadProjectLocal layout = %#v", cfg.Layout)
	}
}

func TestLoadProjectLocalYamlFallback(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".peakypanes.yaml")
	if err := os.WriteFile(path, []byte("session: demo\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := LoadProjectLocal(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectLocal() error: %v", err)
	}
	if cfg.Session != "demo" {
		t.Fatalf("LoadProjectLocal session = %q", cfg.Session)
	}
}

func TestLoadProjectLocalDashboardSidebar(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".peakypanes.yml")
	if err := os.WriteFile(path, []byte("dashboard:\n  sidebar:\n    hidden: true\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := LoadProjectLocal(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectLocal() error: %v", err)
	}
	if cfg.Dashboard.Sidebar.Hidden == nil || !*cfg.Dashboard.Sidebar.Hidden {
		t.Fatalf("LoadProjectLocal dashboard sidebar hidden = %#v", cfg.Dashboard.Sidebar.Hidden)
	}
}

func TestLoadLayoutFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "layout.yml")
	if err := os.WriteFile(path, []byte("name: sample\ngrid: 1x2\n"), 0o644); err != nil {
		t.Fatalf("write layout file: %v", err)
	}

	layout, err := LoadLayoutFile(path)
	if err != nil {
		t.Fatalf("LoadLayoutFile() error: %v", err)
	}
	if layout.Name != "sample" || layout.Grid != "1x2" {
		t.Fatalf("LoadLayoutFile layout = %#v", layout)
	}
}

func TestDefaultPaths(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfgPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	if cfgPath != filepath.Join(tmpHome, ".config", "peakypanes", "config.yml") {
		t.Fatalf("DefaultConfigPath() = %q", cfgPath)
	}

	layoutsDir, err := DefaultLayoutsDir()
	if err != nil {
		t.Fatalf("DefaultLayoutsDir() error: %v", err)
	}
	if layoutsDir != filepath.Join(tmpHome, ".config", "peakypanes", "layouts") {
		t.Fatalf("DefaultLayoutsDir() = %q", layoutsDir)
	}
}

func TestLayoutToYAML(t *testing.T) {
	layout := &LayoutConfig{Name: "demo", Grid: "2x2"}
	yaml, err := layout.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error: %v", err)
	}
	if !strings.Contains(yaml, "name: demo") || !strings.Contains(yaml, "grid: 2x2") {
		t.Fatalf("ToYAML() output = %q", yaml)
	}
}
