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

	layout, err := GetBuiltinLayout("dev-3")
	if err != nil {
		t.Fatalf("GetBuiltinLayout(dev-3) error: %v", err)
	}
	if layout.Name == "" {
		t.Fatalf("GetBuiltinLayout(dev-3) missing name")
	}
}

func TestLoaderGlobalAndProjectPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "layouts")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatalf("mkdir global layouts: %v", err)
	}

	globalLayoutPath := filepath.Join(globalDir, "custom.yml")
	if err := os.WriteFile(globalLayoutPath, []byte("name: custom\ngrid: 1x2\n"), 0o644); err != nil {
		t.Fatalf("write global layout: %v", err)
	}

	globalConfigPath := filepath.Join(tmpDir, "config.yml")
	globalConfig := "layouts:\n  dev-3:\n    name: dev-3\n    grid: 1x2\n"
	if err := os.WriteFile(globalConfigPath, []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projectPath := filepath.Join(projectDir, ".peakypanes.yml")
	if err := os.WriteFile(projectPath, []byte("layout:\n  name: project-layout\n  grid: 2x2\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	loader := NewLoaderWithPaths(globalConfigPath, globalDir, projectDir)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	layout, source, err := loader.GetLayout("custom")
	if err != nil {
		t.Fatalf("GetLayout(custom) error: %v", err)
	}
	if source != "global" || layout.Grid != "1x2" {
		t.Fatalf("custom layout source=%q grid=%q", source, layout.Grid)
	}

	layout, source, err = loader.GetLayout("dev-3")
	if err != nil {
		t.Fatalf("GetLayout(dev-3) error: %v", err)
	}
	if source != "global" || layout.Grid != "1x2" {
		t.Fatalf("dev-3 layout source=%q grid=%q", source, layout.Grid)
	}

	layout, source, err = loader.GetLayout("")
	if err != nil {
		t.Fatalf("GetLayout(empty) error: %v", err)
	}
	if source != "project" || layout.Name != "project-layout" {
		t.Fatalf("project layout source=%q name=%q", source, layout.Name)
	}

	infos := loader.ListLayouts()
	foundProject := false
	for _, info := range infos {
		if info.Source == "project" {
			foundProject = true
			if !strings.HasSuffix(info.Path, ".peakypanes.yml") {
				t.Fatalf("project layout path = %q", info.Path)
			}
		}
	}
	if !foundProject {
		t.Fatalf("ListLayouts() missing project entry: %#v", infos)
	}

	yaml, err := loader.ExportLayout("custom")
	if err != nil {
		t.Fatalf("ExportLayout() error: %v", err)
	}
	if !strings.Contains(yaml, "name: custom") {
		t.Fatalf("ExportLayout() output = %q", yaml)
	}
}

func TestLoaderConfigPresence(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "config.yml")
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

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
