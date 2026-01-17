package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func encodeKeyMsg(msg tea.KeyMsg) []byte {
	if msg.Type == tea.KeyRunes {
		if msg.Alt {
			return []byte("\x1b" + string(msg.Runes))
		}
		return []byte(string(msg.Runes))
	}
	if msg.Type == tea.KeySpace {
		if msg.Alt {
			return []byte{0x1b, ' '}
		}
		return []byte{' '}
	}

	if seq, ok := keySequences[msg.String()]; ok {
		return seq
	}

	if seq := ctrlSequence(msg.String()); seq != nil {
		return seq
	}

	if seq := altSequence(msg.String()); seq != nil {
		return seq
	}

	return nil
}

var keySequences = map[string][]byte{
	" ":                    {' '},
	"space":                {' '},
	"enter":                {'\r'},
	"tab":                  {'\t'},
	"shift+tab":            []byte("\x1b[Z"),
	"esc":                  {0x1b},
	"backspace":            {0x7f},
	"up":                   []byte("\x1b[A"),
	"down":                 []byte("\x1b[B"),
	"right":                []byte("\x1b[C"),
	"left":                 []byte("\x1b[D"),
	"shift+up":             []byte("\x1b[1;2A"),
	"shift+down":           []byte("\x1b[1;2B"),
	"shift+right":          []byte("\x1b[1;2C"),
	"shift+left":           []byte("\x1b[1;2D"),
	"alt+up":               []byte("\x1b[1;3A"),
	"alt+down":             []byte("\x1b[1;3B"),
	"alt+right":            []byte("\x1b[1;3C"),
	"alt+left":             []byte("\x1b[1;3D"),
	"alt+shift+up":         []byte("\x1b[1;4A"),
	"alt+shift+down":       []byte("\x1b[1;4B"),
	"alt+shift+right":      []byte("\x1b[1;4C"),
	"alt+shift+left":       []byte("\x1b[1;4D"),
	"ctrl+up":              []byte("\x1b[1;5A"),
	"ctrl+down":            []byte("\x1b[1;5B"),
	"ctrl+right":           []byte("\x1b[1;5C"),
	"ctrl+left":            []byte("\x1b[1;5D"),
	"ctrl+shift+up":        []byte("\x1b[1;6A"),
	"ctrl+shift+down":      []byte("\x1b[1;6B"),
	"ctrl+shift+right":     []byte("\x1b[1;6C"),
	"ctrl+shift+left":      []byte("\x1b[1;6D"),
	"alt+ctrl+up":          []byte("\x1b[1;7A"),
	"alt+ctrl+down":        []byte("\x1b[1;7B"),
	"alt+ctrl+right":       []byte("\x1b[1;7C"),
	"alt+ctrl+left":        []byte("\x1b[1;7D"),
	"alt+ctrl+shift+up":    []byte("\x1b[1;8A"),
	"alt+ctrl+shift+down":  []byte("\x1b[1;8B"),
	"alt+ctrl+shift+right": []byte("\x1b[1;8C"),
	"alt+ctrl+shift+left":  []byte("\x1b[1;8D"),
	"home":                 []byte("\x1b[H"),
	"end":                  []byte("\x1b[F"),
	"shift+home":           []byte("\x1b[1;2H"),
	"shift+end":            []byte("\x1b[1;2F"),
	"alt+home":             []byte("\x1b[1;3H"),
	"alt+end":              []byte("\x1b[1;3F"),
	"alt+shift+home":       []byte("\x1b[1;4H"),
	"alt+shift+end":        []byte("\x1b[1;4F"),
	"ctrl+home":            []byte("\x1b[1;5H"),
	"ctrl+end":             []byte("\x1b[1;5F"),
	"ctrl+shift+home":      []byte("\x1b[1;6H"),
	"ctrl+shift+end":       []byte("\x1b[1;6F"),
	"alt+ctrl+home":        []byte("\x1b[1;7H"),
	"alt+ctrl+end":         []byte("\x1b[1;7F"),
	"alt+ctrl+shift+home":  []byte("\x1b[1;8H"),
	"alt+ctrl+shift+end":   []byte("\x1b[1;8F"),
	"pgup":                 []byte("\x1b[5~"),
	"pgdown":               []byte("\x1b[6~"),
	"alt+pgup":             []byte("\x1b[5;3~"),
	"alt+pgdown":           []byte("\x1b[6;3~"),
	"ctrl+pgup":            []byte("\x1b[5;5~"),
	"ctrl+pgdown":          []byte("\x1b[6;5~"),
	"alt+ctrl+pgup":        []byte("\x1b[5;7~"),
	"alt+ctrl+pgdown":      []byte("\x1b[6;7~"),
	"delete":               []byte("\x1b[3~"),
	"insert":               []byte("\x1b[2~"),
}

func ctrlSequence(key string) []byte {
	if !strings.HasPrefix(key, "ctrl+") || len(key) != len("ctrl+a") {
		return nil
	}
	ch := key[len("ctrl+") : len("ctrl+")+1][0]
	if ch < 'a' || ch > 'z' {
		return nil
	}
	return []byte{ch - 'a' + 1}
}

func altSequence(key string) []byte {
	if !strings.HasPrefix(key, "alt+") || len(key) != len("alt+a") {
		return nil
	}
	ch := key[len("alt+") : len("alt+")+1]
	return []byte("\x1b" + ch)
}
