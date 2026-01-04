package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/limits"
)

// Scrollback represents a scrollback buffer that stores lines that have
// scrolled off the top of the visible screen.
//
// Design: ring buffer + soft-wrap metadata + reflow-on-resize.
type Scrollback struct {
	lines    [][]uv.Cell
	maxLines int
	head     int
	tail     int
	full     bool

	// Width captured when lines were added. Used for resize reflow.
	lastWidthCaptured int

	// softWrapped indicates whether a stored physical line is a soft wrap
	// (not a hard newline). Soft-wrapped groups can be merged and rewrapped.
	softWrapped []bool
}

func NewScrollback(maxLines int) *Scrollback {
	maxLines = normalizeScrollbackMaxLines(maxLines)
	return &Scrollback{
		lines:       make([][]uv.Cell, maxLines),
		maxLines:    maxLines,
		softWrapped: make([]bool, maxLines),
	}
}

func (sb *Scrollback) PushLine(line []uv.Cell) {
	sb.PushLineWithWrap(line, true)
}

func (sb *Scrollback) PushLineWithWrap(line []uv.Cell, isSoftWrapped bool) {
	if len(line) == 0 || sb.maxLines <= 0 {
		return
	}

	// Copy to avoid aliasing.
	lineCopy := make([]uv.Cell, len(line))
	copy(lineCopy, line)

	sb.lines[sb.tail] = lineCopy
	sb.softWrapped[sb.tail] = isSoftWrapped

	sb.tail = (sb.tail + 1) % sb.maxLines
	if sb.full {
		sb.head = (sb.head + 1) % sb.maxLines
	}
	if sb.tail == sb.head {
		sb.full = true
	}
}

func (sb *Scrollback) Len() int {
	if sb.maxLines <= 0 {
		return 0
	}
	if sb.full {
		return sb.maxLines
	}
	if sb.tail >= sb.head {
		return sb.tail - sb.head
	}
	return sb.maxLines - sb.head + sb.tail
}

func (sb *Scrollback) Line(index int) []uv.Cell {
	length := sb.Len()
	if index < 0 || index >= length || sb.maxLines <= 0 {
		return nil
	}
	physical := (sb.head + index) % sb.maxLines
	if physical < 0 || physical >= len(sb.lines) {
		return nil
	}
	return sb.lines[physical]
}

func (sb *Scrollback) Lines() [][]uv.Cell {
	length := sb.Len()
	if length == 0 {
		return nil
	}
	out := make([][]uv.Cell, length)
	for i := 0; i < length; i++ {
		physical := (sb.head + i) % sb.maxLines
		out[i] = sb.lines[physical]
	}
	return out
}

func (sb *Scrollback) Clear() {
	sb.head = 0
	sb.tail = 0
	sb.full = false
	for i := 0; i < len(sb.lines); i++ {
		sb.lines[i] = nil
		sb.softWrapped[i] = false
	}
}

func (sb *Scrollback) SetCaptureWidth(width int) {
	if width > 0 {
		sb.lastWidthCaptured = width
	}
}

func (sb *Scrollback) CaptureWidth() int { return sb.lastWidthCaptured }
func (sb *Scrollback) MaxLines() int     { return sb.maxLines }

func (sb *Scrollback) SetMaxLines(maxLines int) {
	maxLines = normalizeScrollbackMaxLines(maxLines)
	if maxLines == sb.maxLines {
		return
	}
	if maxLines == 0 {
		sb.lines = nil
		sb.softWrapped = nil
		sb.maxLines = 0
		sb.head, sb.tail, sb.full = 0, 0, false
		return
	}

	oldLen := sb.Len()
	old := sb.exportLinear()

	sb.lines = make([][]uv.Cell, maxLines)
	sb.softWrapped = make([]bool, maxLines)
	sb.maxLines = maxLines
	sb.head, sb.tail, sb.full = 0, 0, false

	// Keep most recent lines.
	if oldLen == 0 {
		return
	}
	if len(old.lines) > maxLines {
		start := len(old.lines) - maxLines
		old.lines = old.lines[start:]
		old.wrap = old.wrap[start:]
	}

	for i := 0; i < len(old.lines); i++ {
		sb.lines[i] = old.lines[i]
		sb.softWrapped[i] = old.wrap[i]
	}
	sb.tail = len(old.lines) % sb.maxLines
	sb.full = len(old.lines) == sb.maxLines
}

