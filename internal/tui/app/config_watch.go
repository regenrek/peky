package app

import (
	"os"
	"path/filepath"
	"time"
)

type projectConfigState struct {
	path    string
	modTime time.Time
	size    int64
	exists  bool
}

func (s projectConfigState) equal(other projectConfigState) bool {
	return s.exists == other.exists &&
		s.size == other.size &&
		s.modTime.Equal(other.modTime) &&
		s.path == other.path
}

func projectConfigStateForPath(projectPath string) projectConfigState {
	if projectPath == "" {
		return projectConfigState{}
	}
	if state, ok := statConfigFile(filepath.Join(projectPath, ".peakypanes.yml")); ok {
		return state
	}
	if state, ok := statConfigFile(filepath.Join(projectPath, ".peakypanes.yaml")); ok {
		return state
	}
	return projectConfigState{}
}

func statConfigFile(path string) (projectConfigState, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return projectConfigState{}, false
	}
	return projectConfigState{
		path:    path,
		modTime: info.ModTime(),
		size:    info.Size(),
		exists:  true,
	}, true
}
