package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessionpolicy"
	"github.com/regenrek/peakypanes/internal/terminal"
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
	sessions = d.mergeOfflineSessions(sessions)
	d.debugSnapshot(req.PreviewLines, sessions)
	focusedSession, focusedPane := d.focusState()
	resp := SnapshotResponse{
		Version:        manager.Version(),
		Sessions:       sessions,
		FocusedSession: focusedSession,
		FocusedPaneID:  focusedPane,
		PaneGit:        d.collectPaneGit(ctx, sessions),
	}
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
	if d.restore != nil {
		d.restore.MarkSessionDirty(context.Background(), d.manager, resp.Name)
	}
	return encodePayload(resp)
}

func (d *Daemon) handleKillSession(payload []byte) ([]byte, error) {
	var req KillSessionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	name, err := sessionpolicy.ValidateSessionName(req.Name)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	if d.restore != nil {
		d.dropSessionSnapshots(context.Background(), manager, name)
	}
	if err := manager.KillSession(name); err != nil {
		return nil, err
	}
	d.broadcast(Event{Type: EventSessionChanged, Session: name})
	return nil, nil
}

func (d *Daemon) handleRenameSession(payload []byte) ([]byte, error) {
	var req RenameSessionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	oldName, err := sessionpolicy.ValidateSessionName(req.OldName)
	if err != nil {
		return nil, err
	}
	newName, err := sessionpolicy.ValidateSessionName(req.NewName)
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
	if d.restore != nil {
		d.restore.MarkSessionDirty(context.Background(), manager, newName)
	}
	return encodePayload(RenameSessionResponse{NewName: newName})
}

