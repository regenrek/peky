package peakypanes

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/mux"
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
	StateCommandPalette
	StateRenameSession
	StateRenameWindow
	StateProjectRootSetup
)

// Status describes the multiplexer lifecycle state of a session.
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

// SessionItem represents a multiplexer session in the dashboard.
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

// WindowItem represents a multiplexer window/tab.
type WindowItem struct {
	Index  string
	Name   string
	Active bool
	Panes  []PaneItem
}

// PaneItem represents a multiplexer pane with preview content.
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
	ProjectRoots    []string
}

// selectionState tracks the current selection by name/index.
type selectionState struct {
	Project string
	Session string
	Window  string
}

// muxSnapshotInput carries the state needed for refresh.
type muxSnapshotInput struct {
	Selection selectionState
	Version   uint64
	Config    *layout.Config
	Settings  DashboardConfig
}

// muxSnapshotResult is returned by a refresh.
type muxSnapshotResult struct {
	Data      DashboardData
	Resolved  selectionState
	Err       error
	Warning   string
	RawConfig *layout.Config
	Settings  DashboardConfig
	Version   uint64
}

// muxSnapshotMsg is sent back to the model.
type muxSnapshotMsg struct {
	Result muxSnapshotResult
}

// refreshTickMsg triggers the next refresh.
type refreshTickMsg struct{}

// selectionRefreshMsg triggers a debounced refresh for selection changes.
type selectionRefreshMsg struct {
	Version uint64
}

// exitAfterAttachMsg exits the dashboard after an attach returns.
type exitAfterAttachMsg struct{}

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

// CommandItem represents a selectable command in the palette.
type CommandItem struct {
	Label string
	Desc  string
	Run   func(*Model) tea.Cmd
}

func (c CommandItem) Title() string       { return c.Label }
func (c CommandItem) Description() string { return c.Desc }
func (c CommandItem) FilterValue() string { return strings.ToLower(c.Label + " " + c.Desc) }

// paneFromMux converts pane info to dashboard pane item.
func paneFromMux(p mux.PaneInfo) PaneItem {
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
