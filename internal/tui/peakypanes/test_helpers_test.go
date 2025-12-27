package peakypanes

import (
	"context"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	model, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel() error: %v", err)
	}
	model.width = 80
	model.height = 24
	model.state = StateDashboard
	if model.native != nil {
		t.Cleanup(func() { model.native.Close() })
	}
	return model
}

func startNativeSession(t *testing.T, model *Model, name string) native.SessionSnapshot {
	t.Helper()
	if model == nil || model.native == nil {
		t.Fatalf("native manager unavailable")
	}
	if name == "" {
		name = "sess"
	}
	path := t.TempDir()
	layoutCfg := &layout.LayoutConfig{
		Name: "test",
		Windows: []layout.WindowDef{{
			Name: "main",
			Panes: []layout.PaneDef{{
				Title: "pane",
				Cmd:   "cat",
			}},
		}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := model.native.StartSession(ctx, native.SessionSpec{
		Name:       name,
		Path:       path,
		Layout:     layoutCfg,
		LayoutName: layoutCfg.Name,
	})
	if err != nil {
		t.Fatalf("StartSession() error: %v", err)
	}
	for _, snap := range model.native.Snapshot(2) {
		if snap.Name == name {
			return snap
		}
	}
	t.Fatalf("session snapshot missing for %q", name)
	return native.SessionSnapshot{}
}
