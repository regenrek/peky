package native

import (
	"errors"

	"github.com/regenrek/peakypanes/internal/tool"
)

// SetToolRegistry updates the tool detection registry.
func (m *Manager) SetToolRegistry(reg *tool.Registry) error {
	if m == nil {
		return errors.New("native: manager is nil")
	}
	if reg == nil {
		return errors.New("native: tool registry is nil")
	}
	m.toolRegistry.Store(reg)
	return nil
}

func (m *Manager) toolRegistryRef() *tool.Registry {
	if m == nil {
		return nil
	}
	if reg, ok := m.toolRegistry.Load().(*tool.Registry); ok && reg != nil {
		return reg
	}
	return nil
}
