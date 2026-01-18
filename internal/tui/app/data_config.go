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
	defaultRefreshMS             = 2000
	defaultPreviewLines          = 12
	defaultIdleSeconds           = 20
	minDashboardPreview          = 10
	defaultResizeMouseThrottleMS = 16
	minResizeMouseThrottleMS     = 4
	maxResizeMouseThrottleMS     = 100
)

var (
	defaultSuccessRegex = "(?i)done|finished|success|completed|✅"
	defaultErrorRegex   = "(?i)error|failed|panic|❌"
	defaultRunningRegex = "(?i)running|in progress|building|installing|▶"
)

var (
	paneViewPerfLow = PaneViewPerformance{
		MaxConcurrency:        2,
		MaxInFlightBatches:    1,
		MaxBatch:              4,
		MinIntervalFocused:    60 * time.Millisecond,
		MinIntervalSelected:   180 * time.Millisecond,
		MinIntervalBackground: 400 * time.Millisecond,
		TimeoutFocused:        2000 * time.Millisecond,
		TimeoutSelected:       1400 * time.Millisecond,
		TimeoutBackground:     1100 * time.Millisecond,
		PumpBaseDelay:         25 * time.Millisecond,
		PumpMaxDelay:          150 * time.Millisecond,
		ForceAfter:            400 * time.Millisecond,
		FallbackMinInterval:   250 * time.Millisecond,
	}
	paneViewPerfMedium = PaneViewPerformance{
		MaxConcurrency:        4,
		MaxInFlightBatches:    2,
		MaxBatch:              8,
		MinIntervalFocused:    33 * time.Millisecond,
		MinIntervalSelected:   100 * time.Millisecond,
		MinIntervalBackground: 250 * time.Millisecond,
		TimeoutFocused:        1500 * time.Millisecond,
		TimeoutSelected:       1000 * time.Millisecond,
		TimeoutBackground:     800 * time.Millisecond,
		PumpBaseDelay:         0,
		PumpMaxDelay:          50 * time.Millisecond,
		ForceAfter:            250 * time.Millisecond,
		FallbackMinInterval:   150 * time.Millisecond,
	}
	paneViewPerfHigh = PaneViewPerformance{
		MaxConcurrency:        6,
		MaxInFlightBatches:    3,
		MaxBatch:              12,
		MinIntervalFocused:    16 * time.Millisecond,
		MinIntervalSelected:   60 * time.Millisecond,
		MinIntervalBackground: 150 * time.Millisecond,
		TimeoutFocused:        1000 * time.Millisecond,
		TimeoutSelected:       800 * time.Millisecond,
		TimeoutBackground:     600 * time.Millisecond,
		PumpBaseDelay:         0,
		PumpMaxDelay:          25 * time.Millisecond,
		ForceAfter:            150 * time.Millisecond,
		FallbackMinInterval:   100 * time.Millisecond,
	}
	paneViewPerfMax = PaneViewPerformance{
		MaxConcurrency:        8,
		MaxInFlightBatches:    4,
		MaxBatch:              16,
		MinIntervalFocused:    0,
		MinIntervalSelected:   0,
		MinIntervalBackground: 0,
		TimeoutFocused:        1000 * time.Millisecond,
		TimeoutSelected:       800 * time.Millisecond,
		TimeoutBackground:     600 * time.Millisecond,
		PumpBaseDelay:         0,
		PumpMaxDelay:          0,
		ForceAfter:            0,
		FallbackMinInterval:   0,
	}
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
	paneTopbarEnabled := boolOrDefault(cfg.PaneTopbar.Enabled, true)
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
	projectRootsAllowNonGit := boolOrDefault(cfg.ProjectRootsAllowNonGit, true)
	hiddenProjects := hiddenProjectKeySet(cfg.HiddenProjects)
	matcher, err := compileStatusMatcher(cfg.StatusRegex)
	if err != nil {
		return DashboardConfig{}, err
	}
	resizeSettings, err := resolveResizeConfig(cfg.Resize)
	if err != nil {
		return DashboardConfig{}, err
	}
	performance, err := resolvePerformanceConfig(cfg.Performance)
	if err != nil {
		return DashboardConfig{}, err
	}
	return DashboardConfig{
		RefreshInterval:         time.Duration(refreshMS) * time.Millisecond,
		PreviewLines:            previewLines,
		PreviewCompact:          previewCompact,
		IdleThreshold:           time.Duration(idleSeconds) * time.Second,
		StatusMatcher:           matcher,
		SidebarHidden:           sidebarHidden,
		Resize:                  resizeSettings,
		ProjectRoots:            projectRoots,
		ProjectRootsAllowNonGit: projectRootsAllowNonGit,
		AgentDetection:          agentDetection,
		PaneTopbar:              PaneTopbarSettings{Enabled: paneTopbarEnabled},
		AttachBehavior:          attachBehavior,
		PaneNavigationMode:      paneNavigationMode,
		QuitBehavior:            quitBehavior,
		HiddenProjects:          hiddenProjects,
		Performance:             performance,
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

func resolveResizeConfig(cfg layout.DashboardResizeConfig) (DashboardResizeSettings, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.MouseApply))
	if mode == "" {
		mode = ResizeMouseApplyLive
	}
	switch mode {
	case ResizeMouseApplyLive, ResizeMouseApplyCommit:
	default:
		return DashboardResizeSettings{}, fmt.Errorf("invalid dashboard.resize.mouse_apply %q (use live or commit)", mode)
	}
	throttleMS := cfg.MouseThrottleMS
	if throttleMS <= 0 {
		throttleMS = defaultResizeMouseThrottleMS
	}
	if throttleMS < minResizeMouseThrottleMS {
		throttleMS = minResizeMouseThrottleMS
	}
	if throttleMS > maxResizeMouseThrottleMS {
		throttleMS = maxResizeMouseThrottleMS
	}
	freeze := boolOrDefault(cfg.FreezeContentDuringDrag, true)
	return DashboardResizeSettings{
		MouseApply:              mode,
		MouseThrottle:           time.Duration(throttleMS) * time.Millisecond,
		FreezeContentDuringDrag: freeze,
	}, nil
}

