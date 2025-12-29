package views

// Render is the entry point for rendering the active TUI view.
func Render(m Model) string {
	switch m.ActiveView {
	case viewDashboard:
		return m.viewDashboard()
	case viewProjectPicker:
		return appStyle.Render(m.ProjectPicker.View())
	case viewLayoutPicker:
		return m.viewLayoutPicker()
	case viewPaneSplitPicker:
		return m.viewPaneSplitPicker()
	case viewPaneSwapPicker:
		return m.viewPaneSwapPicker()
	case viewConfirmKill:
		return m.viewConfirmKill()
	case viewConfirmCloseProject:
		return m.viewConfirmCloseProject()
	case viewConfirmCloseAllProjects:
		return m.viewConfirmCloseAllProjects()
	case viewConfirmClosePane:
		return m.viewConfirmClosePane()
	case viewConfirmRestart:
		return m.viewConfirmRestart()
	case viewHelp:
		return m.viewHelp()
	case viewCommandPalette:
		return m.viewCommandPalette()
	case viewSettingsMenu:
		return m.viewSettingsMenu()
	case viewDebugMenu:
		return m.viewDebugMenu()
	case viewRenameSession, viewRenamePane:
		return m.viewRename()
	case viewProjectRootSetup:
		return m.viewProjectRootSetup()
	default:
		return m.viewDashboard()
	}
}
