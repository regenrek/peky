package mux

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Type identifies the terminal multiplexer backend.
type Type string

const (
	Tmux   Type = "tmux"
	Zellij Type = "zellij"
)

func (t Type) String() string {
	return string(t)
}

// ParseType normalizes a mux string into a Type.
func ParseType(value string) (Type, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "tmux":
		return Tmux, nil
	case "zellij":
		return Zellij, nil
	default:
		return "", fmt.Errorf("unknown multiplexer %q", value)
	}
}

// SessionInfo describes a multiplexer session with optional path metadata.
type SessionInfo struct {
	Name string
	Path string
}

// WindowInfo describes a multiplexer window/tab.
type WindowInfo struct {
	Index  string
	Name   string
	Active bool
}

// PaneInfo describes a multiplexer pane with geometry and status metadata.
type PaneInfo struct {
	ID         string
	Index      string
	Active     bool
	Title      string
	Command    string
	Left       int
	Top        int
	Width      int
	Height     int
	Dead       bool
	DeadStatus int
	LastActive time.Time
}

// PopupOptions controls popup rendering when supported.
type PopupOptions struct {
	Width    string
	Height   string
	StartDir string
}

// Client abstracts per-multiplexer operations needed by the dashboard and CLI.
type Client interface {
	Type() Type
	Binary() string
	IsInside() bool

	ListSessions(ctx context.Context) ([]string, error)
	ListSessionsInfo(ctx context.Context) ([]SessionInfo, error)
	CurrentSession(ctx context.Context) (string, error)
	ListWindows(ctx context.Context, session string) ([]WindowInfo, error)
	ListPanesDetailed(ctx context.Context, target string) ([]PaneInfo, error)
	CapturePaneLines(ctx context.Context, target string, lines int) ([]string, error)
	SessionHasClients(ctx context.Context, session string) (bool, error)

	RenameSession(ctx context.Context, session, newName string) error
	RenameWindow(ctx context.Context, session, windowTarget, newName string) error
	KillSession(ctx context.Context, session string) error
	SendKeys(ctx context.Context, target string, keys ...string) error

	Attach(ctx context.Context, target string, inside bool) error
	AttachCommand(target string, inside bool) (string, []string, []string)

	SupportsPopup(ctx context.Context) bool
	DisplayPopup(ctx context.Context, opts PopupOptions, command []string) error
	OpenDashboardWindow(ctx context.Context, session, windowName string, command []string) error
}
