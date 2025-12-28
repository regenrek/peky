package vt

import "github.com/charmbracelet/x/ansi"

// registerDefaultCcHandlers registers the default control character handlers.
func (e *Emulator) registerDefaultCcHandlers() {
	e.registerC0Handlers()
	e.registerC1Handlers()
}

func (e *Emulator) registerC0Handlers() {
	for i := byte(ansi.NUL); i <= ansi.US; i++ {
		switch i {
		case ansi.NUL: // Null [ansi.NUL]
			// Ignored
			e.registerCcHandler(i, func() bool {
				return true
			})
		case ansi.BEL: // Bell [ansi.BEL]
			e.registerCcHandler(i, func() bool {
				if e.cb.Bell != nil {
					e.cb.Bell()
				}
				return true
			})
		case ansi.BS: // Backspace [ansi.BS]
			e.registerCcHandler(i, func() bool {
				e.backspace()
				return true
			})
		case ansi.HT: // Horizontal Tab [ansi.HT]
			e.registerCcHandler(i, func() bool {
				e.nextTab(1)
				return true
			})
		case ansi.LF, ansi.VT, ansi.FF:
			// Line Feed [ansi.LF]
			// Vertical Tab [ansi.VT]
			// Form Feed [ansi.FF]
			e.registerCcHandler(i, func() bool {
				e.linefeed()
				return true
			})
		case ansi.CR: // Carriage Return [ansi.CR]
			e.registerCcHandler(i, func() bool {
				e.carriageReturn()
				return true
			})
		}
	}
}

func (e *Emulator) registerC1Handlers() {
	for i := byte(ansi.PAD); i <= byte(ansi.APC); i++ {
		switch i {
		case ansi.HTS: // Horizontal Tab Set [ansi.HTS]
			e.registerCcHandler(i, func() bool {
				e.horizontalTabSet()
				return true
			})
		case ansi.RI: // Reverse Index [ansi.RI]
			e.registerCcHandler(i, func() bool {
				e.reverseIndex()
				return true
			})
		case ansi.SO: // Shift Out [ansi.SO]
			e.registerCcHandler(i, func() bool {
				e.gl = 1
				return true
			})
		case ansi.SI: // Shift In [ansi.SI]
			e.registerCcHandler(i, func() bool {
				e.gl = 0
				return true
			})
		case ansi.IND: // Index [ansi.IND]
			e.registerCcHandler(i, func() bool {
				e.index()
				return true
			})
		case ansi.SS2: // Single Shift 2 [ansi.SS2]
			e.registerCcHandler(i, func() bool {
				e.gsingle = 2
				return true
			})
		case ansi.SS3: // Single Shift 3 [ansi.SS3]
			e.registerCcHandler(i, func() bool {
				e.gsingle = 3
				return true
			})
		}
	}
}
