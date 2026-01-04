package vt

import (
	"unsafe"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/limits"
)

// Scrollback stores content that has scrolled off the top of the visible screen.
//
// Unlike the previous per-line allocation model, this scrollback:
//   - is byte-budgeted (bounded memory per pane),
//   - stores logical lines compactly in paged backing memory,
//   - rewraps by rebuilding lightweight row indices (no per-row allocations),
//   - can render rows into a caller-provided buffer (no escape churn).
//
// Scrollback is not safe for concurrent use; callers must serialize access.
type Scrollback struct {
	maxBytes int64
	wrapW    int

	store cellStore

	// lines and lineRows are append-only slices with a moving head (lineStart)
	// to avoid O(n) shifts when pruning.
	lines     []logicalLine
	lineRows  []int // physical rows contributed by lines[i] at wrapW
	lineStart int

	// rows is an append-only slice with a moving head (rowStart).
	rows     []rowRef
	rowStart int

	// lastSoftWrapped tracks whether the last captured physical line was a soft
	// wrap (not a hard newline).
	lastSoftWrapped bool
}

type logicalLine struct {
	cellStart uint64
	cellLen   int
}

type rowRef struct {
	lineAbs    int
	glyphStart int
	glyphCount int
}

var cellBytes = int64(unsafe.Sizeof(uv.Cell{}))

func NewScrollback(maxBytes int64) *Scrollback {
	sb := &Scrollback{
		maxBytes: normalizeScrollbackMaxBytes(maxBytes),
		store:    newCellStore(),
	}
	return sb
}

func (sb *Scrollback) WrapWidth() int  { return sb.wrapW }
func (sb *Scrollback) MaxBytes() int64 { return sb.maxBytes }

// SetWrapWidth sets the width used to wrap logical lines into physical rows.
// This is called on resize and before capturing rows.
func (sb *Scrollback) SetWrapWidth(width int) {
	if sb == nil || width <= 0 {
		return
	}
	if width == sb.wrapW {
		return
	}
	sb.wrapW = width
	sb.rebuildRows()
}

// SetMaxBytes updates the scrollback byte budget. Negative values disable
// scrollback. Zero uses defaults.
func (sb *Scrollback) SetMaxBytes(maxBytes int64) {
	if sb == nil {
		return
	}
	sb.maxBytes = normalizeScrollbackMaxBytes(maxBytes)
	if sb.maxBytes <= 0 {
		sb.Clear()
		return
	}
	sb.enforceBudget()
}

// Len returns the number of physical rows in scrollback at the current wrap width.
func (sb *Scrollback) Len() int {
	if sb == nil {
		return 0
	}
	if sb.wrapW <= 0 {
		return 0
	}
	return len(sb.rows) - sb.rowStart
}

// Clear removes all scrollback content and releases backing pages.
func (sb *Scrollback) Clear() {
	if sb == nil {
		return
	}
	sb.store.Reset()
	for i := sb.lineStart; i < len(sb.lines); i++ {
		sb.lines[i] = logicalLine{}
	}
	sb.lines = sb.lines[:0]
	sb.lineRows = sb.lineRows[:0]
	sb.lineStart = 0
	sb.rows = sb.rows[:0]
	sb.rowStart = 0
	sb.lastSoftWrapped = false
}

// CopyRow copies the physical scrollback row at index into dst.
// Index 0 is the oldest row. Returns false if index is out of range.
//
// dst must have length >= WrapWidth(). Only the first WrapWidth() entries are written.
func (sb *Scrollback) CopyRow(index int, dst []uv.Cell) bool {
	if sb == nil || sb.wrapW <= 0 {
		return false
	}
	if index < 0 || index >= sb.Len() {
		return false
	}
	if len(dst) < sb.wrapW {
		return false
	}

	ref := sb.rows[sb.rowStart+index]
	line := sb.lines[ref.lineAbs]

	blank := uv.EmptyCell
	blank.Width = 1
	for i := 0; i < sb.wrapW; i++ {
		dst[i] = blank
	}

	col := 0
	startAbs := line.cellStart + uint64(ref.glyphStart)
	endAbs := startAbs + uint64(ref.glyphCount)
	for abs := startAbs; abs < endAbs && col < sb.wrapW; abs++ {
		cell, ok := sb.store.CellAt(abs)
		if !ok {
			break
		}

		w := glyphWidth(cell, sb.wrapW)
		if col+w > sb.wrapW {
			break
		}

		dst[col] = cell
		if w > 1 {
			for k := 1; k < w && (col+k) < sb.wrapW; k++ {
				cont := blank
				cont.Width = 0
				cont.Style = cell.Style
				cont.Link = cell.Link
				dst[col+k] = cont
			}
		}
		col += w
	}
	return true
}

