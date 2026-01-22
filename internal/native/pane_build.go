package native

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
	"github.com/regenrek/peakypanes/internal/terminal"
)

var newWindow = terminal.NewWindow

const (
	defaultPaneSpawnThreshold    = 8
	defaultPaneSpawnSpacingMS    = 25
	defaultPaneSpawnWaitOutputMS = 0
	maxPaneSpawnSpacingMS        = 1000
	maxPaneSpawnWaitOutputMS     = 10000

	paneSpawnThresholdEnv  = "PEKY_PANE_SPAWN_THRESHOLD"
	paneSpawnSpacingEnv    = "PEKY_PANE_SPAWN_SPACING_MS"
	paneSpawnWaitOutputEnv = "PEKY_PANE_SPAWN_WAIT_OUTPUT_MS"
)

type paneSpawnPaceConfig struct {
	threshold  int
	spacing    time.Duration
	waitOutput time.Duration
}

var (
	paneSpawnPaceOnce sync.Once
	paneSpawnPace     paneSpawnPaceConfig
)

func resolvePaneSpawnPace() paneSpawnPaceConfig {
	paneSpawnPaceOnce.Do(func() {
		threshold := parsePaneSpawnInt(paneSpawnThresholdEnv, defaultPaneSpawnThreshold)
		if threshold < 1 {
			threshold = 0
		}
		spacingMS := clampPaneSpawnInt(parsePaneSpawnInt(paneSpawnSpacingEnv, defaultPaneSpawnSpacingMS), 0, maxPaneSpawnSpacingMS)
		waitOutputMS := clampPaneSpawnInt(parsePaneSpawnInt(paneSpawnWaitOutputEnv, defaultPaneSpawnWaitOutputMS), 0, maxPaneSpawnWaitOutputMS)
		paneSpawnPace = paneSpawnPaceConfig{
			threshold:  threshold,
			spacing:    time.Duration(spacingMS) * time.Millisecond,
			waitOutput: time.Duration(waitOutputMS) * time.Millisecond,
		}
	})
	return paneSpawnPace
}

func parsePaneSpawnInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func clampPaneSpawnInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (m *Manager) buildPanes(ctx context.Context, spec SessionSpec) ([]*Pane, error) {
	if spec.Layout == nil {
		return nil, errors.New("native: layout is nil")
	}
	layoutCfg := spec.Layout
	if strings.TrimSpace(layoutCfg.Grid) != "" {
		return m.buildGridPanes(ctx, spec.Path, layoutCfg, spec.Env)
	}
	if len(layoutCfg.Panes) == 0 {
		return nil, errors.New("native: layout has no panes defined")
	}
	return m.buildSplitPanes(ctx, spec.Path, layoutCfg.Panes, spec.Env)
}

func (m *Manager) buildGridPanes(ctx context.Context, path string, layoutCfg *layout.LayoutConfig, env []string) ([]*Pane, error) {
	grid, err := layout.Parse(layoutCfg.Grid)
	if err != nil {
		return nil, fmt.Errorf("native: parse grid %q: %w", layoutCfg.Grid, err)
	}
	commands := layout.ResolveGridCommands(layoutCfg, grid.Panes())
	titles := layout.ResolveGridTitles(layoutCfg, grid.Panes())
	paneDefs := layoutCfg.Panes

	panes := make([]*Pane, 0, grid.Panes())
	total := grid.Panes()
	for r := 0; r < grid.Rows; r++ {
		for c := 0; c < grid.Columns; c++ {
			idx := r*grid.Columns + c
			title := ""
			cmd := ""
			if idx < len(titles) {
				title = titles[idx]
			}
			if idx < len(commands) {
				cmd = commands[idx]
			}
			if idx < len(paneDefs) {
				paneDef := paneDefs[idx]
				if strings.TrimSpace(paneDef.Title) != "" {
					title = paneDef.Title
				}
				if strings.TrimSpace(paneDef.Cmd) != "" {
					cmd = paneDef.Cmd
				}
			}
			pane, err := m.createPane(ctx, path, title, cmd, env)
			if err != nil {
				m.closePanes(panes)
				return nil, err
			}
			if idx < len(paneDefs) {
				pane.RestoreMode = resolvePaneRestoreMode(paneDefs[idx])
			}
			pane.Index = strconv.Itoa(idx)
			if idx == 0 {
				pane.Active = true
			}
			panes = append(panes, pane)
			m.pacePaneSpawn(ctx, pane, len(panes), total)
		}
	}
	return panes, nil
}

