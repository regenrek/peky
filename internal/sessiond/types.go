package sessiond

import (
	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/native"
)

// Op identifies the request/response operation.
type Op string

const (
	OpHello          Op = "hello"
	OpSessionNames   Op = "session_names"
	OpSnapshot       Op = "snapshot"
	OpStartSession   Op = "start_session"
	OpKillSession    Op = "kill_session"
	OpRenameSession  Op = "rename_session"
	OpRenamePane     Op = "rename_pane"
	OpSplitPane      Op = "split_pane"
	OpClosePane      Op = "close_pane"
	OpSwapPanes      Op = "swap_panes"
	OpSendInput      Op = "send_input"
	OpSendMouse      Op = "send_mouse"
	OpResizePane     Op = "resize_pane"
	OpPaneView       Op = "pane_view"
	OpTerminalAction Op = "terminal_action"
	OpHandleKey      Op = "handle_key"
)

// EventType identifies async daemon events.
type EventType string

const (
	EventPaneUpdated    EventType = "pane_updated"
	EventSessionChanged EventType = "session_changed"
)

// Event is broadcast from daemon to clients.
type Event struct {
	Type    EventType
	PaneID  string
	Session string
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

// SnapshotResponse returns dashboard snapshots.
type SnapshotResponse struct {
	Version  uint64
	Sessions []native.SessionSnapshot
}

// StartSessionRequest starts a new session.
type StartSessionRequest struct {
	Name       string
	Path       string
	LayoutName string
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

// RenamePaneRequest updates a pane title.
type RenamePaneRequest struct {
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

// SplitPaneResponse returns the new pane index.
type SplitPaneResponse struct {
	NewIndex string
}

// ClosePaneRequest closes a pane.
type ClosePaneRequest struct {
	SessionName string
	PaneIndex   string
}

// SwapPanesRequest swaps two panes in a session.
type SwapPanesRequest struct {
	SessionName string
	PaneA       string
	PaneB       string
}

// SendInputRequest forwards raw input.
type SendInputRequest struct {
	PaneID string
	Input  []byte
}

// MouseAction is a mouse action type.
type MouseAction int

const (
	MouseActionUnknown MouseAction = iota
	MouseActionPress
	MouseActionRelease
	MouseActionMotion
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
}

// SendMouseRequest forwards a mouse event.
type SendMouseRequest struct {
	PaneID string
	Event  MouseEventPayload
}

// ResizePaneRequest resizes the PTY for a pane.
type ResizePaneRequest struct {
	PaneID string
	Cols   int
	Rows   int
}

// PaneViewMode controls how the daemon renders a pane.
type PaneViewMode int

const (
	PaneViewANSI PaneViewMode = iota
	PaneViewLipgloss
)

// PaneViewRequest asks for a rendered pane view.
type PaneViewRequest struct {
	PaneID       string
	Cols         int
	Rows         int
	Mode         PaneViewMode
	ShowCursor   bool
	ColorProfile termenv.Profile
	// DeadlineUnixNano carries the client-side deadline for this request.
	// Zero means no deadline provided.
	DeadlineUnixNano int64
}

// PaneViewResponse returns a rendered pane view.
type PaneViewResponse struct {
	PaneID       string
	Cols         int
	Rows         int
	Mode         PaneViewMode
	ShowCursor   bool
	ColorProfile termenv.Profile
	View         string
	HasMouse     bool
	AllowMotion  bool
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
