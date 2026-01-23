package views

import (
	"strings"
	"testing"
)

func TestUpdateDialogsRender(t *testing.T) {
	m := Model{
		Width:  80,
		Height: 24,
		UpdateDialog: UpdateDialog{
			CurrentVersion: "1.0.0",
			LatestVersion:  "1.1.0",
			Channel:        "npm_global",
			Command:        "npm update -g peakypanes",
			CanInstall:     true,
			RemindLabel:    "Later (3d)",
		},
	}
	dialog := m.viewUpdateDialog()
	if !strings.Contains(dialog, "Update Available") {
		t.Fatalf("expected update dialog title")
	}
	if !strings.Contains(dialog, "v1.0.0") || !strings.Contains(dialog, "v1.1.0") {
		t.Fatalf("expected version values in dialog")
	}
	if !strings.Contains(dialog, "npm") {
		t.Fatalf("expected channel text in dialog")
	}

	m.UpdateProgress = UpdateProgress{Step: "Download", Percent: 120}
	progress := m.viewUpdateProgress()
	if !strings.Contains(progress, "Installing Update") {
		t.Fatalf("expected progress dialog title")
	}
	if !strings.Contains(progress, "100%") {
		t.Fatalf("expected clamped percent")
	}

	restart := m.viewUpdateRestart()
	if !strings.Contains(restart, "Restart Required") {
		t.Fatalf("expected restart dialog title")
	}
}

func TestUpdateFormattingHelpers(t *testing.T) {
	if formatUpdateChannel("") != "-" {
		t.Fatalf("expected default channel format")
	}
	if formatUpdateChannel("npm_global") != "npm (global)" {
		t.Fatalf("unexpected channel format")
	}
	if clampPercent(-10) != 0 || clampPercent(200) != 100 {
		t.Fatalf("expected clamped percent")
	}
	bar := renderProgressBar(5, 50)
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Fatalf("unexpected progress bar: %q", bar)
	}
}
