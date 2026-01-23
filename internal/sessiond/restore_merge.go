package sessiond

import (
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
)

func (d *Daemon) mergeOfflineSessions(live []native.SessionSnapshot) []native.SessionSnapshot {
	if d == nil || d.restore == nil {
		return live
	}
	livePaneIDs, liveSessions := collectLivePaneIDs(live)
	offline := d.restore.Snapshots()
	if len(offline) == 0 {
		return live
	}
	grouped := groupOfflineSnapshots(offline, livePaneIDs, liveSessions)
	if len(grouped) == 0 {
		return live
	}
	return appendOfflineSessions(live, grouped)
}

func collectLivePaneIDs(live []native.SessionSnapshot) (map[string]struct{}, map[string]struct{}) {
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
	return livePaneIDs, liveSessions
}

func groupOfflineSnapshots(
	offline []sessionrestore.PaneSnapshot,
	livePaneIDs map[string]struct{},
	liveSessions map[string]struct{},
) map[string]*native.SessionSnapshot {
	grouped := make(map[string]*native.SessionSnapshot)
	for _, snap := range offline {
		if shouldSkipOfflineSnapshot(snap, livePaneIDs, liveSessions) {
			continue
		}
		group := grouped[snap.SessionName]
		if group == nil {
			group = newOfflineSessionSnapshot(snap)
			grouped[snap.SessionName] = group
		}
		group.Panes = append(group.Panes, buildOfflinePaneSnapshot(snap))
	}
	return grouped
}

func shouldSkipOfflineSnapshot(
	snap sessionrestore.PaneSnapshot,
	livePaneIDs map[string]struct{},
	liveSessions map[string]struct{},
) bool {
	if snap.PaneID == "" || snap.SessionName == "" {
		return true
	}
	if _, ok := livePaneIDs[snap.PaneID]; ok {
		return true
	}
	if _, ok := liveSessions[snap.SessionName]; ok {
		return true
	}
	return false
}

func newOfflineSessionSnapshot(snap sessionrestore.PaneSnapshot) *native.SessionSnapshot {
	return &native.SessionSnapshot{
		Name:       snap.SessionName,
		Path:       snap.SessionPath,
		LayoutName: snap.SessionLayout,
		CreatedAt:  snap.SessionCreated,
		Env:        append([]string(nil), snap.SessionEnv...),
	}
}

func buildOfflinePaneSnapshot(snap sessionrestore.PaneSnapshot) native.PaneSnapshot {
	mode, _ := sessionrestore.ParseMode(snap.RestoreMode)
	preview := make([]string, 0, len(snap.Terminal.ScrollbackLines)+len(snap.Terminal.ScreenLines))
	preview = append(preview, snap.Terminal.ScrollbackLines...)
	preview = append(preview, snap.Terminal.ScreenLines...)
	return native.PaneSnapshot{
		ID:            snap.PaneID,
		Index:         snap.PaneIndex,
		Title:         snap.PaneTitle,
		Command:       snap.PaneCommand,
		StartCommand:  snap.PaneStart,
		Tool:          snap.PaneTool,
		Cwd:           snap.PaneCwd,
		Active:        snap.PaneActive,
		Background:    normalizePaneBackground(snap.PaneBackground),
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
	}
}

func normalizePaneBackground(value int) int {
	if value < limits.PaneBackgroundMin || value > limits.PaneBackgroundMax {
		return limits.PaneBackgroundDefault
	}
	return value
}

func appendOfflineSessions(live []native.SessionSnapshot, grouped map[string]*native.SessionSnapshot) []native.SessionSnapshot {
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
