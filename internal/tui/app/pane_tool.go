package app

import (
	"context"
	"strings"

	"github.com/regenrek/peakypanes/internal/agenttool"
)

func (m *Model) setPaneTool(paneID string, tool agenttool.Tool) {
	if m == nil || m.client == nil {
		return
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" || tool == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
	defer cancel()
	if err := m.client.SetPaneTool(ctx, paneID, string(tool)); err != nil {
		logQuickReplySendError(paneID, err)
	}
}
