package peakypanes

import (
	"time"

	"github.com/kregenrek/tmuxman/internal/layout"
	"github.com/kregenrek/tmuxman/internal/tmuxctl"
)

// ViewState represents the current UI view.
type ViewState int

const (
	StateDashboard ViewState = iota
	StateProjectPicker
	StateLayoutPicker
	StateConfirmKill
	StateConfirmCloseProject
	StateHelp
)

// Status describes the tmux lifecycle state of a session.
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

// SessionItem represents a tmux session in the dashboard.
type SessionItem struct {
	Name         string
	Path         string
	LayoutName   string
	Status       Status
	WindowCount  int
	ActiveWindow string
	Windows      []WindowItem
	Thumbnail    PaneSummary
	Config       *layout.ProjectConfig
}

// WindowItem represents a tmux window.
type WindowItem struct {
	Index  string
	Name   string
	Active bool
	Panes  []PaneItem
}

// PaneItem represents a tmux pane with preview content.
type PaneItem struct {
	ID         string
	Index      string
	Title      string
	Command    string
	Active     bool
	Left       int
	Top        int
	Width      int
	Height     int
	Dead       bool
	DeadStatus int
	LastActive time.Time
	Preview    []string
	Status     PaneStatus
}

// PaneSummary holds lightweight preview info for thumbnails.
type PaneSummary struct {
	Line   string
	Status PaneStatus
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
}

// selectionState tracks the current selection by name/index.
type selectionState struct {
	Project string
	Session string
	Window  string
}

// tmuxSnapshotInput carries the state needed for refresh.
type tmuxSnapshotInput struct {
	Selection selectionState
	Config    *layout.Config
	Settings  DashboardConfig
}

// tmuxSnapshotResult is returned by a refresh.
type tmuxSnapshotResult struct {
	Data      DashboardData
	Resolved  selectionState
	Err       error
	Warning   string
	RawConfig *layout.Config
	Settings  DashboardConfig
}

// tmuxSnapshotMsg is sent back to the model.
type tmuxSnapshotMsg struct {
	Result tmuxSnapshotResult
}

// refreshTickMsg triggers the next refresh.
type refreshTickMsg struct{}

// GitProject represents a project directory with .git.
type GitProject struct {
	Name string
	Path string
}

func (g GitProject) Title() string       { return "📁 " + g.Name }
func (g GitProject) Description() string { return shortenPath(g.Path) }
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

// paneFromTmux converts tmux pane info to dashboard pane item.
func paneFromTmux(p tmuxctl.PaneInfo) PaneItem {
	return PaneItem{
		ID:         p.ID,
		Index:      p.Index,
		Title:      p.Title,
		Command:    p.Command,
		Active:     p.Active,
		Left:       p.Left,
		Top:        p.Top,
		Width:      p.Width,
		Height:     p.Height,
		Dead:       p.Dead,
		DeadStatus: p.DeadStatus,
		LastActive: p.LastActive,
	}
}
