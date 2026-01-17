package input

import (
	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
)

func toTeaMsg(ev uv.Event) (tea.Msg, bool) {
	switch e := ev.(type) {
	case uv.KeyPressEvent:
		return KeyMsg{Key: uv.Key(e)}, true
	case uv.MouseClickEvent:
		return tea.MouseMsg(toTeaMouse(uv.Mouse(e), tea.MouseActionPress)), true
	case uv.MouseReleaseEvent:
		return tea.MouseMsg(toTeaMouse(uv.Mouse(e), tea.MouseActionRelease)), true
	case uv.MouseWheelEvent:
		return tea.MouseMsg(toTeaMouse(uv.Mouse(e), tea.MouseActionPress)), true
	case uv.MouseMotionEvent:
		return tea.MouseMsg(toTeaMouse(uv.Mouse(e), tea.MouseActionMotion)), true
	case uv.PasteEvent:
		return KeyMsg{Key: uv.Key{Code: uv.KeyExtended, Text: e.Content}, Paste: true}, true
	default:
		return nil, false
	}
}

func toTeaMouse(m uv.Mouse, action tea.MouseAction) tea.MouseEvent {
	return tea.MouseEvent{
		X:      m.X,
		Y:      m.Y,
		Shift:  m.Mod.Contains(uv.ModShift),
		Alt:    m.Mod.Contains(uv.ModAlt),
		Ctrl:   m.Mod.Contains(uv.ModCtrl),
		Action: action,
		Button: tea.MouseButton(m.Button),
	}
}

func toTeaKey(k uv.Key) tea.Key {
	alt := k.Mod.Contains(uv.ModAlt)
	ctrl := k.Mod.Contains(uv.ModCtrl)
	shift := k.Mod.Contains(uv.ModShift)

	if t, ok := toTeaKeyCtrl(k.Code, ctrl); ok {
		return tea.Key{Type: t, Alt: alt}
	}
	if out, ok := toTeaKeySpecial(k.Code, alt, shift); ok {
		return out
	}
	if out, ok := toTeaKeyNav(k.Code, alt, shift, ctrl); ok {
		return out
	}
	if out, ok := toTeaKeyPg(k.Code, alt, ctrl); ok {
		return out
	}

	if k.Text != "" {
		return tea.Key{Type: tea.KeyRunes, Runes: []rune(k.Text), Alt: alt}
	}
	if k.Code != 0 {
		return tea.Key{Type: tea.KeyRunes, Runes: []rune{k.Code}, Alt: alt}
	}
	return tea.Key{}
}

func toTeaKeyCtrl(code rune, ctrl bool) (tea.KeyType, bool) {
	if !ctrl {
		return 0, false
	}
	return ctrlKeyType(code)
}

func toTeaKeySpecial(code rune, alt bool, shift bool) (tea.Key, bool) {
	switch code {
	case uv.KeySpace:
		return tea.Key{Type: tea.KeySpace, Alt: alt}, true
	case uv.KeyEnter:
		return tea.Key{Type: tea.KeyEnter, Alt: alt}, true
	case uv.KeyTab:
		if shift {
			return tea.Key{Type: tea.KeyShiftTab, Alt: alt}, true
		}
		return tea.Key{Type: tea.KeyTab, Alt: alt}, true
	case uv.KeyBackspace:
		return tea.Key{Type: tea.KeyBackspace, Alt: alt}, true
	case uv.KeyEscape:
		return tea.Key{Type: tea.KeyEsc, Alt: alt}, true
	case uv.KeyDelete:
		return tea.Key{Type: tea.KeyDelete, Alt: alt}, true
	case uv.KeyInsert:
		return tea.Key{Type: tea.KeyInsert, Alt: alt}, true
	default:
		return tea.Key{}, false
	}
}

func toTeaKeyNav(code rune, alt bool, shift bool, ctrl bool) (tea.Key, bool) {
	switch code {
	case uv.KeyUp:
		return tea.Key{Type: navKeyType(tea.KeyUp, tea.KeyShiftUp, tea.KeyCtrlUp, tea.KeyCtrlShiftUp, shift, ctrl), Alt: alt}, true
	case uv.KeyDown:
		return tea.Key{Type: navKeyType(tea.KeyDown, tea.KeyShiftDown, tea.KeyCtrlDown, tea.KeyCtrlShiftDown, shift, ctrl), Alt: alt}, true
	case uv.KeyLeft:
		return tea.Key{Type: navKeyType(tea.KeyLeft, tea.KeyShiftLeft, tea.KeyCtrlLeft, tea.KeyCtrlShiftLeft, shift, ctrl), Alt: alt}, true
	case uv.KeyRight:
		return tea.Key{Type: navKeyType(tea.KeyRight, tea.KeyShiftRight, tea.KeyCtrlRight, tea.KeyCtrlShiftRight, shift, ctrl), Alt: alt}, true
	case uv.KeyHome:
		return tea.Key{Type: navKeyType(tea.KeyHome, tea.KeyShiftHome, tea.KeyCtrlHome, tea.KeyCtrlShiftHome, shift, ctrl), Alt: alt}, true
	case uv.KeyEnd:
		return tea.Key{Type: navKeyType(tea.KeyEnd, tea.KeyShiftEnd, tea.KeyCtrlEnd, tea.KeyCtrlShiftEnd, shift, ctrl), Alt: alt}, true
	default:
		return tea.Key{}, false
	}
}

func toTeaKeyPg(code rune, alt bool, ctrl bool) (tea.Key, bool) {
	switch code {
	case uv.KeyPgUp:
		if ctrl {
			return tea.Key{Type: tea.KeyCtrlPgUp, Alt: alt}, true
		}
		return tea.Key{Type: tea.KeyPgUp, Alt: alt}, true
	case uv.KeyPgDown:
		if ctrl {
			return tea.Key{Type: tea.KeyCtrlPgDown, Alt: alt}, true
		}
		return tea.Key{Type: tea.KeyPgDown, Alt: alt}, true
	default:
		return tea.Key{}, false
	}
}

func navKeyType(base, shiftT, ctrlT, ctrlShiftT tea.KeyType, shift, ctrl bool) tea.KeyType {
	switch {
	case shift && ctrl:
		return ctrlShiftT
	case ctrl:
		return ctrlT
	case shift:
		return shiftT
	default:
		return base
	}
}

func ctrlKeyType(code rune) (tea.KeyType, bool) {
	switch {
	case code >= 'a' && code <= 'z':
		return tea.KeyType(code - 'a' + 1), true
	case code >= 'A' && code <= 'Z':
		return tea.KeyType(code - 'A' + 1), true
	case code == '[':
		return tea.KeyCtrlOpenBracket, true
	case code == '\\':
		return tea.KeyCtrlBackslash, true
	case code == ']':
		return tea.KeyCtrlCloseBracket, true
	case code == '^':
		return tea.KeyCtrlCaret, true
	case code == '_':
		return tea.KeyCtrlUnderscore, true
	case code == '?':
		return tea.KeyCtrlQuestionMark, true
	default:
		return 0, false
	}
}
