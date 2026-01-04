package native

import (
	"context"
	"log/slog"
	"time"
)

const (
	perfPanePtyCreated = 1 << iota
	perfPaneProcStart
	perfPaneIOStart
	perfPaneFirstUpdate
	perfPaneFirstRead
	perfPaneFirstWrite
	perfPaneFirstReadAfterWrite
)

type panePerfSnapshot struct {
	created             time.Time
	ptyCreated          time.Time
	procStarted         time.Time
	ioStarted           time.Time
	firstUpdate         time.Time
	firstRead           time.Time
	firstWrite          time.Time
	firstReadAfterWrite time.Time
}

func (m *Manager) logPanePerf(pane *Pane) {
	if m == nil || pane == nil || !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	w := pane.window
	if w == nil {
		return
	}
	snap := panePerfSnapshot{
		created:             w.CreatedAt(),
		ptyCreated:          w.PtyCreatedAt(),
		procStarted:         w.ProcessStartedAt(),
		ioStarted:           w.IOStartedAt(),
		firstUpdate:         w.FirstUpdateAt(),
		firstRead:           w.FirstReadAt(),
		firstWrite:          w.FirstWriteAt(),
		firstReadAfterWrite: w.FirstReadAfterWriteAt(),
	}

	m.perfMu.Lock()
	if m.perfLogged == nil {
		m.perfLogged = make(map[string]uint8)
	}
	flags := m.perfLogged[pane.ID]
	flags = logPanePerfStage(flags, perfPanePtyCreated, snap.ptyCreated, func() {
		slog.Debug("native: pane pty ready", slog.String("pane_id", pane.ID), slog.Duration("since_start", snap.ptyCreated.Sub(snap.created)))
	})
	flags = logPanePerfStage(flags, perfPaneProcStart, snap.procStarted, func() {
		slog.Debug("native: pane process started", slog.String("pane_id", pane.ID), slog.Int("pid", pane.PID), slog.Duration("since_start", snap.procStarted.Sub(snap.created)))
	})
	flags = logPanePerfStage(flags, perfPaneIOStart, snap.ioStarted, func() {
		slog.Debug("native: pane io started", slog.String("pane_id", pane.ID), slog.Duration("since_start", snap.ioStarted.Sub(snap.created)))
	})
	flags = logPanePerfStage(flags, perfPaneFirstUpdate, snap.firstUpdate, func() {
		slog.Debug("native: pane first update", slog.String("pane_id", pane.ID), slog.Duration("since_start", snap.firstUpdate.Sub(snap.created)))
	})
	flags = logPanePerfFirstRead(flags, pane.ID, snap)
	flags = logPanePerfStage(flags, perfPaneFirstWrite, snap.firstWrite, func() {
		slog.Debug("native: pane first input sent", slog.String("pane_id", pane.ID), slog.Duration("since_start", snap.firstWrite.Sub(snap.created)))
	})
	flags = logPanePerfFirstReadAfterWrite(flags, pane.ID, snap)
	m.perfLogged[pane.ID] = flags
	m.perfMu.Unlock()
}

func logPanePerfStage(flags uint8, flag uint8, at time.Time, logFn func()) uint8 {
	if at.IsZero() || flags&flag != 0 {
		return flags
	}
	flags |= flag
	logFn()
	return flags
}

func logPanePerfFirstRead(flags uint8, paneID string, snap panePerfSnapshot) uint8 {
	if snap.firstRead.IsZero() || flags&perfPaneFirstRead != 0 {
		return flags
	}
	flags |= perfPaneFirstRead
	slog.Debug("native: pane first output", slog.String("pane_id", paneID), slog.Duration("since_start", snap.firstRead.Sub(snap.created)))
	if !snap.ptyCreated.IsZero() {
		delta := snap.firstRead.Sub(snap.ptyCreated)
		if delta < 0 {
			delta = 0
		}
		slog.Debug("native: pane first output after pty ready", slog.String("pane_id", paneID), slog.Duration("since_pty_ready", delta))
	}
	return flags
}

func logPanePerfFirstReadAfterWrite(flags uint8, paneID string, snap panePerfSnapshot) uint8 {
	if snap.firstReadAfterWrite.IsZero() || flags&perfPaneFirstReadAfterWrite != 0 {
		return flags
	}
	flags |= perfPaneFirstReadAfterWrite
	var delta time.Duration
	if !snap.firstWrite.IsZero() {
		delta = snap.firstReadAfterWrite.Sub(snap.firstWrite)
	}
	slog.Debug(
		"native: pane first output after input",
		slog.String("pane_id", paneID),
		slog.Duration("since_start", snap.firstReadAfterWrite.Sub(snap.created)),
		slog.Duration("since_input", delta),
	)
	return flags
}
