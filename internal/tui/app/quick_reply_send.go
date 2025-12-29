package app

import (
	"strings"

	"github.com/kballard/go-shellquote"
)

var (
	bracketedPasteStart = [...]byte{0x1b, '[', '2', '0', '0', '~'}
	bracketedPasteEnd   = [...]byte{0x1b, '[', '2', '0', '1', '~'}
)

func quickReplyTextBytes(pane PaneItem, text string) []byte {
	if quickReplyTargetIsCodex(pane) {
		return bracketedPasteBytes(text)
	}
	return []byte(text)
}

func quickReplyTargetIsCodex(pane PaneItem) bool {
	return commandIsCodex(pane.StartCommand) || commandIsCodex(pane.Command)
}

func commandIsCodex(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	args, err := shellquote.Split(command)
	if err != nil || len(args) == 0 {
		return false
	}
	switch strings.ToLower(baseNameAnySeparator(args[0])) {
	case "codex", "codex.exe":
		return true
	default:
		return false
	}
}

func baseNameAnySeparator(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.TrimRight(path, "/\\")
	if path == "" {
		return ""
	}
	idx := strings.LastIndexAny(path, "/\\")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}

func bracketedPasteBytes(text string) []byte {
	out := make([]byte, len(bracketedPasteStart)+len(text)+len(bracketedPasteEnd))
	n := copy(out, bracketedPasteStart[:])
	n += copy(out[n:], text)
	copy(out[n:], bracketedPasteEnd[:])
	return out
}
