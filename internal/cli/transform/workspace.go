package transform

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/workspace"
)

// LoadWorkspace loads workspace data from the default config path.
func LoadWorkspace() (workspace.Workspace, error) {
	path, err := layout.DefaultConfigPath()
	if err != nil {
		return workspace.Workspace{}, fmt.Errorf("resolve config path: %w", err)
	}
	return workspace.ListWorkspace(path)
}
