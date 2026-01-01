package native

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/terminal"
)

var newWindow = terminal.NewWindow

const (
	defaultPaneSpawnThreshold    = 8
	defaultPaneSpawnSpacingMS    = 25
	defaultPaneSpawnWaitOutputMS = 0
	maxPaneSpawnSpacingMS        = 1000
	maxPaneSpawnWaitOutputMS     = 10000

	paneSpawnThresholdEnv  = "PEAKYPANES_PANE_SPAWN_THRESHOLD"
	paneSpawnSpacingEnv    = "PEAKYPANES_PANE_SPAWN_SPACING_MS"
	paneSpawnWaitOutputEnv = "PEAKYPANES_PANE_SPAWN_WAIT_OUTPUT_MS"
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

	cellW := LayoutBaseSize / grid.Columns
	cellH := LayoutBaseSize / grid.Rows
	remainderW := LayoutBaseSize % grid.Columns
	remainderH := LayoutBaseSize % grid.Rows

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
			left := c * cellW
			top := r * cellH
			width := cellW
			height := cellH
			if c == grid.Columns-1 {
				width += remainderW
			}
			if r == grid.Rows-1 {
				height += remainderH
			}
			pane, err := m.createPane(ctx, path, title, cmd, env)
			if err != nil {
				m.closePanes(panes)
				return nil, err
			}
			pane.Index = strconv.Itoa(idx)
			pane.Left = left
			pane.Top = top
			pane.Width = width
			pane.Height = height
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
	active := (*Pane)(nil)
	total := len(defs)
	for i, paneDef := range defs {
		pane, err := m.createPane(ctx, path, paneDef.Title, paneDef.Cmd, env)
		if err != nil {
			m.closePanes(panes)
			return nil, err
		}
		pane.Index = strconv.Itoa(i)
		if i == 0 {
			pane.Active = true
			pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, LayoutBaseSize, LayoutBaseSize
			active = pane
		} else if active != nil {
			vertical := strings.EqualFold(paneDef.Split, "vertical") || strings.EqualFold(paneDef.Split, "v")
			percent := parsePercent(paneDef.Size)
			oldRect, newRect := splitRect(rectFromPane(active), vertical, percent)
			active.Left, active.Top, active.Width, active.Height = oldRect.x, oldRect.y, oldRect.w, oldRect.h
			pane.Left, pane.Top, pane.Width, pane.Height = newRect.x, newRect.y, newRect.w, newRect.h
		} else {
			pane.Left, pane.Top, pane.Width, pane.Height = 0, 0, LayoutBaseSize, LayoutBaseSize
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
	opts.OnToast = func(message string) {
		m.notifyToast(id, message)
	}
	opts.OnFirstRead = func() {
		m.markPaneOutputReady(id)
	}
	startCommand := strings.TrimSpace(command)
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
		window:       win,
		LastActive:   time.Now(),
	}
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
	view, ready := win.ViewANSICached()
	if !ready {
		win.RequestANSIRender()
	}
	if view == "" {
		return nil, ready
	}
	plain := ansi.Strip(view)
	lines := strings.Split(plain, "\n")
	if len(lines) <= max {
		return lines, ready
	}
	return lines[len(lines)-max:], ready
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
