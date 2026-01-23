package app

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/views"
	"github.com/regenrek/peakypanes/internal/userpath"
)

func (m *Model) viewModel() views.Model {
	columns := collectDashboardColumns(m.data.Projects)
	filteredColumns := m.filteredDashboardColumns(columns)
	selectedProject := dashboardSelectedProject(filteredColumns, m.selection)

	var sidebarSessions []SessionItem
	var sidebarProject *ProjectGroup
	if project := m.selectedProject(); project != nil {
		sidebarProject = project
		sidebarSessions = m.filteredSessions(project.Sessions)
	}

	var previewSession *SessionItem
	if session := m.selectedSession(); session != nil {
		previewSession = m.previewSessionForView(session)
	}

	menu := m.quickReplyMenuState()
	bannerLabel, bannerHint, bannerVisible := m.updateBannerInfo()
	updateDialog := m.updateDialogView()
	logoLabel := "PEKY"
	if updateDialog.CurrentVersion != "" {
		logoLabel = fmt.Sprintf("PEKY v%s", updateDialog.CurrentVersion)
	}
	vm := views.Model{
		Width:                     m.width,
		Height:                    m.height,
		ActiveView:                int(m.state),
		Tab:                       int(m.tab),
		HeaderLine:                singleLine(headerLine(m.headerParts())),
		LogoLabel:                 logoLabel,
		EmptyStateMessage:         m.emptyStateMessage(),
		SplashInfo:                m.splashInfo(),
		Projects:                  toViewProjects(m.data.Projects),
		DashboardColumns:          toViewColumns(filteredColumns),
		DashboardSelectedProject:  selectedProject,
		SidebarProject:            toViewProjectPtr(sidebarProject),
		SidebarSessions:           toViewSessions(sidebarSessions),
		SidebarHidden:             m.sidebarHidden(sidebarProject),
		PreviewSession:            toViewSessionPtr(previewSession),
		SelectionProject:          m.selection.ProjectID,
		SelectionSession:          m.selection.Session,
		SelectionPane:             m.selection.Pane,
		ExpandedSessions:          m.expandedSessions,
		FilterActive:              m.filterActive,
		FilterInput:               m.filterInput,
		QuickReplyInput:           m.quickReplyInput,
		QuickReplyMode:            m.quickReplyModeLabel(),
		QuickReplySelectionActive: m.quickReplyMouseSel.active,
		QuickReplySelectionStart:  m.quickReplyMouseSel.start,
		QuickReplySelectionEnd:    m.quickReplyMouseSel.end,
		QuickReplySuggestions:     toViewQuickReplySuggestions(menu.suggestions),
		QuickReplySelected:        m.quickReplyMenuIndex,
		QuickReplyEnabled:         m.quickReplyEnabled(),
		HardRaw:                   m.hardRaw,
		PaneCursor:                m.shouldShowPaneCursor(),
		ProjectPicker:             m.projectPicker,
		LayoutPicker:              m.layoutPicker,
		PaneSwapPicker:            m.paneSwapPicker,
		CommandPalette:            m.commandPalette,
		SettingsMenu:              m.settingsMenu,
		PerformanceMenu:           m.perfMenu,
		DebugMenu:                 m.debugMenu,
		AuthDialog: views.AuthDialog{
			Title:  m.authDialogTitle,
			Body:   m.authDialogBody,
			Input:  m.authDialogInput,
			Footer: m.authDialogFooter,
		},
		PekyDialogTitle:    m.pekyDialogTitle,
		PekyDialogFooter:   m.pekyDialogFooter,
		PekyDialogViewport: m.pekyViewport,
		PekyDialogIsError:  m.pekyDialogIsError,
		PekyPromptLine:     singleLine(m.pekyPromptLine),
		ConfirmKill: views.ConfirmKill{
			Session: m.confirmSession,
			Project: m.confirmProject,
		},
		ConfirmQuit: views.ConfirmQuit{
			RunningPanes: m.confirmQuitRunning,
		},
		ConfirmCloseProject: views.ConfirmCloseProject{
			Project:         m.confirmClose,
			RunningSessions: runningSessionsForProject(m.data.Projects, m.confirmCloseID),
		},
		ConfirmCloseAllProjects: views.ConfirmCloseAllProjects{
			ProjectCount:    len(m.data.Projects),
			RunningSessions: runningSessionsCount(m.data.Projects),
		},
		ConfirmClosePane: views.ConfirmClosePane{
			Title:   m.confirmPaneTitle,
			Session: m.confirmPaneSession,
			Running: m.confirmPaneRunning,
		},
		Rename: views.Rename{
			IsPane:    m.state == StateRenamePane,
			Session:   m.renameSession,
			Pane:      m.renamePane,
			PaneIndex: m.renamePaneIndex,
			Input:     m.renameInput,
		},
		ProjectRootInput:      m.projectRootInput,
		Keys:                  buildKeyHints(m.keys, m.quickReplyEnabled()),
		Toast:                 m.toastText(),
		PreviewCompact:        m.settings.PreviewCompact,
		DashboardPreviewLines: dashboardPreviewLines(m.settings),
		PaneTopbarEnabled:     m.settings.PaneTopbar.Enabled,
		PaneTopbarSpinner:     m.paneTopbarSpinnerFrame(),
		PaneView:              m.paneViewProvider(),
		DialogHelp:            m.dialogHelpView(),
		Resize:                m.resizeOverlayView(),
		ContextMenu:           m.contextMenuView(),
		ServerStatus:          m.serverStatus(),
		UpdateBanner: views.UpdateBanner{
			Visible: bannerVisible,
			Label:   bannerLabel,
			Hint:    bannerHint,
		},
		UpdateDialog: views.UpdateDialog{
			CurrentVersion: updateDialog.CurrentVersion,
			LatestVersion:  updateDialog.LatestVersion,
			Channel:        string(updateDialog.Channel),
			Command:        updateDialog.Command,
			CanInstall:     updateDialog.CanInstall,
			RemindLabel:    updateDialog.RemindLabel,
		},
		UpdateProgress: views.UpdateProgress{
			Step:    m.updateProgress.Step,
			Percent: m.updateProgress.Percent,
		},
	}

	return vm
}

