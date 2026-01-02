package app

import (
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/agenttool"
)

var (
	bracketedPasteStart = [...]byte{0x1b, '[', '2', '0', '0', '~'}
	bracketedPasteEnd   = [...]byte{0x1b, '[', '2', '0', '1', '~'}
)

const (
	quickReplyClaudeSubmitDelay = 30 * time.Millisecond
	quickReplyCodexSubmitDelay  = 30 * time.Millisecond
)

func quickReplyTextBytes(pane PaneItem, text string) []byte {
	tool := quickReplyTargetTool(pane)
	if quickReplyToolUsesBracketedPaste(tool) {
		return bracketedPasteBytes(text)
	}
	return []byte(text)
}

func quickReplyInputBytes(pane PaneItem, text string) []byte {
	payload := quickReplyTextBytes(pane, text)
	submit := quickReplySubmitBytes(pane)
	if len(submit) == 0 || !quickReplyTargetCombineSubmit(pane) {
		return payload
	}
	out := make([]byte, 0, len(payload)+len(submit))
	out = append(out, payload...)
	out = append(out, submit...)
	return out
}

func quickReplySubmitBytes(pane PaneItem) []byte {
	return []byte{'\r'}
}

func quickReplyTargetIsCodex(pane PaneItem) bool {
	return quickReplyTargetTool(pane) == agenttool.ToolCodex
}

func quickReplyTargetCombineSubmit(pane PaneItem) bool {
	return false
}

func quickReplySubmitDelay(pane PaneItem) time.Duration {
	switch quickReplyTargetTool(pane) {
	case agenttool.ToolClaude:
		return quickReplyClaudeSubmitDelay
	case agenttool.ToolCodex:
		return quickReplyCodexSubmitDelay
	default:
		return 0
	}
}

func quickReplyTargetTool(pane PaneItem) agenttool.Tool {
	if tool := agenttool.Normalize(pane.Tool); tool != "" {
		return tool
	}
	if tool := agenttool.DetectFromCommand(pane.StartCommand); tool != "" {
		return tool
	}
	if tool := agenttool.DetectFromCommand(pane.Command); tool != "" {
		return tool
	}
	if tool := agenttool.DetectFromTitle(pane.Title); tool != "" {
		return tool
	}
	return ""
}

func quickReplyToolUsesBracketedPaste(tool agenttool.Tool) bool {
	return tool == agenttool.ToolCodex
}

func quickReplyToolFromText(text string) agenttool.Tool {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.ContainsAny(text, "\r\n") {
		return ""
	}
	return agenttool.DetectFromCommand(text)
}

func bracketedPasteBytes(text string) []byte {
	out := make([]byte, len(bracketedPasteStart)+len(text)+len(bracketedPasteEnd))
	n := copy(out, bracketedPasteStart[:])
	n += copy(out[n:], text)
	copy(out[n:], bracketedPasteEnd[:])
	return out
}
