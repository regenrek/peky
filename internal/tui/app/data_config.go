package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/userpath"
	"gopkg.in/yaml.v3"
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
	refreshMS := cfg.RefreshMS
	if refreshMS <= 0 {
		refreshMS = defaultRefreshMS
	}
	previewLines := cfg.PreviewLines
	if previewLines <= 0 {
		previewLines = defaultPreviewLines
	}
	idleSeconds := cfg.IdleSeconds
	if idleSeconds <= 0 {
		idleSeconds = defaultIdleSeconds
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
		return DashboardConfig{}, fmt.Errorf("invalid attach_behavior %q (use current or detached)", cfg.AttachBehavior)
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
		IdleThreshold:   time.Duration(idleSeconds) * time.Second,
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
	path = userpath.ExpandUser(path)
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
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
		root = normalizeProjectPath(root)
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
