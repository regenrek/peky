package app

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/tui/ansi"
)

func TestDefaultDashboardConfigDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	assertDuration(t, "RefreshInterval", cfg.RefreshInterval, 2*time.Second)
	assertInt(t, "PreviewLines", cfg.PreviewLines, 12)
	assertDuration(t, "IdleThreshold", cfg.IdleThreshold, 20*time.Second)
	assertBool(t, "PreviewCompact", cfg.PreviewCompact, true)
	assertBool(t, "SidebarHidden", cfg.SidebarHidden, false)
	assertString(t, "Resize.MouseApply", cfg.Resize.MouseApply, ResizeMouseApplyLive)
	assertDuration(t, "Resize.MouseThrottle", cfg.Resize.MouseThrottle, 16*time.Millisecond)
	assertBool(t, "Resize.FreezeContentDuringDrag", cfg.Resize.FreezeContentDuringDrag, true)
	assertString(t, "AttachBehavior", cfg.AttachBehavior, AttachBehaviorCurrent)
	assertString(t, "PaneNavigationMode", cfg.PaneNavigationMode, PaneNavigationSpatial)
	assertString(t, "QuitBehavior", cfg.QuitBehavior, QuitBehaviorPrompt)
	assertStringSlice(t, "ProjectRoots", cfg.ProjectRoots, []string{filepath.Join(home, "projects")})
	assertString(t, "Performance.Preset", cfg.Performance.Preset, PerfPresetMax)
	assertString(t, "Performance.RenderPolicy", cfg.Performance.RenderPolicy, RenderPolicyVisible)
	assertString(t, "Performance.PreviewRender.Mode", cfg.Performance.PreviewRender.Mode, PreviewRenderDirect)
	assertInt(t, "Performance.PaneViews.MaxConcurrency", cfg.Performance.PaneViews.MaxConcurrency, paneViewPerfMax.MaxConcurrency)
}

func TestDefaultDashboardConfigOverrides(t *testing.T) {
	compact := false
	sidebarHidden := true
	cfg, err := defaultDashboardConfig(layout.DashboardConfig{
		RefreshMS:          500,
		PreviewLines:       5,
		IdleSeconds:        3,
		PreviewCompact:     &compact,
		Sidebar:            layout.DashboardSidebarConfig{Hidden: &sidebarHidden},
		Resize:             layout.DashboardResizeConfig{MouseApply: "commit", MouseThrottleMS: 40, FreezeContentDuringDrag: boolPtr(false)},
		AttachBehavior:     "detached",
		PaneNavigationMode: "memory",
		QuitBehavior:       "keep",
		ProjectRoots:       []string{"/tmp", "/tmp"},
		Performance: layout.PerformanceConfig{
			Preset:       PerfPresetCustom,
			RenderPolicy: RenderPolicyAll,
			PreviewRender: layout.PreviewRenderConfig{
				Mode: PreviewRenderDirect,
			},
			PaneViews: layout.PaneViewPerformanceConfig{
				MaxConcurrency:       2,
				MinIntervalFocusedMS: 10,
			},
		},
	})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	assertDuration(t, "RefreshInterval", cfg.RefreshInterval, 500*time.Millisecond)
	assertInt(t, "PreviewLines", cfg.PreviewLines, 5)
	assertDuration(t, "IdleThreshold", cfg.IdleThreshold, 3*time.Second)
	assertBool(t, "PreviewCompact", cfg.PreviewCompact, false)
	assertBool(t, "SidebarHidden", cfg.SidebarHidden, true)
	assertString(t, "Resize.MouseApply", cfg.Resize.MouseApply, ResizeMouseApplyCommit)
	assertDuration(t, "Resize.MouseThrottle", cfg.Resize.MouseThrottle, 40*time.Millisecond)
	assertBool(t, "Resize.FreezeContentDuringDrag", cfg.Resize.FreezeContentDuringDrag, false)
	assertString(t, "AttachBehavior", cfg.AttachBehavior, AttachBehaviorDetached)
	assertString(t, "PaneNavigationMode", cfg.PaneNavigationMode, PaneNavigationMemory)
	assertString(t, "QuitBehavior", cfg.QuitBehavior, QuitBehaviorKeep)
	assertStringSlice(t, "ProjectRoots", cfg.ProjectRoots, []string{"/tmp"})
	assertString(t, "Performance.Preset", cfg.Performance.Preset, PerfPresetCustom)
	assertString(t, "Performance.RenderPolicy", cfg.Performance.RenderPolicy, RenderPolicyAll)
	assertString(t, "Performance.PreviewRender.Mode", cfg.Performance.PreviewRender.Mode, PreviewRenderDirect)
	assertInt(t, "Performance.PaneViews.MaxConcurrency", cfg.Performance.PaneViews.MaxConcurrency, 2)
	assertDuration(t, "Performance.PaneViews.MinIntervalFocused", cfg.Performance.PaneViews.MinIntervalFocused, 10*time.Millisecond)
}