func (m *Model) serverStatus() string {
	if m == nil {
		return "up"
	}
	if m.daemonDisconnected {
		return "down"
	}
	if m.restartNoticePending {
		return "restored"
	}
	return "up"
}

func (m *Model) previewSessionForView(session *SessionItem) *SessionItem {
	if m == nil || session == nil {
		return session
	}
	engine := m.layoutEngineFor(session.Name)
	if engine == nil || engine.Tree == nil {
		return session
	}
	rects := engine.Tree.ViewRects()
	if len(rects) == 0 {
		return session
	}
	out := *session
	out.Panes = append([]PaneItem(nil), session.Panes...)
	for i := range out.Panes {
		rect, ok := rects[out.Panes[i].ID]
		if ok {
			out.Panes[i].Left = rect.X
			out.Panes[i].Top = rect.Y
			out.Panes[i].Width = rect.W
			out.Panes[i].Height = rect.H
			continue
		}
		if engine.Tree.ZoomedPaneID != "" {
			out.Panes[i].Left = 0
			out.Panes[i].Top = 0
			out.Panes[i].Width = 0
			out.Panes[i].Height = 0
		}
	}
	return &out
}

func toViewQuickReplySuggestions(entries []quickReplySuggestion) []views.QuickReplySuggestion {
	if len(entries) == 0 {
		return nil
	}
	out := make([]views.QuickReplySuggestion, len(entries))
	for i, entry := range entries {
		out[i] = views.QuickReplySuggestion{
			Text:         entry.Text,
			MatchLen:     entry.MatchLen,
			MatchIndexes: entry.MatchIndexes,
			Desc:         entry.Desc,
		}
	}
	return out
}

func (m *Model) paneViewProvider() func(id string, width, height int, showCursor bool) string {
	if m == nil || m.client == nil {
		return nil
	}
	return func(id string, width, height int, showCursor bool) string {
		if strings.TrimSpace(id) == "" || width <= 0 || height <= 0 {
			return ""
		}
		if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
			return ""
		}
		if pane := m.paneByID(id); pane != nil && pane.Disconnected {
			return m.offlinePaneView(pane, width, height)
		}
		return m.paneViewWithFallback(id, width, height, showCursor)
	}
}

func runningSessionsForProject(projects []ProjectGroup, projectID string) int {
	if strings.TrimSpace(projectID) == "" {
		return 0
	}
	project := findProjectByID(projects, projectID)
	if project == nil {
		return 0
	}
	running := 0
	for _, s := range project.Sessions {
		if s.Status != StatusStopped {
			running++
		}
	}
	return running
}

func runningSessionsCount(projects []ProjectGroup) int {
	running := 0
	for _, project := range projects {
		for _, session := range project.Sessions {
			if session.Status != StatusStopped {
				running++
			}
		}
	}
	return running
}

