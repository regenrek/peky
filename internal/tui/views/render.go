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
	case viewConfirmClosePane:
		return m.viewConfirmClosePane()
	case viewHelp:
		return m.viewHelp()
	case viewCommandPalette:
		return m.viewCommandPalette()
	case viewRenameSession, viewRenamePane:
		return m.viewRename()
	case viewProjectRootSetup:
		return m.viewProjectRootSetup()
	default:
		return m.viewDashboard()
	}
}
