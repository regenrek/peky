package sessiond

import (
	"fmt"
	"time"
)

func (d *Daemon) recordEvent(event Event) Event {
	if d == nil {
		return event
	}
	if event.TS.IsZero() {
		event.TS = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", d.eventSeq.Add(1))
	}
	d.eventMu.Lock()
	if d.eventLog != nil {
		d.eventLog.add(event)
	}
	d.eventMu.Unlock()
	return event
}
