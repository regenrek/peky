package sessiond

import (
	"context"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/termframe"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type paneViewWindow interface {
	FrameCacheSeq() uint64
	UpdateSeq() uint64
	CopyModeActive() bool
	ViewFrameCtx(ctx context.Context) (termframe.Frame, error)
	ViewFrameDirectCtx(ctx context.Context) (termframe.Frame, error)
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
	SnapshotPlain(opts terminal.PlainSnapshotOptions) (terminal.PlainSnapshot, error)
}

type sessionManager interface {
	SessionNames() []string
	Snapshot(ctx context.Context, previewLines int) []native.SessionSnapshot
	Version() uint64
	StartSession(ctx context.Context, spec native.SessionSpec) (*native.Session, error)
	KillSession(name string) error
	RenameSession(oldName, newName string) error
	RenamePane(sessionName, paneIndex, newTitle string) error
	SplitPane(ctx context.Context, sessionName, paneIndex string, vertical bool, percent int) (string, error)
	ClosePane(ctx context.Context, sessionName, paneIndex string) error
	SwapPanes(sessionName, paneA, paneB string) error
	ResizePaneEdge(sessionName, paneID string, edge layout.ResizeEdge, delta int, snap bool, snapState layout.SnapState) (layout.ApplyResult, error)
	ResetPaneSizes(sessionName, paneID string) (layout.ApplyResult, error)
	ZoomPane(sessionName, paneID string, toggle bool) (layout.ApplyResult, error)
	SetPaneTool(paneID, tool string) error
	SetPaneBackground(paneID string, background int) error
	SendInput(ctx context.Context, paneID string, input []byte) error
	SendMouse(paneID string, event uv.MouseEvent, route terminal.MouseRoute) error
	Window(paneID string) paneWindow
	PaneTags(paneID string) ([]string, error)
	AddPaneTags(paneID string, tags []string) ([]string, error)
	RemovePaneTags(paneID string, tags []string) ([]string, error)
	OutputSnapshot(paneID string, limit int) ([]native.OutputLine, error)
	OutputLinesSince(paneID string, seq uint64) ([]native.OutputLine, uint64, bool, error)
	WaitForOutput(ctx context.Context, paneID string) bool
	SubscribeRawOutput(paneID string, buffer int) (<-chan native.OutputChunk, func(), error)
	PaneScrollbackSnapshot(paneID string, rows int) (string, bool, error)
	SignalPane(paneID string, signalName string) error
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