func normalizeScrollbackMaxLines(maxLines int) int {
	switch {
	case maxLines == 0:
		maxLines = limits.TerminalScrollbackMaxLinesDefault
	case maxLines < 0:
		maxLines = 0
	}
	if maxLines > limits.TerminalScrollbackMaxLinesMax {
		maxLines = limits.TerminalScrollbackMaxLinesMax
	}
	return maxLines
}

// Reflow reconstructs scrollback lines for a different terminal width.
// This merges soft-wrapped groups back into logical lines, then re-wraps them
// at newWidth while preserving uv.Cell styles.
func (sb *Scrollback) Reflow(newWidth int) {
	if newWidth <= 0 {
		return
	}
	if sb.maxLines <= 0 {
		sb.lastWidthCaptured = newWidth
		return
	}

	oldWidth := sb.lastWidthCaptured
	if oldWidth <= 0 {
		// Fallback: derive from first non-nil line.
		for i := 0; i < sb.Len(); i++ {
			if line := sb.Line(i); len(line) > 0 {
				oldWidth = len(line)
				break
			}
		}
		if oldWidth <= 0 {
			sb.lastWidthCaptured = newWidth
			return
		}
	}
	if newWidth == oldWidth {
		return
	}

	linear := sb.exportLinear()
	lines := linear.lines
	wrap := linear.wrap
	if len(lines) == 0 {
		sb.lastWidthCaptured = newWidth
		return
	}

	var outLines [][]uv.Cell
	var outWrap []bool

	i := 0
	for i < len(lines) {
		// Build one logical line by concatenating physical soft-wrapped lines.
		var logical []uv.Cell
		var lastSoft bool

		for {
			logical = append(logical, flattenForReflow(lines[i])...)
			lastSoft = wrap[i]
			if !wrap[i] || i == len(lines)-1 {
				break
			}
			i++
		}

		rewrappedLines, rewrappedWrap := wrapCells(logical, newWidth, lastSoft)
		outLines = append(outLines, rewrappedLines...)
		outWrap = append(outWrap, rewrappedWrap...)

		i++
	}

	// Enforce maxLines (keep most recent).
	if len(outLines) > sb.maxLines {
		start := len(outLines) - sb.maxLines
		outLines = outLines[start:]
		outWrap = outWrap[start:]
	}

	// Rebuild ring.
	sb.lines = make([][]uv.Cell, sb.maxLines)
	sb.softWrapped = make([]bool, sb.maxLines)
	for idx := 0; idx < len(outLines); idx++ {
		// copy line slice
		lineCopy := make([]uv.Cell, len(outLines[idx]))
		copy(lineCopy, outLines[idx])
		sb.lines[idx] = lineCopy
		sb.softWrapped[idx] = outWrap[idx]
	}
	sb.head = 0
	sb.tail = len(outLines) % sb.maxLines
	sb.full = len(outLines) == sb.maxLines
	sb.lastWidthCaptured = newWidth
}

type linearExport struct {
	lines [][]uv.Cell
	wrap  []bool
}

func (sb *Scrollback) exportLinear() linearExport {
	n := sb.Len()
	out := linearExport{
		lines: make([][]uv.Cell, 0, n),
		wrap:  make([]bool, 0, n),
	}
	if n == 0 || sb.maxLines <= 0 {
		return out
	}
	for i := 0; i < n; i++ {
		phys := (sb.head + i) % sb.maxLines
		if sb.lines[phys] == nil {
			continue
		}
		out.lines = append(out.lines, sb.lines[phys])
		out.wrap = append(out.wrap, sb.softWrapped[phys])
	}
	return out
}

