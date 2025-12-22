package peakypanes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/mux"
	"gopkg.in/yaml.v3"
)

const (
	defaultRefreshMS      = 2000
	defaultPreviewLines   = 12
	defaultThumbnailLines = 1
	defaultIdleSeconds    = 20
	previewSlackLines     = 20
	maxCaptureLines       = 400
)

var (
	defaultSuccessRegex = "(?i)done|finished|success|completed|✅"
	defaultErrorRegex   = "(?i)error|failed|panic|❌"
	defaultRunningRegex = "(?i)running|in progress|building|installing|▶"
)

type statusMatcher struct {
	success *regexp.Regexp
	error   *regexp.Regexp
	running *regexp.Regexp
}

func compileStatusMatcher(cfg layout.StatusRegexConfig) (statusMatcher, error) {
	success := cfg.Success
	if strings.TrimSpace(success) == "" {
		success = defaultSuccessRegex
	}
	errorRe := cfg.Error
	if strings.TrimSpace(errorRe) == "" {
		errorRe = defaultErrorRegex
	}
	running := cfg.Running
	if strings.TrimSpace(running) == "" {
		running = defaultRunningRegex
	}
	matcher := statusMatcher{}
	var err error
	if matcher.success, err = regexp.Compile(success); err != nil {
		return matcher, fmt.Errorf("invalid success regex: %w", err)
	}
	if matcher.error, err = regexp.Compile(errorRe); err != nil {
		return matcher, fmt.Errorf("invalid error regex: %w", err)
	}
	if matcher.running, err = regexp.Compile(running); err != nil {
		return matcher, fmt.Errorf("invalid running regex: %w", err)
	}
	return matcher, nil
}

func defaultDashboardConfig(cfg layout.DashboardConfig) (DashboardConfig, error) {
	refreshMS := cfg.RefreshMS
	if refreshMS <= 0 {
		refreshMS = defaultRefreshMS
	}
	previewLines := cfg.PreviewLines
	if previewLines <= 0 {
		previewLines = defaultPreviewLines
	}
	thumbnailLines := cfg.ThumbnailLines
	if thumbnailLines <= 0 {
		thumbnailLines = defaultThumbnailLines
	}
	idleSeconds := cfg.IdleSeconds
	if idleSeconds <= 0 {
		idleSeconds = defaultIdleSeconds
	}
	showThumbnails := true
	if cfg.ShowThumbnails != nil {
		showThumbnails = *cfg.ShowThumbnails
	}
	previewCompact := true
	if cfg.PreviewCompact != nil {
		previewCompact = *cfg.PreviewCompact
	}
	previewMode := strings.TrimSpace(cfg.PreviewMode)
	if previewMode == "" {
		previewMode = "grid"
	}
	if previewMode != "grid" && previewMode != "layout" {
		return DashboardConfig{}, fmt.Errorf("invalid preview_mode %q (use grid or layout)", previewMode)
	}
	projectRoots := normalizeProjectRoots(cfg.ProjectRoots)
	if len(projectRoots) == 0 {
		projectRoots = defaultProjectRoots()
	}
	matcher, err := compileStatusMatcher(cfg.StatusRegex)
	if err != nil {
		return DashboardConfig{}, err
	}
	return DashboardConfig{
		RefreshInterval: time.Duration(refreshMS) * time.Millisecond,
		PreviewLines:    previewLines,
		PreviewCompact:  previewCompact,
		ThumbnailLines:  thumbnailLines,
		IdleThreshold:   time.Duration(idleSeconds) * time.Second,
		ShowThumbnails:  showThumbnails,
		StatusMatcher:   matcher,
		PreviewMode:     previewMode,
		ProjectRoots:    projectRoots,
	}, nil
}

func defaultProjectRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "projects")}
}

func normalizeProjectRoots(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = filepath.Clean(expandPath(root))
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}

