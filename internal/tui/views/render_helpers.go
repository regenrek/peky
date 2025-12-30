package views

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"

	tuiansi "github.com/regenrek/peakypanes/internal/tui/ansi"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func pathOrDash(path string) string {
	if strings.TrimSpace(path) == "" {
		return "-"
	}
	return path
}

func renderBadge(status int) string {
	switch status {
	case paneStatusDone:
		return theme.StatusBadgeDone.Render("done")
	case paneStatusError:
		return theme.StatusBadgeError.Render("error")
	case paneStatusRunning:
		return theme.StatusBadgeRunning.Render("running")
	default:
		return theme.StatusBadgeIdle.Render("idle")
	}
}

func truncateTileLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(text, width, "…")
}

func truncateTileLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = truncateTileLine(line, width)
	}
	return out
}

func tailLines(lines []string, max int) []string {
	if max <= 0 {
		return nil
	}
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func trimTrailingBlankLines(lines []string) []string {
	end := len(lines)
	for end > 0 {
		if !tuiansi.IsBlank(lines[end-1]) {
			break
		}
		end--
	}
	return lines[:end]
}

func compactPreviewLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		if tuiansi.IsBlank(line) {
			continue
		}
		compact = append(compact, line)
	}
	return compact
}

func truncateLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runeWidth(text) <= width {
		return text
	}
	if width <= 1 {
		return "…"
	}
	trim := truncateRunes(text, width-1)
	return trim + "…"
}

func fitLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	lineWidth := lipgloss.Width(text)
	if lineWidth == width {
		return text
	}
	if lineWidth < width {
		return text + strings.Repeat(" ", width-lineWidth)
	}

	truncated := ansi.Truncate(text, width, "")
	padding := width - lipgloss.Width(truncated)
	if padding <= 0 {
		return truncated
	}
	return truncated + strings.Repeat(" ", padding)
}

func centerLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	lineWidth := lipgloss.Width(text)
	if lineWidth >= width {
		return ansi.Truncate(text, width, "")
	}
	leftPad := (width - lineWidth) / 2
	rightPad := width - lineWidth - leftPad
	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

func truncateRunes(text string, width int) string {
	if width <= 0 {
		return ""
	}
	count := 0
	for i := range text {
		if count >= width {
			return text[:i]
		}
		count++
	}
	return text
}

func runeWidth(text string) int {
	return utf8.RuneCountInString(text)
}

func padLines(text string, width, height int) string {
	lines := strings.Split(text, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = padRight(line, width)
	}
	for len(lines) < height {
		lines = append(lines, padRight("", width))
	}
	return strings.Join(lines, "\n")
}

func padRight(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return fitLine(text, width)
}

func overlayCentered(base, overlay string, width, height int) string {
	overlayW := lipgloss.Width(overlay)
	overlayH := lipgloss.Height(overlay)
	return overlayCenteredSized(base, overlay, width, height, overlayW, overlayH)
}

func overlayCenteredSized(base, overlay string, width, height, overlayW, overlayH int) string {
	if width <= 0 || height <= 0 {
		return base
	}
	base = padLines(base, width, height)
	baseBuf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(baseBuf, base)

	if overlayW > width {
		overlayW = width
	}
	if overlayH > height {
		overlayH = height
	}
	if overlayW <= 0 || overlayH <= 0 {
		return renderBufferLines(baseBuf)
	}
	x := (width - overlayW) / 2
	y := (height - overlayH) / 2
	rect := cellbuf.Rect(x, y, overlayW, overlayH)

	bgLine := lipgloss.NewStyle().Background(theme.Background).Render(strings.Repeat(" ", overlayW))
	bgBlock := strings.Repeat(bgLine+"\n", overlayH-1) + bgLine
	cellbuf.SetContentRect(baseBuf, bgBlock, rect)
	cellbuf.SetContentRect(baseBuf, overlay, rect)

	return renderBufferLines(baseBuf)
}

func overlayAt(base, overlay string, width, height, x, y int) string {
	if width <= 0 || height <= 0 {
		return base
	}
	base = padLines(base, width, height)
	baseBuf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(baseBuf, base)

	overlayW := lipgloss.Width(overlay)
	overlayH := lipgloss.Height(overlay)
	if overlayW <= 0 || overlayH <= 0 {
		return renderBufferLines(baseBuf)
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+overlayW > width {
		overlayW = width - x
	}
	if y+overlayH > height {
		overlayH = height - y
	}
	if overlayW <= 0 || overlayH <= 0 {
		return renderBufferLines(baseBuf)
	}
	rect := cellbuf.Rect(x, y, overlayW, overlayH)

	bgLine := lipgloss.NewStyle().Background(theme.Background).Render(strings.Repeat(" ", overlayW))
	bgBlock := strings.Repeat(bgLine+"\n", overlayH-1) + bgLine
	cellbuf.SetContentRect(baseBuf, bgBlock, rect)
	cellbuf.SetContentRect(baseBuf, overlay, rect)

	return renderBufferLines(baseBuf)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func renderBufferLines(buf *cellbuf.Buffer) string {
	height := buf.Bounds().Dy()
	lines := make([]string, height)
	for y := 0; y < height; y++ {
		_, line := cellbuf.RenderLine(buf, y)
		lines[y] = line
	}
	return strings.Join(lines, "\n")
}
