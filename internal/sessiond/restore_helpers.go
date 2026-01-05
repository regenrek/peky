package sessiond

import (
	"context"
	"strings"
)

func (d *Daemon) dropSessionSnapshots(ctx context.Context, manager sessionManager, sessionName string) {
	if d == nil || d.restore == nil || manager == nil {
		return
	}
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sessions := manager.Snapshot(ctx, 0)
	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}
		for _, pane := range session.Panes {
			if pane.ID != "" {
				d.restore.DeletePane(pane.ID)
			}
		}
	}
}

func paneIDForIndex(ctx context.Context, manager sessionManager, sessionName, paneIndex string) string {
	if manager == nil {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || paneIndex == "" {
		return ""
	}
	sessions := manager.Snapshot(ctx, 0)
	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}
		for _, pane := range session.Panes {
			if pane.Index == paneIndex {
				return pane.ID
			}
		}
	}
	return ""
}
