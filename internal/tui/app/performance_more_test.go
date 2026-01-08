package app

import (
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestPerformanceMenuActionsPersistConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	m.settings.Performance.Preset = PerfPresetMax
	m.settings.Performance.RenderPolicy = RenderPolicyVisible
	m.settings.Performance.PreviewRender.Mode = PreviewRenderDirect

	_ = m.cyclePerformancePreset()
	if m.toast.Level != toastSuccess {
		t.Fatalf("toast level=%v", m.toast.Level)
	}

	_ = m.toggleRenderPolicy()
	if m.settings.Performance.RenderPolicy != RenderPolicyAll {
		t.Fatalf("render policy=%q", m.settings.Performance.RenderPolicy)
	}

	_ = m.cyclePreviewRenderMode()
	if m.settings.Performance.PreviewRender.Mode != PreviewRenderOff {
		t.Fatalf("preview mode=%q", m.settings.Performance.PreviewRender.Mode)
	}
}
