package sessiond

import (
	"context"
	"strings"
)

func (d *Daemon) focusState() (string, string) {
	if d == nil {
		return "", ""
	}
	d.focusMu.RLock()
	session := d.focusedSession
	pane := d.focusedPane
	d.focusMu.RUnlock()
	return session, pane
}

func (d *Daemon) setFocusSession(name string) {
	if d == nil {
		return
	}
	name = strings.TrimSpace(name)
	d.focusMu.Lock()
	d.focusedSession = name
	d.focusedPane = ""
	d.focusMu.Unlock()
	d.broadcast(Event{
		Type:    EventFocus,
		Session: name,
		Payload: map[string]any{"session": name},
	})
}

func (d *Daemon) setFocusPane(paneID string) {
	if d == nil {
		return
	}
	paneID = strings.TrimSpace(paneID)
	session := ""
	if paneID != "" {
		session = d.sessionForPane(paneID)
	}
	d.focusMu.Lock()
	d.focusedPane = paneID
	if session != "" {
		d.focusedSession = session
	}
	d.focusMu.Unlock()
	d.broadcast(Event{
		Type:    EventFocus,
		PaneID:  paneID,
		Session: session,
		Payload: map[string]any{"pane_id": paneID, "session": session},
	})
}

func (d *Daemon) sessionForPane(paneID string) string {
	if d == nil || d.manager == nil || strings.TrimSpace(paneID) == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := d.manager.Snapshot(ctx, 0)
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == paneID {
				return session.Name
			}
		}
	}
	return ""
}
