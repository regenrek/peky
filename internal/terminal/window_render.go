package terminal

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"reflect"
	"strings"

	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/muesli/termenv"
)

// ViewANSI returns the VT's own ANSI-rendered screen.
// This is often the most correct rendering (attrs, reverse-video, etc).
// It is cached for speed and assumes cursor styling is handled separately.
func (w *Window) ViewANSI() string {
	if w == nil {
		return ""
	}

	s, _ := w.ViewANSICtx(context.Background())
	return s
}

// ViewANSICtx returns the ANSI render with cancellation support.
func (w *Window) ViewANSICtx(ctx context.Context) (string, error) {
	if w == nil {
		return "", nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Only force sync render on first paint or after dimension changes.
	cols := w.cols
	rows := w.rows
	alt := w.altScreen.Load()

	w.cacheMu.Lock()
	cached := w.cacheANSI
	dirty := w.cacheDirty
	cacheCols := w.cacheCols
	cacheRows := w.cacheRows
	cacheAlt := w.cacheAltScreen
	w.cacheMu.Unlock()

	// Fast path: cache is clean and matches current dimensions.
	if !dirty && cached != "" && cacheCols == cols && cacheRows == rows && cacheAlt == alt {
		return cached, nil
	}

	// If we have a usable cached frame for current dims, return it and let the background renderer catch up.
	if cached != "" && cacheCols == cols && cacheRows == rows && cacheAlt == alt {
		w.RequestANSIRender()
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return cached, nil
	}

	// Slow path: no usable cached frame (startup or resize). Render once synchronously.
	w.refreshANSICache()
	cached, _ = w.ViewANSICached()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return cached, nil
}

// ViewANSICached returns the cached ANSI render and whether it is up to date.
func (w *Window) ViewANSICached() (string, bool) {
	if w == nil {
		return "", false
	}
	w.cacheMu.Lock()
	defer w.cacheMu.Unlock()
	return w.cacheANSI, !w.cacheDirty
}

func (w *Window) refreshANSICache() {
	if w == nil {
		return
	}

	startSeq := w.UpdateSeq()

	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return
	}
	s := term.Render()
	cols := term.Width()
	rows := term.Height()
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	alt := w.altScreen.Load()
	w.termMu.Unlock()
	endSeq := w.UpdateSeq()

	w.cacheMu.Lock()
	w.cacheANSI = s
	w.cacheCols = cols
	w.cacheRows = rows
	w.cacheAltScreen = alt
	w.cacheDirty = endSeq != startSeq
	w.cacheMu.Unlock()

	if endSeq != startSeq {
		w.RequestANSIRender()
	}
}

// ViewLipgloss renders the VT screen by walking cells and applying lipgloss styles.
// This is useful when you need to composite the pane inside other lipgloss layouts.
func (w *Window) ViewLipgloss(showCursor bool, profile termenv.Profile) string {
	if w == nil {
		return ""
	}
	out, _ := w.ViewLipglossCtx(context.Background(), showCursor, profile)
	return out
}

// ViewLipglossCtx renders the VT screen with cancellation support.
func (w *Window) ViewLipglossCtx(ctx context.Context, showCursor bool, profile termenv.Profile) (string, error) {
	if w == nil {
		return "", nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	snapshot := w.snapshotViewState()

	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return "", nil
	}

	state := buildViewRenderState(w, term, snapshot, showCursor)
	rows := term.Height()
	cols := term.Width()
	if rows <= 0 || cols <= 0 {
		w.termMu.Unlock()
		return "", nil
	}

	cellAt := makeCellAccessor(term, rows, state.topAbsY)

	blank := uv.EmptyCell
	if blank.Width <= 0 {
		blank.Width = 1
	}

	cells := make([]uv.Cell, cols*rows)
	for y := 0; y < rows; y++ {
		if err := ctx.Err(); err != nil {
			w.termMu.Unlock()
			return "", err
		}
		rowOff := y * cols
		for x := 0; x < cols; x++ {
			c := cellAt(x, y)
			if c != nil {
				cells[rowOff+x] = *c
			} else {
				cells[rowOff+x] = blank
			}
		}
	}
	w.termMu.Unlock()

	snapCellAt := func(x, y int) *uv.Cell {
		if x < 0 || x >= cols || y < 0 || y >= rows {
			return nil
		}
		return &cells[y*cols+x]
	}

	opts := RenderOptions{
		ShowCursor: state.showCursor,
		CursorX:    state.cursorX,
		CursorY:    state.cursorY,
		Highlight:  state.highlight,
		Profile:    profile,
	}

	return renderCellsLipglossCtx(ctx, cols, rows, snapCellAt, opts)
}

//
// VT -> Lipgloss rendering
//

// RenderOptions controls VT cell rendering.
type RenderOptions struct {
	ShowCursor bool
	CursorX    int
	CursorY    int

	Profile termenv.Profile

	// Optional: override cursor/selection highlights.
	Highlight func(x, y int) (cursor bool, selection bool)
}

type viewSnapshot struct {
	offset int
	sbMode bool
	cm     *CopyMode
}