func loadConfig(path string) (*layout.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &layout.Config{}, nil
		}
		return nil, err
	}
	var cfg layout.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func buildDashboardData(ctx context.Context, client mux.Client, input muxSnapshotInput) muxSnapshotResult {
	result := muxSnapshotResult{Settings: input.Settings, RawConfig: input.Config, Version: input.Version}
	cfg := input.Config
	settings := input.Settings
	matcher := settings.StatusMatcher
	selected := input.Selection

	currentSession, _ := client.CurrentSession(ctx)
	info, err := client.ListSessionsInfo(ctx)
	if err != nil {
		result.Err = err
		return result
	}

	groups := make([]ProjectGroup, 0)
	groupByKey := make(map[string]*ProjectGroup)

	addGroup := func(key string, group ProjectGroup) *ProjectGroup {
		groups = append(groups, group)
		groupByKey[key] = &groups[len(groups)-1]
		return groupByKey[key]
	}

	projectBySession := make(map[string]*ProjectGroup)
	projectByPath := make(map[string]*ProjectGroup)

	for i := range cfg.Projects {
		pc := &cfg.Projects[i]
		name := strings.TrimSpace(pc.Name)
		if name == "" {
			name = strings.TrimSpace(pc.Session)
		}
		if name == "" {
			name = "project"
		}
		session := strings.TrimSpace(pc.Session)
		if session == "" {
			session = sanitizeSessionName(name)
		}
		path := strings.TrimSpace(pc.Path)
		if path != "" {
			path = expandPath(path)
		}
		groupKey := projectKey(path, name)
		group := addGroup(groupKey, ProjectGroup{
			Name:       name,
			Path:       path,
			FromConfig: true,
		})
		group.Sessions = append(group.Sessions, SessionItem{
			Name:       session,
			Path:       path,
			LayoutName: layoutName(pc.Layout),
			Status:     StatusStopped,
			Config:     pc,
		})
		projectBySession[session] = group
		if path != "" {
			projectByPath[path] = group
		}
	}

	for _, s := range info {
		status := StatusRunning
		if s.Name == currentSession {
			status = StatusCurrent
		}
		path := strings.TrimSpace(s.Path)
		var group *ProjectGroup
		if g := projectBySession[s.Name]; g != nil {
			group = g
		} else if path != "" {
			if g := projectByPath[path]; g != nil {
				group = g
			}
		}
		if group == nil {
			name := groupNameFromPath(path, s.Name)
			group = addGroup(projectKey(path, name), ProjectGroup{
				Name:       name,
				Path:       path,
				FromConfig: false,
			})
			if path != "" {
				projectByPath[path] = group
			}
		}

		windows, winErr := client.ListWindows(ctx, s.Name)
		if winErr != nil {
			result.Err = winErr
			return result
		}
		activeWindow := activeWindowIndex(windows)
		windowCount := len(windows)

		item := findSession(group, s.Name)
		if item == nil {
			group.Sessions = append(group.Sessions, SessionItem{
				Name:         s.Name,
				Path:         path,
				Status:       status,
				WindowCount:  windowCount,
				ActiveWindow: activeWindow,
				Windows:      windowsToItems(windows),
			})
			item = &group.Sessions[len(group.Sessions)-1]
		} else {
			item.Status = status
			item.Path = path
			item.WindowCount = windowCount
			item.ActiveWindow = activeWindow
			item.Windows = windowsToItems(windows)
		}
	}

	resolved := resolveSelection(groups, selected)

	// Populate thumbnails and preview panes
	for gi := range groups {
		for si := range groups[gi].Sessions {
			session := &groups[gi].Sessions[si]
			if session.Status == StatusStopped {
				continue
			}
			if settings.ShowThumbnails {
				thumb, err := sessionThumbnail(ctx, client, *session, settings, matcher)
				if err != nil {
					result.Err = err
					return result
				}
				session.Thumbnail = thumb
			}
		}
	}

	if resolved.Session != "" {
		if target := findSessionByName(groups, resolved.Session); target != nil {
			windowIndex := resolved.Window
			if windowIndex == "" {
				windowIndex = target.ActiveWindow
			}
			if windowIndex != "" {
				panes, err := sessionWindowPanes(ctx, client, target.Name, windowIndex, settings.PreviewLines)
				if err != nil {
					result.Err = err
					return result
				}
				for i := range panes {
					panes[i].Status = classifyPane(panes[i], panes[i].Preview, matcher, settings.IdleThreshold, time.Now())
				}
				attachPanesToWindow(groups, resolved.Session, windowIndex, panes)
			}
		}
	}

	result.Data = DashboardData{Projects: groups, RefreshedAt: time.Now()}
	result.Resolved = resolved
	return result
}

func resolveSelection(groups []ProjectGroup, desired selectionState) selectionState {
	resolved := selectionState{}
	if len(groups) == 0 {
		return resolved
	}
	project := findProject(groups, desired.Project)
	if project == nil {
		project = &groups[0]
	}
	resolved.Project = project.Name
	if len(project.Sessions) == 0 {
		return resolved
	}
	session := findSession(project, desired.Session)
	if session == nil {
		session = &project.Sessions[0]
	}
	resolved.Session = session.Name
	if desired.Window != "" && windowExists(session.Windows, desired.Window) {
		resolved.Window = desired.Window
	} else {
		resolved.Window = session.ActiveWindow
	}
	if resolved.Window == "" && len(session.Windows) > 0 {
		resolved.Window = session.Windows[0].Index
	}
	return resolved
}

func findProject(groups []ProjectGroup, name string) *ProjectGroup {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i]
		}
	}
	return nil
}

func findSession(group *ProjectGroup, name string) *SessionItem {
	if group == nil {
		return nil
	}
	for i := range group.Sessions {
		if group.Sessions[i].Name == name {
			return &group.Sessions[i]
		}
	}
	return nil
}

func findSessionByName(groups []ProjectGroup, name string) *SessionItem {
	for gi := range groups {
		for si := range groups[gi].Sessions {
			if groups[gi].Sessions[si].Name == name {
				return &groups[gi].Sessions[si]
			}
		}
	}
	return nil
}

