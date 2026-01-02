package sessiond

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
)

func (d *Daemon) requireManager() (sessionManager, error) {
	if d.manager == nil {
		return nil, errors.New("sessiond: manager unavailable")
	}
	return d.manager, nil
}

func (d *Daemon) handleHello(payload []byte) ([]byte, error) {
	var req HelloRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	resp := HelloResponse{Version: d.version, PID: os.Getpid()}
	return encodePayload(resp)
}

func (d *Daemon) handleSessionNames() ([]byte, error) {
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	return encodePayload(SessionNamesResponse{Names: manager.SessionNames()})
}

func (d *Daemon) handleSnapshot(payload []byte) ([]byte, error) {
	var req SnapshotRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := manager.Snapshot(ctx, req.PreviewLines)
	d.debugSnapshot(req.PreviewLines, sessions)
	resp := SnapshotResponse{Version: manager.Version(), Sessions: sessions}
	return encodePayload(resp)
}

func (d *Daemon) handleStartSession(payload []byte) ([]byte, error) {
	var req StartSessionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	resp, err := d.startSession(req)
	if err != nil {
		return nil, err
	}
	d.broadcast(Event{Type: EventSessionChanged, Session: resp.Name})
	d.queuePersistState()
	return encodePayload(resp)
}

func (d *Daemon) handleKillSession(payload []byte) ([]byte, error) {
	var req KillSessionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	name, err := validateSessionName(req.Name)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.KillSession(name); err != nil {
		return nil, err
	}
	d.broadcast(Event{Type: EventSessionChanged, Session: name})
	d.queuePersistState()
	return nil, nil
}

func (d *Daemon) handleRenameSession(payload []byte) ([]byte, error) {
	var req RenameSessionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	oldName, err := validateSessionName(req.OldName)
	if err != nil {
		return nil, err
	}
	newName, err := validateSessionName(req.NewName)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.RenameSession(oldName, newName); err != nil {
		return nil, err
	}
	d.broadcast(Event{Type: EventSessionChanged, Session: newName})
	d.queuePersistState()
	return encodePayload(RenameSessionResponse{NewName: newName})
}

