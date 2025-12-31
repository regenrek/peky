package native

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// SignalPane sends a signal to a pane process.
func (m *Manager) SignalPane(paneID string, signalName string) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return errors.New("native: pane id is required")
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil || pane.window == nil {
		return fmt.Errorf("native: pane %q not found", paneID)
	}
	pid := pane.window.PID()
	if pid == 0 {
		return fmt.Errorf("native: pane %q has no process", paneID)
	}
	sig, err := signalFromName(signalName)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("native: find process %d: %w", pid, err)
	}
	if err := proc.Signal(sig); err != nil {
		return fmt.Errorf("native: signal %d: %w", pid, err)
	}
	return nil
}
