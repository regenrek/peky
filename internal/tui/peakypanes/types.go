package peakypanes

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/pathutil"
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
	StateConfirmClosePane
	StateHelp
	StateCommandPalette
	StateRenameSession
	StateRenamePane
	StateProjectRootSetup
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

// DashboardData contains all data required to render the dashboard.
type DashboardData struct {
	Projects    []ProjectGroup
	RefreshedAt time.Time
}

// ProjectGroup represents a project grouping with sessions.
type ProjectGroup struct {
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
	Thumbnail  PaneSummary
	Config     *layout.ProjectConfig
}

// DashboardPane represents a pane with project metadata for the dashboard.
type DashboardPane struct {
	ProjectName string
	ProjectPath string
	SessionName string
	Pane        PaneItem
}

// DashboardProjectColumn represents a dashboard column for a project.
type DashboardProjectColumn struct {
	ProjectName string
	ProjectPath string
	Panes       []DashboardPane
}

// PaneItem represents a pane with preview content.
type PaneItem struct {
	ID           string
	Index        string
	Title        string
	Command      string
	StartCommand string
	PID          int
	Active       bool
	Left         int
	Top          int
	Width        int
	Height       int
	Dead         bool
	DeadStatus   int
	LastActive   time.Time
	Preview      []string
	Status       PaneStatus
}

// PaneSummary holds lightweight preview info for thumbnails.
type PaneSummary struct {
	Line   string
	Status PaneStatus
}

// AgentDetectionConfig enables agent-specific status detection.
type AgentDetectionConfig struct {
	Codex  bool
	Claude bool
}

// DashboardConfig wraps dashboard settings after defaults applied.
type DashboardConfig struct {
	RefreshInterval time.Duration
	PreviewLines    int
	PreviewCompact  bool
	ThumbnailLines  int
	IdleThreshold   time.Duration
	ShowThumbnails  bool
	StatusMatcher   statusMatcher
	PreviewMode     string
	ProjectRoots    []string
	AgentDetection  AgentDetectionConfig
	AttachBehavior  string
	HiddenProjects  map[string]struct{}
}

// selectionState tracks the current selection by name/index.
type selectionState struct {
	Project string
	Session string
	Pane    string
}

// dashboardSnapshotInput carries the state needed for refresh.
type dashboardSnapshotInput struct {
	Selection selectionState
	Tab       DashboardTab
	Version   uint64
	Config    *layout.Config
	Settings  DashboardConfig
	Native    *native.Manager
}

// dashboardSnapshotResult is returned by a refresh.
type dashboardSnapshotResult struct {
	Data      DashboardData
	Resolved  selectionState
	Err       error
	Warning   string
	RawConfig *layout.Config
	Settings  DashboardConfig
	Keymap    *dashboardKeyMap
	Version   uint64
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

// nativePaneUpdatedMsg is emitted when a native pane updates.
type nativePaneUpdatedMsg struct {
	PaneID string
}

// nativeSessionStartedMsg signals a native session creation result.
type nativeSessionStartedMsg struct {
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

// GitProject represents a project directory with .git.
type GitProject struct {
	Name string
	Path string
}

func (g GitProject) Title() string       { return "📁 " + g.Name }
func (g GitProject) Description() string { return pathutil.ShortenUser(g.Path) }
func (g GitProject) FilterValue() string { return g.Name }

// LayoutChoice represents a selectable layout in the picker.
type LayoutChoice struct {
	Label      string
	Desc       string
	LayoutName string
}

func (l LayoutChoice) Title() string       { return l.Label }
func (l LayoutChoice) Description() string { return l.Desc }
func (l LayoutChoice) FilterValue() string { return l.Label }

// PaneSwapChoice represents a target pane for swapping.
type PaneSwapChoice struct {
	Label     string
	Desc      string
	PaneIndex string
}

func (p PaneSwapChoice) Title() string       { return p.Label }
func (p PaneSwapChoice) Description() string { return p.Desc }
func (p PaneSwapChoice) FilterValue() string { return strings.ToLower(p.Label + " " + p.Desc) }

// CommandItem represents a selectable command in the palette.
type CommandItem struct {
	Label string
	Desc  string
	Run   func(*Model) tea.Cmd
}

func (c CommandItem) Title() string       { return c.Label }
func (c CommandItem) Description() string { return c.Desc }
func (c CommandItem) FilterValue() string { return strings.ToLower(c.Label + " " + c.Desc) }
