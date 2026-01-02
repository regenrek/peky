package app

import (
	"strings"
	"testing"
)

func TestDialogHelpForUnknownKey(t *testing.T) {
	content := dialogHelpFor("", "")
	if content.Title != "" || content.Summary != "" || len(content.Body) != 0 {
		t.Fatalf("expected empty content, got %#v", content)
	}

	content = dialogHelpFor("missing", "value")
	if content.Title != "" || content.Summary != "" || len(content.Body) != 0 {
		t.Fatalf("expected empty content for unknown key, got %#v", content)
	}
}

func TestDialogHelpPresetMaxSeverity(t *testing.T) {
	content := dialogHelpFor(helpKeyPerfPreset, "max")
	if content.Severity != dialogHelpHeavy {
		t.Fatalf("expected heavy severity for max preset")
	}
	if !strings.Contains(content.Summary, "max") {
		t.Fatalf("expected summary to mention max, got %q", content.Summary)
	}
}

func TestDialogHelpViewOpen(t *testing.T) {
	m := newTestModelLite()
	m.openPerformanceMenu()
	m.state = StatePerformanceMenu
	m.dialogHelpOpen = true

	view := m.dialogHelpView()
	if !view.Open {
		t.Fatalf("expected help view open")
	}
	if strings.TrimSpace(view.Title) == "" {
		t.Fatalf("expected help title")
	}
	if strings.TrimSpace(view.Line) == "" {
		t.Fatalf("expected help line")
	}
}
