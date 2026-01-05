package views

// Render is the entry point for rendering the active TUI view.
func Render(m Model) string {
	if render := viewRenderers[m.ActiveView]; render != nil {
		return render(m)
	}
	return m.viewDashboard()
}

var viewRenderers = map[int]func(Model) string{
	viewDashboard:               func(m Model) string { return m.viewDashboard() },
	viewProjectPicker:           func(m Model) string { return appStyle.Render(m.ProjectPicker.View()) },
	viewLayoutPicker:            func(m Model) string { return m.viewLayoutPicker() },
	viewPaneSplitPicker:         func(m Model) string { return m.viewPaneSplitPicker() },
	viewPaneSwapPicker:          func(m Model) string { return m.viewPaneSwapPicker() },
	viewConfirmKill:             func(m Model) string { return m.viewConfirmKill() },
	viewConfirmQuit:             func(m Model) string { return m.viewConfirmQuit() },
	viewConfirmCloseProject:     func(m Model) string { return m.viewConfirmCloseProject() },
	viewConfirmCloseAllProjects: func(m Model) string { return m.viewConfirmCloseAllProjects() },
	viewConfirmClosePane:        func(m Model) string { return m.viewConfirmClosePane() },
	viewConfirmRestart:          func(m Model) string { return m.viewConfirmRestart() },
	viewHelp:                    func(m Model) string { return m.viewHelp() },
	viewCommandPalette:          func(m Model) string { return m.viewCommandPalette() },
	viewSettingsMenu:            func(m Model) string { return m.viewSettingsMenu() },
	viewPerformanceMenu:         func(m Model) string { return m.viewPerformanceMenu() },
	viewDebugMenu:               func(m Model) string { return m.viewDebugMenu() },
	viewRenameSession:           func(m Model) string { return m.viewRename() },
	viewRenamePane:              func(m Model) string { return m.viewRename() },
	viewProjectRootSetup:        func(m Model) string { return m.viewProjectRootSetup() },
	viewPekyDialog:              func(m Model) string { return m.viewPekyDialog() },
	viewAuthDialog:              func(m Model) string { return m.viewAuthDialog() },
}
