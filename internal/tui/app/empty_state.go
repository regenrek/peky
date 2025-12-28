package app

import (
	"fmt"
	"strings"
)

func (m *Model) emptyStateMessage() string {
	openKey := ""
	if m != nil && m.keys != nil {
		openKey = keyLabel(m.keys.openProject)
	}
	if strings.TrimSpace(openKey) == "" {
		openKey = "ctrl+o"
	}
	return strings.Join([]string{
		"No sessions found.",
		"",
		"Tips:",
		"  • Run 'peakypanes init' to create a global config",
		"  • Run 'peakypanes start' in a project directory",
		fmt.Sprintf("  • Press %s to open a project (set dashboard.project_roots)", openKey),
	}, "\n")
}
