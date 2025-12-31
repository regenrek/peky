package native

import (
	"log"
	"time"
)

const (
	perfPaneFirstRead = 1 << iota
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
	firstRead := w.FirstReadAt()
	firstWrite := w.FirstWriteAt()
	firstReadAfterWrite := w.FirstReadAfterWriteAt()

	m.perfMu.Lock()
	if m.perfLogged == nil {
		m.perfLogged = make(map[string]uint8)
	}
	flags := m.perfLogged[pane.ID]
	if !firstRead.IsZero() && flags&perfPaneFirstRead == 0 {
		flags |= perfPaneFirstRead
		log.Printf("native: pane first output pane=%s since_start=%s", pane.ID, firstRead.Sub(created))
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
