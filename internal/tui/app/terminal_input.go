package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func encodeKeyMsg(msg tea.KeyMsg) []byte {
	if msg.Type == tea.KeyRunes {
		return []byte(string(msg.Runes))
	}
	if msg.Type == tea.KeySpace {
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
	" ":         {' '},
	"space":     {' '},
	"enter":     {'\r'},
	"tab":       {'\t'},
	"esc":       {0x1b},
	"backspace": {0x7f},
	"up":        []byte("\x1b[A"),
	"down":      []byte("\x1b[B"),
	"right":     []byte("\x1b[C"),
	"left":      []byte("\x1b[D"),
	"home":      []byte("\x1b[H"),
	"end":       []byte("\x1b[F"),
	"pgup":      []byte("\x1b[5~"),
	"pgdown":    []byte("\x1b[6~"),
	"delete":    []byte("\x1b[3~"),
	"insert":    []byte("\x1b[2~"),
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
