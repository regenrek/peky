package sessiond

import (
	"time"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/termframe"
	"github.com/regenrek/peakypanes/internal/terminal"
)

// Op identifies the request/response operation.
type Op string

const (
	OpHello             Op = "hello"
	OpSessionNames      Op = "session_names"
	OpSnapshot          Op = "snapshot"
	OpStartSession      Op = "start_session"
	OpKillSession       Op = "kill_session"
	OpRenameSession     Op = "rename_session"
	OpSessionFocus      Op = "session_focus"
	OpRenamePane        Op = "rename_pane"
	OpSplitPane         Op = "split_pane"
	OpClosePane         Op = "close_pane"
	OpSwapPanes         Op = "swap_panes"
	OpSetPaneTool       Op = "set_pane_tool"
	OpSetPaneBackground Op = "set_pane_background"
	OpSendInput         Op = "send_input"
	OpSendInputTool     Op = "send_input_tool"
	OpSendMouse         Op = "send_mouse"
	OpResizePane        Op = "resize_pane"
	OpResetPaneSizes    Op = "reset_pane_sizes"
	OpZoomPane          Op = "zoom_pane"
	OpPaneView          Op = "pane_view"
	OpPaneOutput        Op = "pane_output"
	OpPaneSnapshot      Op = "pane_snapshot"
	OpPaneHistory       Op = "pane_history"
	OpPaneWait          Op = "pane_wait"
	OpPaneTagAdd        Op = "pane_tag_add"
	OpPaneTagRemove     Op = "pane_tag_remove"
	OpPaneTagList       Op = "pane_tag_list"
	OpPaneFocus         Op = "pane_focus"
	OpPaneSignal        Op = "pane_signal"
	OpRelayCreate       Op = "relay_create"
	OpRelayList         Op = "relay_list"
	OpRelayStop         Op = "relay_stop"
	OpRelayStopAll      Op = "relay_stop_all"
	OpEventsReplay      Op = "events_replay"
	OpTerminalAction    Op = "terminal_action"
	OpHandleKey         Op = "handle_key"
)

// EventType identifies async daemon events.
type EventType string

const (
	EventPaneUpdated     EventType = "pane_updated"
	EventPaneMetaChanged EventType = "pane_meta_changed"
	EventSessionChanged  EventType = "session_changed"
	EventToast           EventType = "toast"
	EventFocus           EventType = "focus"
	EventPaneOutput      EventType = "pane_output"
	EventRelay           EventType = "relay"
)

// Event is broadcast from daemon to clients.
type Event struct {
	ID            string
	Type          EventType
	TS            time.Time
	PaneID        string
	PaneUpdateSeq uint64
	Session       string
	Toast         string
	ToastKind     ToastLevel
	Payload       map[string]any
}

// HelloRequest begins a connection handshake.
type HelloRequest struct {
	Version  string
	ClientID string
}

// HelloResponse acknowledges the handshake.
type HelloResponse struct {
	Version string
	PID     int
}

// SessionNamesResponse returns known session names.
type SessionNamesResponse struct {
	Names []string
}

// SnapshotRequest requests a dashboard snapshot.
type SnapshotRequest struct {
	PreviewLines int
}

// PaneGitMeta describes git metadata for a pane's current working directory.
type PaneGitMeta struct {
	Root      string
	Branch    string
	Dirty     bool
	Worktree  bool
	UpdatedAt time.Time
}

// SnapshotResponse returns dashboard snapshots.
type SnapshotResponse struct {
	Version        uint64
	Sessions       []native.SessionSnapshot
	FocusedSession string
	FocusedPaneID  string
	PaneGit        map[string]PaneGitMeta
}

// StartSessionRequest starts a new session.
type StartSessionRequest struct {
	Name       string
	Path       string
	LayoutName string
	PaneCount  int
	Env        []string
}

// StartSessionResponse confirms session creation.
type StartSessionResponse struct {
	Name       string
	Path       string
	LayoutName string
}

// KillSessionRequest stops a session.
type KillSessionRequest struct {
	Name string
}

// RenameSessionRequest updates a session name.
type RenameSessionRequest struct {
	OldName string
	NewName string
}

// RenameSessionResponse confirms rename.
type RenameSessionResponse struct {
	NewName string
}

// FocusSessionRequest updates the focused session in the daemon.
type FocusSessionRequest struct {
	Name string
}

// RenamePaneRequest updates a pane title.
type RenamePaneRequest struct {
	PaneID      string
	SessionName string
	PaneIndex   string
	NewTitle    string
}

// SplitPaneRequest splits a pane.
type SplitPaneRequest struct {
	SessionName string
	PaneIndex   string
	Vertical    bool
	Percent     int
}

// SplitPaneResponse returns the new pane index and ID.
type SplitPaneResponse struct {
	NewIndex  string
	NewPaneID string
}

