package terminal

import (
	"context"
	"log/slog"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/logging"
	"github.com/regenrek/peakypanes/internal/termframe"
)

// ViewFrame returns the terminal's cached frame snapshot.
func (w *Window) ViewFrame() termframe.Frame {
	if w == nil {
		return termframe.Frame{}
	}
	frame, _ := w.ViewFrameCtx(context.Background())
	return frame
}

// ViewFrameCtx returns the frame snapshot with cancellation support.
func (w *Window) ViewFrameCtx(ctx context.Context) (termframe.Frame, error) {
	if w == nil {
		return termframe.Frame{}, nil
	}
	w.TouchFrameDemand(0)
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return termframe.Frame{}, err
	}

	cols := w.cols
	rows := w.rows
	alt := w.altScreen.Load()

	w.cacheMu.Lock()
	cached := w.cacheFrame
	dirty := w.cacheDirty
	cacheCols := w.cacheCols
	cacheRows := w.cacheRows
	cacheAlt := w.cacheAltScreen
	w.cacheMu.Unlock()

	if !dirty && !cached.Empty() && cacheCols == cols && cacheRows == rows && cacheAlt == alt {
		return cached, nil
	}

	if !cached.Empty() && cacheCols == cols && cacheRows == rows && cacheAlt == alt {
		w.RequestFrameRender()
		if err := ctx.Err(); err != nil {
			return termframe.Frame{}, err
		}
		return cached, nil
	}

	w.refreshFrameCache()
	cached, _ = w.ViewFrameCached()
	if err := ctx.Err(); err != nil {
		return termframe.Frame{}, err
	}
	return cached, nil
}

// ViewFrameDirectCtx renders a frame snapshot without using the cache.
func (w *Window) ViewFrameDirectCtx(ctx context.Context) (termframe.Frame, error) {
	if w == nil {
		return termframe.Frame{}, nil
	}
	w.TouchFrameDemand(0)
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return termframe.Frame{}, err
	}
	cells, cols, rows, state, err := w.collectFrameCells(ctx)
	if err != nil || cols <= 0 || rows <= 0 {
		return termframe.Frame{}, err
	}
	return frameFromCells(cells, cols, rows, state), nil
}

// ViewFrameCached returns the cached frame and whether it is up to date.
func (w *Window) ViewFrameCached() (termframe.Frame, bool) {
	if w == nil {
		return termframe.Frame{}, false
	}
	w.TouchFrameDemand(0)
	w.cacheMu.Lock()
	defer w.cacheMu.Unlock()
	return w.cacheFrame, !w.cacheDirty
}

