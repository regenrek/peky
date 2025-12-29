package views

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

// Model supplies all view state needed to render the TUI. ActiveView and Tab
// values must match the ordering of the app package enums.
type Model struct {
	Width                    int
	Height                   int
	ActiveView               int
	Tab                      int
	HeaderLine               string
	EmptyStateMessage        string
	SplashInfo               string
	Projects                 []Project
	DashboardColumns         []DashboardColumn
	DashboardSelectedProject string
	SidebarProject           *Project
	SidebarSessions          []Session
	PreviewSession           *Session
	SelectionProject         string
	SelectionSession         string
	SelectionPane            string
	ExpandedSessions         map[string]bool
	FilterActive             bool
	FilterInput              textinput.Model
	QuickReplyInput          textinput.Model
	TerminalFocus            bool
	SupportsTerminalFocus    bool
	ProjectPicker            list.Model
	LayoutPicker             list.Model
	PaneSwapPicker           list.Model
	CommandPalette           list.Model
	ConfirmKill              ConfirmKill
	ConfirmCloseProject      ConfirmCloseProject
	ConfirmCloseAllProjects  ConfirmCloseAllProjects
	ConfirmClosePane         ConfirmClosePane
	Rename                   Rename
	ProjectRootInput         textinput.Model
	Keys                     KeyHints
	Toast                    string
	PreviewCompact           bool
	PreviewMode              string
	DashboardPreviewLines    int
	PaneView                 func(id string, width, height int, showCursor bool) string
}

type Project struct {
	Name     string
	Path     string
	Sessions []Session
}

type Session struct {
	Name       string
	Status     int
	PaneCount  int
	ActivePane string
	Panes      []Pane
}

type Pane struct {
	ID          string
	Index       string
	Title       string
	Command     string
	Active      bool
	Left        int
	Top         int
	Width       int
	Height      int
	Preview     []string
	Status      int
	SummaryLine string
}

type DashboardColumn struct {
	ProjectID   string
	ProjectName string
	ProjectPath string
	Panes       []DashboardPane
}

type DashboardPane struct {
	ProjectID   string
	ProjectName string
	ProjectPath string
	SessionName string
	Pane        Pane
}

type KeyHints struct {
	ProjectKeys     string
	SessionKeys     string
	SessionOnlyKeys string
	PaneKeys        string
	OpenProject     string
	CloseProject    string
	NewSession      string
	KillSession     string
	TogglePanes     string
	TerminalFocus   string
	Scrollback      string
	CopyMode        string
	Refresh         string
	EditConfig      string
	CommandPalette  string
	Filter          string
	Help            string
	Quit            string
}

type ConfirmKill struct {
	Session string
	Project string
}

type ConfirmCloseProject struct {
	Project         string
	RunningSessions int
}

type ConfirmCloseAllProjects struct {
	ProjectCount    int
	RunningSessions int
}

type ConfirmClosePane struct {
	Title   string
	Session string
	Running bool
}

type Rename struct {
	IsPane    bool
	Session   string
	Pane      string
	PaneIndex string
	Input     textinput.Model
}
