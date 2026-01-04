package sessionrestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(Config{Enabled: true, BaseDir: base})
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	snap := PaneSnapshot{
		CapturedAt:    time.Now().UTC(),
		SessionName:   "demo",
		SessionPath:   "/tmp/demo",
		SessionLayout: "layout",
		PaneID:        "p-1",
		PaneIndex:     "0",
		PaneTitle:     "title",
		PaneCommand:   "cmd",
		PaneTool:      "tool",
		PaneLastAct:   time.Now().UTC(),
		Terminal: TerminalSnapshot{
			Cols:            80,
			Rows:            2,
			CursorX:         1,
			CursorY:         1,
			CursorVisible:   true,
			AltScreen:       false,
			ScreenLines:     []string{"hello", "world"},
			ScrollbackLines: []string{"old"},
		},
	}
	if err := store.Save(context.Background(), snap); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := NewStore(Config{Enabled: true, BaseDir: base})
	if err != nil {
		t.Fatalf("NewStore(load) error: %v", err)
	}
	if err := loaded.Load(context.Background()); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	got, ok := loaded.Snapshot("p-1")
	if !ok {
		t.Fatalf("expected snapshot")
	}
	if got.SessionName != snap.SessionName || got.PaneTitle != snap.PaneTitle {
		t.Fatalf("snapshot mismatch: %#v", got)
	}
	if len(got.Terminal.ScreenLines) != 2 || got.Terminal.ScreenLines[0] != "hello" {
		t.Fatalf("screen lines mismatch: %#v", got.Terminal.ScreenLines)
	}
}

func TestTrimLinesByBytes(t *testing.T) {
	lines := []string{"one", "two", "three"}
	out := TrimLinesByBytes(lines, 6)
	if len(out) != 1 || out[0] != "three" {
		t.Fatalf("TrimLinesByBytes() = %#v", out)
	}
}

func TestStoreDelete(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(Config{Enabled: true, BaseDir: base})
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	snap := PaneSnapshot{
		SessionName: "demo",
		PaneID:      "p-1",
		Terminal: TerminalSnapshot{
			Cols:        1,
			Rows:        1,
			ScreenLines: []string{"x"},
		},
	}
	if err := store.Save(context.Background(), snap); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	store.Delete("p-1")
	if _, ok := store.Snapshot("p-1"); ok {
		t.Fatalf("expected snapshot to be deleted")
	}
	path := filepath.Join(base, paneDirName, "p-1"+snapshotExt)
	if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected file removed, got %v", err)
	}
}

func TestStoreSnapshotsSorted(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(Config{Enabled: true, BaseDir: base})
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	first := time.Now().Add(-time.Minute)
	second := time.Now()
	for _, snap := range []PaneSnapshot{
		{CapturedAt: second, SessionName: "demo", PaneID: "p-2", Terminal: TerminalSnapshot{Cols: 1, Rows: 1}},
		{CapturedAt: first, SessionName: "demo", PaneID: "p-1", Terminal: TerminalSnapshot{Cols: 1, Rows: 1}},
	} {
		if err := store.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
	}
	out := store.Snapshots()
	if len(out) != 2 || out[0].PaneID != "p-1" || out[1].PaneID != "p-2" {
		t.Fatalf("Snapshots() = %#v", out)
	}
}

func TestStoreGCTTL(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(Config{Enabled: true, BaseDir: base, TTLInactive: time.Hour})
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	old := time.Now().Add(-2 * time.Hour)
	snap := PaneSnapshot{
		CapturedAt:  old,
		SessionName: "demo",
		PaneID:      "p-1",
		PaneLastAct: old,
		Terminal: TerminalSnapshot{
			Cols:        1,
			Rows:        1,
			ScreenLines: []string{"x"},
		},
	}
	if err := store.Save(context.Background(), snap); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if err := store.GC(context.Background(), nil); err != nil {
		t.Fatalf("GC() error: %v", err)
	}
	if _, ok := store.Snapshot("p-1"); ok {
		t.Fatalf("expected snapshot to expire")
	}
}
