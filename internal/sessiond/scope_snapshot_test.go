package sessiond

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/native"
)

func TestResolveScopeTargetsWithSnapshotSession(t *testing.T) {
	d := &Daemon{}
	sessions := []native.SessionSnapshot{
		{
			Name:  "demo",
			Path:  "/tmp/demo",
			Panes: []native.PaneSnapshot{{ID: "p-1"}, {ID: "p-2"}},
		},
	}
	ids, err := d.resolveScopeTargetsWithSnapshot("session", sessions)
	if err != nil {
		t.Fatalf("resolveScopeTargetsWithSnapshot: %v", err)
	}
	if len(ids) != 2 || ids[0] != "p-1" || ids[1] != "p-2" {
		t.Fatalf("unexpected ids: %#v", ids)
	}
}

func TestResolveScopeTargetsWithSnapshotProject(t *testing.T) {
	d := &Daemon{}
	sessions := []native.SessionSnapshot{
		{Name: "a", Path: "/tmp/demo", Panes: []native.PaneSnapshot{{ID: "p-1"}}},
		{Name: "b", Path: "/tmp/demo", Panes: []native.PaneSnapshot{{ID: "p-2"}}},
	}
	ids, err := d.resolveScopeTargetsWithSnapshot("project", sessions)
	if err != nil {
		t.Fatalf("resolveScopeTargetsWithSnapshot: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
}

func TestResolveScopeTargetsWithSnapshotInvalidScope(t *testing.T) {
	d := &Daemon{}
	if _, err := d.resolveScopeTargetsWithSnapshot("nope", []native.SessionSnapshot{{Name: "s"}}); err == nil {
		t.Fatalf("expected error")
	}
}
