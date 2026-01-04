package terminal

import (
	"strings"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
)

type wordClass uint8

const (
	wordSpace wordClass = iota
	wordAlpha
	wordPunct
)

func (w *Window) wordBoundsAt(absY, x int) (startX, endX int, ok bool) {
	cols, _, get, ok := w.wordCellGetter(absY)
	if !ok || cols <= 0 {
		return 0, 0, false
	}

	x = clampInt(x, 0, cols-1)
	x = moveToWordBase(get, x)

	base := get(x)
	class := classifyWordCell(base)

	start := scanWordLeft(get, x, class)
	end := scanWordRight(get, x, cols, class)
	return start, end, true
}

func (w *Window) wordCellGetter(absY int) (cols int, sbLen int, get func(int) uv.Cell, ok bool) {
	if w == nil {
		return 0, 0, nil, false
	}
	cols = w.cols
	rows := w.rows
	if cols <= 0 || rows <= 0 {
		return 0, 0, nil, false
	}

	w.termMu.Lock()
	term := w.term
	w.termMu.Unlock()
	if term == nil {
		return 0, 0, nil, false
	}

	w.termMu.Lock()
	sbLen = term.ScrollbackLen()
	w.termMu.Unlock()

	total := sbLen + rows
	if absY < 0 || absY >= total {
		return 0, 0, nil, false
	}

	var sbRow []uv.Cell
	get = func(cx int) uv.Cell {
		if cx < 0 || cx >= cols {
			return uv.EmptyCell
		}
		if absY < sbLen {
			if sbRow == nil {
				sbRow = make([]uv.Cell, cols)
				w.termMu.Lock()
				_ = term.CopyScrollbackRow(absY, sbRow)
				w.termMu.Unlock()
			}
			return sbRow[cx]
		}
		screenY := absY - sbLen
		w.termMu.Lock()
		cell := term.CellAt(cx, screenY)
		w.termMu.Unlock()
		if cell == nil {
			return uv.EmptyCell
		}
		return *cell
	}

	return cols, sbLen, get, true
}

func moveToWordBase(get func(int) uv.Cell, x int) int {
	for x > 0 && get(x).Width == 0 {
		x--
	}
	return x
}

func scanWordLeft(get func(int) uv.Cell, x int, class wordClass) int {
	start := x
	for start > 0 {
		prev := get(start - 1)
		if prev.Width == 0 {
			start--
			continue
		}
		if classifyWordCell(prev) != class {
			break
		}
		start--
	}
	return start
}

func scanWordRight(get func(int) uv.Cell, x, cols int, class wordClass) int {
	end := x
	for end < cols-1 {
		next := get(end + 1)
		if next.Width == 0 {
			end++
			continue
		}
		if classifyWordCell(next) != class {
			break
		}
		end++
	}
	return end
}

func classifyWordCell(c uv.Cell) wordClass {
	if c.IsZero() || c.Equal(&uv.EmptyCell) {
		return wordSpace
	}
	if strings.TrimSpace(c.Content) == "" {
		return wordSpace
	}
	r, _ := utf8.DecodeRuneInString(c.Content)
	if r == utf8.RuneError {
		return wordPunct
	}
	if unicode.IsSpace(r) {
		return wordSpace
	}
	if r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) {
		return wordAlpha
	}
	return wordPunct
}
