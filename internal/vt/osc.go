// Package vt provides a virtual terminal implementation.
// SKIP: Fix typecheck errors - function signature mismatches and undefined types
package vt

import (
	"bytes"
	"image/color"
	"io"

	"github.com/charmbracelet/x/ansi"
)

// handleOsc handles an OSC escape sequence.
func (e *Emulator) handleOsc(cmd int, data []byte) {
	e.flushGrapheme() // Flush any pending grapheme before handling OSC sequences.
	if !e.handlers.handleOsc(cmd, data) {
		e.logf("unhandled sequence: OSC %q", data)
	}
}

func (e *Emulator) handleTitle(cmd int, data []byte) {
	parts := bytes.Split(data, []byte{';'})
	if len(parts) != 2 {
		// Invalid, ignore
		return
	}
	switch cmd {
	case 0: // Set window title and icon name
		name := string(parts[1])
		e.iconName, e.title = name, name
		if e.cb.Title != nil {
			e.cb.Title(name)
		}
		if e.cb.IconName != nil {
			e.cb.IconName(name)
		}
	case 1: // Set icon name
		name := string(parts[1])
		e.iconName = name
		if e.cb.IconName != nil {
			e.cb.IconName(name)
		}
	case 2: // Set window title
		name := string(parts[1])
		e.title = name
		if e.cb.Title != nil {
			e.cb.Title(name)
		}
	}
}

func (e *Emulator) handleDefaultColor(cmd int, data []byte) {
	kind, ok := defaultColorKindForCmd(cmd)
	if !ok {
		return
	}
	parts := bytes.Split(data, []byte{';'})
	if len(parts) == 0 {
		return
	}
	switch len(parts) {
	case 1:
		e.applyDefaultColor(kind, nil)
	case 2:
		arg := string(parts[1])
		if arg == "?" {
			e.queryDefaultColor(kind)
			return
		}
		if c := ansi.XParseColor(arg); c != nil {
			e.applyDefaultColor(kind, c)
		}
	}
}

type defaultColorKind int

const (
	defaultColorForeground defaultColorKind = iota
	defaultColorBackground
	defaultColorCursor
)

func defaultColorKindForCmd(cmd int) (defaultColorKind, bool) {
	switch cmd {
	case 10, 110:
		return defaultColorForeground, true
	case 11, 111:
		return defaultColorBackground, true
	case 12, 112:
		return defaultColorCursor, true
	default:
		return 0, false
	}
}

func (e *Emulator) applyDefaultColor(kind defaultColorKind, c color.Color) {
	switch kind {
	case defaultColorForeground:
		e.SetForegroundColor(c)
	case defaultColorBackground:
		e.SetBackgroundColor(c)
	case defaultColorCursor:
		e.SetCursorColor(c)
	}
}

func (e *Emulator) queryDefaultColor(kind defaultColorKind) {
	var xrgb ansi.XRGBColor
	switch kind {
	case defaultColorForeground:
		xrgb.Color = e.ForegroundColor()
		if xrgb.Color != nil {
			io.WriteString(e.pw, ansi.SetForegroundColor(xrgb.String())) //nolint:errcheck,gosec
		}
	case defaultColorBackground:
		xrgb.Color = e.BackgroundColor()
		if xrgb.Color != nil {
			io.WriteString(e.pw, ansi.SetBackgroundColor(xrgb.String())) //nolint:errcheck,gosec
		}
	case defaultColorCursor:
		xrgb.Color = e.CursorColor()
		if xrgb.Color != nil {
			io.WriteString(e.pw, ansi.SetCursorColor(xrgb.String())) //nolint:errcheck,gosec
		}
	}
}

func (e *Emulator) handleWorkingDirectory(cmd int, data []byte) {
	if cmd != 7 {
		// Invalid, ignore
		return
	}

	// The data is the working directory path.
	parts := bytes.Split(data, []byte{';'})
	if len(parts) != 2 {
		// Invalid, ignore
		return
	}

	path := string(parts[1])
	e.cwd = path

	if e.cb.WorkingDirectory != nil {
		e.cb.WorkingDirectory(path)
	}
}

func (e *Emulator) handleHyperlink(cmd int, data []byte) {
	parts := bytes.Split(data, []byte{';'})
	if len(parts) != 3 || cmd != 8 {
		// Invalid, ignore
		return
	}

	e.scr.cur.Link.URL = string(parts[1])
	e.scr.cur.Link.Params = string(parts[2])
}
