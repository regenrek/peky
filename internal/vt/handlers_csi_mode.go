package vt

import "github.com/charmbracelet/x/ansi"

func (e *Emulator) registerCsiModeHandlers() {
	e.RegisterCsiHandler('h', func(params ansi.Params) bool {
		// Set Mode [ansi.SM] - ANSI
		e.handleMode(params, true, true)
		return true
	})

	e.RegisterCsiHandler(ansi.Command('?', 0, 'h'), func(params ansi.Params) bool {
		// Set Mode [ansi.SM] - DEC
		e.handleMode(params, true, false)
		return true
	})

	e.RegisterCsiHandler('l', func(params ansi.Params) bool {
		// Reset Mode [ansi.RM] - ANSI
		e.handleMode(params, false, true)
		return true
	})

	e.RegisterCsiHandler(ansi.Command('?', 0, 'l'), func(params ansi.Params) bool {
		// Reset Mode [ansi.RM] - DEC
		e.handleMode(params, false, false)
		return true
	})

	e.RegisterCsiHandler('m', func(params ansi.Params) bool {
		// Select Graphic Rendition [ansi.SGR]
		e.handleSgr(params)
		return true
	})
}
