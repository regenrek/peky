package layout

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/runenv"
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
	submitDelay := 150
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
			{Title: "${BAR}", Cmd: "${EXTRA}", Setup: []string{"${FOO}"}, DirectSend: []SendAction{{Text: "${FOO} ${PROJECT_NAME}", Submit: true, SubmitDelayMS: &submitDelay, WaitForOutput: true}}},
		},
		BroadcastSend: []SendAction{{Text: "${BAR} ${PROJECT_PATH}", Submit: true, WaitForOutput: true}},
	}

	extra := map[string]string{
		"FOO":   "override",
		"EXTRA": "extra",
	}

	expanded := ExpandLayoutVars(layout, extra, "/work/app", "myapp")
	assertEqual(t, "expanded.Vars[FOO]", expanded.Vars["FOO"], "override")
	assertEqual(t, "expanded.Vars[BAR]", expanded.Vars["BAR"], "two")
	assertEqual(t, "expanded.Vars[EXTRA]", expanded.Vars["EXTRA"], "extra")
	assertEqual(t, "expanded.Grid", expanded.Grid, "override")
	assertEqual(t, "expanded.Command", expanded.Command, "extra")
	assertDeepEqual(t, "expanded.Commands", expanded.Commands, []string{"override", "two"})
	assertDeepEqual(t, "expanded.Titles", expanded.Titles, []string{"/work/app"})
	if len(expanded.Panes) != 1 {
		t.Fatalf("expanded.Panes = %#v", expanded.Panes)
	}
	assertEqual(t, "expanded.Panes[0].Title", expanded.Panes[0].Title, "two")
	assertEqual(t, "expanded.Panes[0].Cmd", expanded.Panes[0].Cmd, "extra")
	assertDeepEqual(t, "expanded.Panes[0].Setup", expanded.Panes[0].Setup, []string{"override"})
	assertSendActions(t, expanded.Panes[0].DirectSend, "override myapp", &submitDelay, true, true)
	assertSendActions(t, expanded.BroadcastSend, "two /work/app", nil, true, true)
}

func assertEqual[T comparable](t *testing.T, label string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v", label, got)
	}
}

func assertDeepEqual(t *testing.T, label string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v", label, got)
	}
}

func assertSendActions(t *testing.T, actions []SendAction, text string, submitDelay *int, submit bool, waitForOutput bool) {
	t.Helper()
	if len(actions) != 1 {
		t.Fatalf("send actions = %#v", actions)
	}
	action := actions[0]
	if action.Text != text {
		t.Fatalf("send action text = %q", action.Text)
	}
	if action.Submit != submit {
		t.Fatalf("send action submit = %v", action.Submit)
	}
	if action.WaitForOutput != waitForOutput {
		t.Fatalf("send action wait_for_output = %v", action.WaitForOutput)
	}
	if submit {
		if submitDelay == nil {
			if action.SubmitDelayMS != nil {
				t.Fatalf("send action submit_delay = %#v", action.SubmitDelayMS)
			}
			return
		}
		if action.SubmitDelayMS == nil || *action.SubmitDelayMS != *submitDelay {
			t.Fatalf("send action submit_delay = %#v", action.SubmitDelayMS)
		}
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
	t.Setenv(runenv.ConfigDirEnv, "")
	t.Setenv(runenv.FreshConfigEnv, "")

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

func TestDefaultPathsConfigDirOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(runenv.ConfigDirEnv, dir)
	t.Setenv(runenv.FreshConfigEnv, "")

	cfgPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	if cfgPath != filepath.Join(dir, "config.yml") {
		t.Fatalf("DefaultConfigPath() = %q", cfgPath)
	}

	layoutsDir, err := DefaultLayoutsDir()
	if err != nil {
		t.Fatalf("DefaultLayoutsDir() error: %v", err)
	}
	if layoutsDir != filepath.Join(dir, "layouts") {
		t.Fatalf("DefaultLayoutsDir() = %q", layoutsDir)
	}
}

func TestDefaultConfigPathFreshConfig(t *testing.T) {
	t.Setenv(runenv.FreshConfigEnv, "1")
	t.Setenv(runenv.ConfigDirEnv, "")

	cfgPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	if cfgPath != "" {
		t.Fatalf("DefaultConfigPath() = %q", cfgPath)
	}
}

func TestDefaultConfigPathConfigDirFreshConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(runenv.ConfigDirEnv, dir)
	t.Setenv(runenv.FreshConfigEnv, "1")

	cfgPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	if cfgPath != filepath.Join(dir, "config.yml") {
		t.Fatalf("DefaultConfigPath() = %q", cfgPath)
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
