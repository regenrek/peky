package app

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/diag"
)

const quickReplyLogPreviewLimit = 256
const quickReplyLogFieldLimit = 160

func logQuickReplySendAttempt(pane PaneItem, payload []byte) {
	if !diag.Enabled() {
		return
	}
	paneID := strings.TrimSpace(pane.ID)
	diag.Logf(
		"quick-reply: send pane=%s codex=%v title=%s command=%s start=%s bytes=%d payload=%s",
		paneID,
		quickReplyTargetIsCodex(pane),
		quickReplyLogField("title", pane.Title),
		quickReplyLogField("command", pane.Command),
		quickReplyLogField("start", pane.StartCommand),
		len(payload),
		quickReplyPayloadPreview(payload),
	)
}

func logQuickReplySendError(paneID string, err error) {
	if !diag.Enabled() || err == nil {
		return
	}
	diag.Logf("quick-reply: send error pane=%s err=%v", paneID, err)
}

func quickReplyPayloadPreview(payload []byte) string {
	if len(payload) == 0 {
		return `""`
	}
	if len(payload) <= quickReplyLogPreviewLimit {
		return fmt.Sprintf("%q", payload)
	}
	head := payload[:quickReplyLogPreviewLimit]
	extra := len(payload) - quickReplyLogPreviewLimit
	return fmt.Sprintf("%q...(+%d bytes)", head, extra)
}

func quickReplyLogField(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return `""`
	}
	if len(value) <= quickReplyLogFieldLimit {
		return fmt.Sprintf("%q", value)
	}
	head := value[:quickReplyLogFieldLimit]
	extra := len(value) - quickReplyLogFieldLimit
	return fmt.Sprintf("%q...(+%d chars)", head, extra)
}
