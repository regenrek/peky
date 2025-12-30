package app

import (
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// ViewState represents the current UI view.
type ViewState int

const (
	StateDashboard ViewState = iota
	StateProjectPicker
	StateLayoutPicker
	StatePaneSplitPicker
	StatePaneSwapPicker
	StateConfirmKill
	StateConfirmCloseProject
	StateConfirmCloseAllProjects
	StateConfirmClosePane
	StateConfirmRestart
	StateHelp
	StateCommandPalette
	StateRenameSession
	StateRenamePane
	StateProjectRootSetup
	StateSettingsMenu
	StateDebugMenu
)

// DashboardTab represents the active tab within the dashboard view.
type DashboardTab int

const (
	TabDashboard DashboardTab = iota
	TabProject
)

// Status describes the lifecycle state of a session.
type Status int

const (
	StatusStopped Status = iota
	StatusRunning
	StatusCurrent
)

// PaneStatus represents the activity/status of a pane.
type PaneStatus int

const (
	PaneStatusIdle PaneStatus = iota
	PaneStatusRunning
	PaneStatusDone
	PaneStatusError
)

const (
	AttachBehaviorCurrent  = "current"
	AttachBehaviorDetached = "detached"
)

const (
	PaneNavigationSpatial = "spatial"
	PaneNavigationMemory  = "memory"
)

// DashboardData contains all data required to render the dashboard.
type DashboardData struct {
	Projects    []ProjectGroup
	RefreshedAt time.Time
}

// ProjectGroup represents a project grouping with sessions.
type ProjectGroup struct {
	ID         string
	Name       string
	Path       string
	FromConfig bool
	Sessions   []SessionItem
}

// SessionItem represents a session in the dashboard.
type SessionItem struct {
	Name       string
	Path       string
	LayoutName string
	Status     Status
	PaneCount  int
	ActivePane string
	Panes      []PaneItem
	Config     *layout.ProjectConfig
}

// DashboardPane represents a pane with project metadata for the dashboard.
type DashboardPane struct {
	ProjectID   string
	ProjectName string
	ProjectPath string
	SessionName string
	Pane        PaneItem
}

// DashboardProjectColumn represents a dashboard column for a project.
type DashboardProjectColumn struct {
	ProjectID   string
	ProjectName string
	ProjectPath string
	Panes       []DashboardPane
}

// PaneItem represents a pane with preview content.
type PaneItem struct {
	ID            string
	Index         string
	Title         string
	Command       string
	StartCommand  string
	PID           int
	Active        bool
	Left          int
	Top           int
	Width         int
	Height        int
	Dead          bool
	DeadStatus    int
	RestoreFailed bool
	RestoreError  string
	LastActive    time.Time
	Preview       []string
	Status        PaneStatus
}

// AgentDetectionConfig enables agent-specific status detection.
type AgentDetectionConfig struct {
	Codex  bool
	Claude bool
}

// DashboardConfig wraps dashboard settings after defaults applied.
type DashboardConfig struct {
	RefreshInterval    time.Duration
	PreviewLines       int
	PreviewCompact     bool
	IdleThreshold      time.Duration
	StatusMatcher      statusMatcher
	PreviewMode        string
	SidebarHidden      bool
	ProjectRoots       []string
	AgentDetection     AgentDetectionConfig
	AttachBehavior     string
	PaneNavigationMode string
	HiddenProjects     map[string]struct{}
}

// selectionState tracks the current selection by stable project ID.
type selectionState struct {
	ProjectID string
	Session   string
	Pane      string
}

// dashboardSnapshotInput carries the state needed for refresh.
type dashboardSnapshotInput struct {
	Selection  selectionState
	Tab        DashboardTab
	Version    uint64
	RefreshSeq uint64
	Config     *layout.Config
	Settings   DashboardConfig
	Sessions   []native.SessionSnapshot
}

// dashboardSnapshotResult is returned by a refresh.
type dashboardSnapshotResult struct {
	Data       DashboardData
	Resolved   selectionState
	Err        error
	Warning    string
	RawConfig  *layout.Config
	Settings   DashboardConfig
	Keymap     *dashboardKeyMap
	Version    uint64
	RefreshSeq uint64
}

// dashboardSnapshotMsg is sent back to the model.
type dashboardSnapshotMsg struct {
	Result dashboardSnapshotResult
}

// refreshTickMsg triggers the next refresh.
type refreshTickMsg struct{}

// selectionRefreshMsg triggers a debounced refresh for selection changes.
type selectionRefreshMsg struct {
	Version uint64
}

// daemonEventMsg wraps an async daemon event.
type daemonEventMsg struct {
	Event sessiond.Event
}

// paneViewsMsg carries pane view updates from the daemon.
type paneViewsMsg struct {
	Views []sessiond.PaneViewResponse
	Err   error
}

type daemonRestartMsg struct {
	Client         *sessiond.Client
	PaneViewClient *sessiond.Client
	Err            error
}

// PaneClosedMsg signals the selected pane can no longer accept input.
type PaneClosedMsg struct {
	PaneID  string
	Message string
}

// sessionStartedMsg signals a session creation result.
type sessionStartedMsg struct {
	Name  string
	Path  string
	Err   error
	Focus bool
}

// AutoStartSpec instructs the TUI to start a session on launch.
type AutoStartSpec struct {
	Session string
	Path    string
	Layout  string
	Focus   bool
}

// PaneSwapChoice represents a target pane for swapping.
type PaneSwapChoice struct {
	Label     string
	Desc      string
	PaneIndex string
}

func (p PaneSwapChoice) Title() string       { return p.Label }
func (p PaneSwapChoice) Description() string { return p.Desc }
func (p PaneSwapChoice) FilterValue() string { return strings.ToLower(p.Label + " " + p.Desc) }
