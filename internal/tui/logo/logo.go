// Package logo renders the Peaky Panes wordmark in a stylized ASCII form.
package logo

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

const (
	word          = "PEAKYPANES"
	fullSpacing   = 2
	compactLabel  = "PEAKY PANES"
	fallbackWidth = 5
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
	maxWidth := 0
	for _, line := range lines {
		if width := utf8.RuneCountInString(line); width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// FullHeight reports the height of the full wordmark.
func FullHeight() int {
	maxHeight := 0
	for _, form := range letterForms {
		if len(form) > maxHeight {
			maxHeight = len(form)
		}
	}
	if maxHeight == 0 {
		return 0
	}
	return maxHeight
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
		form = normalizeLetterform(form, height)
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
	if height <= 0 {
		return nil
	}
	line := strings.Repeat(" ", fallbackWidth)
	out := make([]string, height)
	center := height / 2
	for i := range out {
		out[i] = line
	}
	if fallbackWidth >= 3 {
		out[center] = " " + string(r) + strings.Repeat(" ", fallbackWidth-2)
	} else {
		out[center] = string(r)
	}
	return out
}

var letterForms = map[rune][]string{
	'P': {
		"████ ",
		"█   █",
		"█   █",
		"████ ",
		"█    ",
		"█    ",
		"█    ",
	},
	'E': {
		"█████",
		"█    ",
		"█    ",
		"████ ",
		"█    ",
		"█    ",
		"█████",
	},
	'A': {
		" ███ ",
		"█   █",
		"█   █",
		"█████",
		"█   █",
		"█   █",
		"█   █",
	},
	'K': {
		"█   █",
		"█  █ ",
		"█ █  ",
		"██   ",
		"█ █  ",
		"█  █ ",
		"█   █",
	},
	'Y': {
		"█   █",
		"█   █",
		" █ █ ",
		"  █  ",
		"  █  ",
		"  █  ",
		"  █  ",
	},
	'N': {
		"█   █",
		"██  █",
		"███ █",
		"█ ██ ",
		"█  ██",
		"█   █",
		"█   █",
	},
	'S': {
		" ████",
		"█    ",
		"█    ",
		" ███ ",
		"    █",
		"    █",
		"████ ",
	},
}

func normalizeLetterform(form []string, height int) []string {
	if height <= 0 {
		return nil
	}
	maxWidth := 0
	for _, line := range form {
		if width := utf8.RuneCountInString(line); width > maxWidth {
			maxWidth = width
		}
	}
	if maxWidth == 0 {
		maxWidth = fallbackWidth
	}
	out := make([]string, height)
	for i := 0; i < height; i++ {
		line := ""
		if i < len(form) {
			line = form[i]
		}
		lineWidth := utf8.RuneCountInString(line)
		if lineWidth < maxWidth {
			line += strings.Repeat(" ", maxWidth-lineWidth)
		}
		out[i] = line
	}
	return out
}
