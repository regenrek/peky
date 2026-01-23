package views

import "testing"

func TestHelpLineWrapping(t *testing.T) {
	lines := wrapHelpText("hello world", 5)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped lines, got %#v", lines)
	}
	clamped := clampHelpLines(lines, 5, 1)
	if len(clamped) != 1 {
		t.Fatalf("expected clamped lines")
	}
	if truncateHelpLine("hello", 2) == "" {
		t.Fatalf("expected truncated help line")
	}
	help := DialogHelp{Line: "short help", Body: "expanded body", Open: true}
	out := buildHelpLines(help, 10, 2)
	if len(out) != 2 {
		t.Fatalf("expected help lines with padding, got %d", len(out))
	}
}
