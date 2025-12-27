package tmuxstream

import (
	"context"
	"strings"
	"time"
)

func (m *Manager) startPane(ctx context.Context, spec PaneSpec) error {
	id := strings.TrimSpace(spec.PaneID)
	if id == "" {
		return nil
	}

	ps := &paneState{id: id, lastTouch: time.Now()}
	ps.resetTerm(spec.Cols, spec.Rows)

	m.mu.Lock()
	if _, exists := m.panes[id]; exists {
		m.mu.Unlock()
		return nil
	}
	m.panes[id] = ps
	m.mu.Unlock()

	{
		snapCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
		lines, err := m.tmux.CapturePaneLines(snapCtx, id, maxInt(1, ps.rows))
		cancel()
		if err == nil && len(lines) > 0 {
			ps.applySnapshot(lines)
		}
	}

	{
		pipeCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		cmd, _, err := m.tmux.PanePipe(pipeCtx, id)
		cancel()
		if err == nil && strings.TrimSpace(cmd) != "" {
			return nil
		}
	}

	pipeCmd := m.buildPipeCommand(id)
	{
		pipeCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
		err := m.tmux.PipePane(pipeCtx, id, pipeCmd)
		cancel()
		if err != nil {
			return err
		}
	}

	ps.pipeCommandStr = pipeCmd
	ps.pipeByUs = true
	return nil
}

func (m *Manager) resyncPane(ctx context.Context, spec PaneSpec) error {
	id := strings.TrimSpace(spec.PaneID)
	if id == "" {
		return nil
	}

	m.mu.Lock()
	ps := m.panes[id]
	m.mu.Unlock()
	if ps == nil {
		return nil
	}

	ps.resetTerm(spec.Cols, spec.Rows)

	snapCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	lines, err := m.tmux.CapturePaneLines(snapCtx, id, maxInt(1, ps.rows))
	cancel()
	if err == nil && len(lines) > 0 {
		ps.applySnapshot(lines)
	}

	m.notify()
	return nil
}

func (m *Manager) stopPane(ctx context.Context, paneID string, _ bool) error {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil
	}

	m.mu.Lock()
	ps := m.panes[paneID]
	if ps == nil {
		m.mu.Unlock()
		return nil
	}
	delete(m.panes, paneID)
	m.mu.Unlock()

	ps.streamMu.Lock()
	if ps.stream != nil {
		_ = ps.stream.Close()
		ps.stream = nil
	}
	ps.streamMu.Unlock()

	if !ps.pipeByUs {
		return nil
	}

	checkCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	cmd, _, err := m.tmux.PanePipe(checkCtx, paneID)
	cancel()
	if err == nil && strings.TrimSpace(cmd) != "" && cmd != ps.pipeCommandStr {
		return nil
	}

	unpipeCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	return m.tmux.PipePane(unpipeCtx, paneID, "")
}

func (m *Manager) buildPipeCommand(paneID string) string {
	parts := []string{
		shellQuote(m.exe),
		"pipe",
		"--socket", shellQuote(m.socketPath),
		"--token", shellQuote(m.token),
		"--pane-id", shellQuote(paneID),
	}
	return strings.Join(parts, " ")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