// PushLineWithWrap captures a physical line with wrap metadata. This is primarily
// used by tests; production capture should use CaptureRowFromBuffer.
func (sb *Scrollback) PushLineWithWrap(line []uv.Cell, isSoftWrapped bool) {
	if sb == nil || sb.maxBytes <= 0 || sb.wrapW <= 0 {
		return
	}
	sb.pushFromCells(line, isSoftWrapped)
}

// CaptureRowFromBuffer captures the row y from buf into scrollback.
// It derives soft-wrap metadata from the row’s last visible cell and maintains
// logical-line grouping for correct rewrapping on resize.
func (sb *Scrollback) CaptureRowFromBuffer(buf *uv.Buffer, y int) {
	if sb == nil || sb.maxBytes <= 0 || buf == nil {
		return
	}
	width := buf.Width()
	if width <= 0 {
		return
	}
	sb.SetWrapWidth(width)
	soft := guessSoftWrappedRow(buf, y, width)
	sb.pushFromBuffer(buf, y, width, soft)
}

func normalizeScrollbackMaxBytes(maxBytes int64) int64 {
	switch {
	case maxBytes == 0:
		maxBytes = limits.TerminalScrollbackMaxBytesDefault
	case maxBytes < 0:
		maxBytes = 0
	}
	if maxBytes > limits.TerminalScrollbackMaxBytesMax {
		maxBytes = limits.TerminalScrollbackMaxBytesMax
	}
	return maxBytes
}

func (sb *Scrollback) pushFromCells(line []uv.Cell, isSoftWrapped bool) {
	startAbs := sb.store.NextAbs()
	n := appendGlyphsFromCells(&sb.store, line)
	sb.appendLogicalLine(startAbs, n, isSoftWrapped)
	sb.enforceBudget()
}

func (sb *Scrollback) pushFromBuffer(buf *uv.Buffer, y, width int, isSoftWrapped bool) {
	startAbs := sb.store.NextAbs()
	n := appendGlyphsFromBuffer(&sb.store, buf, y, width)
	sb.appendLogicalLine(startAbs, n, isSoftWrapped)
	sb.enforceBudget()
}

func (sb *Scrollback) appendLogicalLine(startAbs uint64, appended int, isSoftWrapped bool) {
	if sb.lastSoftWrapped && sb.activeLineCount() > 0 {
		lastAbs := len(sb.lines) - 1
		if sb.tryExtendTailLine(lastAbs, startAbs, appended) {
			sb.lastSoftWrapped = isSoftWrapped
			return
		}
	}

	lineAbs := len(sb.lines)
	sb.lines = append(sb.lines, logicalLine{cellStart: startAbs, cellLen: appended})
	sb.lineRows = append(sb.lineRows, 0)

	added := sb.appendRowsForLine(lineAbs)
	sb.lineRows[lineAbs] = added
	sb.lastSoftWrapped = isSoftWrapped
}

func (sb *Scrollback) tryExtendTailLine(lineAbs int, startAbs uint64, appended int) bool {
	line := &sb.lines[lineAbs]
	if appended > 0 {
		want := line.cellStart + uint64(line.cellLen)
		if want != startAbs {
			// Defensive: refuse to merge if caller invariants are violated.
			return false
		}
		line.cellLen += appended
	}

	// Remove existing row refs for this logical line and recompute with new content.
	prevRows := sb.lineRows[lineAbs]
	if prevRows > 0 {
		sb.rows = sb.rows[:len(sb.rows)-prevRows]
	}
	added := sb.appendRowsForLine(lineAbs)
	sb.lineRows[lineAbs] = added
	return true
}