// PreviewPlainLines returns the last max lines as plain text and whether the view is stable.
// It avoids full frame rendering to keep snapshot preview generation cheap.
func (w *Window) PreviewPlainLines(max int) ([]string, bool) {
	if w == nil || max <= 0 {
		return nil, false
	}
	if w.FirstReadAt().IsZero() {
		return nil, false
	}

	startSeq := w.UpdateSeq()

	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return nil, false
	}
	cols := term.Width()
	rows := term.Height()
	if cols <= 0 || rows <= 0 {
		w.termMu.Unlock()
		return nil, true
	}
	if max > rows {
		max = rows
	}
	startRow := rows - max
	lines := make([]string, 0, max)
	for y := startRow; y < rows; y++ {
		var b strings.Builder
		b.Grow(cols)
		for x := 0; x < cols; {
			cell := term.CellAt(x, y)
			if cell != nil && cell.Width == 0 {
				x++
				continue
			}
			ch := " "
			width := 1
			if cell != nil {
				if cell.Content != "" {
					ch = cell.Content
				}
				if cell.Width > 1 {
					width = cell.Width
				}
			}
			b.WriteString(ch)
			x += width
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	w.termMu.Unlock()

	endSeq := w.UpdateSeq()
	return lines, startSeq == endSeq
}

func (w *Window) refreshFrameCache() {
	if w == nil {
		return
	}
	if w.closed.Load() {
		return
	}

	startSeq := w.UpdateSeq()

	perf := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	var start time.Time
	if perf {
		start = time.Now()
	}

	cells, cols, rows, state, err := w.collectFrameCells(context.Background())
	if err != nil || cols < 0 || rows < 0 {
		return
	}
	frame := frameFromCells(cells, cols, rows, state)
	alt := w.altScreen.Load()
	endSeq := w.UpdateSeq()

	if perf {
		total := time.Since(start)
		if total > perfSlowFrameRender {
			logging.LogEvery(
				context.Background(),
				"term.frame.render",
				perfLogInterval,
				slog.LevelDebug,
				"terminal: frame render slow",
				slog.Duration("dur", total),
				slog.Int("cols", cols),
				slog.Int("rows", rows),
			)
		}
	}

	w.cacheMu.Lock()
	wasDirty := w.cacheDirty
	w.cacheFrame = frame
	w.cacheSeq = startSeq
	w.cacheCols = cols
	w.cacheRows = rows
	w.cacheAltScreen = alt
	w.cacheDirty = endSeq != startSeq
	nowDirty := w.cacheDirty
	w.cacheMu.Unlock()

	// When the cache transitions from dirty -> clean, publish an update even if no
	// new output arrived. This lets pane view consumers pull the settled frame and
	// avoids "cut off" snapshots after resizes.
	if wasDirty && !nowDirty && w.frameDemandActive() {
		select {
		case w.updates <- struct{}{}:
		default:
		}
	}

	if endSeq != startSeq && w.frameDemandActive() {
		w.RequestFrameRender()
	}
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (w *Window) collectFrameCells(ctx context.Context) ([]uv.Cell, int, int, viewRenderState, error) {
	snapshot := w.snapshotViewState()

	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return nil, 0, 0, viewRenderState{}, nil
	}

	state := buildViewRenderState(w, term, snapshot)
	rows := term.Height()
	cols := term.Width()
	if rows <= 0 || cols <= 0 {
		w.termMu.Unlock()
		return nil, 0, 0, viewRenderState{}, nil
	}

	cellAt := makeCellAccessor(term, cols, rows, state.topAbsY)

	blank := uv.EmptyCell
	if blank.Width <= 0 {
		blank.Width = 1
	}

	cells := make([]uv.Cell, cols*rows)
	for y := 0; y < rows; y++ {
		if err := ctx.Err(); err != nil {
			w.termMu.Unlock()
			return nil, 0, 0, viewRenderState{}, err
		}
		rowOff := y * cols
		for x := 0; x < cols; x++ {
			if c := cellAt(x, y); c != nil {
				cells[rowOff+x] = *c
			} else {
				cells[rowOff+x] = blank
			}
		}
	}
	w.termMu.Unlock()
	return cells, cols, rows, state, nil
}

func frameFromCells(cells []uv.Cell, cols, rows int, state viewRenderState) termframe.Frame {
	if cols <= 0 || rows <= 0 {
		return termframe.Frame{}
	}
	frame := termframe.Frame{
		Cols:  cols,
		Rows:  rows,
		Cells: make([]termframe.Cell, len(cells)),
		Cursor: termframe.Cursor{
			X:       state.cursorX,
			Y:       state.cursorY,
			Visible: state.showCursor,
		},
	}
	for i, cell := range cells {
		c := termframe.Cell{
			Content: cell.Content,
			Width:   cell.Width,
			Style:   frameStyleFromUV(cell.Style),
			Link: termframe.Link{
				URL:    cell.Link.URL,
				Params: cell.Link.Params,
			},
		}
		if state.highlight != nil {
			x := i % cols
			y := i / cols
			cursor, selection := state.highlight(x, y)
			if cursor || selection {
				c.Style.Attrs |= termframe.AttrReverse
				if cursor {
					c.Style.Attrs |= termframe.AttrBold
				}
			}
		}
		frame.Cells[i] = c
	}
	return frame
}

func frameStyleFromUV(style uv.Style) termframe.Style {
	return termframe.Style{
		Fg:             termframe.ColorFromColor(style.Fg),
		Bg:             termframe.ColorFromColor(style.Bg),
		UnderlineColor: termframe.ColorFromColor(style.UnderlineColor),
		UnderlineStyle: termframe.UnderlineStyle(style.Underline),
		Attrs:          termframe.Attrs(style.Attrs),
	}
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

func buildViewRenderState(w *Window, term vtEmulator, snapshot viewSnapshot) viewRenderState {
	offset := snapshot.offset
	sbMode := snapshot.sbMode
	cm := snapshot.cm

	if term.IsAltScreen() {
		if cm == nil || !cm.Active || !w.mouseSel.fromMouse {
			cm = nil
		}
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

	state.showCursor = w.cursorVisible.Load() && offset == 0 && (cm == nil || !cm.Active)
	if sbMode || offset > 0 {
		state.showCursor = false
	}
	if cm != nil && cm.Active {
		state.showCursor = false
		state.highlight = selectionHighlighter(topAbsY, cm)
	}

	return state
}

func makeCellAccessor(term vtEmulator, cols, rows, topAbsY int) func(x, y int) *uv.Cell {
	sbLen := term.ScrollbackLen()
	var sbRow []uv.Cell
	lastAbsY := -1
	return func(x, y int) *uv.Cell {
		absY := topAbsY + y
		if absY < sbLen {
			if cols <= 0 || x < 0 || x >= cols {
				return nil
			}
			if sbRow == nil || len(sbRow) != cols {
				sbRow = make([]uv.Cell, cols)
				lastAbsY = -1
			}
			if absY != lastAbsY {
				if ok := term.CopyScrollbackRow(absY, sbRow); !ok {
					return nil
				}
				lastAbsY = absY
			}
			return &sbRow[x]
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
