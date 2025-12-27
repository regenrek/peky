package main

import (
	"os"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestRunInitGlobalCreatesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	out := captureStdout(func() {
		runInit([]string{})
	})
	if !strings.Contains(out, "Initialized Peaky Panes") {
		t.Fatalf("runInit(global) output = %q", out)
	}

	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config missing: %v", err)
	}

	layoutsDir, err := layout.DefaultLayoutsDir()
	if err != nil {
		t.Fatalf("DefaultLayoutsDir() error: %v", err)
	}
	if _, err := os.Stat(layoutsDir); err != nil {
		t.Fatalf("layouts dir missing: %v", err)
	}
}

func TestListLayoutsPrintsHeader(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir() error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	out := captureStdout(func() {
		listLayouts()
	})
	if !strings.Contains(out, "Available Layouts") {
		t.Fatalf("listLayouts() output = %q", out)
	}
	if !strings.Contains(out, "builtin") {
		t.Fatalf("listLayouts() missing builtin layouts")
	}
}

func TestExportLayoutPrintsYaml(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	out := captureStdout(func() {
		exportLayout("dev-3")
	})
	if !strings.Contains(out, "Peaky Panes Layout: dev-3") {
		t.Fatalf("exportLayout() output = %q", out)
	}
	if !strings.Contains(out, "layout:") {
		t.Fatalf("exportLayout() missing layout block")
	}
}