func TestDefaultDashboardConfigInvalidResizeMouseApply(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{Resize: layout.DashboardResizeConfig{MouseApply: "bad"}})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigResizeThrottleClamp(t *testing.T) {
	cfg, err := defaultDashboardConfig(layout.DashboardConfig{Resize: layout.DashboardResizeConfig{MouseThrottleMS: 1}})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.Resize.MouseThrottle != 4*time.Millisecond {
		t.Fatalf("Resize.MouseThrottle = %s", cfg.Resize.MouseThrottle)
	}
	cfg, err = defaultDashboardConfig(layout.DashboardConfig{Resize: layout.DashboardResizeConfig{MouseThrottleMS: 1000}})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.Resize.MouseThrottle != 100*time.Millisecond {
		t.Fatalf("Resize.MouseThrottle = %s", cfg.Resize.MouseThrottle)
	}
}

func assertDuration(t *testing.T, name string, got time.Duration, want time.Duration) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v", name, got)
	}
}

func assertInt(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d", name, got)
	}
}

func assertBool(t *testing.T, name string, got bool, want bool) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v", name, got)
	}
}

func assertString(t *testing.T, name string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q", name, got)
	}
}

func assertStringSlice(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v", name, got)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func TestDefaultDashboardConfigInvalidAttachBehavior(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{AttachBehavior: "weird"})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigInvalidPaneNavigationMode(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{PaneNavigationMode: "sideways"})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigInvalidQuitBehavior(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{QuitBehavior: "nope"})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigInvalidPerformancePreset(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{Performance: layout.PerformanceConfig{Preset: "fast"}})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigMaxPreset(t *testing.T) {
	cfg, err := defaultDashboardConfig(layout.DashboardConfig{Performance: layout.PerformanceConfig{Preset: PerfPresetMax}})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.Performance.PaneViews.MaxConcurrency != paneViewPerfMax.MaxConcurrency {
		t.Fatalf("Performance.PaneViews.MaxConcurrency = %d", cfg.Performance.PaneViews.MaxConcurrency)
	}
	if cfg.Performance.PaneViews.MinIntervalFocused != 0 {
		t.Fatalf("Performance.PaneViews.MinIntervalFocused = %s", cfg.Performance.PaneViews.MinIntervalFocused)
	}
}

func TestDefaultDashboardConfigLowPreset(t *testing.T) {
	cfg, err := defaultDashboardConfig(layout.DashboardConfig{Performance: layout.PerformanceConfig{Preset: PerfPresetLow}})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.Performance.PaneViews.MaxConcurrency != paneViewPerfLow.MaxConcurrency {
		t.Fatalf("Performance.PaneViews.MaxConcurrency = %d", cfg.Performance.PaneViews.MaxConcurrency)
	}
	if cfg.Performance.PaneViews.MinIntervalBackground != paneViewPerfLow.MinIntervalBackground {
		t.Fatalf("Performance.PaneViews.MinIntervalBackground = %s", cfg.Performance.PaneViews.MinIntervalBackground)
	}
}

func TestDefaultDashboardConfigHighPreset(t *testing.T) {
	cfg, err := defaultDashboardConfig(layout.DashboardConfig{Performance: layout.PerformanceConfig{Preset: PerfPresetHigh}})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.Performance.PaneViews.MaxConcurrency != paneViewPerfHigh.MaxConcurrency {
		t.Fatalf("Performance.PaneViews.MaxConcurrency = %d", cfg.Performance.PaneViews.MaxConcurrency)
	}
	if cfg.Performance.PaneViews.MinIntervalFocused != paneViewPerfHigh.MinIntervalFocused {
		t.Fatalf("Performance.PaneViews.MinIntervalFocused = %s", cfg.Performance.PaneViews.MinIntervalFocused)
	}
}

