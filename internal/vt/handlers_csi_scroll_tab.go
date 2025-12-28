package vt

import "github.com/charmbracelet/x/ansi"

func (e *Emulator) registerCsiScrollTabHandlers() {
	e.RegisterCsiHandler('S', func(params ansi.Params) bool {
		// Scroll Up [ansi.SU]
		n, _, _ := params.Param(0, 1)
		e.scr.ScrollUp(n)
		return true
	})

	e.RegisterCsiHandler('T', func(params ansi.Params) bool {
		// Scroll Down [ansi.SD]
		n, _, _ := params.Param(0, 1)
		e.scr.ScrollDown(n)
		return true
	})

	e.RegisterCsiHandler(ansi.Command('?', 0, 'W'), func(params ansi.Params) bool {
		// Set Tab at Every 8 Columns [ansi.DECST8C]
		if len(params) == 1 && params[0] == 5 {
			e.resetTabStops()
			return true
		}
		return false
	})

	e.RegisterCsiHandler('Z', func(params ansi.Params) bool {
		// Cursor Backward Tabulation [ansi.CBT]
		n, _, _ := params.Param(0, 1)
		e.prevTab(n)
		return true
	})

	e.RegisterCsiHandler('b', func(params ansi.Params) bool {
		// Repeat Previous Character [ansi.REP]
		n, _, _ := params.Param(0, 1)
		e.repeatPreviousCharacter(n)
		return true
	})

	e.RegisterCsiHandler('g', func(params ansi.Params) bool {
		// Tab Clear [ansi.TBC]
		value, _, _ := params.Param(0, 0)
		switch value {
		case 0:
			x, _ := e.scr.CursorPosition()
			e.tabstops.Reset(x)
		case 3:
			e.tabstops.Clear()
		default:
			return false
		}

		return true
	})
}
