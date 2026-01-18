package views

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"time"
)

// Model supplies all view state needed to render the TUI. ActiveView and Tab
// values must match the ordering of the app package enums.
type Model struct {
	Width                     int
	Height                    int
	ActiveView                int
	Tab                       int
	HeaderLine                string
	EmptyStateMessage         string
	SplashInfo                string
	Projects                  []Project
	DashboardColumns          []DashboardColumn
	DashboardSelectedProject  string
	SidebarProject            *Project
	SidebarSessions           []Session
	SidebarHidden             bool
	PreviewSession            *Session
	SelectionProject          string
	SelectionSession          string
	SelectionPane             string
	ExpandedSessions          map[string]bool
	FilterActive              bool
	FilterInput               textinput.Model
	QuickReplyInput           textinput.Model
	QuickReplyMode            string
	QuickReplySelectionActive bool
	QuickReplySelectionStart  int
	QuickReplySelectionEnd    int
	QuickReplySuggestions     []QuickReplySuggestion
	QuickReplySelected        int
	QuickReplyEnabled         bool
	HardRaw                   bool
	PaneCursor                bool
	ProjectPicker             list.Model
	LayoutPicker              list.Model
	PaneSwapPicker            list.Model
	CommandPalette            list.Model
	SettingsMenu              list.Model
	PerformanceMenu           list.Model
	DebugMenu                 list.Model
	AuthDialog                AuthDialog
	PekyDialogTitle           string
	PekyDialogFooter          string
	PekyDialogViewport        viewport.Model
	PekyDialogIsError         bool
	PekyPromptLine            string
	ConfirmKill               ConfirmKill
	ConfirmQuit               ConfirmQuit
	ConfirmCloseProject       ConfirmCloseProject
	ConfirmCloseAllProjects   ConfirmCloseAllProjects
	ConfirmClosePane          ConfirmClosePane
	Rename                    Rename
	ProjectRootInput          textinput.Model
	Keys                      KeyHints
	Toast                     string
	PreviewCompact            bool
	DashboardPreviewLines     int
	PaneTopbarEnabled         bool
	PaneTopbarSpinner         string
	PaneView                  func(id string, width, height int, showCursor bool) string
	DialogHelp                DialogHelp
	Resize                    ResizeOverlay
	ContextMenu               ContextMenu
	ServerStatus              string
}

type Project struct {
	Name     string
	Path     string
	Sessions []Session
}

type DialogHelp struct {
	Line  string
	Title string
	Body  string
	Open  bool
}

type ResizeOverlay struct {
	Active      bool
	Dragging    bool
	SnapEnabled bool
	SnapActive  bool
	EdgeLabel   string
	ModeKey     string
	Guides      []ResizeGuide
	Label       string
	LabelX      int
	LabelY      int
}

type ResizeGuide struct {
	X      int
	Y      int
	W      int
	H      int
	Active bool
}

type ContextMenu struct {
	Open     bool
	X        int
	Y        int
	Items    []ContextMenuItem
	Selected int
}

type ContextMenuItem struct {
	Label   string
	Enabled bool
}

type Session struct {
	Name       string
	Status     int
	PaneCount  int
	ActivePane string
	Panes      []Pane
}

type Pane struct {
	ID           string
	Index        string
	Title        string
	Command      string
	Cwd          string
	GitRoot      string
	GitBranch    string
	GitDirty     bool
	GitWorktree  bool
	Tool         string
	AgentTool    string
	AgentState   string // running | idle
	AgentUpdated time.Time
	AgentUnread  bool
	Active       bool
	Left         int
	Top          int
	Width        int
	Height       int
	Preview      []string
	Status       int
	SummaryLine  string
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
	ToggleLastPane  string
	FocusAction     string
	OpenProject     string
	CloseProject    string
	NewSession      string
	KillSession     string
	TogglePanes     string
	ToggleSidebar   string
	HardRaw         string
	ResizeMode      string
	Scrollback      string
	CopyMode        string
	Refresh         string
	EditConfig      string
	CommandPalette  string
	Filter          string
	Help            string
	Quit            string
}

type QuickReplySuggestion struct {
	Text         string
	MatchLen     int
	MatchIndexes []int
	Desc         string
}

type ConfirmKill struct {
	Session string
	Project string
}

type ConfirmQuit struct {
	RunningPanes int
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

type AuthDialog struct {
	Title  string
	Body   string
	Input  textinput.Model
	Footer string
}
