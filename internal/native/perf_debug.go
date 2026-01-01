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

func (m *Manager) logPanePerf(pane *Pane) {
	if m == nil || pane == nil || !perfDebugEnabled() {
		return
	}
	w := pane.window
	if w == nil {
		return
	}
	created := w.CreatedAt()
	ptyCreated := w.PtyCreatedAt()
	procStarted := w.ProcessStartedAt()
	ioStarted := w.IOStartedAt()
	firstUpdate := w.FirstUpdateAt()
	firstRead := w.FirstReadAt()
	firstWrite := w.FirstWriteAt()
	firstReadAfterWrite := w.FirstReadAfterWriteAt()

	m.perfMu.Lock()
	if m.perfLogged == nil {
		m.perfLogged = make(map[string]uint8)
	}
	flags := m.perfLogged[pane.ID]
	if !ptyCreated.IsZero() && flags&perfPanePtyCreated == 0 {
		flags |= perfPanePtyCreated
		log.Printf("native: pane pty ready pane=%s since_start=%s", pane.ID, ptyCreated.Sub(created))
	}
	if !procStarted.IsZero() && flags&perfPaneProcStart == 0 {
		flags |= perfPaneProcStart
		log.Printf("native: pane process started pane=%s pid=%d since_start=%s", pane.ID, pane.PID, procStarted.Sub(created))
	}
	if !ioStarted.IsZero() && flags&perfPaneIOStart == 0 {
		flags |= perfPaneIOStart
		log.Printf("native: pane io started pane=%s since_start=%s", pane.ID, ioStarted.Sub(created))
	}
	if !firstUpdate.IsZero() && flags&perfPaneFirstUpdate == 0 {
		flags |= perfPaneFirstUpdate
		log.Printf("native: pane first update pane=%s since_start=%s", pane.ID, firstUpdate.Sub(created))
	}
	if !firstRead.IsZero() && flags&perfPaneFirstRead == 0 {
		flags |= perfPaneFirstRead
		log.Printf("native: pane first output pane=%s since_start=%s", pane.ID, firstRead.Sub(created))
		if !ptyCreated.IsZero() {
			delta := firstRead.Sub(ptyCreated)
			if delta < 0 {
				delta = 0
			}
			log.Printf("native: pane first output after pty ready pane=%s since_pty_ready=%s", pane.ID, delta)
		}
	}
	if !firstWrite.IsZero() && flags&perfPaneFirstWrite == 0 {
		flags |= perfPaneFirstWrite
		log.Printf("native: pane first input sent pane=%s since_start=%s", pane.ID, firstWrite.Sub(created))
	}
	if !firstReadAfterWrite.IsZero() && flags&perfPaneFirstReadAfterWrite == 0 {
		flags |= perfPaneFirstReadAfterWrite
		var delta time.Duration
		if !firstWrite.IsZero() {
			delta = firstReadAfterWrite.Sub(firstWrite)
		}
		log.Printf("native: pane first output after input pane=%s since_start=%s since_input=%s", pane.ID, firstReadAfterWrite.Sub(created), delta)
	}
	m.perfLogged[pane.ID] = flags
	m.perfMu.Unlock()
}
