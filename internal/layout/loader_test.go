package layout

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBuiltinLayouts(t *testing.T) {
	names, err := ListBuiltinLayouts()
	if err != nil {
		t.Fatalf("ListBuiltinLayouts() error: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("ListBuiltinLayouts() returned no layouts")
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("ListBuiltinLayouts() not sorted: %#v", names)
	}

	layout, err := GetBuiltinLayout(DefaultLayoutName)
	if err != nil {
		t.Fatalf("GetBuiltinLayout(%s) error: %v", DefaultLayoutName, err)
	}
	if layout.Name == "" {
		t.Fatalf("GetBuiltinLayout(%s) missing name", DefaultLayoutName)
	}
}

func TestLoaderGlobalAndProjectPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := createLayoutsDir(t, tmpDir)
	writeGlobalLayout(t, globalDir, "custom.yml", readLayoutFixture(t, "global-layout.yml"))
	globalConfigPath := writeGlobalConfig(t, tmpDir, readLayoutFixture(t, "global-config.yml"))
	projectDir := createProjectDir(t, tmpDir)
	writeProjectConfig(t, projectDir, readLayoutFixture(t, "project-config.yml"))
	loader := NewLoaderWithPaths(globalConfigPath, globalDir, projectDir)
	requireLoadAll(t, loader)
	assertLayout(t, loader, "custom", "global", "1x2")
	assertLayout(t, loader, DefaultLayoutName, "global", "1x2")
	assertProjectLayout(t, loader, "project-layout")
	assertProjectLayoutListed(t, loader)
	assertExportContains(t, loader, "custom", "name: custom")
}

func TestLoaderConfigPresence(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "config.yml")
	projectDir := createProjectDir(t, tmpDir)

	loader := NewLoaderWithPaths(globalPath, "", projectDir)
	if loader.HasGlobalConfig() {
		t.Fatalf("HasGlobalConfig() should be false")
	}
	if loader.HasProjectConfig() {
		t.Fatalf("HasProjectConfig() should be false")
	}

	if err := os.WriteFile(globalPath, []byte("layouts: {}\n"), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}
	projectPath := filepath.Join(projectDir, ".peakypanes.yml")
	if err := os.WriteFile(projectPath, []byte("session: demo\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	if !loader.HasGlobalConfig() {
		t.Fatalf("HasGlobalConfig() should be true")
	}
	if !loader.HasProjectConfig() {
		t.Fatalf("HasProjectConfig() should be true")
	}
}

func TestLoaderProjectAccessors(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := createProjectDir(t, tmpDir)
	writeProjectConfig(t, projectDir, "layout:\n  name: project-layout\n  panes:\n    - cmd: \"\"\n")

	loader := NewLoaderWithPaths("", "", projectDir)
	if err := loader.LoadProjectLayout(); err != nil {
		t.Fatalf("LoadProjectLayout() error: %v", err)
	}
	if loader.GetProjectLayout() == nil {
		t.Fatalf("GetProjectLayout() returned nil")
	}
	if loader.GetProjectConfig() == nil {
		t.Fatalf("GetProjectConfig() returned nil")
	}

	loader.SetProjectDir("custom")
	if loader.projectDir != "custom" {
		t.Fatalf("SetProjectDir() = %q", loader.projectDir)
	}
}

func TestNewLoader(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() error: %v", err)
	}
	if loader == nil {
		t.Fatalf("NewLoader() returned nil")
	}
}

func TestGetLayoutFallbacks(t *testing.T) {
	loader := NewLoaderWithPaths("", "", "")
	if err := loader.LoadBuiltins(); err != nil {
		t.Fatalf("LoadBuiltins() error: %v", err)
	}
	if layout, source, err := loader.GetLayout(""); err != nil || layout == nil || source != "builtin" {
		t.Fatalf("GetLayout(default) = %#v source=%q err=%v", layout, source, err)
	}
	if _, _, err := loader.GetLayout("missing"); err == nil {
		t.Fatalf("expected error for missing layout")
	}
}

func createLayoutsDir(t *testing.T, base string) string {
	t.Helper()
	dir := filepath.Join(base, "layouts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir global layouts: %v", err)
	}
	return dir
}

func createProjectDir(t *testing.T, base string) string {
	t.Helper()
	dir := filepath.Join(base, "project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	return dir
}

func writeGlobalLayout(t *testing.T, dir, name, contents string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name), contents, "write global layout")
}

func writeGlobalConfig(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yml")
	writeFile(t, path, contents, "write global config")
	return path
}

func writeProjectConfig(t *testing.T, dir, contents string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, ".peakypanes.yml"), contents, "write project config")
}

func readLayoutFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "layouts", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read layout fixture %s: %v", name, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, contents, failure string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("%s: %v", failure, err)
	}
}

func requireLoadAll(t *testing.T, loader *Loader) {
	t.Helper()
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
}

func assertLayout(t *testing.T, loader *Loader, name, source, grid string) {
	t.Helper()
	layout, gotSource, err := loader.GetLayout(name)
	if err != nil {
		t.Fatalf("GetLayout(%s) error: %v", name, err)
	}
	if gotSource != source || layout.Grid != grid {
		t.Fatalf("%s layout source=%q grid=%q", name, gotSource, layout.Grid)
	}
}

func assertProjectLayout(t *testing.T, loader *Loader, name string) {
	t.Helper()
	layout, source, err := loader.GetLayout("")
	if err != nil {
		t.Fatalf("GetLayout(empty) error: %v", err)
	}
	if source != "project" || layout.Name != name {
		t.Fatalf("project layout source=%q name=%q", source, layout.Name)
	}
}

func assertProjectLayoutListed(t *testing.T, loader *Loader) {
	t.Helper()
	infos := loader.ListLayouts()
	for _, info := range infos {
		if info.Source != "project" {
			continue
		}
		if !strings.HasSuffix(info.Path, ".peakypanes.yml") {
			t.Fatalf("project layout path = %q", info.Path)
		}
		return
	}
	t.Fatalf("ListLayouts() missing project entry: %#v", infos)
}

func assertExportContains(t *testing.T, loader *Loader, layoutName, fragment string) {
	t.Helper()
	yaml, err := loader.ExportLayout(layoutName)
	if err != nil {
		t.Fatalf("ExportLayout() error: %v", err)
	}
	if !strings.Contains(yaml, fragment) {
		t.Fatalf("ExportLayout() output = %q", yaml)
	}
}
