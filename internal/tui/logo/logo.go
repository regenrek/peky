// Package logo renders the PEKY wordmark in ASCII form.
package logo

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

const (
	compactLabel = "PEKY"
)

const logoASCII = `██████  ███████ ██   ██ ██    ██
██   ██ ██      ██  ██   ██  ██
██████  █████   █████     ████
██      ██      ██  ██     ██
██      ███████ ██   ██    ██`

var logoLines = strings.Split(strings.TrimRight(logoASCII, "\n"), "\n")

// Render returns the full wordmark. Width truncates the output
// per line; set width <= 0 for no truncation.
func Render(width int, compact bool) string {
	if compact {
		return SmallRender(width)
	}
	lines := logoLines
	if width > 0 {
		out := make([]string, len(lines))
		for i, line := range lines {
			out[i] = ansi.Truncate(line, width, "")
		}
		lines = out
	}
	return strings.Join(lines, "\n")
}

// SmallRender returns a compact single-line logo.
func SmallRender(width int) string {
	line := compactLabel
	if width > 0 {
		line = ansi.Truncate(line, width, "")
	}
	return line
}

// FullWidth reports the width of the full wordmark.
func FullWidth() int {
	maxWidth := 0
	for _, line := range logoLines {
		if width := utf8.RuneCountInString(line); width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// FullHeight reports the height of the full wordmark.
func FullHeight() int {
	return len(logoLines)
}
