package app

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"

	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

func keyMsgFromTea(msg tea.KeyMsg) tuiinput.KeyMsg {
	k := uv.Key{}

	if msg.Alt {
		k.Mod |= uv.ModAlt
	}

	switch msg.Type {
	case tea.KeyF1:
		k.Code = uv.KeyF1
	case tea.KeyF2:
		k.Code = uv.KeyF2
	case tea.KeyF3:
		k.Code = uv.KeyF3
	case tea.KeyF4:
		k.Code = uv.KeyF4
	case tea.KeyF5:
		k.Code = uv.KeyF5
	case tea.KeyF6:
		k.Code = uv.KeyF6
	case tea.KeyF7:
		k.Code = uv.KeyF7
	case tea.KeyF8:
		k.Code = uv.KeyF8
	case tea.KeyF9:
		k.Code = uv.KeyF9
	case tea.KeyF10:
		k.Code = uv.KeyF10
	case tea.KeyF11:
		k.Code = uv.KeyF11
	case tea.KeyF12:
		k.Code = uv.KeyF12
	case tea.KeySpace:
		k.Code = uv.KeySpace
		k.Text = " "
	case tea.KeyEnter:
		k.Code = uv.KeyEnter
	case tea.KeyTab:
		k.Code = uv.KeyTab
	case tea.KeyShiftTab:
		k.Code = uv.KeyTab
		k.Mod |= uv.ModShift
	case tea.KeyEsc:
		k.Code = uv.KeyEscape
	case tea.KeyUp:
		k.Code = uv.KeyUp
	case tea.KeyDown:
		k.Code = uv.KeyDown
	case tea.KeyLeft:
		k.Code = uv.KeyLeft
	case tea.KeyRight:
		k.Code = uv.KeyRight
	case tea.KeyHome:
		k.Code = uv.KeyHome
	case tea.KeyEnd:
		k.Code = uv.KeyEnd
	case tea.KeyPgUp:
		k.Code = uv.KeyPgUp
	case tea.KeyPgDown:
		k.Code = uv.KeyPgDown
	case tea.KeyDelete:
		k.Code = uv.KeyDelete
	case tea.KeyInsert:
		k.Code = uv.KeyInsert
	case tea.KeyBackspace:
		k.Code = uv.KeyBackspace
	case tea.KeyRunes:
		k.Text = string(msg.Runes)
		if r, size := utf8.DecodeRuneInString(k.Text); r != utf8.RuneError && size == len(k.Text) {
			k.Code = r
		} else {
			k.Code = uv.KeyExtended
		}
	default:
		// Best-effort ctrl key parsing.
		key := msg.String()
		if strings.HasPrefix(key, "ctrl+") && len(key) == len("ctrl+a") {
			ch := key[len("ctrl+") : len("ctrl+")+1]
			k.Mod |= uv.ModCtrl
			r, _ := utf8.DecodeRuneInString(ch)
			k.Code = r
		}
	}

	return tuiinput.KeyMsg{Key: k, Paste: msg.Paste}
}
