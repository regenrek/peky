package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func (m *Model) cyclePerformancePreset() tea.Cmd {
	if m == nil {
		return nil
	}
	current := strings.ToLower(strings.TrimSpace(m.settings.Performance.Preset))
	order := []string{PerfPresetLow, PerfPresetMedium, PerfPresetHigh, PerfPresetCustom}
	next := order[0]
	for i, preset := range order {
		if preset == current {
			next = order[(i+1)%len(order)]
			break
		}
	}
	return m.savePerformanceSettings(next, m.settings.Performance.RenderPolicy)
}

func (m *Model) toggleRenderPolicy() tea.Cmd {
	if m == nil {
		return nil
	}
	current := strings.ToLower(strings.TrimSpace(m.settings.Performance.RenderPolicy))
	next := RenderPolicyVisible
	if current == RenderPolicyVisible {
		next = RenderPolicyAll
	}
	return m.savePerformanceSettings(m.settings.Performance.Preset, next)
}

func (m *Model) savePerformanceSettings(preset, renderPolicy string) tea.Cmd {
	configPath, err := m.requireConfigPath()
	if err != nil {
		m.setToast("Performance update failed: "+err.Error(), toastError)
		return nil
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		m.setToast("Performance update failed: "+err.Error(), toastError)
		return nil
	}
	cfg.Dashboard.Performance.Preset = strings.ToLower(strings.TrimSpace(preset))
	cfg.Dashboard.Performance.RenderPolicy = strings.ToLower(strings.TrimSpace(renderPolicy))
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		m.setToast("Performance update failed: "+err.Error(), toastError)
		return nil
	}
	if err := layout.SaveConfig(configPath, cfg); err != nil {
		m.setToast("Performance update failed: "+err.Error(), toastError)
		return nil
	}
	m.config = cfg
	settings, err := defaultDashboardConfig(cfg.Dashboard)
	if err != nil {
		m.setToast("Performance update failed: "+err.Error(), toastError)
		return nil
	}
	m.settings = settings

	label := fmt.Sprintf("Performance preset: %s", titleCase(m.settings.Performance.Preset))
	if m.settings.Performance.RenderPolicy == RenderPolicyAll {
		label += " (all panes live)"
	}
	m.setToast(label, toastSuccess)
	return m.requestRefreshCmd()
}
