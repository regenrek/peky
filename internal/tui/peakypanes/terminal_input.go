package peakypanes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func encodeKeyMsg(msg tea.KeyMsg) []byte {
	if msg.Type == tea.KeyRunes {
		return []byte(string(msg.Runes))
	}

	switch msg.String() {
	case "enter":
		return []byte{'\r'}
	case "tab":
		return []byte{'\t'}
	case "esc":
		return []byte{0x1b}
	case "backspace":
		return []byte{0x7f}
	case "up":
		return []byte("\x1b[A")
	case "down":
		return []byte("\x1b[B")
	case "right":
		return []byte("\x1b[C")
	case "left":
		return []byte("\x1b[D")
	case "home":
		return []byte("\x1b[H")
	case "end":
		return []byte("\x1b[F")
	case "pgup":
		return []byte("\x1b[5~")
	case "pgdown":
		return []byte("\x1b[6~")
	case "delete":
		return []byte("\x1b[3~")
	case "insert":
		return []byte("\x1b[2~")
	}

	// ctrl+<letter>
	if strings.HasPrefix(msg.String(), "ctrl+") && len(msg.String()) == len("ctrl+a") {
		ch := msg.String()[len("ctrl+") : len("ctrl+")+1][0]
		if ch >= 'a' && ch <= 'z' {
			return []byte{ch - 'a' + 1}
		}
	}

	// alt+<letter> as ESC prefix.
	if strings.HasPrefix(msg.String(), "alt+") && len(msg.String()) == len("alt+a") {
		ch := msg.String()[len("alt+") : len("alt+")+1]
		return []byte("\x1b" + ch)
	}

	return nil
}
