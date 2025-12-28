package app

func (m *Model) sessionExpanded(name string) bool {
	if m == nil || m.expandedSessions == nil {
		return true
	}
	expanded, ok := m.expandedSessions[name]
	if !ok {
		return true
	}
	return expanded
}