type viewRenderState struct {
	topAbsY    int
	showCursor bool
	cursorX    int
	cursorY    int
	highlight  func(x, y int) (cursor bool, selection bool)
}

func (w *Window) snapshotViewState() viewSnapshot {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return viewSnapshot{
		offset: w.ScrollbackOffset,
		sbMode: w.ScrollbackMode,
		cm:     w.CopyMode,
	}
}

func buildViewRenderState(w *Window, term vtEmulator, snapshot viewSnapshot, showCursor bool) viewRenderState {
	offset := snapshot.offset
	sbMode := snapshot.sbMode
	cm := snapshot.cm

	if term.IsAltScreen() {
		offset = 0
		sbMode = false
		cm = nil
	}

	sbLen := term.ScrollbackLen()
	offset = clampInt(offset, 0, sbLen)
	topAbsY := sbLen - offset
	if topAbsY < 0 {
		topAbsY = 0
	}

	cur := term.CursorPosition()
	state := viewRenderState{
		topAbsY: topAbsY,
		cursorX: cur.X,
		cursorY: cur.Y,
	}

	state.showCursor = showCursor && w.cursorVisible.Load() && offset == 0 && (cm == nil || !cm.Active)
	if sbMode || offset > 0 {
		state.showCursor = false
	}
	if cm != nil && cm.Active {
		state.showCursor = false
		state.highlight = selectionHighlighter(topAbsY, cm)
	}

	return state
}

func makeCellAccessor(term vtEmulator, rows, topAbsY int) func(x, y int) *uv.Cell {
	sbLen := term.ScrollbackLen()
	return func(x, y int) *uv.Cell {
		absY := topAbsY + y
		if absY < sbLen {
			line := term.ScrollbackLine(absY)
			if line == nil || x < 0 || x >= len(line) {
				return nil
			}
			return &line[x]
		}
		screenY := absY - sbLen
		if screenY < 0 || screenY >= rows {
			return nil
		}
		return term.CellAt(x, screenY)
	}
}

func selectionHighlighter(topAbsY int, cm *CopyMode) func(x, y int) (cursor bool, selection bool) {
	startX, startY := cm.SelStartX, cm.SelStartAbsY
	endX, endY := cm.SelEndX, cm.SelEndAbsY
	if startY > endY || (startY == endY && startX > endX) {
		startX, endX = endX, startX
		startY, endY = endY, startY
	}

	return func(x, y int) (cursor bool, selection bool) {
		absY := topAbsY + y
		cursor = (absY == cm.CursorAbsY && x == cm.CursorX)
		if !cm.Selecting {
			return cursor, false
		}
		if absY < startY || absY > endY {
			return cursor, false
		}
		if startY == endY {
			return cursor, absY == startY && x >= startX && x <= endX
		}
		if absY == startY {
			return cursor, x >= startX
		}
		if absY == endY {
			return cursor, x <= endX
		}
		return cursor, true
	}
}

func cellContent(cell *uv.Cell) (string, int) {
	if cell == nil {
		return " ", 1
	}
	ch := " "
	if cell.Content != "" {
		ch = cell.Content
	}
	width := 1
	if cell.Width > 1 {
		width = cell.Width
	}
	return ch, width
}

// renderCellsLipgloss renders a cols x rows viewport using a cellAt accessor.
func renderCellsLipgloss(cols, rows int, cellAt func(x, y int) *uv.Cell, opts RenderOptions) string {
	out, _ := renderCellsLipglossCtx(context.Background(), cols, rows, cellAt, opts)
	return out
}

// renderCellsLipglossCtx renders a cols x rows viewport using a cellAt accessor with cancellation support.
func renderCellsLipglossCtx(ctx context.Context, cols, rows int, cellAt func(x, y int) *uv.Cell, opts RenderOptions) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if cols <= 0 || rows <= 0 || cellAt == nil {
		return "", nil
	}
	renderer := newLipglossRenderer(cols, rows, cellAt, opts)
	var b strings.Builder
	b.Grow(cols * rows)
	for y := 0; y < rows; y++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if y > 0 {
			b.WriteByte('\n')
		}
		renderer.renderRow(y, &b)
	}
	return b.String(), nil
}

type renderKey struct {
	fg, bg                  string
	bold, italic, underline bool
	reverse, strike, blink  bool
	cursor                  bool
	selection               bool
}

type lipglossRenderer struct {
	cols       int
	rows       int
	cellAt     func(x, y int) *uv.Cell
	opts       RenderOptions
	renderer   *lipgloss.Renderer
	styleCache map[renderKey]lipgloss.Style
}

func newLipglossRenderer(cols, rows int, cellAt func(x, y int) *uv.Cell, opts RenderOptions) *lipglossRenderer {
	renderer := lipgloss.NewRenderer(io.Discard)
	renderer.SetColorProfile(normalizeProfile(opts.Profile))
	return &lipglossRenderer{
		cols:       cols,
		rows:       rows,
		cellAt:     cellAt,
		opts:       opts,
		renderer:   renderer,
		styleCache: make(map[renderKey]lipgloss.Style, 128),
	}
}

