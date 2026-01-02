package sessiond

import (
	"context"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/native"
)

type paneViewWindow interface {
	ANSICacheSeq() uint64
	UpdateSeq() uint64
	CopyModeActive() bool
	ViewLipglossCtx(ctx context.Context, showCursor bool, profile termenv.Profile) (string, error)
	ViewANSICtx(ctx context.Context) (string, error)
	ViewANSIDirectCtx(ctx context.Context) (string, error)
	ViewLipgloss(showCursor bool, profile termenv.Profile) string
	ViewANSI() string
}

type paneWindow interface {
	paneViewWindow

	CopySelectionActive() bool
	CopySelectionFromMouseActive() bool
	CopyMove(dx, dy int)
	CopyPageDown()
	CopyPageUp()
	CopyToggleSelect()
	CopyYankText() string
	EnterCopyMode()
	EnterScrollback()
	ExitCopyMode()
	ExitScrollback()
	GetScrollbackOffset() int
	IsAltScreen() bool
	PageDown()
	PageUp()
	ScrollDown(lines int)
	ScrollToBottom()
	ScrollToTop()
	ScrollUp(lines int)
	ScrollbackModeActive() bool

	Resize(cols, rows int) error
	HasMouseMode() bool
	AllowsMouseMotion() bool
}

type sessionManager interface {
	SessionNames() []string
	Snapshot(ctx context.Context, previewLines int) []native.SessionSnapshot
	Version() uint64
	StartSession(ctx context.Context, spec native.SessionSpec) (*native.Session, error)
	RestoreSession(ctx context.Context, spec native.SessionRestoreSpec) (*native.Session, error)
	KillSession(name string) error
	RenameSession(oldName, newName string) error
	RenamePane(sessionName, paneIndex, newTitle string) error
	SplitPane(ctx context.Context, sessionName, paneIndex string, vertical bool, percent int) (string, error)
	ClosePane(ctx context.Context, sessionName, paneIndex string) error
	SwapPanes(sessionName, paneA, paneB string) error
	SetPaneTool(paneID, tool string) error
	SendInput(paneID string, input []byte) error
	SendMouse(paneID string, event uv.MouseEvent) error
	Window(paneID string) paneWindow
	Events() <-chan native.PaneEvent
	Close()
}

type nativeManagerAdapter struct {
	*native.Manager
}

func (m nativeManagerAdapter) Window(paneID string) paneWindow {
	if m.Manager == nil {
		return nil
	}
	win := m.Manager.Window(paneID)
	if win == nil {
		return nil
	}
	return win
}

func wrapManager(mgr *native.Manager) sessionManager {
	if mgr == nil {
		return nil
	}
	return nativeManagerAdapter{Manager: mgr}
}