// ClosePaneRequest closes a pane.
type ClosePaneRequest struct {
	PaneID      string
	SessionName string
	PaneIndex   string
}

// SwapPanesRequest swaps two panes in a session.
type SwapPanesRequest struct {
	SessionName string
	PaneA       string
	PaneB       string
}

// SetPaneToolRequest updates the recorded tool for a pane.
type SetPaneToolRequest struct {
	PaneID string
	Tool   string
}

// SetPaneBackgroundRequest updates the background palette index for a pane.
type SetPaneBackgroundRequest struct {
	PaneID      string
	SessionName string
	PaneIndex   string
	Background  int
}

// SendInputRequest forwards raw input.
type SendInputRequest struct {
	PaneID       string
	Scope        string
	Input        []byte
	RecordAction bool
	Action       string
	Summary      string
}

// SendInputToolRequest forwards input using tool-aware profiles.
type SendInputToolRequest struct {
	PaneID        string
	Scope         string
	Input         []byte
	RecordAction  bool
	Action        string
	Summary       string
	Submit        bool
	SubmitDelayMS *int
	Raw           bool
	ToolFilter    string
	DetectTool    bool
}

// SendInputResult captures a send attempt result.
type SendInputResult struct {
	PaneID  string
	Status  string
	Message string
}

// SendInputResponse returns send results (for scoped sends).
type SendInputResponse struct {
	Results []SendInputResult
}

// MouseAction is a mouse action type.
type MouseAction int

const (
	MouseActionUnknown MouseAction = iota
	MouseActionPress
	MouseActionRelease
	MouseActionMotion
)

// MouseRoute controls how mouse input is routed within a pane.
type MouseRoute = terminal.MouseRoute

const (
	MouseRouteAuto          = terminal.MouseRouteAuto
	MouseRouteApp           = terminal.MouseRouteApp
	MouseRouteHostSelection = terminal.MouseRouteHostSelection
)

// MouseEventPayload is a serializable mouse event.
type MouseEventPayload struct {
	X      int
	Y      int
	Button int
	Action MouseAction
	Shift  bool
	Alt    bool
	Ctrl   bool
	Wheel  bool
	// WheelCount is the number of wheel ticks represented by this payload.
	// When Wheel is true, zero means 1.
	WheelCount int
	Route      MouseRoute
}

// SendMouseRequest forwards a mouse event.
type SendMouseRequest struct {
	PaneID string
	Event  MouseEventPayload
}

type ResizeEdge string

const (
	ResizeEdgeLeft  ResizeEdge = "left"
	ResizeEdgeRight ResizeEdge = "right"
	ResizeEdgeUp    ResizeEdge = "up"
	ResizeEdgeDown  ResizeEdge = "down"
)

type SnapState struct {
	Active bool
	Target int
}

// ResizePaneRequest resizes a split edge for a pane.
type ResizePaneRequest struct {
	SessionName string
	PaneID      string
	Edge        ResizeEdge
	Delta       int
	Snap        bool
	SnapState   SnapState
}

type LayoutOpResponse struct {
	Changed   bool
	Snapped   bool
	SnapState SnapState
	Affected  []string
}

// ResetSizesRequest resets pane sizes.
type ResetSizesRequest struct {
	SessionName string
	PaneID      string
}

// ZoomPaneRequest toggles pane zoom.
type ZoomPaneRequest struct {
	SessionName string
	PaneID      string
	Toggle      bool
}

type PaneViewPriority int

const (
	PaneViewPriorityUnset PaneViewPriority = iota
	PaneViewPriorityBackground
	PaneViewPriorityNormal
	PaneViewPriorityFocused
)

// PaneViewRequest asks for a rendered pane view.
type PaneViewRequest struct {
	PaneID string
	Cols   int
	Rows   int
	// DirectRender bypasses frame caching and renders synchronously.
	DirectRender bool
	KnownSeq     uint64
	Priority     PaneViewPriority
	// DeadlineUnixNano carries the client-side deadline for this request.
	// Zero means no deadline provided.
	DeadlineUnixNano int64
}

// PaneViewResponse returns a rendered pane view.
type PaneViewResponse struct {
	PaneID      string
	Cols        int
	Rows        int
	UpdateSeq   uint64
	NotModified bool
	Frame       termframe.Frame
	HasMouse    bool
	AllowMotion bool
}

// PaneOutputRequest asks for output lines since a sequence.
type PaneOutputRequest struct {
	PaneID   string
	SinceSeq uint64
	Limit    int
	Wait     bool
}

// PaneOutputResponse returns output lines and the next sequence.
type PaneOutputResponse struct {
	PaneID    string
	Lines     []native.OutputLine
	NextSeq   uint64
	Truncated bool
}

