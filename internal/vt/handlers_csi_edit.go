package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func (e *Emulator) registerCsiEditHandlers() {
	e.RegisterCsiHandler('J', func(params ansi.Params) bool {
		// Erase in Display [ansi.ED]
		n, _, _ := params.Param(0, 0)
		width, height := e.Width(), e.Height()
		x, y := e.scr.CursorPosition()
		switch n {
		case 0: // Erase screen below (from after cursor position)
			rect1 := uv.Rect(x, y, width, 1)            // cursor to end of line
			rect2 := uv.Rect(0, y+1, width, height-y-1) // next line onwards
			e.scr.FillArea(e.scr.blankCell(), rect1)
			e.scr.FillArea(e.scr.blankCell(), rect2)
		case 1: // Erase screen above (including cursor)
			rect := uv.Rect(0, 0, width, y+1)
			e.scr.FillArea(e.scr.blankCell(), rect)
		case 2: // erase screen
			fallthrough
		case 3: // erase display
			//nolint:godox
			// TODO: Scrollback buffer support?
			e.scr.Clear()
		default:
			return false
		}
		return true
	})

	e.RegisterCsiHandler('K', func(params ansi.Params) bool {
		// Erase in Line [ansi.EL]
		n, _, _ := params.Param(0, 0)
		// NOTE: Erase Line (EL) erases all character attributes but not cell
		// bg color.
		x, y := e.scr.CursorPosition()
		w := e.scr.Width()

		switch n {
		case 0: // Erase from cursor to end of line
			e.eraseCharacter(w - x)
		case 1: // Erase from start of line to cursor
			rect := uv.Rect(0, y, x+1, 1)
			e.scr.FillArea(e.scr.blankCell(), rect)
		case 2: // Erase entire line
			rect := uv.Rect(0, y, w, 1)
			e.scr.FillArea(e.scr.blankCell(), rect)
		default:
			return false
		}
		return true
	})

	e.RegisterCsiHandler('L', func(params ansi.Params) bool {
		// Insert Line [ansi.IL]
		n, _, _ := params.Param(0, 1)
		if e.scr.InsertLine(n) {
			// Move the cursor to the left margin.
			e.scr.setCursorX(0, true)
		}
		return true
	})

	e.RegisterCsiHandler('M', func(params ansi.Params) bool {
		// Delete Line [ansi.DL]
		n, _, _ := params.Param(0, 1)
		if e.scr.DeleteLine(n) {
			// If the line was deleted successfully, move the cursor to the
			// left.
			// Move the cursor to the left margin.
			e.scr.setCursorX(0, true)
		}
		return true
	})

	e.RegisterCsiHandler('P', func(params ansi.Params) bool {
		// Delete Character [ansi.DCH]
		n, _, _ := params.Param(0, 1)
		e.scr.DeleteCell(n)
		return true
	})

	e.RegisterCsiHandler('X', func(params ansi.Params) bool {
		// Erase Character [ansi.ECH]
		n, _, _ := params.Param(0, 1)
		e.eraseCharacter(n)
		return true
	})
}
