package sessiond

import (
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
)

func (d *Daemon) mergeOfflineSessions(live []native.SessionSnapshot, previewLines int) []native.SessionSnapshot {
	if d == nil || d.restore == nil {
		return live
	}
	livePaneIDs := make(map[string]struct{})
	liveSessions := make(map[string]struct{})
	for _, session := range live {
		if session.Name != "" {
			liveSessions[session.Name] = struct{}{}
		}
		for _, pane := range session.Panes {
			if pane.ID != "" {
				livePaneIDs[pane.ID] = struct{}{}
			}
		}
	}

	offline := d.restore.Snapshots()
	if len(offline) == 0 {
		return live
	}

	grouped := make(map[string]*native.SessionSnapshot)
	for _, snap := range offline {
		if snap.PaneID == "" || snap.SessionName == "" {
			continue
		}
		if _, ok := livePaneIDs[snap.PaneID]; ok {
			continue
		}
		if _, ok := liveSessions[snap.SessionName]; ok {
			continue
		}
		group := grouped[snap.SessionName]
		if group == nil {
			group = &native.SessionSnapshot{
				Name:       snap.SessionName,
				Path:       snap.SessionPath,
				LayoutName: snap.SessionLayout,
				CreatedAt:  snap.SessionCreated,
				Env:        append([]string(nil), snap.SessionEnv...),
			}
			grouped[snap.SessionName] = group
		}
		mode, _ := sessionrestore.ParseMode(snap.RestoreMode)
		preview := append([]string(nil), snap.Terminal.ScreenLines...)
		if previewLines > 0 && len(preview) > previewLines {
			preview = preview[len(preview)-previewLines:]
		}
		group.Panes = append(group.Panes, native.PaneSnapshot{
			ID:            snap.PaneID,
			Index:         snap.PaneIndex,
			Title:         snap.PaneTitle,
			Command:       snap.PaneCommand,
			StartCommand:  snap.PaneStart,
			Tool:          snap.PaneTool,
			Cwd:           snap.PaneCwd,
			Active:        snap.PaneActive,
			Left:          snap.PaneLeft,
			Top:           snap.PaneTop,
			Width:         snap.PaneWidth,
			Height:        snap.PaneHeight,
			Dead:          snap.PaneDead,
			DeadStatus:    snap.PaneDeadCode,
			LastActive:    snap.PaneLastAct,
			RestoreFailed: snap.PaneRestoreFailed,
			RestoreError:  snap.PaneRestoreErr,
			RestoreMode:   mode,
			Disconnected:  true,
			SnapshotAt:    snap.CapturedAt,
			Tags:          append([]string(nil), snap.PaneTags...),
			BytesIn:       snap.PaneBytesIn,
			BytesOut:      snap.PaneBytesOut,
			Preview:       preview,
		})
	}

	if len(grouped) == 0 {
		return live
	}
	out := append([]native.SessionSnapshot(nil), live...)
	for _, session := range grouped {
		if session.CreatedAt.IsZero() {
			session.CreatedAt = time.Now()
		}
		sortPanesByIndex(session.Panes)
		out = append(out, *session)
	}
	return out
}

func sortPanesByIndex(panes []native.PaneSnapshot) {
	sort.Slice(panes, func(i, j int) bool {
		left := strings.TrimSpace(panes[i].Index)
		right := strings.TrimSpace(panes[j].Index)
		if left == right {
			return panes[i].ID < panes[j].ID
		}
		return left < right
	})
}
