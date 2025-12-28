package picker

import "testing"

func TestNewProjectPicker(t *testing.T) {
	picker := NewProjectPicker()
	if picker.Title != "📁 Open Project" {
		t.Fatalf("unexpected title %q", picker.Title)
	}
	if !picker.ShowStatusBar() {
		t.Fatalf("expected status bar enabled")
	}
	if !picker.FilteringEnabled() {
		t.Fatalf("expected filtering enabled")
	}
}

func TestNewLayoutPicker(t *testing.T) {
	picker := NewLayoutPicker()
	if picker.Title != "🧩 New Session Layout" {
		t.Fatalf("unexpected title %q", picker.Title)
	}
	if !picker.ShowStatusBar() {
		t.Fatalf("expected status bar enabled")
	}
	if !picker.FilteringEnabled() {
		t.Fatalf("expected filtering enabled")
	}
}

func TestNewCommandPalette(t *testing.T) {
	picker := NewCommandPalette()
	if picker.Title != "⌘ Command Palette" {
		t.Fatalf("unexpected title %q", picker.Title)
	}
	if !picker.ShowStatusBar() {
		t.Fatalf("expected status bar enabled")
	}
	if !picker.FilteringEnabled() {
		t.Fatalf("expected filtering enabled")
	}
}
