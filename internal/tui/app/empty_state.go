package app

import (
	"fmt"
	"strings"
)

func (m *Model) emptyStateMessage() string {
	info := m.splashInfo()
	if strings.TrimSpace(info) == "" {
		info = "Hit ctrl+o to open a project and ctrl+g for help."
	}
	return strings.Join([]string{
		"No sessions found.",
		"",
		info,
	}, "\n")
}

func (m *Model) splashInfo() string {
	openKey := ""
	helpKey := ""
	if m != nil && m.keys != nil {
		openKey = keyLabel(m.keys.openProject)
		helpKey = keyLabel(m.keys.help)
	}
	if strings.TrimSpace(openKey) == "" {
		openKey = "ctrl+o"
	}
	if strings.TrimSpace(helpKey) == "" {
		helpKey = "ctrl+g"
	}
	return fmt.Sprintf("Hit %s to open a project and %s for help.", openKey, helpKey)
}