func normalizeProfile(profile termenv.Profile) termenv.Profile {
	switch profile {
	case termenv.TrueColor, termenv.ANSI256, termenv.ANSI, termenv.Ascii:
		return profile
	default:
		return termenv.TrueColor
	}
}

func (r *lipglossRenderer) renderRow(y int, b *strings.Builder) {
	var run strings.Builder
	var prev renderKey
	var hasPrev bool

	flush := func() {
		if run.Len() == 0 {
			return
		}
		b.WriteString(r.renderText(prev, run.String()))
		run.Reset()
	}

	for x := 0; x < r.cols; {
		cell := r.cellAt(x, y)
		if cell != nil && cell.Width == 0 {
			x++
			continue
		}

		ch, w := cellContent(cell)
		kc := r.cellKey(x, y, cell)

		if !hasPrev {
			prev = kc
			hasPrev = true
		} else if kc != prev {
			flush()
			prev = kc
		}

		run.WriteString(ch)
		x += w
	}
	flush()
}

func (r *lipglossRenderer) renderText(k renderKey, text string) string {
	if text == "" {
		return ""
	}
	if k == (renderKey{}) {
		return text
	}
	return r.styleForKey(k).Render(text)
}

func (r *lipglossRenderer) styleForKey(k renderKey) lipgloss.Style {
	if st, ok := r.styleCache[k]; ok {
		return st
	}
	st := r.renderer.NewStyle()
	if k.fg != "" {
		st = st.Foreground(lipgloss.Color(k.fg))
	}
	if k.bg != "" {
		st = st.Background(lipgloss.Color(k.bg))
	}
	if k.bold {
		st = st.Bold(true)
	}
	if k.italic {
		st = st.Italic(true)
	}
	if k.underline {
		st = st.Underline(true)
	}
	if k.strike {
		st = st.Strikethrough(true)
	}
	if k.blink {
		st = st.Blink(true)
	}
	if k.reverse || k.selection {
		st = st.Reverse(true)
	}
	if k.cursor {
		st = st.Reverse(true).Bold(true)
	}
	r.styleCache[k] = st
	return st
}

func (r *lipglossRenderer) cellKey(x, y int, cell *uv.Cell) renderKey {
	kc := keyFromCell(cell)
	cursor, selection := r.highlightAt(x, y)
	if cursor {
		kc.cursor = true
	}
	if selection {
		kc.selection = true
	}
	return kc
}

func (r *lipglossRenderer) highlightAt(x, y int) (bool, bool) {
	if r.opts.Highlight != nil {
		return r.opts.Highlight(x, y)
	}
	if r.opts.ShowCursor && x == r.opts.CursorX && y == r.opts.CursorY {
		return true, false
	}
	return false, false
}

// RenderEmulatorLipgloss converts a VT emulator screen into a lipgloss-compatible string.
// It walks uv.Cells and batches runs with the same style to reduce ANSI churn.
func RenderEmulatorLipgloss(term interface {
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
}, cols, rows int, opts RenderOptions) string {
	if term == nil || cols <= 0 || rows <= 0 {
		return ""
	}

	cursor := term.CursorPosition()
	return renderCellsLipgloss(cols, rows, func(x, y int) *uv.Cell {
		return term.CellAt(x, y)
	}, RenderOptions{
		ShowCursor: opts.ShowCursor,
		CursorX:    cursor.X,
		CursorY:    cursor.Y,
		Highlight:  opts.Highlight,
		Profile:    opts.Profile,
	})
}

func keyFromCell(cell *uv.Cell) (k renderKey) {
	if cell == nil {
		return k
	}

	k.fg = colorToHex(cell.Style.Fg)
	k.bg = colorToHex(cell.Style.Bg)

	attrs := cell.Style.Attrs
	// Reflective feature detection keeps this resilient across uv.Attrs implementations.
	k.bold = attrsBool(attrs, "Bold")
	k.italic = attrsBool(attrs, "Italic")
	k.underline = attrsBool(attrs, "Underline")
	k.blink = attrsBool(attrs, "Blink")

	// Reverse is sometimes named Reverse or Inverse depending on implementation.
	k.reverse = attrsBool(attrs, "Reverse") || attrsBool(attrs, "Inverse")

	// Strikethrough naming varies.
	k.strike = attrsBool(attrs, "Strikethrough") || attrsBool(attrs, "Strike")

	return k
}

func attrsBool(attrs any, method string) bool {
	if attrs == nil || strings.TrimSpace(method) == "" {
		return false
	}
	v := reflect.ValueOf(attrs)

	// Try method on value.
	m := v.MethodByName(method)
	if !m.IsValid() && v.Kind() != reflect.Pointer && v.CanAddr() {
		// Try pointer receiver.
		m = v.Addr().MethodByName(method)
	}
	if !m.IsValid() {
		return false
	}
	t := m.Type()
	if t.NumIn() != 0 || t.NumOut() != 1 || t.Out(0).Kind() != reflect.Bool {
		return false
	}
	out := m.Call(nil)
	return len(out) == 1 && out[0].Bool()
}

func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
