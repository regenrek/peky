package native

import (
	"log"
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
	if m == nil || pane == nil || !perfDebugEnabled() {
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
		log.Printf("native: pane pty ready pane=%s since_start=%s", pane.ID, snap.ptyCreated.Sub(snap.created))
	})
	flags = logPanePerfStage(flags, perfPaneProcStart, snap.procStarted, func() {
		log.Printf("native: pane process started pane=%s pid=%d since_start=%s", pane.ID, pane.PID, snap.procStarted.Sub(snap.created))
	})
	flags = logPanePerfStage(flags, perfPaneIOStart, snap.ioStarted, func() {
		log.Printf("native: pane io started pane=%s since_start=%s", pane.ID, snap.ioStarted.Sub(snap.created))
	})
	flags = logPanePerfStage(flags, perfPaneFirstUpdate, snap.firstUpdate, func() {
		log.Printf("native: pane first update pane=%s since_start=%s", pane.ID, snap.firstUpdate.Sub(snap.created))
	})
	flags = logPanePerfFirstRead(flags, pane.ID, snap)
	flags = logPanePerfStage(flags, perfPaneFirstWrite, snap.firstWrite, func() {
		log.Printf("native: pane first input sent pane=%s since_start=%s", pane.ID, snap.firstWrite.Sub(snap.created))
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
	log.Printf("native: pane first output pane=%s since_start=%s", paneID, snap.firstRead.Sub(snap.created))
	if !snap.ptyCreated.IsZero() {
		delta := snap.firstRead.Sub(snap.ptyCreated)
		if delta < 0 {
			delta = 0
		}
		log.Printf("native: pane first output after pty ready pane=%s since_pty_ready=%s", paneID, delta)
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
	log.Printf("native: pane first output after input pane=%s since_start=%s since_input=%s", paneID, snap.firstReadAfterWrite.Sub(snap.created), delta)
	return flags
}
