package sessionrestore

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// TrimLinesByBytes keeps the newest lines that fit within maxBytes.
func TrimLinesByBytes(lines []string, maxBytes int64) []string {
	if maxBytes <= 0 || len(lines) == 0 {
		return lines
	}
	total := int64(0)
	start := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		lineBytes := int64(len(lines[i]))
		if total+lineBytes > maxBytes {
			start = i + 1
			break
		}
		total += lineBytes
		start = i
	}
	if start <= 0 {
		return lines
	}
	return lines[start:]
}

// RenderPlainLines renders a fixed-size viewport from scrollback and screen lines.
func RenderPlainLines(cols, rows int, scrollback, screen []string) []string {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	lines := make([]string, 0, len(scrollback)+len(screen))
	lines = append(lines, scrollback...)
	lines = append(lines, screen...)
	if len(lines) > rows {
		lines = lines[len(lines)-rows:]
	}
	if len(lines) < rows {
		padding := make([]string, rows-len(lines))
		lines = append(padding, lines...)
	}
	for i := range lines {
		lines[i] = fitWidth(lines[i], cols)
	}
	return lines
}

func fitWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) == width {
		return text
	}
	if runewidth.StringWidth(text) < width {
		return text + strings.Repeat(" ", width-runewidth.StringWidth(text))
	}
	return runewidth.Truncate(text, width, "")
}