func (m *Manager) buildSplitPanes(ctx context.Context, path string, defs []layout.PaneDef, env []string) ([]*Pane, error) {
	var panes []*Pane
	total := len(defs)
	for i, paneDef := range defs {
		pane, err := m.createPane(ctx, path, paneDef.Title, paneDef.Cmd, env)
		if err != nil {
			m.closePanes(panes)
			return nil, err
		}
		pane.RestoreMode = resolvePaneRestoreMode(paneDef)
		pane.Index = strconv.Itoa(i)
		if i == 0 {
			pane.Active = true
		}
		panes = append(panes, pane)
		m.pacePaneSpawn(ctx, pane, len(panes), total)
	}
	return panes, nil
}

func (m *Manager) pacePaneSpawn(ctx context.Context, pane *Pane, idx, total int) {
	cfg := resolvePaneSpawnPace()
	if cfg.threshold == 0 || total < cfg.threshold {
		return
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	if pane != nil && cfg.waitOutput > 0 {
		_ = m.waitForPaneOutput(pane.ID, cfg.waitOutput)
	}
	if cfg.spacing > 0 && idx < total {
		timer := time.NewTimer(cfg.spacing)
		defer timer.Stop()
		if ctx != nil {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		} else {
			<-timer.C
		}
	}
}

func (m *Manager) createPane(ctx context.Context, path, title, command string, env []string) (*Pane, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	id := m.nextPaneID()
	opts := terminal.Options{
		ID:    id,
		Title: strings.TrimSpace(title),
		Dir:   strings.TrimSpace(path),
		Env:   env,
	}
	output := newOutputLog(0)
	opts.OnToast = func(message string) {
		m.notifyToast(id, message)
	}
	opts.OnOutput = func(payload []byte) {
		if output != nil {
			output.append(payload)
		}
	}
	opts.OnFirstRead = func() {
		m.markPaneOutputReady(id)
	}
	startCommand := strings.TrimSpace(command)
	reg := m.toolRegistryRef()
	if reg == nil {
		return nil, errors.New("native: tool registry unavailable")
	}
	toolID := reg.DetectFromCommand(startCommand)
	if startCommand == "" {
		opts.Command = ""
	} else {
		cmd, args, err := splitCommand(startCommand)
		if err != nil {
			return nil, fmt.Errorf("native: parse command %q: %w", startCommand, err)
		}
		opts.Command = cmd
		opts.Args = args
	}
	win, err := newWindow(opts)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			_ = win.Close()
			return nil, ctx.Err()
		default:
		}
	}
	pane := &Pane{
		ID:           id,
		Title:        strings.TrimSpace(title),
		Command:      startCommand,
		StartCommand: startCommand,
		Tool:         toolID,
		Background:   limits.PaneBackgroundDefault,
		window:       win,
		output:       output,
	}
	pane.SetLastActive(time.Now())
	if win != nil && win.Exited() {
		pane.Dead = true
		pane.DeadStatus = win.ExitStatus()
	}
	if win != nil {
		pane.PID = win.PID()
	}
	return pane, nil
}

func renderPreviewLines(win *terminal.Window, max int) ([]string, bool) {
	if win == nil || max <= 0 {
		return nil, false
	}
	if win.FirstReadAt().IsZero() {
		return nil, false
	}
	lines, ready := win.PreviewPlainLines(max)
	if len(lines) == 0 {
		return nil, ready
	}
	return lines, ready
}

func resolvePaneRestoreMode(def layout.PaneDef) sessionrestore.Mode {
	mode, err := sessionrestore.ParseMode(def.SessionRestore)
	if err == nil {
		return mode
	}
	slog.Warn("native: invalid pane session_restore mode", slog.Any("err", err))
	return sessionrestore.ModeDefault
}

func splitCommand(command string) (string, []string, error) {
	parts, err := shellquote.Split(command)
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 {
		return "", nil, errors.New("empty command")
	}
	return parts[0], parts[1:], nil
}

func validatePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", clean)
	}
	return nil
}
