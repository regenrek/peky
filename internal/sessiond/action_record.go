package sessiond

import (
	"strings"
	"time"
)

func (d *Daemon) recordPaneAction(paneID, action, summary, command, status string) {
	if d == nil {
		return
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return
	}
	entry := PaneHistoryEntry{
		TS:      time.Now().UTC(),
		Action:  strings.TrimSpace(action),
		Summary: strings.TrimSpace(summary),
		Command: strings.TrimSpace(command),
		Status:  strings.TrimSpace(status),
	}
	d.actionMu.Lock()
	log := d.actionLogs[paneID]
	if log == nil {
		log = newActionLog(0)
		d.actionLogs[paneID] = log
	}
	log.add(entry)
	d.actionMu.Unlock()
}

func (d *Daemon) paneHistory(paneID string, limit int, since time.Time) []PaneHistoryEntry {
	if d == nil {
		return nil
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil
	}
	d.actionMu.RLock()
	log := d.actionLogs[paneID]
	d.actionMu.RUnlock()
	if log == nil {
		return nil
	}
	return log.list(limit, since)
}
