package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/workspace"
)

const (
	defaultRefreshMS    = 2000
	defaultPreviewLines = 12
	defaultIdleSeconds  = 20
	minDashboardPreview = 10
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
	refreshMS := positiveIntOrDefault(cfg.RefreshMS, defaultRefreshMS)
	previewLines := positiveIntOrDefault(cfg.PreviewLines, defaultPreviewLines)
	idleSeconds := positiveIntOrDefault(cfg.IdleSeconds, defaultIdleSeconds)
	previewCompact := boolOrDefault(cfg.PreviewCompact, true)
	agentDetection := resolveAgentDetection(cfg.AgentDetection)
	previewMode, err := resolvePreviewMode(cfg.PreviewMode)
	if err != nil {
		return DashboardConfig{}, err
	}
	sidebarHidden := boolOrDefault(cfg.Sidebar.Hidden, false)
	attachBehavior, err := resolveAttachBehavior(cfg.AttachBehavior)
	if err != nil {
		return DashboardConfig{}, err
	}
	paneNavigationMode, err := resolvePaneNavigationMode(cfg.PaneNavigationMode)
	if err != nil {
		return DashboardConfig{}, err
	}
	quitBehavior, err := resolveQuitBehavior(cfg.QuitBehavior)
	if err != nil {
		return DashboardConfig{}, err
	}
	projectRoots := resolveProjectRoots(cfg.ProjectRoots)
	hiddenProjects := hiddenProjectKeySet(cfg.HiddenProjects)
	matcher, err := compileStatusMatcher(cfg.StatusRegex)
	if err != nil {
		return DashboardConfig{}, err
	}
	return DashboardConfig{
		RefreshInterval:    time.Duration(refreshMS) * time.Millisecond,
		PreviewLines:       previewLines,
		PreviewCompact:     previewCompact,
		IdleThreshold:      time.Duration(idleSeconds) * time.Second,
		StatusMatcher:      matcher,
		PreviewMode:        previewMode,
		SidebarHidden:      sidebarHidden,
		ProjectRoots:       projectRoots,
		AgentDetection:     agentDetection,
		AttachBehavior:     attachBehavior,
		PaneNavigationMode: paneNavigationMode,
		QuitBehavior:       quitBehavior,
		HiddenProjects:     hiddenProjects,
	}, nil
}

func positiveIntOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value != nil {
		return *value
	}
	return fallback
}

func resolveAgentDetection(cfg layout.AgentDetectionConfig) AgentDetectionConfig {
	agentDetection := AgentDetectionConfig{Codex: true, Claude: true}
	if cfg.Codex != nil {
		agentDetection.Codex = *cfg.Codex
	}
	if cfg.Claude != nil {
		agentDetection.Claude = *cfg.Claude
	}
	return agentDetection
}

func resolvePreviewMode(value string) (string, error) {
	previewMode := strings.TrimSpace(value)
	if previewMode == "" {
		previewMode = "grid"
	}
	if previewMode != "grid" && previewMode != "layout" {
		return "", fmt.Errorf("invalid preview_mode %q (use grid or layout)", previewMode)
	}
	return previewMode, nil
}

func resolveAttachBehavior(value string) (string, error) {
	attachBehavior, ok := normalizeAttachBehavior(value)
	if !ok {
		return "", fmt.Errorf("invalid attach_behavior %q (use current or detached)", value)
	}
	return attachBehavior, nil
}

func resolvePaneNavigationMode(value string) (string, error) {
	paneNavigationMode, ok := normalizePaneNavigationMode(value)
	if !ok {
		return "", fmt.Errorf("invalid pane_navigation_mode %q (use spatial or memory)", value)
	}
	return paneNavigationMode, nil
}

func resolveQuitBehavior(value string) (string, error) {
	quitBehavior, ok := normalizeQuitBehavior(value)
	if !ok {
		return "", fmt.Errorf("invalid quit_behavior %q (use prompt, keep, or stop)", value)
	}
	return quitBehavior, nil
}

func resolveProjectRoots(roots []string) []string {
	return workspace.ResolveProjectRoots(roots)
}

func normalizeAttachBehavior(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return AttachBehaviorCurrent, true
	}
	switch strings.ToLower(trimmed) {
	case AttachBehaviorCurrent:
		return AttachBehaviorCurrent, true
	case AttachBehaviorDetached:
		return AttachBehaviorDetached, true
	default:
		return "", false
	}
}

func normalizePaneNavigationMode(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return PaneNavigationSpatial, true
	}
	switch strings.ToLower(trimmed) {
	case PaneNavigationSpatial:
		return PaneNavigationSpatial, true
	case PaneNavigationMemory:
		return PaneNavigationMemory, true
	default:
		return "", false
	}
}

func normalizeQuitBehavior(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return QuitBehaviorPrompt, true
	}
	switch strings.ToLower(trimmed) {
	case QuitBehaviorPrompt:
		return QuitBehaviorPrompt, true
	case QuitBehaviorKeep:
		return QuitBehaviorKeep, true
	case QuitBehaviorStop:
		return QuitBehaviorStop, true
	default:
		return "", false
	}
}

func defaultProjectRoots() []string {
	return workspace.DefaultProjectRoots()
}

func normalizeProjectPath(path string) string {
	return workspace.NormalizeProjectPath(path)
}

func normalizeProjectRoots(roots []string) []string {
	return workspace.NormalizeProjectRoots(roots)
}

func normalizeHiddenProjects(entries []layout.HiddenProjectConfig) []layout.HiddenProjectConfig {
	return workspace.NormalizeHiddenProjects(entries)
}

func hiddenProjectKeySet(entries []layout.HiddenProjectConfig) map[string]struct{} {
	return workspace.HiddenProjectKeySet(entries)
}

func loadConfig(path string) (*layout.Config, error) {
	return workspace.LoadConfig(path)
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
