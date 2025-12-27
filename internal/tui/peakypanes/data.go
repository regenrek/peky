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
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/tmuxctl"
	"gopkg.in/yaml.v3"
)

const (
	defaultRefreshMS      = 2000
	defaultPreviewLines   = 12
	defaultThumbnailLines = 1
	defaultIdleSeconds    = 20
	previewSlackLines     = 20
	maxCaptureLines       = 400
	minDashboardPreview   = 10
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
	agentDetection := AgentDetectionConfig{Codex: true, Claude: true}
	if cfg.AgentDetection.Codex != nil {
		agentDetection.Codex = *cfg.AgentDetection.Codex
	}
	if cfg.AgentDetection.Claude != nil {
		agentDetection.Claude = *cfg.AgentDetection.Claude
	}
	previewMode := strings.TrimSpace(cfg.PreviewMode)
	if previewMode == "" {
		previewMode = "grid"
	}
	if previewMode != "grid" && previewMode != "layout" {
		return DashboardConfig{}, fmt.Errorf("invalid preview_mode %q (use grid or layout)", previewMode)
	}
	attachBehavior, ok := normalizeAttachBehavior(cfg.AttachBehavior)
	if !ok {
		return DashboardConfig{}, fmt.Errorf("invalid attach_behavior %q (use current, new_terminal, or detached)", cfg.AttachBehavior)
	}
	projectRoots := normalizeProjectRoots(cfg.ProjectRoots)
	if len(projectRoots) == 0 {
		projectRoots = defaultProjectRoots()
	}
	hiddenProjects := hiddenProjectKeySet(cfg.HiddenProjects)
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
		AgentDetection:  agentDetection,
		AttachBehavior:  attachBehavior,
		HiddenProjects:  hiddenProjects,
	}, nil
}

func normalizeAttachBehavior(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return AttachBehaviorNewTerminal, true
	}
	switch strings.ToLower(trimmed) {
	case AttachBehaviorCurrent, "attach", "default":
		return AttachBehaviorCurrent, true
	case AttachBehaviorNewTerminal, "new-terminal", "newterminal", "terminal":
		return AttachBehaviorNewTerminal, true
	case AttachBehaviorDetached, "detach", "detached-only":
		return AttachBehaviorDetached, true
	default:
		return "", false
	}
}

func defaultProjectRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "projects")}
}

func normalizeProjectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = expandPath(path)
	path = filepath.Clean(path)
	return path
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