func (d *Daemon) handleRenamePane(payload []byte) ([]byte, error) {
	var req RenamePaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName, err := validateSessionName(req.SessionName)
	if err != nil {
		return nil, err
	}
	paneIndex, err := validatePaneIndex(req.PaneIndex)
	if err != nil {
		return nil, err
	}
	newTitle := strings.TrimSpace(req.NewTitle)
	if newTitle == "" {
		return nil, errors.New("sessiond: pane title is required")
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.RenamePane(sessionName, paneIndex, newTitle); err != nil {
		return nil, err
	}
	d.queuePersistState()
	return nil, nil
}

func (d *Daemon) handleSplitPane(payload []byte) ([]byte, error) {
	var req SplitPaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName, err := validateSessionName(req.SessionName)
	if err != nil {
		return nil, err
	}
	paneIndex, err := validatePaneIndex(req.PaneIndex)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	newIndex, err := manager.SplitPane(ctx, sessionName, paneIndex, req.Vertical, req.Percent)
	if err != nil {
		return nil, err
	}
	d.queuePersistState()
	return encodePayload(SplitPaneResponse{NewIndex: newIndex})
}

func (d *Daemon) handleClosePane(payload []byte) ([]byte, error) {
	var req ClosePaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName, err := validateSessionName(req.SessionName)
	if err != nil {
		return nil, err
	}
	paneIndex, err := validatePaneIndex(req.PaneIndex)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	if err := manager.ClosePane(ctx, sessionName, paneIndex); err != nil {
		return nil, err
	}
	d.queuePersistState()
	return nil, nil
}

func (d *Daemon) handleSwapPanes(payload []byte) ([]byte, error) {
	var req SwapPanesRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName, err := validateSessionName(req.SessionName)
	if err != nil {
		return nil, err
	}
	paneA, err := validatePaneIndex(req.PaneA)
	if err != nil {
		return nil, err
	}
	paneB, err := validatePaneIndex(req.PaneB)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.SwapPanes(sessionName, paneA, paneB); err != nil {
		return nil, err
	}
	d.queuePersistState()
	return nil, nil
}

func (d *Daemon) handleSetPaneTool(payload []byte) ([]byte, error) {
	var req SetPaneToolRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.SetPaneTool(paneID, req.Tool); err != nil {
		return nil, err
	}
	d.queuePersistState()
	return nil, nil
}

func (d *Daemon) handleSendInput(payload []byte) ([]byte, error) {
	var req SendInputRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.SendInput(paneID, req.Input); err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *Daemon) handleSendMouse(payload []byte) ([]byte, error) {
	var req SendMouseRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	event, ok := mousePayloadToEvent(req.Event)
	if !ok {
		return nil, nil
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if err := manager.SendMouse(paneID, event); err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *Daemon) handleResizePane(payload []byte) ([]byte, error) {
	var req ResizePaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	win := manager.Window(paneID)
	if win == nil {
		return nil, fmt.Errorf("sessiond: pane %q not found", paneID)
	}
	cols, rows := normalizeDimensions(req.Cols, req.Rows)
	if err := win.Resize(cols, rows); err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *Daemon) handleTerminalActionPayload(payload []byte) ([]byte, error) {
	var req TerminalActionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	resp, err := d.terminalAction(req)
	if err != nil {
		return nil, err
	}
	return encodePayload(resp)
}

func (d *Daemon) handleHandleKey(payload []byte) ([]byte, error) {
	var req TerminalKeyRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	resp, err := d.handleTerminalKey(req)
	if err != nil {
		return nil, err
	}
	return encodePayload(resp)
}

func requirePaneID(value string) (string, error) {
	paneID := strings.TrimSpace(value)
	if paneID == "" {
		return "", errors.New("sessiond: pane id is required")
	}
	return paneID, nil
}

func normalizeDimensions(cols, rows int) (int, int) {
	return limits.Clamp(cols, rows)
}

func (d *Daemon) startSession(req StartSessionRequest) (StartSessionResponse, error) {
	if d.manager == nil {
		return StartSessionResponse{}, errors.New("sessiond: manager unavailable")
	}
	path, err := validatePath(req.Path)
	if err != nil {
		return StartSessionResponse{}, err
	}
	nameOverride, err := validateOptionalSessionName(req.Name)
	if err != nil {
		return StartSessionResponse{}, err
	}
	loader, err := layout.NewLoader()
	if err != nil {
		return StartSessionResponse{}, err
	}
	loader.SetProjectDir(path)
	if err := loader.LoadAll(); err != nil {
		return StartSessionResponse{}, err
	}
	sessionName := layout.ResolveSessionName(path, nameOverride, loader.GetProjectConfig())
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return StartSessionResponse{}, errors.New("sessiond: session name is required")
	}
	if _, err := validateSessionName(sessionName); err != nil {
		return StartSessionResponse{}, err
	}

	layoutName := strings.TrimSpace(req.LayoutName)
	var selectedLayout *layout.LayoutConfig
	if layoutName != "" {
		selectedLayout, _, err = loader.GetLayout(layoutName)
		if err != nil {
			return StartSessionResponse{}, err
		}
	} else if loader.HasProjectConfig() {
		selectedLayout = loader.GetProjectLayout()
		if selectedLayout == nil {
			selectedLayout, _, _ = loader.GetLayout("dev-3")
		}
	} else {
		selectedLayout, _, _ = loader.GetLayout("dev-3")
	}
	if selectedLayout == nil {
		return StartSessionResponse{}, errors.New("sessiond: no layout found")
	}

	projectName := filepath.Base(path)
	var projectVars map[string]string
	if loader.GetProjectConfig() != nil {
		projectVars = loader.GetProjectConfig().Vars
	}
	expanded := layout.ExpandLayoutVars(selectedLayout, projectVars, path, projectName)

	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	_, err = d.manager.StartSession(ctx, native.SessionSpec{
		Name:       sessionName,
		Path:       path,
		Layout:     expanded,
		LayoutName: selectedLayout.Name,
	})
	if err != nil {
		return StartSessionResponse{}, err
	}
	return StartSessionResponse{Name: sessionName, Path: path, LayoutName: selectedLayout.Name}, nil
}

func mousePayloadToEvent(payload MouseEventPayload) (uv.MouseEvent, bool) {
	if payload.X < 0 || payload.Y < 0 {
		return nil, false
	}
	mod := uv.KeyMod(0)
	if payload.Shift {
		mod |= uv.ModShift
	}
	if payload.Alt {
		mod |= uv.ModAlt
	}
	if payload.Ctrl {
		mod |= uv.ModCtrl
	}
	mouse := uv.Mouse{X: payload.X, Y: payload.Y, Button: uv.MouseButton(payload.Button), Mod: mod}
	if payload.Wheel {
		return uv.MouseWheelEvent(mouse), true
	}
	switch payload.Action {
	case MouseActionPress:
		return uv.MouseClickEvent(mouse), true
	case MouseActionRelease:
		return uv.MouseReleaseEvent(mouse), true
	case MouseActionMotion:
		return uv.MouseMotionEvent(mouse), true
	default:
		return nil, false
	}
}