func (sb *Scrollback) activeLineCount() int {
	return len(sb.lines) - sb.lineStart
}

func (sb *Scrollback) enforceBudget() {
	if sb.maxBytes <= 0 {
		return
	}
	for sb.activeLineCount() > 0 {
		bytesUsed := int64(sb.store.Len()) * cellBytes
		if bytesUsed <= sb.maxBytes {
			break
		}
		sb.dropOldestLine()
	}
	sb.maybeCompact()
}

func (sb *Scrollback) dropOldestLine() {
	idx := sb.lineStart
	if idx < 0 || idx >= len(sb.lines) {
		return
	}
	line := sb.lines[idx]
	sb.store.DropPrefix(line.cellLen)
	if idx < len(sb.lineRows) {
		sb.rowStart += sb.lineRows[idx]
		sb.lineRows[idx] = 0
	}
	sb.lines[idx] = logicalLine{}
	sb.lineStart++
	if sb.activeLineCount() == 0 {
		sb.lastSoftWrapped = false
	}
}

func (sb *Scrollback) maybeCompact() {
	const (
		minShift = 1024
	)
	if sb.lineStart < minShift {
		return
	}
	if sb.lineStart*2 <= len(sb.lines) {
		return
	}

	// Drop row head first so we only touch active rows.
	if sb.rowStart > 0 {
		sb.rows = append([]rowRef(nil), sb.rows[sb.rowStart:]...)
		sb.rowStart = 0
	}

	delta := sb.lineStart
	sb.lines = append([]logicalLine(nil), sb.lines[sb.lineStart:]...)
	sb.lineRows = append([]int(nil), sb.lineRows[sb.lineStart:]...)
	sb.lineStart = 0

	for i := range sb.rows {
		sb.rows[i].lineAbs -= delta
	}
}

func (sb *Scrollback) rebuildRows() {
	sb.rows = sb.rows[:0]
	sb.rowStart = 0
	for i := sb.lineStart; i < len(sb.lines); i++ {
		sb.lineRows[i] = 0
	}
	for lineAbs := sb.lineStart; lineAbs < len(sb.lines); lineAbs++ {
		added := sb.appendRowsForLine(lineAbs)
		sb.lineRows[lineAbs] = added
	}
}

func (sb *Scrollback) appendRowsForLine(lineAbs int) int {
	if sb.wrapW <= 0 {
		return 0
	}
	line := sb.lines[lineAbs]
	before := len(sb.rows)

	if line.cellLen == 0 {
		sb.rows = append(sb.rows, rowRef{lineAbs: lineAbs, glyphStart: 0, glyphCount: 0})
		return len(sb.rows) - before
	}

	col := 0
	startGlyph := 0
	for gi := 0; gi < line.cellLen; gi++ {
		cell, ok := sb.store.CellAt(line.cellStart + uint64(gi))
		if !ok {
			break
		}
		w := glyphWidth(cell, sb.wrapW)

		if col+w > sb.wrapW && col > 0 {
			sb.rows = append(sb.rows, rowRef{lineAbs: lineAbs, glyphStart: startGlyph, glyphCount: gi - startGlyph})
			startGlyph = gi
			col = 0
		}

		col += w
		if col == sb.wrapW {
			sb.rows = append(sb.rows, rowRef{lineAbs: lineAbs, glyphStart: startGlyph, glyphCount: gi + 1 - startGlyph})
			startGlyph = gi + 1
			col = 0
		}
	}
	if startGlyph < line.cellLen {
		sb.rows = append(sb.rows, rowRef{lineAbs: lineAbs, glyphStart: startGlyph, glyphCount: line.cellLen - startGlyph})
	}
	return len(sb.rows) - before
}

func glyphWidth(c uv.Cell, maxWidth int) int {
	w := c.Width
	if w <= 0 {
		w = 1
	}
	if maxWidth > 0 && w > maxWidth {
		w = 1
	}
	return w
}
