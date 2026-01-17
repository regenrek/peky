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

	if ctrl {
		if t, ok := ctrlKeyType(k.Code); ok {
			return tea.Key{Type: t, Alt: alt}
		}
	}

	switch k.Code {
	case uv.KeySpace:
		return tea.Key{Type: tea.KeySpace, Alt: alt}
	case uv.KeyEnter:
		return tea.Key{Type: tea.KeyEnter, Alt: alt}
	case uv.KeyTab:
		if shift {
			return tea.Key{Type: tea.KeyShiftTab, Alt: alt}
		}
		return tea.Key{Type: tea.KeyTab, Alt: alt}
	case uv.KeyBackspace:
		return tea.Key{Type: tea.KeyBackspace, Alt: alt}
	case uv.KeyEscape:
		return tea.Key{Type: tea.KeyEsc, Alt: alt}
	case uv.KeyUp:
		return tea.Key{Type: navKeyType(tea.KeyUp, tea.KeyShiftUp, tea.KeyCtrlUp, tea.KeyCtrlShiftUp, shift, ctrl), Alt: alt}
	case uv.KeyDown:
		return tea.Key{Type: navKeyType(tea.KeyDown, tea.KeyShiftDown, tea.KeyCtrlDown, tea.KeyCtrlShiftDown, shift, ctrl), Alt: alt}
	case uv.KeyLeft:
		return tea.Key{Type: navKeyType(tea.KeyLeft, tea.KeyShiftLeft, tea.KeyCtrlLeft, tea.KeyCtrlShiftLeft, shift, ctrl), Alt: alt}
	case uv.KeyRight:
		return tea.Key{Type: navKeyType(tea.KeyRight, tea.KeyShiftRight, tea.KeyCtrlRight, tea.KeyCtrlShiftRight, shift, ctrl), Alt: alt}
	case uv.KeyHome:
		return tea.Key{Type: navKeyType(tea.KeyHome, tea.KeyShiftHome, tea.KeyCtrlHome, tea.KeyCtrlShiftHome, shift, ctrl), Alt: alt}
	case uv.KeyEnd:
		return tea.Key{Type: navKeyType(tea.KeyEnd, tea.KeyShiftEnd, tea.KeyCtrlEnd, tea.KeyCtrlShiftEnd, shift, ctrl), Alt: alt}
	case uv.KeyPgUp:
		if ctrl {
			return tea.Key{Type: tea.KeyCtrlPgUp, Alt: alt}
		}
		return tea.Key{Type: tea.KeyPgUp, Alt: alt}
	case uv.KeyPgDown:
		if ctrl {
			return tea.Key{Type: tea.KeyCtrlPgDown, Alt: alt}
		}
		return tea.Key{Type: tea.KeyPgDown, Alt: alt}
	case uv.KeyDelete:
		return tea.Key{Type: tea.KeyDelete, Alt: alt}
	case uv.KeyInsert:
		return tea.Key{Type: tea.KeyInsert, Alt: alt}
	}

	if k.Text != "" {
		return tea.Key{Type: tea.KeyRunes, Runes: []rune(k.Text), Alt: alt}
	}
	if k.Code != 0 {
		return tea.Key{Type: tea.KeyRunes, Runes: []rune{k.Code}, Alt: alt}
	}
	return tea.Key{}
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