func windowsToItems(windows []mux.WindowInfo) []WindowItem {
	items := make([]WindowItem, len(windows))
	for i, w := range windows {
		items[i] = WindowItem{Index: w.Index, Name: w.Name, Active: w.Active}
	}
	return items
}

func windowExists(windows []WindowItem, index string) bool {
	for _, w := range windows {
		if w.Index == index {
			return true
		}
	}
	return false
}

func activeWindowIndex(windows []mux.WindowInfo) string {
	for _, w := range windows {
		if w.Active {
			return w.Index
		}
	}
	if len(windows) > 0 {
		return windows[0].Index
	}
	return ""
}

func sessionWindowPanes(ctx context.Context, client mux.Client, session, window string, lines int) ([]PaneItem, error) {
	target := fmt.Sprintf("%s:%s", session, window)
	panes, err := client.ListPanesDetailed(ctx, target)
	if err != nil {
		return nil, err
	}
	items := make([]PaneItem, len(panes))
	for i, p := range panes {
		item := paneFromMux(p)
		if lines > 0 {
			captureLines := captureLinesForPreview(p.Height, lines)
			preview, err := client.CapturePaneLines(ctx, fmt.Sprintf("%s.%s", target, p.Index), captureLines)
			if err != nil {
				return nil, err
			}
			item.Preview = preview
		}
		items[i] = item
	}
	return items, nil
}

func captureLinesForPreview(paneHeight, previewLines int) int {
	lines := previewLines
	if paneHeight > 0 {
		if candidate := paneHeight + previewSlackLines; candidate > lines {
			lines = candidate
		}
	}
	if lines <= 0 {
		lines = defaultPreviewLines
	}
	if lines > maxCaptureLines {
		lines = maxCaptureLines
	}
	return lines
}

func sessionThumbnail(ctx context.Context, client mux.Client, session SessionItem, settings DashboardConfig, matcher statusMatcher) (PaneSummary, error) {
	if session.Name == "" || session.ActiveWindow == "" {
		return PaneSummary{}, nil
	}
	target := fmt.Sprintf("%s:%s", session.Name, session.ActiveWindow)
	panes, err := client.ListPanesDetailed(ctx, target)
	if err != nil {
		return PaneSummary{}, err
	}
	active := pickActivePane(panes)
	if active == nil {
		return PaneSummary{}, nil
	}
	lines, err := client.CapturePaneLines(ctx, fmt.Sprintf("%s.%s", target, active.Index), settings.ThumbnailLines)
	if err != nil {
		return PaneSummary{}, err
	}
	status := classifyPane(paneFromMux(*active), lines, matcher, settings.IdleThreshold, time.Now())
	return PaneSummary{Line: lastNonEmpty(lines), Status: status}, nil
}

func pickActivePane(panes []mux.PaneInfo) *mux.PaneInfo {
	for i := range panes {
		if panes[i].Active {
			return &panes[i]
		}
	}
	if len(panes) > 0 {
		return &panes[0]
	}
	return nil
}

func classifyPane(pane PaneItem, lines []string, matcher statusMatcher, idle time.Duration, now time.Time) PaneStatus {
	if pane.Dead {
		if pane.DeadStatus != 0 {
			return PaneStatusError
		}
		return PaneStatusDone
	}
	joined := strings.Join(lines, "\n")
	if joined != "" {
		if matcher.error != nil && matcher.error.MatchString(joined) {
			return PaneStatusError
		}
		if matcher.success != nil && matcher.success.MatchString(joined) {
			return PaneStatusDone
		}
		if matcher.running != nil && matcher.running.MatchString(joined) {
			return PaneStatusRunning
		}
	}
	if !pane.LastActive.IsZero() && now.Sub(pane.LastActive) > idle {
		return PaneStatusIdle
	}
	return PaneStatusRunning
}

func lastNonEmpty(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}

func attachPanesToWindow(groups []ProjectGroup, sessionName, windowIndex string, panes []PaneItem) {
	for gi := range groups {
		for wi := range groups[gi].Sessions {
			if groups[gi].Sessions[wi].Name != sessionName {
				continue
			}
			for wj := range groups[gi].Sessions[wi].Windows {
				if groups[gi].Sessions[wi].Windows[wj].Index == windowIndex {
					groups[gi].Sessions[wi].Windows[wj].Panes = panes
					return
				}
			}
		}
	}
}

func projectKey(path, name string) string {
	if strings.TrimSpace(path) != "" {
		return strings.ToLower(path)
	}
	return strings.ToLower(name)
}

func groupNameFromPath(path, fallback string) string {
	if path == "" {
		return fallback
	}
	return filepath.Base(path)
}

func layoutName(layoutValue interface{}) string {
	switch v := layoutValue.(type) {
	case string:
		return v
	case *layout.LayoutConfig:
		if v != nil && v.Name != "" {
			return v.Name
		}
		return "inline"
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok && name != "" {
			return name
		}
		return "inline"
	case map[interface{}]interface{}:
		if name, ok := v["name"].(string); ok && name != "" {
			return name
		}
		return "inline"
	default:
		return ""
	}
}
