package sessiond

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type restoreService struct {
	store *sessionrestore.Store
	cfg   sessionrestore.Config

	mu    sync.Mutex
	dirty map[string]struct{}
}

func newRestoreService(store *sessionrestore.Store, cfg sessionrestore.Config) *restoreService {
	return &restoreService{
		store: store,
		cfg:   cfg.Normalized(),
		dirty: make(map[string]struct{}),
	}
}

func (r *restoreService) Load(ctx context.Context) error {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.Load(ctx)
}

func (r *restoreService) MarkDirty(paneID string) {
	if r == nil {
		return
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return
	}
	r.mu.Lock()
	r.dirty[paneID] = struct{}{}
	r.mu.Unlock()
}

func (r *restoreService) MarkSessionDirty(ctx context.Context, mgr sessionManager, sessionName string) {
	if r == nil {
		return
	}
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" || mgr == nil {
		return
	}
	ctx = ensureContext(ctx)
	sessions := mgr.Snapshot(ctx, 0)
	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}
		for _, pane := range session.Panes {
			r.MarkDirty(pane.ID)
		}
	}
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (r *restoreService) DeletePane(paneID string) {
	if r == nil || r.store == nil {
		return
	}
	r.store.Delete(paneID)
}

func (r *restoreService) Snapshot(paneID string) (sessionrestore.PaneSnapshot, bool) {
	if r == nil || r.store == nil {
		return sessionrestore.PaneSnapshot{}, false
	}
	return r.store.Snapshot(paneID)
}

func (r *restoreService) Snapshots() []sessionrestore.PaneSnapshot {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.Snapshots()
}

func (r *restoreService) Flush(ctx context.Context, mgr sessionManager) error {
	if r == nil || r.store == nil {
		return nil
	}
	if mgr == nil {
		return errors.New("sessiond: manager unavailable")
	}
	ctx = ensureContext(ctx)
	sessions := mgr.Snapshot(ctx, 0)
	dirty := r.consumeDirty()
	live := make(map[string]struct{})

	for _, session := range sessions {
		for _, pane := range session.Panes {
			live[pane.ID] = struct{}{}
			if _, ok := dirty[pane.ID]; !ok && len(dirty) > 0 {
				continue
			}
			if !allowPanePersistence(pane, r.cfg) {
				r.store.Delete(pane.ID)
				continue
			}
			if err := r.snapshotPane(ctx, mgr, session, pane); err != nil {
				slog.Warn("sessiond: restore snapshot failed", slog.String("pane", pane.ID), slog.Any("err", err))
				r.MarkDirty(pane.ID)
			}
		}
	}
	return r.store.GC(ctx, live)
}

func (r *restoreService) consumeDirty() map[string]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.dirty) == 0 {
		return nil
	}
	out := r.dirty
	r.dirty = make(map[string]struct{})
	return out
}

func (r *restoreService) snapshotPane(ctx context.Context, mgr sessionManager, session native.SessionSnapshot, pane native.PaneSnapshot) error {
	win := mgr.Window(pane.ID)
	if win == nil {
		return errors.New("sessiond: pane window unavailable")
	}
	termSnap, err := win.SnapshotPlain(terminal.PlainSnapshotOptions{
		MaxScrollbackLines: r.cfg.MaxScrollbackLines,
	})
	if err != nil {
		return err
	}
	scrollback := termSnap.Scrollback
	if r.cfg.MaxScrollbackLines > 0 && len(scrollback) > r.cfg.MaxScrollbackLines {
		scrollback = scrollback[len(scrollback)-r.cfg.MaxScrollbackLines:]
	}
	if r.cfg.MaxScrollbackBytes > 0 {
		scrollback = sessionrestore.TrimLinesByBytes(scrollback, r.cfg.MaxScrollbackBytes)
	}
	term := sessionrestore.TerminalSnapshot{
		Cols:            termSnap.Cols,
		Rows:            termSnap.Rows,
		CursorX:         termSnap.CursorX,
		CursorY:         termSnap.CursorY,
		CursorVisible:   termSnap.CursorVisible,
		AltScreen:       termSnap.AltScreen,
		ScreenLines:     termSnap.ScreenLines,
		ScrollbackLines: scrollback,
	}
	restoreMode := pane.RestoreMode.String()
	snap := sessionrestore.PaneSnapshot{
		CapturedAt:        termSnap.CapturedAt,
		SessionName:       session.Name,
		SessionPath:       session.Path,
		SessionLayout:     session.LayoutName,
		SessionCreated:    session.CreatedAt,
		SessionEnv:        append([]string(nil), session.Env...),
		PaneID:            pane.ID,
		PaneIndex:         pane.Index,
		PaneTitle:         pane.Title,
		PaneCommand:       pane.Command,
		PaneStart:         pane.StartCommand,
		PaneTool:          pane.Tool,
		PaneCwd:           pane.Cwd,
		PaneActive:        pane.Active,
		PaneBackground:    pane.Background,
		PaneLeft:          pane.Left,
		PaneTop:           pane.Top,
		PaneWidth:         pane.Width,
		PaneHeight:        pane.Height,
		PaneDead:          pane.Dead,
		PaneDeadCode:      pane.DeadStatus,
		PaneLastAct:       pane.LastActive,
		PaneRestoreFailed: pane.RestoreFailed,
		PaneRestoreErr:    pane.RestoreError,
		PaneTags:          append([]string(nil), pane.Tags...),
		PaneBytesIn:       pane.BytesIn,
		PaneBytesOut:      pane.BytesOut,
		RestoreMode:       restoreMode,
		Private:           pane.RestoreMode.IsPrivate(),
		Terminal:          term,
	}
	return r.store.Save(ctx, snap)
}

func allowPanePersistence(pane native.PaneSnapshot, cfg sessionrestore.Config) bool {
	return pane.RestoreMode.AllowsPersistence(cfg.Enabled)
}