// PaneSnapshotRequest asks for scrollback snapshot.
type PaneSnapshotRequest struct {
	PaneID string
	Rows   int
}

// PaneSnapshotResponse returns scrollback snapshot.
type PaneSnapshotResponse struct {
	PaneID    string
	Rows      int
	Content   string
	Truncated bool
}

// PaneHistoryRequest requests pane action history.
type PaneHistoryRequest struct {
	PaneID string
	Limit  int
	Since  time.Time
}

// PaneHistoryEntry is a single action log entry.
type PaneHistoryEntry struct {
	TS      time.Time
	Action  string
	Summary string
	Command string
	Status  string
}

// PaneHistoryResponse returns pane action history.
type PaneHistoryResponse struct {
	PaneID  string
	Entries []PaneHistoryEntry
}

// PaneWaitRequest waits for output match.
type PaneWaitRequest struct {
	PaneID  string
	Pattern string
	Timeout time.Duration
}

// PaneWaitResponse returns wait result.
type PaneWaitResponse struct {
	PaneID  string
	Pattern string
	Matched bool
	Match   string
	Elapsed time.Duration
}

// PaneTagRequest adds/removes tags.
type PaneTagRequest struct {
	PaneID string
	Tags   []string
}

// PaneTagListResponse returns tags.
type PaneTagListResponse struct {
	PaneID string
	Tags   []string
}

// PaneFocusRequest sets focused pane.
type PaneFocusRequest struct {
	PaneID string
}

// PaneSignalRequest sends a signal to a pane process.
type PaneSignalRequest struct {
	PaneID string
	Signal string
}

// RelayMode describes relay behavior.
type RelayMode string

const (
	RelayModeLine RelayMode = "line"
	RelayModeRaw  RelayMode = "raw"
)

// RelayStatus describes relay state.
type RelayStatus string

const (
	RelayStatusRunning RelayStatus = "running"
	RelayStatusStopped RelayStatus = "stopped"
	RelayStatusFailed  RelayStatus = "failed"
)

// RelayConfig describes a relay creation request.
type RelayConfig struct {
	FromPaneID string
	ToPaneIDs  []string
	Scope      string
	Mode       RelayMode
	Delay      time.Duration
	Prefix     string
	TTL        time.Duration
}

// RelayStats captures relay statistics.
type RelayStats struct {
	Lines        uint64
	Bytes        uint64
	LastActivity time.Time
}

// RelayInfo describes a relay.
type RelayInfo struct {
	ID        string
	FromPane  string
	ToPanes   []string
	Scope     string
	Mode      RelayMode
	Status    RelayStatus
	Delay     time.Duration
	Prefix    string
	TTL       time.Duration
	CreatedAt time.Time
	Stats     RelayStats
}

// RelayCreateRequest requests a new relay.
type RelayCreateRequest struct {
	Config RelayConfig
}

// RelayCreateResponse returns created relay.
type RelayCreateResponse struct {
	Relay RelayInfo
}

// RelayListResponse returns relays.
type RelayListResponse struct {
	Relays []RelayInfo
}

// RelayStopRequest stops a relay by id.
type RelayStopRequest struct {
	ID string
}

// EventsReplayRequest requests recent events.
type EventsReplayRequest struct {
	Since time.Time
	Until time.Time
	Limit int
	Types []EventType
}

// EventsReplayResponse returns replayed events.
type EventsReplayResponse struct {
	Events []Event
}

// TerminalAction identifies scrollback/copy-mode actions.
type TerminalAction int

const (
	TerminalActionUnknown TerminalAction = iota
	TerminalEnterScrollback
	TerminalExitScrollback
	TerminalScrollUp
	TerminalScrollDown
	TerminalPageUp
	TerminalPageDown
	TerminalScrollTop
	TerminalScrollBottom
	TerminalEnterCopyMode
	TerminalExitCopyMode
	TerminalCopyMove
	TerminalCopyPageUp
	TerminalCopyPageDown
	TerminalCopyToggleSelect
	TerminalCopyYank
)

// TerminalActionRequest runs a terminal action.
type TerminalActionRequest struct {
	PaneID string
	Action TerminalAction
	DeltaX int
	DeltaY int
	Lines  int
}

// TerminalActionResponse returns optional data from an action.
type TerminalActionResponse struct {
	PaneID string
	Text   string
}

// ToastLevel indicates toast severity.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
)

// TerminalKeyRequest asks the daemon to handle a key in scrollback/copy modes.
type TerminalKeyRequest struct {
	PaneID           string
	Key              string
	ScrollbackToggle bool
	CopyToggle       bool
}

// TerminalKeyResponse returns handling info for a key.
type TerminalKeyResponse struct {
	Handled   bool
	Toast     string
	ToastKind ToastLevel
	YankText  string
}
