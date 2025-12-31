package app

import (
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestDefaultProjectRootsUsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	roots := defaultProjectRoots()
	if len(roots) != 1 || roots[0] != filepath.Join(home, "projects") {
		t.Fatalf("unexpected roots: %#v", roots)
	}
}

func TestNormalizeProjectConfigAndHidden(t *testing.T) {
	name, session, _ := normalizeProjectConfig(nil)
	if name == "" || session == "" {
		t.Fatalf("expected defaults, got name=%q session=%q", name, session)
	}

	var path string
	cfg := &layout.ProjectConfig{Name: "App", Session: "", Path: "/tmp/app"}
	name, session, path = normalizeProjectConfig(cfg)
	if name != "App" || session == "" || path != "/tmp/app" {
		t.Fatalf("unexpected normalize project config")
	}

	settings := DashboardConfig{HiddenProjects: map[string]struct{}{"/tmp/app": {}}}
	if !isHiddenProject(settings, "/tmp/app", "") {
		t.Fatalf("expected hidden project by path")
	}
}

func TestNormalizeProjectPathRelative(t *testing.T) {
	want, err := filepath.Abs(filepath.Join("relative", "proj"))
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	got := normalizeProjectPath(filepath.Join("relative", "proj"))
	if got != want {
		t.Fatalf("normalizeProjectPath() = %q, want %q", got, want)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	cfg, err := loadConfig(filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil || cfg == nil {
		t.Fatalf("expected empty config, err=%v", err)
	}
}

func TestLoadConfigEmptyPath(t *testing.T) {
	cfg, err := loadConfig(" ")
	if err != nil || cfg == nil {
		t.Fatalf("expected empty config for empty path, err=%v", err)
	}
}