func normalizeHiddenProjects(entries []layout.HiddenProjectConfig) []layout.HiddenProjectConfig {
	if len(entries) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]layout.HiddenProjectConfig, 0, len(entries))
	for _, entry := range entries {
		entry.Name = strings.TrimSpace(entry.Name)
		entry.Path = normalizeProjectPath(entry.Path)
		key := normalizeProjectKey(entry.Path, entry.Name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func hiddenProjectKeySet(entries []layout.HiddenProjectConfig) map[string]struct{} {
	if len(entries) == 0 {
		return nil
	}
	keys := make(map[string]struct{})
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name)
		path := normalizeProjectPath(entry.Path)
		if path != "" {
			keys[strings.ToLower(path)] = struct{}{}
		}
		if name != "" {
			keys[strings.ToLower(name)] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return keys
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

func buildDashboardData(ctx context.Context, client *tmuxctl.Client, input tmuxSnapshotInput) tmuxSnapshotResult {
	result := tmuxSnapshotResult{Settings: input.Settings, RawConfig: input.Config, Version: input.Version}
	cfg := input.Config
	settings := input.Settings
	selected := input.Selection

	var currentSession string
	var info []tmuxctl.SessionInfo
	if client != nil {
		currentSession, _ = client.CurrentSession(ctx)
		var err error
		info, err = client.ListSessionsInfo(ctx)
		if err != nil {
			result.Err = err
			return result
		}
	}

	var nativeSessions []native.SessionSnapshot
	if input.Native != nil {
		previewLines := settings.PreviewLines
		if dashboard := dashboardPreviewLines(settings); dashboard > previewLines {
			previewLines = dashboard
		}
		nativeSessions = input.Native.Snapshot(previewLines)
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
	localConfigs := make(map[string]*layout.ProjectLocalConfig)
	resolveMux := func(pc *layout.ProjectConfig) string {
		if pc == nil {
			return layout.ResolveMultiplexer(nil, nil, cfg)
		}
		path := normalizeProjectPath(pc.Path)
		if path == "" {
			return layout.ResolveMultiplexer(nil, pc, cfg)
		}
		if cached, ok := localConfigs[path]; ok {
			return layout.ResolveMultiplexer(cached, pc, cfg)
		}
		localCfg, err := layout.LoadProjectLocal(path)
		if err != nil {
			localConfigs[path] = nil
			return layout.ResolveMultiplexer(nil, pc, cfg)
		}
		localConfigs[path] = localCfg
		return layout.ResolveMultiplexer(localCfg, pc, cfg)
	}

	for i := range cfg.Projects {
		pc := &cfg.Projects[i]
		name, session, path := normalizeProjectConfig(pc)
		groupKey := projectKey(path, name)
		if isHiddenProject(settings, path, name) {
			continue
		}
		group := addGroup(groupKey, ProjectGroup{
			Name:       name,
			Path:       path,
			FromConfig: true,
		})
		group.Sessions = append(group.Sessions, SessionItem{
			Name:        session,
			Path:        path,
			LayoutName:  layoutName(pc.Layout),
			Multiplexer: resolveMux(pc),
			Status:      StatusStopped,
			Config:      pc,
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
		path := normalizeProjectPath(s.Path)
		name := groupNameFromPath(path, s.Name)
		if isHiddenProject(settings, path, name) {
			continue
		}
		var group *ProjectGroup
		if g := projectBySession[s.Name]; g != nil {
			group = g
		} else if path != "" {
			if g := projectByPath[path]; g != nil {
				group = g
			}
		}
		if group == nil {
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
				Multiplexer:  layout.MultiplexerTmux,
				Status:       status,
				WindowCount:  windowCount,
				ActiveWindow: activeWindow,
				Windows:      windowsToItems(windows),
			})
			item = &group.Sessions[len(group.Sessions)-1]
		} else {
			item.Status = status
			item.Path = path
			item.Multiplexer = layout.MultiplexerTmux
			item.WindowCount = windowCount
			item.ActiveWindow = activeWindow
			item.Windows = windowsToItems(windows)
		}
	}

	for _, s := range nativeSessions {
		status := StatusRunning
		path := normalizeProjectPath(s.Path)
		name := groupNameFromPath(path, s.Name)
		if isHiddenProject(settings, path, name) {
			continue
		}
		var group *ProjectGroup
		if g := projectBySession[s.Name]; g != nil {
			group = g
		} else if path != "" {
			if g := projectByPath[path]; g != nil {
				group = g
			}
		}
		if group == nil {
			group = addGroup(projectKey(path, name), ProjectGroup{
				Name:       name,
				Path:       path,
				FromConfig: false,
			})
			if path != "" {
				projectByPath[path] = group
			}
		}

		activeWindow := ""
		if len(s.Windows) > 0 {
			activeWindow = s.Windows[0].Index
		}
		windows := windowsFromNative(s.Windows, activeWindow, settings)
		windowCount := len(windows)

		item := findSession(group, s.Name)
		if item == nil {
			group.Sessions = append(group.Sessions, SessionItem{
				Name:         s.Name,
				Path:         path,
				LayoutName:   s.LayoutName,
				Multiplexer:  layout.MultiplexerNative,
				Status:       status,
				WindowCount:  windowCount,
				ActiveWindow: activeWindow,
				Windows:      windows,
			})
			item = &group.Sessions[len(group.Sessions)-1]
		} else {
			item.Status = status
			item.Path = path
			item.LayoutName = s.LayoutName
			item.Multiplexer = layout.MultiplexerNative
			item.WindowCount = windowCount
			item.ActiveWindow = activeWindow
			item.Windows = windows
		}
	}

	var resolved selectionState
	if input.Tab == TabDashboard {
		previewLines := dashboardPreviewLines(settings)
		if err := populateAllProjectPanes(ctx, client, groups, previewLines, settings); err != nil {
			result.Err = err
			return result
		}
		resolved = resolveDashboardSelection(groups, selected)
	} else {
		resolved = resolveSelection(groups, selected)
		if resolved.Project != "" {
			if err := populateProjectPanes(ctx, client, groups, resolved.Project, 0, settings); err != nil {
				result.Err = err
				return result
			}
		}
	}

	// Populate thumbnails
	for gi := range groups {
		for si := range groups[gi].Sessions {
			session := &groups[gi].Sessions[si]
			if session.Status == StatusStopped {
				continue
			}
			if !settings.ShowThumbnails {
				continue
			}
			if input.Tab == TabDashboard {
				session.Thumbnail = sessionThumbnailFromData(session)
				continue
			}
			if session.Multiplexer == layout.MultiplexerNative || client == nil {
				session.Thumbnail = sessionThumbnailFromData(session)
				continue
			}
			thumb, err := sessionThumbnail(ctx, client, *session, settings)
			if err != nil {
				result.Err = err
				return result
			}
			session.Thumbnail = thumb
		}
	}

	if input.Tab == TabProject && resolved.Session != "" {
		if target := findSessionByName(groups, resolved.Session); target != nil {
			windowIndex := resolved.Window
			if windowIndex == "" {
				windowIndex = target.ActiveWindow
			}
			if windowIndex != "" {
				if target.Multiplexer == layout.MultiplexerNative || client == nil {
					window := selectedWindow(target, windowIndex)
					if window != nil {
						resolved.Pane = resolvePaneSelection(resolved.Pane, window.Panes)
					}
				} else {
					panes, err := sessionWindowPanes(ctx, client, target.Name, windowIndex, settings.PreviewLines)
					if err != nil {
						result.Err = err
						return result
					}
					for i := range panes {
						panes[i].Status = classifyPane(panes[i], panes[i].Preview, settings, time.Now())
					}
					attachPanesToWindow(groups, resolved.Session, windowIndex, panes)
					resolved.Pane = resolvePaneSelection(resolved.Pane, panes)
				}
			}
		}
	}

	result.Data = DashboardData{Projects: groups, RefreshedAt: time.Now()}
	result.Resolved = resolved
	return result
}

func normalizeProjectConfig(pc *layout.ProjectConfig) (name, session, path string) {
	if pc == nil {
		return "project", layout.SanitizeSessionName("project"), ""
	}
	name = strings.TrimSpace(pc.Name)
	if name == "" {
		name = strings.TrimSpace(pc.Session)
	}
	if name == "" {
		name = "project"
	}
	session = strings.TrimSpace(pc.Session)
	if session == "" {
		session = layout.SanitizeSessionName(name)
	}
	path = normalizeProjectPath(pc.Path)
	return name, session, path
}

func normalizeProjectKey(path, name string) string {
	path = normalizeProjectPath(path)
	if path != "" {
		return strings.ToLower(path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return strings.ToLower(name)
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
	if desired.Window != "" {
		if windowExists(session.Windows, desired.Window) {
			resolved.Window = desired.Window
		} else {
			resolved.Window = session.ActiveWindow
		}
		if resolved.Window == "" && len(session.Windows) > 0 {
			resolved.Window = session.Windows[0].Index
		}
	}
	resolved.Pane = desired.Pane
	return resolved
}

func resolveDashboardSelection(groups []ProjectGroup, desired selectionState) selectionState {
	columns := collectDashboardColumns(groups)
	if len(columns) == 0 {
		return selectionState{}
	}
	if selected := resolveDashboardSelectionFromColumns(columns, desired); selected.Project != "" {
		return selected
	}
	return selectionState{}
}

func resolveDashboardSelectionFromColumns(columns []DashboardProjectColumn, desired selectionState) selectionState {
	if len(columns) == 0 {
		return selectionState{}
	}
	if desired.Session != "" {
		for _, column := range columns {
			if len(column.Panes) == 0 {
				continue
			}
			if idx := dashboardPaneIndex(column.Panes, desired); idx >= 0 {
				pane := column.Panes[idx]
				return selectionState{
					Project: column.ProjectName,
					Session: pane.SessionName,
					Window:  pane.WindowIndex,
					Pane:    pane.Pane.Index,
				}
			}
		}
	}
	if desired.Project != "" {
		for _, column := range columns {
			if column.ProjectName != desired.Project {
				continue
			}
			if len(column.Panes) == 0 {
				return selectionState{Project: column.ProjectName}
			}
			idx := dashboardPaneIndex(column.Panes, desired)
			if idx < 0 {
				idx = 0
			}
			pane := column.Panes[idx]
			return selectionState{
				Project: column.ProjectName,
				Session: pane.SessionName,
				Window:  pane.WindowIndex,
				Pane:    pane.Pane.Index,
			}
		}
	}
	for _, column := range columns {
		if len(column.Panes) == 0 {
			continue
		}
		pane := column.Panes[0]
		return selectionState{
			Project: column.ProjectName,
			Session: pane.SessionName,
			Window:  pane.WindowIndex,
			Pane:    pane.Pane.Index,
		}
	}
	if len(columns) > 0 {
		return selectionState{Project: columns[0].ProjectName}
	}
	return selectionState{}
}

func resolvePaneSelection(desired string, panes []PaneItem) string {
	if desired != "" && paneExists(panes, desired) {
		return desired
	}
	if active := activePaneIndex(panes); active != "" {
		return active
	}
	if len(panes) > 0 {
		return panes[0].Index
	}
	return ""
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

func findProjectForSession(groups []ProjectGroup, name string) (*ProjectGroup, *SessionItem) {
	for gi := range groups {
		for si := range groups[gi].Sessions {
			if groups[gi].Sessions[si].Name == name {
				return &groups[gi], &groups[gi].Sessions[si]
			}
		}
	}
	return nil, nil
}

func windowsToItems(windows []tmuxctl.WindowInfo) []WindowItem {
	items := make([]WindowItem, len(windows))
	for i, w := range windows {
		items[i] = WindowItem{Index: w.Index, Name: w.Name, Active: w.Active}
	}
	return items
}

func windowsFromNative(windows []native.WindowSnapshot, activeWindow string, settings DashboardConfig) []WindowItem {
	if len(windows) == 0 {
		return nil
	}
	if strings.TrimSpace(activeWindow) == "" {
		activeWindow = windows[0].Index
	}
	items := make([]WindowItem, len(windows))
	now := time.Now()
	for i, w := range windows {
		panes := panesFromNative(w.Panes, settings, now)
		items[i] = WindowItem{
			Index:  w.Index,
			Name:   w.Name,
			Active: w.Index == activeWindow,
			Panes:  panes,
		}
	}
	return items
}

func panesFromNative(panes []native.PaneSnapshot, settings DashboardConfig, now time.Time) []PaneItem {
	if len(panes) == 0 {
		return nil
	}
	items := make([]PaneItem, len(panes))
	for i, p := range panes {
		item := PaneItem{
			ID:           p.ID,
			Multiplexer:  layout.MultiplexerNative,
			Index:        p.Index,
			Title:        p.Title,
			Command:      p.Command,
			StartCommand: p.StartCommand,
			PID:          p.PID,
			Active:       p.Active,
			Left:         p.Left,
			Top:          p.Top,
			Width:        p.Width,
			Height:       p.Height,
			Dead:         p.Dead,
			DeadStatus:   p.DeadStatus,
			LastActive:   p.LastActive,
			Preview:      p.Preview,
		}
		item.Status = classifyPane(item, item.Preview, settings, now)
		items[i] = item
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

func paneExists(panes []PaneItem, index string) bool {
	for _, p := range panes {
		if p.Index == index {
			return true
		}
	}
	return false
}

func activeWindowIndex(windows []tmuxctl.WindowInfo) string {
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

func activePaneIndex(panes []PaneItem) string {
	for _, p := range panes {
		if p.Active {
			return p.Index
		}
	}
	if len(panes) > 0 {
		return panes[0].Index
	}
	return ""
}

func sessionWindowPanes(ctx context.Context, client *tmuxctl.Client, session, window string, lines int) ([]PaneItem, error) {
	target := fmt.Sprintf("%s:%s", session, window)
	panes, err := client.ListPanesDetailed(ctx, target)
	if err != nil {
		return nil, err
	}
	items := make([]PaneItem, len(panes))
	for i, p := range panes {
		item := paneFromTmux(p)
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

func dashboardPreviewLines(settings DashboardConfig) int {
	lines := settings.PreviewLines
	if lines < minDashboardPreview {
		lines = minDashboardPreview
	}
	if lines <= 0 {
		lines = minDashboardPreview
	}
	return lines
}

func sessionThumbnail(ctx context.Context, client *tmuxctl.Client, session SessionItem, settings DashboardConfig) (PaneSummary, error) {
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
	status := classifyPane(paneFromTmux(*active), lines, settings, time.Now())
	return PaneSummary{Line: lastNonEmpty(lines), Status: status}, nil
}

func sessionThumbnailFromData(session *SessionItem) PaneSummary {
	if session == nil {
		return PaneSummary{}
	}
	window := selectedWindow(session, session.ActiveWindow)
	if window == nil || len(window.Panes) == 0 {
		return PaneSummary{}
	}
	var active *PaneItem
	for i := range window.Panes {
		if window.Panes[i].Active {
			active = &window.Panes[i]
			break
		}
	}
	if active == nil {
		active = &window.Panes[0]
	}
	return PaneSummary{Line: lastNonEmpty(active.Preview), Status: active.Status}
}

func pickActivePane(panes []tmuxctl.PaneInfo) *tmuxctl.PaneInfo {
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

func classifyPane(pane PaneItem, lines []string, settings DashboardConfig, now time.Time) PaneStatus {
	if pane.Dead {
		if pane.DeadStatus != 0 {
			return PaneStatusError
		}
		return PaneStatusDone
	}
	matcher := settings.StatusMatcher
	if status, ok := classifyAgentState(pane, settings, now); ok {
		if status != PaneStatusIdle {
			return status
		}
		joined := stripANSI(strings.Join(lines, "\n"))
		if joined != "" && matcher.running != nil && matcher.running.MatchString(joined) {
			return PaneStatusRunning
		}
		return status
	}
	joined := stripANSI(strings.Join(lines, "\n"))
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
	if !pane.LastActive.IsZero() && now.Sub(pane.LastActive) > settings.IdleThreshold {
		return PaneStatusIdle
	}
	return PaneStatusRunning
}

func lastNonEmpty(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(stripANSI(lines[i]))
		if line != "" {
			return line
		}
	}
	return ""
}

func populateProjectPanes(ctx context.Context, client *tmuxctl.Client, groups []ProjectGroup, projectName string, previewLines int, settings DashboardConfig) error {
	project := findProject(groups, projectName)
	if project == nil || client == nil {
		return nil
	}
	for si := range project.Sessions {
		session := &project.Sessions[si]
		if session.Status == StatusStopped {
			continue
		}
		if session.Multiplexer != layout.MultiplexerTmux {
			continue
		}
		if strings.TrimSpace(session.Name) == "" {
			continue
		}
		for wi := range session.Windows {
			window := &session.Windows[wi]
			if strings.TrimSpace(window.Index) == "" {
				continue
			}
			panes, err := sessionWindowPanes(ctx, client, session.Name, window.Index, previewLines)
			if err != nil {
				return err
			}
			if previewLines > 0 {
				now := time.Now()
				for i := range panes {
					panes[i].Status = classifyPane(panes[i], panes[i].Preview, settings, now)
				}
			}
			window.Panes = panes
		}
	}
	return nil
}

func populateAllProjectPanes(ctx context.Context, client *tmuxctl.Client, groups []ProjectGroup, previewLines int, settings DashboardConfig) error {
	for i := range groups {
		if err := populateProjectPanes(ctx, client, groups, groups[i].Name, previewLines, settings); err != nil {
			return err
		}
	}
	return nil
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
	return normalizeProjectKey(path, name)
}

func isHiddenProject(settings DashboardConfig, path, name string) bool {
	if len(settings.HiddenProjects) == 0 {
		return false
	}
	path = normalizeProjectPath(path)
	if path != "" {
		if _, ok := settings.HiddenProjects[strings.ToLower(path)]; ok {
			return true
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	_, ok := settings.HiddenProjects[strings.ToLower(name)]
	return ok
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
