package vt

import uv "github.com/charmbracelet/ultraviolet"

// DamageState summarizes changes since last Consume.
// It is optimized for incremental renderers that cache per-row strings.
type DamageState struct {
	Width  int
	Height int

	// Full indicates the entire screen should be treated as dirty.
	Full bool

	// ScrollDy is the net full-screen vertical scroll delta since last Consume.
	// Negative means content moved up, positive means content moved down.
	ScrollDy int

	// DirtyRows contains row indices (0-based) that changed.
	// Rows are in the coordinate space AFTER applying ScrollDy to the previous cache.
	DirtyRows []int
}

// DamageTracker tracks conservative screen damage at row granularity plus full-screen scroll deltas.
// It is not thread safe; callers should serialize access (typically via the terminal mutex).
type DamageTracker struct {
	width  int
	height int

	full      bool
	dirtyRows []bool
	scrollDy  int
}

func (d *DamageTracker) Resize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	d.width = width
	d.height = height
	d.dirtyRows = make([]bool, height)
	d.full = true
	d.scrollDy = 0
}

func (d *DamageTracker) MarkAll() {
	d.full = true
	d.scrollDy = 0
	if len(d.dirtyRows) == 0 {
		return
	}
	for i := range d.dirtyRows {
		d.dirtyRows[i] = true
	}
}

func (d *DamageTracker) MarkRow(y int) {
	if d.full {
		return
	}
	if y < 0 || y >= d.height {
		return
	}
	if d.dirtyRows == nil || len(d.dirtyRows) != d.height {
		d.dirtyRows = make([]bool, d.height)
	}
	d.dirtyRows[y] = true
}

func (d *DamageTracker) MarkRect(rect uv.Rectangle) {
	if d.full {
		return
	}
	if d.height <= 0 {
		return
	}
	minY := rect.Min.Y
	maxY := rect.Max.Y
	if minY < 0 {
		minY = 0
	}
	if maxY > d.height {
		maxY = d.height
	}
	if minY >= maxY {
		return
	}
	if d.dirtyRows == nil || len(d.dirtyRows) != d.height {
		d.dirtyRows = make([]bool, d.height)
	}
	for y := minY; y < maxY; y++ {
		d.dirtyRows[y] = true
	}
}

// MarkScroll records a full-screen vertical scroll and shifts any pending dirty-row markers
// so they remain correct in the post-scroll coordinate space.
// It also marks the newly introduced blank rows as dirty so the renderer can paint correct
// background attributes for those rows.
func (d *DamageTracker) MarkScroll(dy int) {
	if d.shouldSkipScroll(dy) {
		return
	}
	if scrollOutOfBounds(dy, d.height) {
		d.MarkAll()
		return
	}

	d.scrollDy += dy
	if scrollOutOfBounds(d.scrollDy, d.height) {
		d.MarkAll()
		return
	}

	d.ensureDirtyRows()
	shiftDirtyRows(d.dirtyRows, dy)
	markScrollFillRows(d.dirtyRows, dy, d.height)
}

func (d *DamageTracker) shouldSkipScroll(dy int) bool {
	if d.full {
		d.scrollDy = 0
		return true
	}
	if d.height <= 0 || dy == 0 {
		return true
	}
	return false
}

func (d *DamageTracker) ensureDirtyRows() {
	if d.dirtyRows == nil || len(d.dirtyRows) != d.height {
		d.dirtyRows = make([]bool, d.height)
	}
}

func scrollOutOfBounds(dy, height int) bool {
	return dy >= height || dy <= -height
}

func markScrollFillRows(rows []bool, dy, height int) {
	if dy > 0 {
		limit := dy
		if limit > height {
			limit = height
		}
		for y := 0; y < limit; y++ {
			rows[y] = true
		}
		return
	}
	fill := -dy
	start := height - fill
	if start < 0 {
		start = 0
	}
	for y := start; y < height; y++ {
		rows[y] = true
	}
}

func shiftDirtyRows(rows []bool, dy int) {
	if dy == 0 || len(rows) == 0 {
		return
	}
	h := len(rows)
	if dy > 0 {
		for y := h - 1; y >= dy; y-- {
			rows[y] = rows[y-dy]
		}
		for y := 0; y < dy; y++ {
			rows[y] = false
		}
		return
	}

	dy = -dy
	for y := 0; y < h-dy; y++ {
		rows[y] = rows[y+dy]
	}
	for y := h - dy; y < h; y++ {
		rows[y] = false
	}
}

func (d *DamageTracker) Consume() DamageState {
	st := DamageState{
		Width:    d.width,
		Height:   d.height,
		Full:     d.full,
		ScrollDy: d.scrollDy,
	}
	if !d.full && d.height > 0 && d.dirtyRows != nil {
		dirty := make([]int, 0, 8)
		for y := 0; y < d.height; y++ {
			if d.dirtyRows[y] {
				dirty = append(dirty, y)
			}
		}
		st.DirtyRows = dirty
	}

	d.full = false
	d.scrollDy = 0
	if d.dirtyRows != nil {
		for i := range d.dirtyRows {
			d.dirtyRows[i] = false
		}
	}
	return st
}