func (d *Daemon) handleRenamePane(payload []byte) ([]byte, error) {
	var req RenamePaneRequest
	if err := decodePayload(payload, &req); err != nil {
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
	sessionName := strings.TrimSpace(req.SessionName)
	paneIndex := strings.TrimSpace(req.PaneIndex)
	if paneID := strings.TrimSpace(req.PaneID); paneID != "" {
		sessionName, paneIndex, err = resolvePaneTargetByID(manager, paneID)
		if err != nil {
			return nil, err
		}
	}
	sessionName, err = sessionpolicy.ValidateSessionName(sessionName)
	if err != nil {
		return nil, err
	}
	paneIndex, err = sessionpolicy.ValidatePaneIndex(paneIndex)
	if err != nil {
		return nil, err
	}
	if err := manager.RenamePane(sessionName, paneIndex, newTitle); err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *Daemon) handleSplitPane(payload []byte) ([]byte, error) {
	var req SplitPaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName, err := sessionpolicy.ValidateSessionName(req.SessionName)
	if err != nil {
		return nil, err
	}
	paneIndex, err := sessionpolicy.ValidatePaneIndex(req.PaneIndex)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	newIndex, newPaneID, err := manager.SplitPane(ctx, sessionName, paneIndex, req.Vertical, req.Percent)
	if err != nil {
		return nil, err
	}
	if d.restore != nil {
		d.restore.MarkSessionDirty(context.Background(), manager, sessionName)
	}
	return encodePayload(SplitPaneResponse{NewIndex: newIndex, NewPaneID: newPaneID})
}

func (d *Daemon) handleClosePane(payload []byte) ([]byte, error) {
	var req ClosePaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	sessionName := strings.TrimSpace(req.SessionName)
	paneIndex := strings.TrimSpace(req.PaneIndex)
	paneID := strings.TrimSpace(req.PaneID)
	if paneID != "" {
		sessionName, paneIndex, err = resolvePaneTargetByID(manager, paneID)
		if err != nil {
			return nil, err
		}
	}
	sessionName, err = sessionpolicy.ValidateSessionName(sessionName)
	if err != nil {
		return nil, err
	}
	paneIndex, err = sessionpolicy.ValidatePaneIndex(paneIndex)
	if err != nil {
		return nil, err
	}
	if paneID == "" {
		paneID = paneIDForIndex(context.Background(), manager, sessionName, paneIndex)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	if err := manager.ClosePane(ctx, sessionName, paneIndex); err != nil {
		return nil, err
	}
	if d.restore != nil {
		if paneID != "" {
			d.restore.DeletePane(paneID)
		}
		d.restore.MarkSessionDirty(context.Background(), manager, sessionName)
	}
	return nil, nil
}

func (d *Daemon) handleSwapPanes(payload []byte) ([]byte, error) {
	var req SwapPanesRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName, err := sessionpolicy.ValidateSessionName(req.SessionName)
	if err != nil {
		return nil, err
	}
	paneA, err := sessionpolicy.ValidatePaneIndex(req.PaneA)
	if err != nil {
		return nil, err
	}
	paneB, err := sessionpolicy.ValidatePaneIndex(req.PaneB)
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
	if d.restore != nil {
		d.restore.MarkSessionDirty(context.Background(), manager, sessionName)
	}
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
	return nil, nil
}

func (d *Daemon) handleSetPaneBackground(payload []byte) ([]byte, error) {
	var req SetPaneBackgroundRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	background, err := sessionpolicy.ValidatePaneBackground(req.Background)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	paneID := strings.TrimSpace(req.PaneID)
	if paneID == "" {
		sessionName, err := sessionpolicy.ValidateSessionName(req.SessionName)
		if err != nil {
			return nil, err
		}
		paneIndex, err := sessionpolicy.ValidatePaneIndex(req.PaneIndex)
		if err != nil {
			return nil, err
		}
		paneID, err = resolvePaneID(manager, sessionName, paneIndex)
		if err != nil {
			return nil, err
		}
	}
	if err := manager.SetPaneBackground(paneID, background); err != nil {
		return nil, err
	}
	if d.restore != nil {
		d.restore.MarkDirty(paneID)
	}
	return nil, nil
}

func (d *Daemon) handleSendInput(payload []byte) ([]byte, error) {
	var req SendInputRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	target, err := resolveSendInputTarget(req)
	if err != nil {
		return nil, err
	}
	if target.PaneID != "" {
		return d.sendInputToPane(manager, req, target.PaneID)
	}
	return d.sendInputToScope(manager, req, target.Scope)
}

type sendInputTarget struct {
	PaneID string
	Scope  string
}

func resolveSendInputTarget(req SendInputRequest) (sendInputTarget, error) {
	paneID := strings.TrimSpace(req.PaneID)
	scope := strings.TrimSpace(req.Scope)
	if paneID != "" && scope != "" {
		return sendInputTarget{}, errors.New("sessiond: pane id and scope are mutually exclusive")
	}
	if paneID == "" && scope == "" {
		return sendInputTarget{}, errors.New("sessiond: pane id or scope is required")
	}
	return sendInputTarget{PaneID: paneID, Scope: scope}, nil
}

func (d *Daemon) sendInputToPane(manager sessionManager, req SendInputRequest, paneID string) ([]byte, error) {
	paneID, err := requirePaneID(paneID)
	if err != nil {
		return nil, err
	}
	start := time.Time{}
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		start = time.Now()
		slog.Debug(
			"sessiond: send_input start",
			slog.String("pane_id", paneID),
			slog.Int("bytes", len(req.Input)),
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	err = manager.SendInput(ctx, paneID, req.Input)
	cancel()
	if err != nil {
		if !start.IsZero() {
			slog.Debug(
				"sessiond: send_input done",
				slog.String("pane_id", paneID),
				slog.Duration("dur", time.Since(start)),
				slog.Any("err", err),
			)
		}
		return nil, err
	}
	if !start.IsZero() {
		slog.Debug(
			"sessiond: send_input done",
			slog.String("pane_id", paneID),
			slog.Duration("dur", time.Since(start)),
		)
	}
	d.recordSendInputAction(paneID, req, "ok")
	return encodePayload(SendInputResponse{
		Results: []SendInputResult{{PaneID: paneID, Status: "ok"}},
	})
}

func (d *Daemon) sendInputToScope(manager sessionManager, req SendInputRequest, scope string) ([]byte, error) {
	targets, err := d.resolveScopeTargets(scope)
	if err != nil {
		return nil, err
	}
	results := make([]SendInputResult, len(targets))
	if len(targets) == 0 {
		return encodePayload(SendInputResponse{Results: results})
	}

	action := resolveSendInputAction(req.Action)
	workers := scopeSendConcurrency(len(targets))
	jobs := make(chan scopeSendJob)
	responses := make(chan scopeSendResult, len(targets))

	for i := 0; i < workers; i++ {
		go func() {
			for job := range jobs {
				status, message := sendInputToTargetWithTimeout(manager, job.PaneID, req.Input, scopeSendTimeout)
				if req.RecordAction {
					d.recordPaneAction(job.PaneID, action, req.Summary, "", status)
				}
				responses <- scopeSendResult{
					Index:   job.Index,
					PaneID:  job.PaneID,
					Status:  status,
					Message: message,
				}
			}
		}()
	}

	for idx, target := range targets {
		jobs <- scopeSendJob{Index: idx, PaneID: target}
	}
	close(jobs)

	for i := 0; i < len(targets); i++ {
		res := <-responses
		results[res.Index] = SendInputResult{
			PaneID:  res.PaneID,
			Status:  res.Status,
			Message: res.Message,
		}
	}

	return encodePayload(SendInputResponse{Results: results})
}

func sendInputToTarget(manager sessionManager, ctx context.Context, paneID string, input []byte) (string, string) {
	if err := manager.SendInput(ctx, paneID, input); err != nil {
		return "failed", err.Error()
	}
	return "ok", ""
}

func (d *Daemon) recordSendInputAction(paneID string, req SendInputRequest, status string) {
	if !req.RecordAction {
		return
	}
	action := resolveSendInputAction(req.Action)
	d.recordPaneAction(paneID, action, req.Summary, "", status)
}

func resolveSendInputAction(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return "send"
	}
	return action
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
	event, route, ok := mousePayloadToEvent(req.Event)
	if !ok {
		return nil, nil
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	sendCount := 1
	if _, isWheel := event.(uv.MouseWheelEvent); isWheel {
		sendCount = clampWheelCount(mousePayloadWheelCount(req.Event))
	}
	for i := 0; i < sendCount; i++ {
		if err := manager.SendMouse(paneID, event, route); err != nil {
			if d.restore != nil {
				if _, ok := d.restore.Snapshot(paneID); ok {
					return nil, nil
				}
			}
			return nil, err
		}
	}
	return nil, nil
}

const sessiondMouseWheelCountMax = 4096

func mousePayloadWheelCount(payload MouseEventPayload) int {
	if payload.WheelCount <= 0 {
		return 1
	}
	return payload.WheelCount
}

func clampWheelCount(count int) int {
	if count < 1 {
		return 1
	}
	if count > sessiondMouseWheelCountMax {
		return sessiondMouseWheelCountMax
	}
	return count
}

func (d *Daemon) handleResizePane(payload []byte) ([]byte, error) {
	var req ResizePaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName := strings.TrimSpace(req.SessionName)
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	if sessionName == "" {
		return nil, errors.New("sessiond: session is required")
	}
	edge, err := resolveResizeEdge(req.Edge)
	if err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	snapState := layout.SnapState{Active: req.SnapState.Active, Target: req.SnapState.Target}
	result, err := manager.ResizePaneEdge(sessionName, paneID, edge, req.Delta, req.Snap, snapState)
	if err != nil {
		return nil, err
	}
	return encodePayload(layoutOpResponse(result))
}

func (d *Daemon) handleResetPaneSizes(payload []byte) ([]byte, error) {
	var req ResetSizesRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName := strings.TrimSpace(req.SessionName)
	if sessionName == "" {
		return nil, errors.New("sessiond: session is required")
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	result, err := manager.ResetPaneSizes(sessionName, strings.TrimSpace(req.PaneID))
	if err != nil {
		return nil, err
	}
	return encodePayload(layoutOpResponse(result))
}

func (d *Daemon) handleZoomPane(payload []byte) ([]byte, error) {
	var req ZoomPaneRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	sessionName := strings.TrimSpace(req.SessionName)
	paneID := strings.TrimSpace(req.PaneID)
	if sessionName == "" || paneID == "" {
		return nil, errors.New("sessiond: session and pane are required")
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	result, err := manager.ZoomPane(sessionName, paneID, req.Toggle)
	if err != nil {
		return nil, err
	}
	return encodePayload(layoutOpResponse(result))
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

func resolveResizeEdge(edge ResizeEdge) (layout.ResizeEdge, error) {
	value := strings.ToLower(strings.TrimSpace(string(edge)))
	switch value {
	case string(ResizeEdgeLeft):
		return layout.ResizeEdgeLeft, nil
	case string(ResizeEdgeRight):
		return layout.ResizeEdgeRight, nil
	case string(ResizeEdgeUp):
		return layout.ResizeEdgeUp, nil
	case string(ResizeEdgeDown):
		return layout.ResizeEdgeDown, nil
	default:
		return layout.ResizeEdgeLeft, fmt.Errorf("sessiond: invalid resize edge %q", edge)
	}
}

func layoutOpResponse(result layout.ApplyResult) LayoutOpResponse {
	resp := LayoutOpResponse{
		Changed:   result.Changed,
		Snapped:   result.Snapped,
		SnapState: SnapState{Active: result.SnapState.Active, Target: result.SnapState.Target},
	}
	if len(result.Affected) > 0 {
		resp.Affected = append([]string(nil), result.Affected...)
	}
	return resp
}

func resolvePaneTargetByID(manager sessionManager, paneID string) (string, string, error) {
	if manager == nil {
		return "", "", errors.New("sessiond: manager unavailable")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return "", "", errors.New("sessiond: pane id is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := manager.Snapshot(ctx, 0)
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == paneID {
				return session.Name, pane.Index, nil
			}
		}
	}
	return "", "", fmt.Errorf("sessiond: pane %q not found", paneID)
}

func resolvePaneID(manager sessionManager, sessionName, paneIndex string) (string, error) {
	if manager == nil {
		return "", errors.New("sessiond: manager unavailable")
	}
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || paneIndex == "" {
		return "", errors.New("sessiond: session and pane index are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := manager.Snapshot(ctx, 0)
	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}
		for _, pane := range session.Panes {
			if pane.Index == paneIndex {
				if pane.ID == "" {
					return "", fmt.Errorf("sessiond: pane %q id unavailable", paneIndex)
				}
				return pane.ID, nil
			}
		}
		return "", fmt.Errorf("sessiond: pane %q not found in %q", paneIndex, sessionName)
	}
	return "", fmt.Errorf("sessiond: session %q not found", sessionName)
}

func (d *Daemon) startSession(req StartSessionRequest) (StartSessionResponse, error) {
	if d.manager == nil {
		return StartSessionResponse{}, errors.New("sessiond: manager unavailable")
	}
	path, nameOverride, env, paneCount, err := validateStartSessionRequest(req)
	if err != nil {
		return StartSessionResponse{}, err
	}
	loader, err := loadStartSessionLayouts(path)
	if err != nil {
		return StartSessionResponse{}, err
	}
	sessionName, err := resolveStartSessionName(path, nameOverride, loader)
	if err != nil {
		return StartSessionResponse{}, err
	}
	layoutName := strings.TrimSpace(req.LayoutName)
	var selectedLayout *layout.LayoutConfig
	if paneCount > 0 {
		layoutCfg, generatedName, err := layoutForPaneCount(paneCount)
		if err != nil {
			return StartSessionResponse{}, err
		}
		layoutName = generatedName
		selectedLayout = layoutCfg
	} else {
		selectedLayout, err = selectStartSessionLayout(loader, layoutName)
		if err != nil {
			return StartSessionResponse{}, err
		}
		if selectedLayout == nil {
			return StartSessionResponse{}, errors.New("sessiond: no layout found")
		}
		layoutName = selectedLayout.Name
	}
	expanded := expandStartSessionLayout(selectedLayout, loader, path)
	if err := d.startSessionWithLayout(sessionName, path, layoutName, expanded, env); err != nil {
		return StartSessionResponse{}, err
	}
	return StartSessionResponse{Name: sessionName, Path: path, LayoutName: layoutName}, nil
}

func validateStartSessionRequest(req StartSessionRequest) (string, string, []string, int, error) {
	path, err := sessionpolicy.ValidatePath(req.Path)
	if err != nil {
		return "", "", nil, 0, err
	}
	nameOverride, err := sessionpolicy.ValidateOptionalSessionName(req.Name)
	if err != nil {
		return "", "", nil, 0, err
	}
	env, err := sessionpolicy.ValidateEnvList(req.Env)
	if err != nil {
		return "", "", nil, 0, err
	}
	paneCount, err := sessionpolicy.ValidatePaneCount(req.PaneCount)
	if err != nil {
		return "", "", nil, 0, err
	}
	if paneCount > 0 && strings.TrimSpace(req.LayoutName) != "" {
		return "", "", nil, 0, errors.New("layout cannot be combined with panes")
	}
	return path, nameOverride, env, paneCount, nil
}

func loadStartSessionLayouts(path string) (*layout.Loader, error) {
	loader, err := layout.NewLoader()
	if err != nil {
		return nil, err
	}
	loader.SetProjectDir(path)
	if err := loader.LoadAll(); err != nil {
		return nil, err
	}
	return loader, nil
}

func resolveStartSessionName(path, nameOverride string, loader *layout.Loader) (string, error) {
	sessionName := layout.ResolveSessionName(path, nameOverride, loader.GetProjectConfig())
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return "", errors.New("sessiond: session name is required")
	}
	if _, err := sessionpolicy.ValidateSessionName(sessionName); err != nil {
		return "", err
	}
	return sessionName, nil
}

func selectStartSessionLayout(loader *layout.Loader, layoutName string) (*layout.LayoutConfig, error) {
	if layoutName != "" {
		selectedLayout, _, err := loader.GetLayout(layoutName)
		if err != nil {
			return nil, err
		}
		return selectedLayout, nil
	}
	if loader.HasProjectConfig() {
		selectedLayout := loader.GetProjectLayout()
		if selectedLayout != nil {
			return selectedLayout, nil
		}
	}
	selectedLayout, _, _ := loader.GetLayout("")
	if selectedLayout == nil {
		return nil, errors.New("sessiond: no layout found")
	}
	return selectedLayout, nil
}

func expandStartSessionLayout(selectedLayout *layout.LayoutConfig, loader *layout.Loader, path string) *layout.LayoutConfig {
	projectName := filepath.Base(path)
	var projectVars map[string]string
	if loader.GetProjectConfig() != nil {
		projectVars = loader.GetProjectConfig().Vars
	}
	return layout.ExpandLayoutVars(selectedLayout, projectVars, path, projectName)
}

func layoutForPaneCount(count int) (*layout.LayoutConfig, string, error) {
	grid, err := gridForPaneCount(count)
	if err != nil {
		return nil, "", err
	}
	name := fmt.Sprintf("grid-%s", grid.String())
	return &layout.LayoutConfig{
		Name: name,
		Grid: grid.String(),
	}, name, nil
}

func gridForPaneCount(count int) (layout.Grid, error) {
	if count <= 0 {
		return layout.Grid{}, errors.New("pane count must be positive")
	}
	best := layout.Grid{Rows: 1, Columns: count}
	bestDiff := best.Columns - best.Rows
	for rows := 1; rows*rows <= count; rows++ {
		if count%rows != 0 {
			continue
		}
		cols := count / rows
		diff := cols - rows
		if diff < bestDiff {
			best = layout.Grid{Rows: rows, Columns: cols}
			bestDiff = diff
		}
	}
	if err := best.Validate(); err != nil {
		return layout.Grid{}, err
	}
	return best, nil
}

func (d *Daemon) startSessionWithLayout(name, path, layoutName string, layoutConfig *layout.LayoutConfig, env []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	_, err := d.manager.StartSession(ctx, native.SessionSpec{
		Name:       name,
		Path:       path,
		Layout:     layoutConfig,
		LayoutName: layoutName,
		Env:        env,
	})
	return err
}

func mousePayloadToEvent(payload MouseEventPayload) (uv.MouseEvent, terminal.MouseRoute, bool) {
	if payload.X < 0 || payload.Y < 0 {
		return nil, terminal.MouseRouteAuto, false
	}
	route, ok := terminal.ParseMouseRoute(string(payload.Route))
	if !ok {
		return nil, terminal.MouseRouteAuto, false
	}
	button, wheel, ok := mousePayloadButton(payload)
	if !ok {
		return nil, terminal.MouseRouteAuto, false
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
	mouse := uv.Mouse{X: payload.X, Y: payload.Y, Button: button, Mod: mod}
	if wheel {
		return uv.MouseWheelEvent(mouse), route, true
	}
	switch payload.Action {
	case MouseActionPress:
		return uv.MouseClickEvent(mouse), route, true
	case MouseActionRelease:
		return uv.MouseReleaseEvent(mouse), route, true
	case MouseActionMotion:
		return uv.MouseMotionEvent(mouse), route, true
	default:
		return nil, terminal.MouseRouteAuto, false
	}
}

func mousePayloadButton(payload MouseEventPayload) (uv.MouseButton, bool, bool) {
	// Protocol button codes (do not rely on upstream tea/uv enum values).
	switch payload.Button {
	case 1:
		return uv.MouseLeft, false, true
	case 2:
		return uv.MouseMiddle, false, true
	case 3:
		return uv.MouseRight, false, true
	case 4:
		return uv.MouseWheelUp, true, true
	case 5:
		return uv.MouseWheelDown, true, true
	case 6:
		return uv.MouseWheelLeft, true, true
	case 7:
		return uv.MouseWheelRight, true, true
	default:
		return uv.MouseNone, false, false
	}
}