func buildKeyHints(keys *dashboardKeyMap, quickReplyEnabled bool) views.KeyHints {
	if keys == nil {
		return views.KeyHints{}
	}
	focusAction := ""
	if quickReplyEnabled {
		focusAction = keyLabel(keys.focusAction)
	}
	return views.KeyHints{
		ProjectKeys:     joinKeyLabels(keys.projectLeft, keys.projectRight),
		SessionKeys:     joinKeyLabels(keys.sessionUp, keys.sessionDown),
		SessionOnlyKeys: joinKeyLabels(keys.sessionOnlyUp, keys.sessionOnlyDown),
		PaneKeys:        joinKeyLabels(keys.panePrev, keys.paneNext),
		ToggleLastPane:  keyLabel(keys.toggleLastPane),
		FocusAction:     focusAction,
		OpenProject:     keyLabel(keys.openProject),
		CloseProject:    keyLabel(keys.closeProject),
		NewSession:      keyLabel(keys.newSession),
		KillSession:     keyLabel(keys.kill),
		TogglePanes:     keyLabel(keys.togglePanes),
		ToggleSidebar:   keyLabel(keys.toggleSidebar),
		HardRaw:         keyLabel(keys.hardRaw),
		ResizeMode:      keyLabel(keys.resizeMode),
		Scrollback:      keyLabel(keys.scrollback),
		CopyMode:        keyLabel(keys.copyMode),
		Refresh:         keyLabel(keys.refresh),
		EditConfig:      keyLabel(keys.editConfig),
		CommandPalette:  keyLabel(keys.commandPalette),
		Filter:          keyLabel(keys.filter),
		Help:            keyLabel(keys.help),
		Quit:            keyLabel(keys.quit),
	}
}

func toViewProjects(projects []ProjectGroup) []views.Project {
	out := make([]views.Project, 0, len(projects))
	for _, project := range projects {
		out = append(out, toViewProject(project))
	}
	return out
}

func toViewProjectPtr(project *ProjectGroup) *views.Project {
	if project == nil {
		return nil
	}
	value := toViewProject(*project)
	return &value
}

func toViewProject(project ProjectGroup) views.Project {
	return views.Project{
		Name:     project.Name,
		Path:     displayPath(project.Path),
		Sessions: toViewSessions(project.Sessions),
	}
}

func toViewSessions(sessions []SessionItem) []views.Session {
	out := make([]views.Session, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, toViewSession(session))
	}
	return out
}

func toViewSessionPtr(session *SessionItem) *views.Session {
	if session == nil {
		return nil
	}
	value := toViewSession(*session)
	return &value
}

func toViewSession(session SessionItem) views.Session {
	activePane := session.ActivePane
	if activePane == "" {
		activePane = activePaneIndex(session.Panes)
	}
	return views.Session{
		Name:       session.Name,
		Status:     int(session.Status),
		PaneCount:  session.PaneCount,
		ActivePane: activePane,
		Panes:      toViewPanes(session.Panes),
	}
}

func toViewPanes(panes []PaneItem) []views.Pane {
	out := make([]views.Pane, 0, len(panes))
	for _, pane := range panes {
		out = append(out, toViewPane(pane))
	}
	return out
}

func toViewPane(pane PaneItem) views.Pane {
	return views.Pane{
		ID:           pane.ID,
		Index:        pane.Index,
		Title:        pane.Title,
		Command:      pane.Command,
		Cwd:          displayPath(pane.Cwd),
		GitRoot:      displayPath(pane.GitRoot),
		GitBranch:    pane.GitBranch,
		GitDirty:     pane.GitDirty,
		GitWorktree:  pane.GitWorktree,
		Tool:         pane.Tool,
		AgentTool:    pane.AgentTool,
		AgentState:   pane.AgentState,
		AgentUpdated: pane.AgentUpdated,
		AgentUnread:  pane.AgentUnread,
		Active:       pane.Active,
		Left:         pane.Left,
		Top:          pane.Top,
		Width:        pane.Width,
		Height:       pane.Height,
		Preview:      pane.Preview,
		Status:       int(pane.Status),
		SummaryLine:  paneSummaryLine(pane, 0),
	}
}

func toViewColumns(columns []DashboardProjectColumn) []views.DashboardColumn {
	out := make([]views.DashboardColumn, 0, len(columns))
	for _, column := range columns {
		viewColumn := views.DashboardColumn{
			ProjectID:   column.ProjectID,
			ProjectName: column.ProjectName,
			ProjectPath: displayPath(column.ProjectPath),
			Panes:       make([]views.DashboardPane, 0, len(column.Panes)),
		}
		for _, pane := range column.Panes {
			viewColumn.Panes = append(viewColumn.Panes, views.DashboardPane{
				ProjectID:   pane.ProjectID,
				ProjectName: pane.ProjectName,
				ProjectPath: displayPath(pane.ProjectPath),
				SessionName: pane.SessionName,
				Pane:        toViewPane(pane.Pane),
			})
		}
		out = append(out, viewColumn)
	}
	return out
}

func displayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return userpath.ShortenUser(path)
}
