package views

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestPadLinesKeepsRightBorderForDashboardPaneTiles(t *testing.T) {
	width := 40
	height := 8

	pane := DashboardPane{
		SessionName: "sess",
		Pane: Pane{
			Index:   "0",
			Title:   "shell",
			Command: "bash",
			Status:  paneStatusIdle,
			Preview: []string{"hello", "world"},
		},
	}

	tile := renderDashboardPaneTile(pane, width, height, 2, false, false, paneIconContext{})
	out := padLines(tile, width, height)

	tileStripped := ansi.Strip(tile)
	tileLines := strings.Split(tileStripped, "\n")
	if len(tileLines) != height {
		t.Fatalf("renderDashboardPaneTile() produced %d lines, want %d", len(tileLines), height)
	}
	for i, line := range tileLines {
		if w := lipgloss.Width(line); w != width {
			t.Fatalf("renderDashboardPaneTile() line %d width = %d, want %d: %q", i, w, width, line)
		}
	}

	stripped := ansi.Strip(out)
	lines := strings.Split(stripped, "\n")
	if len(lines) != height {
		t.Fatalf("padLines(tile) produced %d lines, want %d", len(lines), height)
	}

	for i, line := range lines {
		if line == "" {
			t.Fatalf("line %d is empty", i)
		}
		last, _ := utf8.DecodeLastRuneInString(line)
		if last == ' ' {
			t.Fatalf("line %d missing right border; got trailing space: %q", i, line)
		}
	}
}
