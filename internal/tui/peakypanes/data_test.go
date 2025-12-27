package peakypanes

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tmuxctl"
)

func TestDefaultDashboardConfigDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.RefreshInterval != 2*time.Second {
		t.Fatalf("RefreshInterval = %v", cfg.RefreshInterval)
	}
	if cfg.PreviewLines != 12 {
		t.Fatalf("PreviewLines = %d", cfg.PreviewLines)
	}
	if cfg.ThumbnailLines != 1 {
		t.Fatalf("ThumbnailLines = %d", cfg.ThumbnailLines)
	}
	if cfg.IdleThreshold != 20*time.Second {
		t.Fatalf("IdleThreshold = %v", cfg.IdleThreshold)
	}
	if !cfg.ShowThumbnails || !cfg.PreviewCompact {
		t.Fatalf("ShowThumbnails=%v PreviewCompact=%v", cfg.ShowThumbnails, cfg.PreviewCompact)
	}
	if cfg.PreviewMode != "grid" {
		t.Fatalf("PreviewMode = %q", cfg.PreviewMode)
	}
	if cfg.AttachBehavior != AttachBehaviorNewTerminal {
		t.Fatalf("AttachBehavior = %q", cfg.AttachBehavior)
	}
	if len(cfg.ProjectRoots) != 1 || cfg.ProjectRoots[0] != filepath.Join(home, "projects") {
		t.Fatalf("ProjectRoots = %#v", cfg.ProjectRoots)
	}
}

func TestDefaultDashboardConfigOverrides(t *testing.T) {
	show := false
	compact := false
	cfg, err := defaultDashboardConfig(layout.DashboardConfig{
		RefreshMS:      500,
		PreviewLines:   5,
		ThumbnailLines: 2,
		IdleSeconds:    3,
		ShowThumbnails: &show,
		PreviewCompact: &compact,
		PreviewMode:    "layout",
		AttachBehavior: "new-terminal",
		ProjectRoots:   []string{"/tmp", "/tmp"},
	})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	if cfg.RefreshInterval != 500*time.Millisecond {
		t.Fatalf("RefreshInterval = %v", cfg.RefreshInterval)
	}
	if cfg.PreviewLines != 5 || cfg.ThumbnailLines != 2 {
		t.Fatalf("PreviewLines=%d ThumbnailLines=%d", cfg.PreviewLines, cfg.ThumbnailLines)
	}
	if cfg.IdleThreshold != 3*time.Second {
		t.Fatalf("IdleThreshold = %v", cfg.IdleThreshold)
	}
	if cfg.ShowThumbnails || cfg.PreviewCompact {
		t.Fatalf("ShowThumbnails=%v PreviewCompact=%v", cfg.ShowThumbnails, cfg.PreviewCompact)
	}
	if cfg.PreviewMode != "layout" {
		t.Fatalf("PreviewMode = %q", cfg.PreviewMode)
	}
	if cfg.AttachBehavior != AttachBehaviorNewTerminal {
		t.Fatalf("AttachBehavior = %q", cfg.AttachBehavior)
	}
	if !reflect.DeepEqual(cfg.ProjectRoots, []string{"/tmp"}) {
		t.Fatalf("ProjectRoots = %#v", cfg.ProjectRoots)
	}
}

func TestDefaultDashboardConfigInvalidPreviewMode(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{PreviewMode: "bad"})
	if err == nil {
		t.Fatalf("defaultDashboardConfig() expected error")
	}
}

func TestDefaultDashboardConfigInvalidAttachBehavior(t *testing.T) {
	_, err := defaultDashboardConfig(layout.DashboardConfig{AttachBehavior: "weird"})
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
	if deadErr != PaneStatusError {
		t.Fatalf("dead error status = %v", deadErr)
	}
	deadDone := classifyPane(PaneItem{Dead: true, DeadStatus: 0}, nil, settings, time.Now())
	if deadDone != PaneStatusDone {
		t.Fatalf("dead done status = %v", deadDone)
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
	line := lastNonEmpty([]string{"", "  ", "ok", ""})
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
			Name: "A",
			Sessions: []SessionItem{
				{Name: "s1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0"}, {Index: "1"}}},
			},
		},
		{
			Name: "B",
			Sessions: []SessionItem{
				{Name: "s2", ActiveWindow: "2", Windows: []WindowItem{{Index: "2"}}},
			},
		},
	}
	resolved := resolveSelection(groups, selectionState{Project: "B", Session: "s2", Window: "2"})
	if resolved.Project != "B" || resolved.Session != "s2" || resolved.Window != "2" {
		t.Fatalf("resolveSelection() = %#v", resolved)
	}
	resolved = resolveSelection(groups, selectionState{Project: "B", Session: "s2", Window: "missing"})
	if resolved.Window != "2" {
		t.Fatalf("resolveSelection() window fallback = %#v", resolved)
	}
}