func resolvePerformanceConfig(cfg layout.PerformanceConfig) (DashboardPerformance, error) {
	preset := strings.ToLower(strings.TrimSpace(cfg.Preset))
	if preset == "" {
		preset = PerfPresetMax
	}
	switch preset {
	case PerfPresetLow, PerfPresetMedium, PerfPresetHigh, PerfPresetMax, PerfPresetCustom:
	default:
		return DashboardPerformance{}, fmt.Errorf("invalid dashboard.performance.preset %q (use low, medium, high, max, or custom)", preset)
	}

	renderPolicy := strings.ToLower(strings.TrimSpace(cfg.RenderPolicy))
	if renderPolicy == "" {
		renderPolicy = RenderPolicyVisible
	}
	switch renderPolicy {
	case RenderPolicyVisible, RenderPolicyAll:
	default:
		return DashboardPerformance{}, fmt.Errorf("invalid dashboard.performance.render_policy %q (use visible or all)", renderPolicy)
	}

	previewMode := strings.ToLower(strings.TrimSpace(cfg.PreviewRender.Mode))
	if previewMode == "" {
		previewMode = PreviewRenderDirect
	}
	switch previewMode {
	case PreviewRenderCached, PreviewRenderDirect, PreviewRenderOff:
	default:
		return DashboardPerformance{}, fmt.Errorf("invalid dashboard.performance.preview_render.mode %q (use cached, direct, or off)", previewMode)
	}

	base := paneViewPerfMedium
	switch preset {
	case PerfPresetLow:
		base = paneViewPerfLow
	case PerfPresetHigh:
		base = paneViewPerfHigh
	case PerfPresetMax:
		base = paneViewPerfMax
	case PerfPresetCustom:
		base = applyPaneViewOverrides(base, cfg.PaneViews)
	}

	return DashboardPerformance{
		Preset:        preset,
		RenderPolicy:  renderPolicy,
		PreviewRender: PreviewRenderSettings{Mode: previewMode},
		PaneViews:     base,
	}, nil
}

func applyPaneViewOverrides(base PaneViewPerformance, overrides layout.PaneViewPerformanceConfig) PaneViewPerformance {
	base.MaxConcurrency = applyPositiveInt(base.MaxConcurrency, overrides.MaxConcurrency)
	base.MaxInFlightBatches = applyPositiveInt(base.MaxInFlightBatches, overrides.MaxInFlightBatches)
	base.MaxBatch = applyPositiveInt(base.MaxBatch, overrides.MaxBatch)
	base.MinIntervalFocused = applyDurationMS(base.MinIntervalFocused, overrides.MinIntervalFocusedMS)
	base.MinIntervalSelected = applyDurationMS(base.MinIntervalSelected, overrides.MinIntervalSelectedMS)
	base.MinIntervalBackground = applyDurationMS(base.MinIntervalBackground, overrides.MinIntervalBackgroundMS)
	base.TimeoutFocused = applyDurationMS(base.TimeoutFocused, overrides.TimeoutFocusedMS)
	base.TimeoutSelected = applyDurationMS(base.TimeoutSelected, overrides.TimeoutSelectedMS)
	base.TimeoutBackground = applyDurationMS(base.TimeoutBackground, overrides.TimeoutBackgroundMS)
	base.PumpBaseDelay = applyDurationMS(base.PumpBaseDelay, overrides.PumpBaseDelayMS)
	base.PumpMaxDelay = applyDurationMS(base.PumpMaxDelay, overrides.PumpMaxDelayMS)
	base.ForceAfter = applyDurationMS(base.ForceAfter, overrides.ForceAfterMS)
	base.FallbackMinInterval = applyDurationMS(base.FallbackMinInterval, overrides.FallbackMinIntervalMS)
	return base
}

func applyPositiveInt(base, override int) int {
	if override > 0 {
		return override
	}
	return base
}

func applyDurationMS(base time.Duration, override int) time.Duration {
	if override > 0 {
		return time.Duration(override) * time.Millisecond
	}
	return base
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
