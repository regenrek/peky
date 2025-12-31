package sessiond

import (
	"testing"
	"time"
)

func TestActionLogAddAndList(t *testing.T) {
	log := newActionLog(2)
	if entries := log.list(0, time.Time{}); len(entries) != 0 {
		t.Fatalf("expected empty log")
	}

	log.add(PaneHistoryEntry{TS: time.Now().Add(-2 * time.Second), Action: "a"})
	log.add(PaneHistoryEntry{TS: time.Now().Add(-1 * time.Second), Action: "b"})
	log.add(PaneHistoryEntry{TS: time.Now(), Action: "c"})
	entries := log.list(0, time.Time{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Action != "b" || entries[1].Action != "c" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestActionLogListSinceAndLimit(t *testing.T) {
	log := newActionLog(5)
	base := time.Now().Add(-10 * time.Second)
	for i := 0; i < 4; i++ {
		log.add(PaneHistoryEntry{TS: base.Add(time.Duration(i) * time.Second), Action: "a"})
	}
	since := base.Add(2 * time.Second)
	entries := log.list(2, since)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].TS.Before(since) {
		t.Fatalf("expected entries after since")
	}
}

func TestRecordPaneActionAndHistory(t *testing.T) {
	d := &Daemon{actionLogs: make(map[string]*actionLog)}
	d.recordPaneAction("pane-1", "send", "summary", "", "ok")
	d.recordPaneAction("pane-1", "send", "summary", "", "ok")
	d.recordPaneAction("", "send", "summary", "", "ok")

	entries := d.paneHistory("pane-1", 10, time.Time{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Action != "send" {
		t.Fatalf("unexpected action: %+v", entries[0])
	}
	if len(d.paneHistory("", 10, time.Time{})) != 0 {
		t.Fatalf("expected empty history for empty pane")
	}
}
