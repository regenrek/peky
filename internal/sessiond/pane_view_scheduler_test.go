package sessiond

import (
	"testing"
	"time"
)

func TestPaneViewSchedulerPreservesReceived(t *testing.T) {
	s := newPaneViewScheduler()
	req := PaneViewRequest{Priority: PaneViewPriorityBackground}
	s.enqueue("p1", Envelope{ID: 1}, req)

	s.mu.Lock()
	first, ok := s.pending["p1"]
	s.mu.Unlock()
	if !ok {
		t.Fatalf("expected pending job")
	}

	s.enqueue("p1", Envelope{ID: 2}, req)

	s.mu.Lock()
	second, ok := s.pending["p1"]
	s.mu.Unlock()
	if !ok {
		t.Fatalf("expected pending job after re-enqueue")
	}

	if !second.received.Equal(first.received) {
		t.Fatalf("received timestamp should be preserved for pending job")
	}
}

func TestPaneViewSchedulerBoostPrefersOldBackground(t *testing.T) {
	s := newPaneViewScheduler()
	now := time.Now()
	old := now.Add(-paneViewStarvationWindow * 2)

	s.pending["bg"] = paneViewJob{
		req:      PaneViewRequest{Priority: PaneViewPriorityBackground},
		received: old,
	}
	s.pending["fg"] = paneViewJob{
		req:      PaneViewRequest{Priority: PaneViewPriorityFocused},
		received: now,
	}

	s.mu.Lock()
	bestID, _, ok := s.pickLocked()
	s.mu.Unlock()
	if !ok {
		t.Fatalf("expected pickLocked to return a job")
	}
	if bestID != "bg" {
		t.Fatalf("expected background pane to win after boost, got %q", bestID)
	}
}
