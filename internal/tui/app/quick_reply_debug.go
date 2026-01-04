package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/regenrek/peakypanes/internal/logging"
)

const quickReplyLogFieldLimit = 160

func logQuickReplySendAttempt(pane PaneItem, payload []byte) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	paneID := strings.TrimSpace(pane.ID)
	slog.Debug(
		"quick-reply: send",
		slog.String("pane_id", paneID),
		slog.String("tool", strings.TrimSpace(pane.Tool)),
		slog.String("title", quickReplyLogField("title", logging.SanitizeCommand(pane.Title))),
		slog.String("command", quickReplyLogField("command", logging.SanitizeCommand(pane.Command))),
		slog.String("start", quickReplyLogField("start", logging.SanitizeCommand(pane.StartCommand))),
		slog.Int("bytes", len(payload)),
		logging.PayloadAttr("payload", payload),
	)
}

func logQuickReplySendError(paneID string, err error) {
	if err == nil || !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	slog.Debug("quick-reply: send error", slog.String("pane_id", paneID), slog.Any("err", err))
}

func quickReplyLogField(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= quickReplyLogFieldLimit {
		return value
	}
	head := value[:quickReplyLogFieldLimit]
	extra := len(value) - quickReplyLogFieldLimit
	return fmt.Sprintf("%s...(+%d chars)", head, extra)
}
