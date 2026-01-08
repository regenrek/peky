package app

import "github.com/regenrek/peakypanes/internal/layout"

func (m *Model) syncLayoutEngines() {
	if m == nil {
		return
	}
	engines := make(map[string]*layout.Engine)
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			if session.LayoutTree == nil {
				continue
			}
			tree := layout.TreeFromSnapshot(session.LayoutTree)
			if tree == nil {
				continue
			}
			engines[session.Name] = layout.NewEngine(tree)
		}
	}
	m.layoutEngines = engines
	m.layoutEngineVersion++
	m.resize.invalidateCache()
}

func (m *Model) layoutEngineFor(session string) *layout.Engine {
	if m == nil || session == "" {
		return nil
	}
	if m.resize.preview.active && m.resize.preview.session == session && m.resize.preview.engine != nil {
		return m.resize.preview.engine
	}
	return m.layoutEngines[session]
}
