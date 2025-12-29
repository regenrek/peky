package views

// View state ordering must match app.ViewState.
const (
	viewDashboard = iota
	viewProjectPicker
	viewLayoutPicker
	viewPaneSplitPicker
	viewPaneSwapPicker
	viewConfirmKill
	viewConfirmCloseProject
	viewConfirmCloseAllProjects
	viewConfirmClosePane
	viewConfirmRestart
	viewHelp
	viewCommandPalette
	viewRenameSession
	viewRenamePane
	viewProjectRootSetup
)

// Tab ordering must match app.DashboardTab.
const (
	tabDashboard = iota
	tabProject
)

// Session status ordering must match app.Status.
const (
	sessionStopped = iota
	sessionRunning
	sessionCurrent
)

// Pane status ordering must match app.PaneStatus.
const (
	paneStatusIdle = iota
	paneStatusRunning
	paneStatusDone
	paneStatusError
)