func TestDefaultDashboardConfigInvalidRenderPolicy(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{Performance: layout.PerformanceConfig{RenderPolicy: "everywhere"}})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigInvalidPreviewRenderMode(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{Performance: layout.PerformanceConfig{PreviewRender: layout.PreviewRenderConfig{Mode: "fast"}}})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestNormalizeProjectRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	roots := normalizeProjectRoots([]string{"~/work", " ~/work ", "/tmp", ""})
	want := []string{filepath.Join(home, "work"), "/tmp"}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("normalizeProjectRoots() = %#v", roots)
	}
}

func TestCompileStatusMatcherInvalid(t *testing.T) {
	_, err := compileStatusMatcher(layout.StatusRegexConfig{Success: "("})
	if err == nil {
		t.Fatalf("compileStatusMatcher() expected error")
	}
}

func TestDashboardPreviewLinesMinimum(t *testing.T) {
	settings := DashboardConfig{PreviewLines: 3}
	if got := dashboardPreviewLines(settings); got < 10 {
		t.Fatalf("dashboardPreviewLines() = %d", got)
	}
}

func TestClassifyPane(t *testing.T) {
	matcher, err := compileStatusMatcher(layout.StatusRegexConfig{})
	if err != nil {
		t.Fatalf("compileStatusMatcher() error: %v", err)
	}
	settings := DashboardConfig{
		StatusMatcher: matcher,
		IdleThreshold: time.Second,
	}

	deadErr := classifyPane(PaneItem{Dead: true, DeadStatus: 1}, nil, settings, time.Now())
	if deadErr != PaneStatusDead {
		t.Fatalf("dead status = %v", deadErr)
	}
	deadDone := classifyPane(PaneItem{Dead: true, DeadStatus: 0}, nil, settings, time.Now())
	if deadDone != PaneStatusDead {
		t.Fatalf("dead status = %v", deadDone)
	}

	if got := classifyPane(PaneItem{}, []string{"error: boom"}, settings, time.Now()); got != PaneStatusError {
		t.Fatalf("error status = %v", got)
	}
	if got := classifyPane(PaneItem{}, []string{"done"}, settings, time.Now()); got != PaneStatusDone {
		t.Fatalf("success status = %v", got)
	}
	if got := classifyPane(PaneItem{}, []string{"running"}, settings, time.Now()); got != PaneStatusRunning {
		t.Fatalf("running status = %v", got)
	}

	idle := classifyPane(PaneItem{LastActive: time.Now().Add(-10 * time.Second)}, nil, settings, time.Now())
	if idle != PaneStatusIdle {
		t.Fatalf("idle status = %v", idle)
	}
}

func TestLastNonEmpty(t *testing.T) {
	line := ansi.LastNonEmpty([]string{"", "  ", "ok", ""})
	if line != "ok" {
		t.Fatalf("lastNonEmpty() = %q", line)
	}
}

func TestLayoutName(t *testing.T) {
	if layoutName("dev") != "dev" {
		t.Fatalf("layoutName(string) failed")
	}
	if layoutName(&layout.LayoutConfig{Name: "inline"}) != "inline" {
		t.Fatalf("layoutName(LayoutConfig) failed")
	}
	if layoutName(&layout.LayoutConfig{}) != "inline" {
		t.Fatalf("layoutName(LayoutConfig empty) failed")
	}
	if layoutName(map[string]interface{}{"name": "map"}) != "map" {
		t.Fatalf("layoutName(map) failed")
	}
	if layoutName(map[interface{}]interface{}{"name": "legacy"}) != "legacy" {
		t.Fatalf("layoutName(map legacy) failed")
	}
	if layoutName(123) != "" {
		t.Fatalf("layoutName(other) should be empty")
	}
}

func TestProjectKeyAndGroupName(t *testing.T) {
	if projectKey("/path", "Name") != strings.ToLower("/path") {
		t.Fatalf("projectKey(path) failed")
	}
	if projectKey("", "Name") != "name" {
		t.Fatalf("projectKey(name) failed")
	}
	if groupNameFromPath("/tmp/app", "fallback") != "app" {
		t.Fatalf("groupNameFromPath(path) failed")
	}
	if groupNameFromPath("", "fallback") != "fallback" {
		t.Fatalf("groupNameFromPath(empty) failed")
	}
}

