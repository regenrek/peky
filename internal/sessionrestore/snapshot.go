package sessionrestore

import "time"

// CurrentSchemaVersion identifies the persisted schema version.
const CurrentSchemaVersion = 1

// PaneSnapshot captures persisted data for a single pane.
type PaneSnapshot struct {
	SchemaVersion int       `json:"schemaVersion"`
	CapturedAt    time.Time `json:"capturedAt"`

	SessionName    string    `json:"sessionName"`
	SessionPath    string    `json:"sessionPath,omitempty"`
	SessionLayout  string    `json:"sessionLayout,omitempty"`
	SessionCreated time.Time `json:"sessionCreatedAt,omitempty"`
	SessionEnv     []string  `json:"sessionEnv,omitempty"`

	PaneID        string    `json:"paneId"`
	PaneIndex     string    `json:"paneIndex"`
	PaneTitle     string    `json:"paneTitle,omitempty"`
	PaneCommand   string    `json:"paneCommand,omitempty"`
	PaneStart     string    `json:"paneStartCommand,omitempty"`
	PaneTool      string    `json:"paneTool,omitempty"`
	PaneCwd       string    `json:"paneCwd,omitempty"`
	PaneActive    bool      `json:"paneActive,omitempty"`
	PaneLeft      int       `json:"paneLeft,omitempty"`
	PaneTop       int       `json:"paneTop,omitempty"`
	PaneWidth     int       `json:"paneWidth,omitempty"`
	PaneHeight    int       `json:"paneHeight,omitempty"`
	PaneDead      bool      `json:"paneDead,omitempty"`
	PaneDeadCode  int       `json:"paneDeadStatus,omitempty"`
	PaneLastAct   time.Time `json:"paneLastActive,omitempty"`
	PaneRestoreFailed bool   `json:"paneRestoreFailed,omitempty"`
	PaneRestoreErr    string `json:"paneRestoreError,omitempty"`
	PaneTags      []string  `json:"paneTags,omitempty"`
	PaneBytesIn   uint64    `json:"paneBytesIn,omitempty"`
	PaneBytesOut  uint64    `json:"paneBytesOut,omitempty"`

	RestoreMode string           `json:"restoreMode,omitempty"`
	Private     bool             `json:"private,omitempty"`
	Terminal    TerminalSnapshot `json:"terminal"`
}

// TerminalSnapshot captures a plain-text VT snapshot.
type TerminalSnapshot struct {
	Cols           int      `json:"cols"`
	Rows           int      `json:"rows"`
	CursorX        int      `json:"cursorX"`
	CursorY        int      `json:"cursorY"`
	CursorVisible  bool     `json:"cursorVisible"`
	AltScreen      bool     `json:"altScreen"`
	ScreenLines    []string `json:"screen"`
	ScrollbackLines []string `json:"scrollback,omitempty"`
}
