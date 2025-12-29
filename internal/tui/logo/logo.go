// Package logo renders the Peaky Panes wordmark in a stylized ASCII form.
package logo

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

const (
	word         = "PEAKYPANES"
	fullSpacing  = 2
	compactLabel = "PEAKY PANES"
)

// Render returns the full Peaky Panes wordmark. Width truncates the output
// per line; set width <= 0 for no truncation.
func Render(width int, compact bool) string {
	spacing := fullSpacing
	if compact {
		spacing = 1
	}
	lines := renderWord(word, spacing)
	if width > 0 {
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, width, "")
		}
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
	lines := renderWord(word, fullSpacing)
	if len(lines) == 0 {
		return 0
	}
	return utf8.RuneCountInString(lines[0])
}

// FullHeight reports the height of the full wordmark.
func FullHeight() int {
	return len(letterForms['P'])
}

func renderWord(text string, spacing int) []string {
	if spacing < 0 {
		spacing = 0
	}
	height := FullHeight()
	lines := make([]string, height)
	for idx, r := range text {
		form, ok := letterForms[r]
		if !ok {
			form = fallbackLetter(r, height)
		}
		for i := 0; i < height; i++ {
			if idx > 0 && spacing > 0 {
				lines[i] += strings.Repeat(" ", spacing)
			}
			lines[i] += form[i]
		}
	}
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return lines
}

func fallbackLetter(r rune, height int) []string {
	line := strings.Repeat(" ", 3)
	if height <= 0 {
		return nil
	}
	out := make([]string, height)
	center := height / 2
	for i := range out {
		out[i] = line
	}
	out[center] = " " + string(r) + " "
	return out
}

var letterForms = map[rune][]string{
	'P': {
		"████ ",
		"█   █",
		"████ ",
		"█    ",
		"█    ",
	},
	'E': {
		"█████",
		"█    ",
		"████ ",
		"█    ",
		"█████",
	},
	'A': {
		" ███ ",
		"█   █",
		"█████",
		"█   █",
		"█   █",
	},
	'K': {
		"█   █",
		"█  █ ",
		"███  ",
		"█  █ ",
		"█   █",
	},
	'Y': {
		"█   █",
		" █ █ ",
		"  █  ",
		"  █  ",
		"  █  ",
	},
	'N': {
		"█   █",
		"██  █",
		"█ █ █",
		"█  ██",
		"█   █",
	},
	'S': {
		" ████",
		"█    ",
		" ███ ",
		"    █",
		"████ ",
	},
}
