package vt

import "github.com/charmbracelet/x/ansi"

func (e *Emulator) registerCsiCursorHandlers() {
	e.RegisterCsiHandler('@', func(params ansi.Params) bool {
		// Insert Character [ansi.ICH]
		n, _, _ := params.Param(0, 1)
		e.scr.InsertCell(n)
		return true
	})

	e.RegisterCsiHandler('A', func(params ansi.Params) bool {
		// Cursor Up [ansi.CUU]
		n, _, _ := params.Param(0, 1)
		e.moveCursor(0, -n)
		return true
	})

	e.RegisterCsiHandler('B', func(params ansi.Params) bool {
		// Cursor Down [ansi.CUD]
		n, _, _ := params.Param(0, 1)
		e.moveCursor(0, n)
		return true
	})

	e.RegisterCsiHandler('C', func(params ansi.Params) bool {
		// Cursor Forward [ansi.CUF]
		n, _, _ := params.Param(0, 1)
		e.moveCursor(n, 0)
		return true
	})

	e.RegisterCsiHandler('D', func(params ansi.Params) bool {
		// Cursor Backward [ansi.CUB]
		n, _, _ := params.Param(0, 1)
		e.moveCursor(-n, 0)
		return true
	})

	e.RegisterCsiHandler('E', func(params ansi.Params) bool {
		// Cursor Next Line [ansi.CNL]
		n, _, _ := params.Param(0, 1)
		e.moveCursor(0, n)
		e.carriageReturn()
		return true
	})

	e.RegisterCsiHandler('F', func(params ansi.Params) bool {
		// Cursor Previous Line [ansi.CPL]
		n, _, _ := params.Param(0, 1)
		e.moveCursor(0, -n)
		e.carriageReturn()
		return true
	})

	e.RegisterCsiHandler('G', func(params ansi.Params) bool {
		// Cursor Horizontal Absolute [ansi.CHA]
		n, _, _ := params.Param(0, 1)
		_, y := e.scr.CursorPosition()
		e.setCursor(n-1, y)
		return true
	})

	e.RegisterCsiHandler('H', func(params ansi.Params) bool {
		// Cursor Position [ansi.CUP]
		width, height := e.Width(), e.Height()
		row, _, _ := params.Param(0, 1)
		col, _, _ := params.Param(1, 1)
		if row < 1 {
			row = 1
		}
		if col < 1 {
			col = 1
		}
		y := min(height-1, row-1)
		x := min(width-1, col-1)
		e.setCursorPosition(x, y)
		return true
	})

	e.RegisterCsiHandler('I', func(params ansi.Params) bool {
		// Cursor Horizontal Tabulation [ansi.CHT]
		n, _, _ := params.Param(0, 1)
		e.nextTab(n)
		return true
	})

	e.RegisterCsiHandler('`', func(params ansi.Params) bool {
		// Horizontal Position Absolute [ansi.HPA]
		n, _, _ := params.Param(0, 1)
		width := e.Width()
		_, y := e.scr.CursorPosition()
		e.setCursorPosition(min(width-1, n-1), y)
		return true
	})

	e.RegisterCsiHandler('a', func(params ansi.Params) bool {
		// Horizontal Position Relative [ansi.HPR]
		n, _, _ := params.Param(0, 1)
		width := e.Width()
		x, y := e.scr.CursorPosition()
		e.setCursorPosition(min(width-1, x+n), y)
		return true
	})

	e.RegisterCsiHandler('d', func(params ansi.Params) bool {
		// Vertical Position Absolute [ansi.VPA]
		n, _, _ := params.Param(0, 1)
		height := e.Height()
		x, _ := e.scr.CursorPosition()
		e.setCursorPosition(x, min(height-1, n-1))
		return true
	})

	e.RegisterCsiHandler('e', func(params ansi.Params) bool {
		// Vertical Position Relative [ansi.VPR]
		n, _, _ := params.Param(0, 1)
		height := e.Height()
		x, y := e.scr.CursorPosition()
		e.setCursorPosition(x, min(height-1, y+n))
		return true
	})

	e.RegisterCsiHandler('f', func(params ansi.Params) bool {
		// Horizontal and Vertical Position [ansi.HVP]
		width, height := e.Width(), e.Height()
		row, _, _ := params.Param(0, 1)
		col, _, _ := params.Param(1, 1)
		y := min(height-1, row-1)
		x := min(width-1, col-1)
		e.setCursor(x, y)
		return true
	})
}