func TestResolveSelection(t *testing.T) {
	groups := []ProjectGroup{
		{
			ID:   projectKey("", "A"),
			Name: "A",
			Sessions: []SessionItem{
				{Name: "s1", ActivePane: "0", Panes: []PaneItem{{Index: "0"}, {Index: "1"}}},
			},
		},
		{
			ID:   projectKey("", "B"),
			Name: "B",
			Sessions: []SessionItem{
				{Name: "s2", ActivePane: "2", Panes: []PaneItem{{Index: "2"}}},
			},
		},
	}
	resolved := resolveSelection(groups, selectionState{ProjectID: projectKey("", "B"), Session: "s2", Pane: "2"})
	if resolved.ProjectID != projectKey("", "B") || resolved.Session != "s2" || resolved.Pane != "2" {
		t.Fatalf("resolveSelection() = %#v", resolved)
	}
	resolved = resolveSelection(groups, selectionState{ProjectID: projectKey("", "B"), Session: "s2", Pane: "missing"})
	if resolved.Pane != "missing" {
		t.Fatalf("resolveSelection() pane passthrough = %#v", resolved)
	}
}

func TestResolveDashboardSelection(t *testing.T) {
	groups := []ProjectGroup{
		{
			ID:   projectKey("", "A"),
			Name: "A",
			Sessions: []SessionItem{
				{
					Name:   "s1",
					Status: StatusRunning,
					Panes:  []PaneItem{{Index: "0", Active: true}},
				},
			},
		},
		{
			ID:   projectKey("", "B"),
			Name: "B",
			Sessions: []SessionItem{
				{
					Name:   "s2",
					Status: StatusRunning,
					Panes:  []PaneItem{{Index: "2", Active: true}},
				},
			},
		},
	}
	desired := selectionState{Session: "s2", Pane: "2"}
	resolved := resolveDashboardSelection(groups, desired)
	if resolved.ProjectID != projectKey("", "B") || resolved.Session != "s2" || resolved.Pane != "2" {
		t.Fatalf("resolveDashboardSelection() = %#v", resolved)
	}
}

func TestBuildDashboardData(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "app")
	cfg := &layout.Config{Projects: []layout.ProjectConfig{{Name: "App", Session: snap.Name, Path: snap.Path}}}
	settings, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}

	result := buildDashboardData(dashboardSnapshotInput{
		Selection: selectionState{},
		Tab:       TabProject,
		Config:    cfg,
		Settings:  settings,
		Version:   1,
		Sessions:  []native.SessionSnapshot{snap},
	})
	if result.Err != nil {
		t.Fatalf("buildDashboardData() error: %v", result.Err)
	}
	if len(result.Data.Projects) != 1 {
		t.Fatalf("Projects = %#v", result.Data.Projects)
	}
	session := result.Data.Projects[0].Sessions[0]
	if session.Status != StatusRunning {
		t.Fatalf("session status = %v", session.Status)
	}
	if session.PaneCount == 0 || session.ActivePane == "" {
		t.Fatalf("session panes = %d active=%q", session.PaneCount, session.ActivePane)
	}
	if len(session.Panes) == 0 {
		t.Fatalf("panes not attached: %#v", session.Panes)
	}
	if result.Resolved.Session != snap.Name {
		t.Fatalf("Resolved = %#v", result.Resolved)
	}
}

func TestBuildDashboardDataHonorsHiddenProjects(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "app")
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{{Name: "App", Path: snap.Path}},
		Dashboard: layout.DashboardConfig{
			HiddenProjects: []layout.HiddenProjectConfig{{Name: "App", Path: snap.Path}},
		},
	}
	settings, err := defaultDashboardConfig(cfg.Dashboard)
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}

	result := buildDashboardData(dashboardSnapshotInput{
		Selection: selectionState{},
		Tab:       TabProject,
		Config:    cfg,
		Settings:  settings,
		Version:   1,
		Sessions:  []native.SessionSnapshot{snap},
	})
	if result.Err != nil {
		t.Fatalf("buildDashboardData() error: %v", result.Err)
	}
	if len(result.Data.Projects) != 0 {
		t.Fatalf("Projects = %#v", result.Data.Projects)
	}
}

func TestPaneExistsAndActivePaneIndex(t *testing.T) {
	panes := []PaneItem{{Index: "0", Active: false}, {Index: "1", Active: true}}
	if !paneExists(panes, "1") || paneExists(panes, "2") {
		t.Fatalf("paneExists() unexpected")
	}
	if active := activePaneIndex(panes); active != "1" {
		t.Fatalf("activePaneIndex() = %q", active)
	}
	empty := []PaneItem{}
	if active := activePaneIndex(empty); active != "" {
		t.Fatalf("activePaneIndex(empty) = %q", active)
	}
}
