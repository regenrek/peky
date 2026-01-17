package app

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"

	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

var teaKeyTypeToUVCode = map[tea.KeyType]rune{
	tea.KeyF1:        uv.KeyF1,
	tea.KeyF2:        uv.KeyF2,
	tea.KeyF3:        uv.KeyF3,
	tea.KeyF4:        uv.KeyF4,
	tea.KeyF5:        uv.KeyF5,
	tea.KeyF6:        uv.KeyF6,
	tea.KeyF7:        uv.KeyF7,
	tea.KeyF8:        uv.KeyF8,
	tea.KeyF9:        uv.KeyF9,
	tea.KeyF10:       uv.KeyF10,
	tea.KeyF11:       uv.KeyF11,
	tea.KeyF12:       uv.KeyF12,
	tea.KeySpace:     uv.KeySpace,
	tea.KeyEnter:     uv.KeyEnter,
	tea.KeyTab:       uv.KeyTab,
	tea.KeyEsc:       uv.KeyEscape,
	tea.KeyUp:        uv.KeyUp,
	tea.KeyDown:      uv.KeyDown,
	tea.KeyLeft:      uv.KeyLeft,
	tea.KeyRight:     uv.KeyRight,
	tea.KeyHome:      uv.KeyHome,
	tea.KeyEnd:       uv.KeyEnd,
	tea.KeyPgUp:      uv.KeyPgUp,
	tea.KeyPgDown:    uv.KeyPgDown,
	tea.KeyDelete:    uv.KeyDelete,
	tea.KeyInsert:    uv.KeyInsert,
	tea.KeyBackspace: uv.KeyBackspace,
}

func keyMsgFromTea(msg tea.KeyMsg) tuiinput.KeyMsg {
	k := uv.Key{}

	if msg.Alt {
		k.Mod |= uv.ModAlt
	}

	switch msg.Type {
	case tea.KeyRunes:
		k.Text = string(msg.Runes)
		if r, size := utf8.DecodeRuneInString(k.Text); r != utf8.RuneError && size == len(k.Text) {
			k.Code = r
		} else {
			k.Code = uv.KeyExtended
		}
	case tea.KeyShiftTab:
		k.Code = uv.KeyTab
		k.Mod |= uv.ModShift
	default:
		if code, ok := teaKeyTypeToUVCode[msg.Type]; ok {
			k.Code = code
			if msg.Type == tea.KeySpace {
				k.Text = " "
			}
		} else {
			// Best-effort ctrl key parsing.
			key := msg.String()
			if strings.HasPrefix(key, "ctrl+") && len(key) == len("ctrl+a") {
				ch := key[len("ctrl+") : len("ctrl+")+1]
				k.Mod |= uv.ModCtrl
				r, _ := utf8.DecodeRuneInString(ch)
				k.Code = r
			}
		}
	}

	return tuiinput.KeyMsg{Key: k, Paste: msg.Paste}
}
