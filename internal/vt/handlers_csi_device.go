package vt

import (
	"io"

	"github.com/charmbracelet/x/ansi"
)

func (e *Emulator) registerCsiDeviceHandlers() {
	e.RegisterCsiHandler('c', func(params ansi.Params) bool {
		// Primary Device Attributes [ansi.DA1]
		n, _, _ := params.Param(0, 0)
		if n != 0 {
			return false
		}

		// Do we fully support VT220?
		_, _ = io.WriteString(e.pw, ansi.PrimaryDeviceAttributes(
			62, // VT220
			1,  // 132 columns
			6,  // Selective Erase
			22, // ANSI color
		))
		return true
	})

	e.RegisterCsiHandler(ansi.Command('>', 0, 'c'), func(params ansi.Params) bool {
		// Secondary Device Attributes [ansi.DA2]
		n, _, _ := params.Param(0, 0)
		if n != 0 {
			return false
		}

		// Do we fully support VT220?
		_, _ = io.WriteString(e.pw, ansi.SecondaryDeviceAttributes(
			1,  // VT220
			10, // Version 1.0
			0,  // ROM Cartridge is always zero
		))
		return true
	})

	e.RegisterCsiHandler('n', func(params ansi.Params) bool {
		// Device Status Report [ansi.DSR]
		n, _, ok := params.Param(0, 1)
		if !ok || n == 0 {
			return false
		}

		switch n {
		case 5: // Operating Status
			// We're always ready ;)
			// See: https://vt100.net/docs/vt510-rm/DSR-OS.html
			_, _ = io.WriteString(e.pw, ansi.DeviceStatusReport(ansi.DECStatusReport(0)))
		case 6: // Cursor Position Report [ansi.CPR]
			x, y := e.scr.CursorPosition()
			_, _ = io.WriteString(e.pw, ansi.CursorPositionReport(y+1, x+1))
		default:
			return false
		}

		return true
	})

	e.RegisterCsiHandler(ansi.Command('?', 0, 'n'), func(params ansi.Params) bool {
		n, _, ok := params.Param(0, 1)
		if !ok || n == 0 {
			return false
		}

		switch n {
		case 6: // Extended Cursor Position Report [ansi.DECXCPR]
			x, y := e.scr.CursorPosition()
			_, _ = io.WriteString(e.pw, ansi.ExtendedCursorPositionReport(y+1, x+1, 0)) // We don't support page numbers //nolint:errcheck
		default:
			return false
		}

		return true
	})

	e.RegisterCsiHandler(ansi.Command(0, '$', 'p'), func(params ansi.Params) bool {
		// Request Mode [ansi.DECRQM] - ANSI
		e.handleRequestMode(params, true)
		return true
	})

	e.RegisterCsiHandler(ansi.Command('?', '$', 'p'), func(params ansi.Params) bool {
		// Request Mode [ansi.DECRQM] - DEC
		e.handleRequestMode(params, false)
		return true
	})

	e.RegisterCsiHandler(ansi.Command(0, ' ', 'q'), func(params ansi.Params) bool {
		// Set Cursor Style [ansi.DECSCUSR]
		n := 1
		if param, _, ok := params.Param(0, 0); ok && param > n {
			n = param
		}
		blink := n == 0 || n%2 == 1
		style := n / 2
		if !blink {
			style--
		}
		e.scr.setCursorStyle(CursorStyle(style), blink)
		return true
	})
}
