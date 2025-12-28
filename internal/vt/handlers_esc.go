package vt

import "github.com/charmbracelet/x/ansi"

// registerDefaultEscHandlers registers the default ESC escape sequence handlers.
func (e *Emulator) registerDefaultEscHandlers() {
	e.RegisterEscHandler('=', func() bool {
		// Keypad Application Mode [ansi.DECKPAM]
		e.setMode(ansi.ModeNumericKeypad, ansi.ModeSet)
		return true
	})

	e.RegisterEscHandler('>', func() bool {
		// Keypad Numeric Mode [ansi.DECKPNM]
		e.setMode(ansi.ModeNumericKeypad, ansi.ModeReset)
		return true
	})

	e.RegisterEscHandler('7', func() bool {
		// Save Cursor [ansi.DECSC]
		e.scr.SaveCursor()
		return true
	})

	e.RegisterEscHandler('8', func() bool {
		// Restore Cursor [ansi.DECRC]
		e.scr.RestoreCursor()
		return true
	})

	for _, cmd := range []int{
		ansi.Command(0, '(', 'A'), // UK G0
		ansi.Command(0, ')', 'A'), // UK G1
		ansi.Command(0, '*', 'A'), // UK G2
		ansi.Command(0, '+', 'A'), // UK G3
		ansi.Command(0, '(', 'B'), // USASCII G0
		ansi.Command(0, ')', 'B'), // USASCII G1
		ansi.Command(0, '*', 'B'), // USASCII G2
		ansi.Command(0, '+', 'B'), // USASCII G3
		ansi.Command(0, '(', '0'), // Special G0
		ansi.Command(0, ')', '0'), // Special G1
		ansi.Command(0, '*', '0'), // Special G2
		ansi.Command(0, '+', '0'), // Special G3
	} {
		e.RegisterEscHandler(cmd, func() bool {
			// Select Character Set [ansi.SCS]
			c := ansi.Cmd(cmd)
			set := c.Intermediate() - '('
			switch c.Final() {
			case 'A': // UK Character Set
				e.charsets[set] = UK
			case 'B': // USASCII Character Set
				e.charsets[set] = nil // USASCII is the default
			case '0': // Special Drawing Character Set
				e.charsets[set] = SpecialDrawing
			default:
				return false
			}
			return true
		})
	}

	e.RegisterEscHandler('D', func() bool {
		// Index [ansi.IND]
		e.index()
		return true
	})

	e.RegisterEscHandler('H', func() bool {
		// Horizontal Tab Set [ansi.HTS]
		e.horizontalTabSet()
		return true
	})

	e.RegisterEscHandler('M', func() bool {
		// Reverse Index [ansi.RI]
		e.reverseIndex()
		return true
	})

	e.RegisterEscHandler('c', func() bool {
		// Reset Initial State [ansi.RIS]
		e.fullReset()
		return true
	})

	e.RegisterEscHandler('n', func() bool {
		// Locking Shift G2 [ansi.LS2]
		e.gl = 2
		return true
	})

	e.RegisterEscHandler('o', func() bool {
		// Locking Shift G3 [ansi.LS3]
		e.gl = 3
		return true
	})

	e.RegisterEscHandler('|', func() bool {
		// Locking Shift 3 Right [ansi.LS3R]
		e.gr = 3
		return true
	})

	e.RegisterEscHandler('}', func() bool {
		// Locking Shift 2 Right [ansi.LS2R]
		e.gr = 2
		return true
	})

	e.RegisterEscHandler('~', func() bool {
		// Locking Shift 1 Right [ansi.LS1R]
		e.gr = 1
		return true
	})
}
