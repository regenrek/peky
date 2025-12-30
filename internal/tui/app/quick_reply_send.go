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

func quickReplyInputBytes(pane PaneItem, text string) []byte {
	payload := quickReplyTextBytes(pane, text)
	submit := quickReplySubmitBytes(pane)
	if len(submit) == 0 {
		return payload
	}
	out := make([]byte, 0, len(payload)+len(submit))
	out = append(out, payload...)
	out = append(out, submit...)
	return out
}

func quickReplySubmitBytes(pane PaneItem) []byte {
	if quickReplyTargetIsCodex(pane) {
		return []byte{'\r'}
	}
	return []byte{'\r'}
}

func quickReplyTargetIsCodex(pane PaneItem) bool {
	return commandIsCodex(pane.StartCommand) || commandIsCodex(pane.Command) || titleIsCodex(pane.Title)
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
	for _, arg := range args {
		if codexArg(arg) {
			return true
		}
	}
	return false
}

func codexArg(arg string) bool {
	base := strings.ToLower(strings.TrimSpace(baseNameAnySeparator(arg)))
	if base == "" {
		return false
	}
	base = strings.TrimSuffix(base, ".exe")
	if base == "codex" {
		return true
	}
	return strings.HasPrefix(base, "codex@")
}

func titleIsCodex(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	fields := strings.Fields(strings.ToLower(title))
	if len(fields) == 0 {
		return false
	}
	first := strings.TrimSuffix(fields[0], ":")
	if first == "codex" {
		return true
	}
	return strings.HasPrefix(first, "codex@")
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