func TestResolveDashboardSelection(t *testing.T) {
	groups := []ProjectGroup{
		{
			Name: "A",
			Sessions: []SessionItem{
				{
					Name:   "s1",
					Status: StatusRunning,
					Windows: []WindowItem{{
						Index: "0",
						Panes: []PaneItem{{Index: "0", Active: true}},
					}},
				},
			},
		},
		{
			Name: "B",
			Sessions: []SessionItem{
				{
					Name:   "s2",
					Status: StatusRunning,
					Windows: []WindowItem{{
						Index: "1",
						Panes: []PaneItem{{Index: "2", Active: true}},
					}},
				},
			},
		},
	}
	desired := selectionState{Session: "s2", Window: "1", Pane: "2"}
	resolved := resolveDashboardSelection(groups, desired)
	if resolved.Project != "B" || resolved.Session != "s2" || resolved.Window != "1" || resolved.Pane != "2" {
		t.Fatalf("resolveDashboardSelection() = %#v", resolved)
	}
}

func TestBuildDashboardData(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &layout.Config{Projects: []layout.ProjectConfig{{Name: "App", Path: tmpDir}}}
	settings, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}

	fullFormat := "#{pane_id}\t#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}\t#{pane_start_command}\t#{pane_pid}\t#{pane_left}\t#{pane_top}\t#{pane_width}\t#{pane_height}\t#{pane_dead}\t#{pane_dead_status}\t#{pane_last_active}"
	paneLine := fmt.Sprintf("%%1\t0\t1\tTitle\tbash\tbash\t1234\t0\t0\t80\t24\t0\t0\t%d\n", time.Now().Unix())

	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"display-message", "-p", "#S"}, stdout: "app\n", exit: 0},
		{name: "tmux", args: []string{"list-sessions", "-F", "#{session_name}\t#{session_path}"}, stdout: fmt.Sprintf("app\t%s\n", tmpDir), exit: 0},
		{name: "tmux", args: []string{"list-windows", "-t", "app", "-F", "#{window_index}\t#{window_name}\t#{window_active}"}, stdout: "0\tmain\t1\n1\tlogs\t0\n", exit: 0},
		{name: "tmux", args: []string{"list-panes", "-t", "app:0", "-F", fullFormat}, stdout: paneLine, exit: 0},
		{name: "tmux", args: []string{"list-panes", "-t", "app:1", "-F", fullFormat}, stdout: paneLine, exit: 0},
		{name: "tmux", args: []string{"list-panes", "-t", "app:0", "-F", fullFormat}, stdout: paneLine, exit: 0},
		{name: "tmux", args: []string{"list-panes", "-t", "app:0", "-F", fullFormat}, stdout: paneLine, exit: 0},
	}}

	client, err := tmuxctl.NewClient("tmux")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	client.WithExec(runner.run)

	result := buildDashboardData(context.Background(), client, tmuxSnapshotInput{
		Selection: selectionState{},
		Tab:       TabProject,
		Config:    cfg,
		Settings:  settings,
		Version:   1,
	})
	if result.Err != nil {
		t.Fatalf("buildDashboardData() error: %v", result.Err)
	}
	if len(result.Data.Projects) != 1 {
		t.Fatalf("Projects = %#v", result.Data.Projects)
	}
	session := result.Data.Projects[0].Sessions[0]
	if session.Status != StatusCurrent {
		t.Fatalf("session status = %v", session.Status)
	}
	if session.WindowCount != 2 || session.ActiveWindow != "0" {
		t.Fatalf("session windows = %d active=%q", session.WindowCount, session.ActiveWindow)
	}
	if session.Thumbnail.Line != "Title" {
		t.Fatalf("thumbnail line = %q", session.Thumbnail.Line)
	}
	if len(session.Windows) == 0 || len(session.Windows[0].Panes) == 0 {
		t.Fatalf("panes not attached: %#v", session.Windows)
	}
	if result.Resolved.Session != "app" || result.Resolved.Window != "" {
		t.Fatalf("Resolved = %#v", result.Resolved)
	}

	runner.assertDone()
}

func TestBuildDashboardDataHonorsHiddenProjects(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{{Name: "App", Path: tmpDir}},
		Dashboard: layout.DashboardConfig{
			HiddenProjects: []layout.HiddenProjectConfig{{Name: "App", Path: tmpDir}},
		},
	}
	settings, err := defaultDashboardConfig(cfg.Dashboard)
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}

	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"display-message", "-p", "#S"}, stdout: "app\n", exit: 0},
		{name: "tmux", args: []string{"list-sessions", "-F", "#{session_name}\t#{session_path}"}, stdout: fmt.Sprintf("app\t%s\n", tmpDir), exit: 0},
	}}

	client, err := tmuxctl.NewClient("tmux")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	client.WithExec(runner.run)

	result := buildDashboardData(context.Background(), client, tmuxSnapshotInput{
		Selection: selectionState{},
		Tab:       TabProject,
		Config:    cfg,
		Settings:  settings,
		Version:   1,
	})
	if result.Err != nil {
		t.Fatalf("buildDashboardData() error: %v", result.Err)
	}
	if len(result.Data.Projects) != 0 {
		t.Fatalf("Projects = %#v", result.Data.Projects)
	}

	runner.assertDone()
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
