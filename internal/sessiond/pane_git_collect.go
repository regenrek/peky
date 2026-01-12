package sessiond

import (
	"context"
	"strings"

	"github.com/regenrek/peakypanes/internal/native"
)

func (d *Daemon) collectPaneGit(ctx context.Context, sessions []native.SessionSnapshot) map[string]PaneGitMeta {
	if d == nil || d.paneGit == nil || len(sessions) == 0 {
		return nil
	}
	out := make(map[string]PaneGitMeta)
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == "" {
				continue
			}
			cwd := strings.TrimSpace(pane.Cwd)
			if cwd == "" {
				cwd = strings.TrimSpace(session.Path)
			}
			if cwd == "" {
				continue
			}
			meta, ok := d.paneGit.Meta(ctx, cwd)
			if !ok {
				continue
			}
			out[pane.ID] = meta
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
