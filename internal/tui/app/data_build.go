package app

import (
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func buildDashboardData(input dashboardSnapshotInput) dashboardSnapshotResult {
	result := dashboardSnapshotResult{
		Settings:   input.Settings,
		RawConfig:  input.Config,
		Version:    input.Version,
		RefreshSeq: input.RefreshSeq,
	}

	index := newDashboardGroupIndex(len(input.Config.Projects) + len(input.Sessions))
	index.addConfigProjects(input.Config, input.Settings)
	index.mergeNativeSessions(input.Sessions, input.Settings)

	groups := index.groups
	sortProjectGroups(groups, input.Config)
	resolved := resolveSelectionForTab(input.Tab, groups, input.Selection)
	applySessionThumbnails(groups, input.Settings)
	resolved = resolveProjectPaneSelection(input.Tab, groups, resolved)

	result.Data = DashboardData{Projects: groups, RefreshedAt: time.Now()}
	result.Resolved = resolved
	return result
}

type dashboardGroupIndex struct {
	groups    []ProjectGroup
	byKey     map[string]int
	bySession map[string]int
}

func newDashboardGroupIndex(capacity int) *dashboardGroupIndex {
	if capacity < 0 {
		capacity = 0
	}
	return &dashboardGroupIndex{
		groups:    make([]ProjectGroup, 0, capacity),
		byKey:     make(map[string]int),
		bySession: make(map[string]int),
	}
}

func (idx *dashboardGroupIndex) addGroup(key string, group ProjectGroup) int {
	if group.ID == "" {
		group.ID = key
	}
	idx.groups = append(idx.groups, group)
	pos := len(idx.groups) - 1
	idx.byKey[key] = pos
	return pos
}

func (idx *dashboardGroupIndex) addConfigProjects(cfg *layout.Config, settings DashboardConfig) {
	for i := range cfg.Projects {
		pc := &cfg.Projects[i]
		name, session, path := normalizeProjectConfig(pc)
		if isHiddenProject(settings, path, name) {
			continue
		}
		groupKey := projectKey(path, name)
		pos, ok := idx.byKey[groupKey]
		if !ok {
			pos = idx.addGroup(groupKey, ProjectGroup{
				ID:         groupKey,
				Name:       name,
				Path:       path,
				FromConfig: true,
			})
		}
		group := &idx.groups[pos]
		group.FromConfig = true
		if group.Name == "" {
			group.Name = name
		}
		if group.Path == "" {
			group.Path = path
		}
		group.Sessions = append(group.Sessions, SessionItem{
			Name:       session,
			Path:       path,
			LayoutName: layoutName(pc.Layout),
			Status:     StatusStopped,
			Config:     pc,
		})
		idx.bySession[session] = pos
	}
}

func (idx *dashboardGroupIndex) mergeNativeSessions(nativeSessions []native.SessionSnapshot, settings DashboardConfig) {
	now := time.Now()
	for _, s := range nativeSessions {
		path := normalizeProjectPath(s.Path)
		name := groupNameFromPath(path, s.Name)
		if isHiddenProject(settings, path, name) {
			continue
		}
		group := idx.groupForSession(s.Name, path)
		if group == nil {
			key := projectKey(path, name)
			pos := idx.addGroup(key, ProjectGroup{
				ID:         key,
				Name:       name,
				Path:       path,
				FromConfig: false,
			})
			group = &idx.groups[pos]
		}
		idx.mergeSession(group, s, settings, now)
	}
}

func (idx *dashboardGroupIndex) groupForSession(name, path string) *ProjectGroup {
	if pos, ok := idx.bySession[name]; ok {
		return &idx.groups[pos]
	}
	key := projectKey(path, name)
	if key == "" {
		return nil
	}
	if pos, ok := idx.byKey[key]; ok {
		return &idx.groups[pos]
	}
	return nil
}

func (idx *dashboardGroupIndex) mergeSession(group *ProjectGroup, session native.SessionSnapshot, settings DashboardConfig, now time.Time) {
	panes := panesFromNative(session.Panes, settings, now)
	activePane := activePaneIndex(panes)
	paneCount := len(panes)

	item := findSession(group, session.Name)
	if item == nil {
		group.Sessions = append(group.Sessions, SessionItem{
			Name:       session.Name,
			Path:       normalizeProjectPath(session.Path),
			LayoutName: session.LayoutName,
			Status:     StatusRunning,
			PaneCount:  paneCount,
			ActivePane: activePane,
			Panes:      panes,
		})
		return
	}
	item.Status = StatusRunning
	item.Path = normalizeProjectPath(session.Path)
	item.LayoutName = session.LayoutName
	item.PaneCount = paneCount
	item.ActivePane = activePane
	item.Panes = panes
}

func resolveSelectionForTab(tab DashboardTab, groups []ProjectGroup, desired selectionState) selectionState {
	if tab == TabDashboard {
		return resolveDashboardSelection(groups, desired)
	}
	return resolveSelection(groups, desired)
}

func applySessionThumbnails(groups []ProjectGroup, settings DashboardConfig) {
	if !settings.ShowThumbnails {
		return
	}
	for gi := range groups {
		for si := range groups[gi].Sessions {
			session := &groups[gi].Sessions[si]
			if session.Status == StatusStopped {
				continue
			}
			session.Thumbnail = sessionThumbnailFromData(session, settings)
		}
	}
}

func resolveProjectPaneSelection(tab DashboardTab, groups []ProjectGroup, resolved selectionState) selectionState {
	if tab != TabProject || resolved.Session == "" {
		return resolved
	}
	if target := findSessionByName(groups, resolved.Session); target != nil {
		resolved.Pane = resolvePaneSelection(resolved.Pane, target.Panes)
	}
	return resolved
}

func sortProjectGroups(groups []ProjectGroup, cfg *layout.Config) {
	if len(groups) < 2 {
		return
	}
	order := configProjectOrder(cfg)
	sort.SliceStable(groups, func(i, j int) bool {
		left := buildProjectGroupSortMeta(groups[i], order)
		right := buildProjectGroupSortMeta(groups[j], order)
		if left.hasOrder || right.hasOrder {
			if left.hasOrder && right.hasOrder && left.order != right.order {
				return left.order < right.order
			}
			if left.hasOrder != right.hasOrder {
				return left.hasOrder
			}
		}
		if left.name != right.name {
			return left.name < right.name
		}
		if left.path != right.path {
			return left.path < right.path
		}
		return left.key < right.key
	})
}

type projectGroupSortMeta struct {
	key      string
	name     string
	path     string
	order    int
	hasOrder bool
}

func buildProjectGroupSortMeta(group ProjectGroup, order map[string]int) projectGroupSortMeta {
	key := group.ID
	if key == "" {
		key = projectKey(group.Path, group.Name)
	}
	meta := projectGroupSortMeta{
		key:  key,
		name: strings.ToLower(strings.TrimSpace(group.Name)),
		path: strings.ToLower(strings.TrimSpace(group.Path)),
	}
	if order != nil {
		if idx, ok := order[key]; ok {
			meta.order = idx
			meta.hasOrder = true
		}
	}
	return meta
}

func configProjectOrder(cfg *layout.Config) map[string]int {
	if cfg == nil || len(cfg.Projects) == 0 {
		return nil
	}
	order := make(map[string]int, len(cfg.Projects))
	for i := range cfg.Projects {
		name, _, path := normalizeProjectConfig(&cfg.Projects[i])
		key := projectKey(path, name)
		if key == "" {
			continue
		}
		if _, exists := order[key]; !exists {
			order[key] = i
		}
	}
	if len(order) == 0 {
		return nil
	}
	return order
}

func panesFromNative(panes []native.PaneSnapshot, settings DashboardConfig, now time.Time) []PaneItem {
	if len(panes) == 0 {
		return nil
	}
	items := make([]PaneItem, len(panes))
	for i, p := range panes {
		item := PaneItem{
			ID:            p.ID,
			Index:         p.Index,
			Title:         p.Title,
			Command:       p.Command,
			StartCommand:  p.StartCommand,
			PID:           p.PID,
			Active:        p.Active,
			Left:          p.Left,
			Top:           p.Top,
			Width:         p.Width,
			Height:        p.Height,
			Dead:          p.Dead,
			DeadStatus:    p.DeadStatus,
			RestoreFailed: p.RestoreFailed,
			RestoreError:  p.RestoreError,
			LastActive:    p.LastActive,
			Preview:       p.Preview,
		}
		item.Status = classifyPane(item, item.Preview, settings, now)
		items[i] = item
	}
	return items
}

func paneExists(panes []PaneItem, index string) bool {
	for _, p := range panes {
		if p.Index == index {
			return true
		}
	}
	return false
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

func sessionThumbnailFromData(session *SessionItem, settings DashboardConfig) PaneSummary {
	if session == nil {
		return PaneSummary{}
	}
	if len(session.Panes) == 0 {
		return PaneSummary{}
	}
	var active *PaneItem
	for i := range session.Panes {
		if session.Panes[i].Active {
			active = &session.Panes[i]
			break
		}
	}
	if active == nil {
		active = &session.Panes[0]
	}
	return PaneSummary{Line: paneSummaryLine(*active, settings.ThumbnailLines), Status: active.Status}
}
