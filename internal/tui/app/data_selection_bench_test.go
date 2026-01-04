package app

import (
	"fmt"
	"testing"
)

func BenchmarkBuildFocusIndexLarge(b *testing.B) {
	groups := benchmarkGroups(25, 8, 16)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildFocusIndex(groups)
	}
}

func BenchmarkSelectionFromFocusLarge(b *testing.B) {
	groups := benchmarkGroups(25, 8, 16)
	index := buildFocusIndex(groups)
	targetGroup := groups[len(groups)-1]
	targetSession := targetGroup.Sessions[len(targetGroup.Sessions)-1]
	targetPane := targetSession.Panes[len(targetSession.Panes)-1]
	target := targetPane.ID
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = selectionFromFocus(index, "", target)
	}
}

func benchmarkGroups(projects, sessionsPerProject, panesPerSession int) []ProjectGroup {
	groups := make([]ProjectGroup, 0, projects)
	paneCounter := 0
	for p := 0; p < projects; p++ {
		group := ProjectGroup{
			ID:   fmt.Sprintf("proj-%d", p),
			Name: fmt.Sprintf("Proj %d", p),
			Path: fmt.Sprintf("/tmp/proj-%d", p),
		}
		group.Sessions = make([]SessionItem, 0, sessionsPerProject)
		for s := 0; s < sessionsPerProject; s++ {
			session := SessionItem{
				Name: fmt.Sprintf("sess-%d-%d", p, s),
				Path: group.Path,
			}
			session.Panes = make([]PaneItem, 0, panesPerSession)
			for i := 0; i < panesPerSession; i++ {
				paneCounter++
				pane := PaneItem{
					ID:     fmt.Sprintf("p-%d", paneCounter),
					Index:  fmt.Sprintf("%d", i+1),
					Active: i == 0,
				}
				session.Panes = append(session.Panes, pane)
			}
			group.Sessions = append(group.Sessions, session)
		}
		groups = append(groups, group)
	}
	return groups
}
