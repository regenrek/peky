package vt

import "github.com/charmbracelet/x/ansi"

func (e *Emulator) registerCsiMarginHandlers() {
	e.RegisterCsiHandler('r', func(params ansi.Params) bool {
		// Set Top and Bottom Margins [ansi.DECSTBM]
		top, _, _ := params.Param(0, 1)
		if top < 1 {
			top = 1
		}

		height := e.Height()
		bottom, _ := e.parser.Param(1, height)
		if bottom < 1 {
			bottom = height
		}

		if top >= bottom {
			return false
		}

		// Rect is [x, y) which means y is exclusive. So the top margin
		// is the top of the screen minus one.
		e.scr.setVerticalMargins(top-1, bottom)

		// Move the cursor to the top-left of the screen or scroll region
		// depending on [ansi.DECOM].
		e.setCursorPosition(0, 0)
		return true
	})

	e.RegisterCsiHandler('s', func(params ansi.Params) bool {
		// Set Left and Right Margins [ansi.DECSLRM]
		// These conflict with each other. When [ansi.DECSLRM] is set, the we
		// set the left and right margins. Otherwise, we save the cursor
		// position.

		if e.isModeSet(ansi.ModeLeftRightMargin) {
			// Set Left Right Margins [ansi.DECSLRM]
			left, _, _ := params.Param(0, 1)
			if left < 1 {
				left = 1
			}

			width := e.Width()
			right, _, _ := params.Param(1, width)
			if right < 1 {
				right = width
			}

			if left >= right {
				return false
			}

			e.scr.setHorizontalMargins(left-1, right)

			// Move the cursor to the top-left of the screen or scroll region
			// depending on [ansi.DECOM].
			e.setCursorPosition(0, 0)
		} else {
			// Save Current Cursor Position [ansi.SCOSC]
			e.scr.SaveCursor()
		}

		return true
	})
}