func flattenForReflow(line []uv.Cell) []uv.Cell {
	if len(line) == 0 {
		return nil
	}

	// Trim right padding that is truly "empty", and trim trailing continuation cells.
	end := len(line)
	for end > 0 {
		c := line[end-1]
		if c.Width == 0 {
			end--
			continue
		}
		if isBlankCell(c) {
			end--
			continue
		}
		break
	}
	if end <= 0 {
		return nil
	}

	out := make([]uv.Cell, 0, end)
	for i := 0; i < end; i++ {
		c := line[i]
		if c.Width == 0 {
			continue
		}
		if c.Width <= 0 {
			c.Width = 1
		}
		// Treat empty content as a space for correct wrapping.
		if c.Content == "" {
			c.Content = " "
		}
		out = append(out, c)
	}
	return out
}

func wrapCells(glyphs []uv.Cell, width int, lastSoftWrapped bool) ([][]uv.Cell, []bool) {
	if width <= 0 {
		return nil, nil
	}

	blank := uv.EmptyCell
	if blank.Width == 0 {
		blank.Width = 1
	}

	makeBlankLine := func() []uv.Cell {
		l := make([]uv.Cell, width)
		for i := 0; i < width; i++ {
			l[i] = blank
		}
		return l
	}

	// Empty logical line -> one blank line.
	if len(glyphs) == 0 {
		return [][]uv.Cell{makeBlankLine()}, []bool{lastSoftWrapped}
	}

	var lines [][]uv.Cell
	var wraps []bool

	line := makeBlankLine()
	col := 0

	flush := func(isSoft bool) {
		lines = append(lines, line)
		wraps = append(wraps, isSoft)
		line = makeBlankLine()
		col = 0
	}

	for gi := 0; gi < len(glyphs); gi++ {
		cell := glyphs[gi]
		w := cell.Width
		if w <= 0 {
			w = 1
			cell.Width = 1
		}
		if w > width {
			// Extremely wide glyph: place it alone (best-effort).
			w = 1
			cell.Width = 1
		}

		if col+w > width {
			flush(true)
		}

		// Place start cell.
		line[col] = cell

		// Place continuation cells (Width=0) for wide glyphs.
		if w > 1 {
			for k := 1; k < w && (col+k) < width; k++ {
				cont := blank
				cont.Width = 0
				cont.Style = cell.Style
				cont.Link = cell.Link
				line[col+k] = cont
			}
		}

		col += w
	}

	// Last line: preserve original last soft-wrap marker.
	lines = append(lines, line)
	wraps = append(wraps, lastSoftWrapped)

	// Any line before last is necessarily a soft wrap introduced by reflow.
	for i := 0; i < len(wraps)-1; i++ {
		wraps[i] = true
	}
	return lines, wraps
}

func isBlankCell(c uv.Cell) bool {
	if c.Content != "" && c.Content != " " {
		return false
	}
	if c.Width == 0 {
		return true
	}
	var zeroStyle uv.Style
	if !c.Style.Equal(&zeroStyle) {
		return false
	}
	if c.Link != (uv.Link{}) {
		return false
	}
	return true
}

func guessSoftWrapped(line []uv.Cell) bool {
	if len(line) == 0 {
		return false
	}

	last := -1
	var cell uv.Cell
	for i := len(line) - 1; i >= 0; i-- {
		if line[i].Width == 0 {
			continue
		}
		if isBlankCell(line[i]) {
			continue
		}
		last = i
		cell = line[i]
		break
	}
	if last == -1 {
		return false
	}
	width := cell.Width
	if width <= 0 {
		width = 1
	}
	end := last + width - 1
	return end >= len(line)-1
}

// extractLine copies a full screen-width line from a buffer.
func extractLine(buf *uv.Buffer, y, width int) []uv.Cell {
	line := make([]uv.Cell, width)
	for x := 0; x < width; x++ {
		if cell := buf.CellAt(x, y); cell != nil {
			line[x] = *cell
		} else {
			line[x] = uv.EmptyCell
		}
	}
	return line
}
