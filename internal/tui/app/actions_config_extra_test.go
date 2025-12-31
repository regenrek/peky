package app

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/workspace"
)

func TestStartSessionNativeAndClosePaneErrors(t *testing.T) {
	m := newTestModelLite()
	m.client = nil

	cmd := m.startSessionNative("demo", "/tmp", "", true)
	msg, ok := cmd().(sessionStartedMsg)
	if !ok || msg.Err == nil {
		t.Fatalf("expected sessionStartedMsg error")
	}

	m.confirmPaneSession = "alpha-1"
	m.confirmPaneIndex = "1"
	m.confirmPaneID = "p1"
	if cmd := m.applyClosePane(); cmd != nil {
		t.Fatalf("expected nil cmd for close pane without client")
	}
	if m.toast.Text == "" {
		t.Fatalf("expected toast set for close pane failure")
	}

	shutdown := m.shutdownCmd()
	if shutdown == nil {
		t.Fatalf("expected shutdown cmd")
	}
	if msg := shutdown(); msg != nil {
		t.Fatalf("expected nil shutdown msg")
	}
}

func TestUnhideProjectInConfigAndLabels(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.yml"
	cfg := &layout.Config{
		Dashboard: layout.DashboardConfig{
			HiddenProjects: []layout.HiddenProjectConfig{{
				Name: "Alpha",
				Path: "/alpha",
			}},
		},
	}
	if err := layout.SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	changed, err := m.unhideProjectInConfig(layout.HiddenProjectConfig{Name: "Alpha"})
	if err != nil || !changed {
		t.Fatalf("expected project unhidden, err=%v changed=%v", err, changed)
	}
	if len(m.settings.HiddenProjects) != 0 {
		t.Fatalf("expected hidden projects cleared")
	}

	label := hiddenProjectLabel(layout.HiddenProjectConfig{Name: "Alpha", Path: "/alpha"})
	if label == "" {
		t.Fatalf("expected hidden project label")
	}
	keys := workspace.HiddenProjectKeySet([]layout.HiddenProjectConfig{{Name: "Alpha"}})
	if len(keys) == 0 {
		t.Fatalf("expected hidden project key set")
	}
}

func TestToastTextLevels(t *testing.T) {
	m := newTestModelLite()
	m.toast = toastMessage{Text: "ok", Level: toastSuccess, Until: time.Now().Add(2 * time.Second)}
	if m.toastText() == "" {
		t.Fatalf("expected success toast text")
	}
	m.toast = toastMessage{Text: "warn", Level: toastWarning, Until: time.Now().Add(2 * time.Second)}
	if m.toastText() == "" {
		t.Fatalf("expected warning toast text")
	}
	m.toast = toastMessage{Text: "err", Level: toastError, Until: time.Now().Add(2 * time.Second)}
	if m.toastText() == "" {
		t.Fatalf("expected error toast text")
	}
	m.toast = toastMessage{Text: "gone", Level: toastInfo, Until: time.Now().Add(-time.Second)}
	if m.toastText() != "" {
		t.Fatalf("expected expired toast to be empty")
	}
}
