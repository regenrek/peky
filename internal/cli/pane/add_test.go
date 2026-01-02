package pane

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

func TestActivePaneIndex(t *testing.T) {
	t.Run("active", func(t *testing.T) {
		sessions := []native.SessionSnapshot{
			{
				Name: "alpha",
				Panes: []native.PaneSnapshot{
					{Index: "1", Active: false},
					{Index: "2", Active: true},
				},
			},
		}
		got, err := activePaneIndex(sessions, "alpha")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "2" {
			t.Fatalf("expected active index 2, got %q", got)
		}
	})
	t.Run("fallback-first", func(t *testing.T) {
		sessions := []native.SessionSnapshot{
			{
				Name: "alpha",
				Panes: []native.PaneSnapshot{
					{Index: "1"},
					{Index: "2"},
				},
			},
		}
		got, err := activePaneIndex(sessions, "alpha")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "1" {
			t.Fatalf("expected fallback index 1, got %q", got)
		}
	})
	t.Run("missing-session", func(t *testing.T) {
		_, err := activePaneIndex(nil, "missing")
		if err == nil {
			t.Fatalf("expected error for missing session")
		}
	})
	t.Run("no-panes", func(t *testing.T) {
		sessions := []native.SessionSnapshot{{Name: "alpha"}}
		_, err := activePaneIndex(sessions, "alpha")
		if err == nil {
			t.Fatalf("expected error for session without panes")
		}
	})
}

func TestFindPaneByID(t *testing.T) {
	ts := time.Now()
	sessions := []native.SessionSnapshot{
		{
			Name: "alpha",
			Panes: []native.PaneSnapshot{
				{ID: "p-1", Index: "0", LastActive: ts},
			},
		},
		{
			Name: "beta",
			Panes: []native.PaneSnapshot{
				{ID: "p-2", Index: "3", LastActive: ts},
			},
		},
	}
	session, index, ok := findPaneByID(sessions, "p-2")
	if !ok {
		t.Fatalf("expected pane found")
	}
	if session != "beta" || index != "3" {
		t.Fatalf("unexpected match %q:%q", session, index)
	}
	_, _, ok = findPaneByID(sessions, "missing")
	if ok {
		t.Fatalf("expected missing pane not found")
	}
}
