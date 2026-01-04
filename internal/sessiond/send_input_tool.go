package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/tool"
)

type toolSendPlan struct {
	Payload     []byte
	Submit      []byte
	SubmitDelay time.Duration
	Combine     bool
	ToolID      string
}

func (d *Daemon) handleSendInputTool(payload []byte) ([]byte, error) {
	var req SendInputToolRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	target, err := resolveSendInputToolTarget(req)
	if err != nil {
		return nil, err
	}
	reg := d.toolRegistryRef()
	if reg == nil {
		return nil, errors.New("sessiond: tool registry unavailable")
	}
	filter, err := normalizeToolFilter(reg, req.ToolFilter)
	if err != nil {
		return nil, err
	}
	if target.PaneID != "" {
		return d.sendInputToolToPane(manager, reg, req, target.PaneID, filter)
	}
	return d.sendInputToolToScope(manager, reg, req, target.Scope, filter)
}

func resolveSendInputToolTarget(req SendInputToolRequest) (sendInputTarget, error) {
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

func normalizeToolFilter(reg *tool.Registry, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	canonical := reg.Normalize(value)
	if canonical == "" {
		return "", fmt.Errorf("sessiond: unknown tool %q", value)
	}
	if !reg.Allowed(canonical) {
		return "", fmt.Errorf("sessiond: tool %q is disabled", canonical)
	}
	return canonical, nil
}

func (d *Daemon) sendInputToolToPane(manager sessionManager, reg *tool.Registry, req SendInputToolRequest, paneID, filter string) ([]byte, error) {
	paneID, err := requirePaneID(paneID)
	if err != nil {
		return nil, err
	}
	paneInfo, err := d.lookupPaneInfo(manager, paneID)
	if err != nil {
		return nil, err
	}
	plan, ok := buildToolSendPlan(reg, paneInfo, req, filter)
	if !ok {
		return encodePayload(SendInputResponse{Results: []SendInputResult{{PaneID: paneID, Status: "skipped", Message: "tool filter mismatch"}}})
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	status, message := sendToolInputWithContext(manager, ctx, paneID, plan)
	cancel()
	if status != "ok" {
		return nil, errors.New(message)
	}
	if req.RecordAction {
		d.recordSendInputAction(paneID, SendInputRequest{RecordAction: true, Action: req.Action, Summary: req.Summary}, "ok")
	}
	if req.DetectTool {
		detectAndSetPaneTool(manager, reg, paneID, req.Input)
	}
	return encodePayload(SendInputResponse{Results: []SendInputResult{{PaneID: paneID, Status: "ok"}}})
}

func (d *Daemon) sendInputToolToScope(manager sessionManager, reg *tool.Registry, req SendInputToolRequest, scope, filter string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := manager.Snapshot(ctx, 0)
	if len(sessions) == 0 {
		return nil, errors.New("sessiond: no sessions available")
	}
	targets, err := d.resolveScopeTargetsWithSnapshot(scope, sessions)
	if err != nil {
		return nil, err
	}
	results := make([]SendInputResult, len(targets))
	if len(targets) == 0 {
		return encodePayload(SendInputResponse{Results: results})
	}
	paneInfo := snapshotPaneInfo(sessions)
	action := resolveSendInputAction(req.Action)
	workers := scopeSendConcurrency(len(targets))
	jobs := make(chan scopeSendJob)
	responses := make(chan scopeSendResult, len(targets))
	for i := 0; i < workers; i++ {
		go func() {
			for job := range jobs {
				info, ok := paneInfo[job.PaneID]
				if !ok {
					responses <- scopeSendResult{Index: job.Index, PaneID: job.PaneID, Status: "failed", Message: "pane not found"}
					continue
				}
				plan, ok := buildToolSendPlan(reg, info, req, filter)
				if !ok {
					responses <- scopeSendResult{Index: job.Index, PaneID: job.PaneID, Status: "skipped", Message: "tool filter mismatch"}
					continue
				}
				status, message := sendToolInputWithTimeout(manager, job.PaneID, plan)
				if req.RecordAction {
					d.recordPaneAction(job.PaneID, action, req.Summary, "", status)
				}
				if req.DetectTool && status == "ok" {
					detectAndSetPaneTool(manager, reg, job.PaneID, req.Input)
				}
				responses <- scopeSendResult{Index: job.Index, PaneID: job.PaneID, Status: status, Message: message}
			}
		}()
	}
	for idx, target := range targets {
		jobs <- scopeSendJob{Index: idx, PaneID: target}
	}
	close(jobs)
	for i := 0; i < len(targets); i++ {
		res := <-responses
		results[res.Index] = SendInputResult{PaneID: res.PaneID, Status: res.Status, Message: res.Message}
	}
	return encodePayload(SendInputResponse{Results: results})
}

func buildToolSendPlan(reg *tool.Registry, info tool.PaneInfo, req SendInputToolRequest, filter string) (toolSendPlan, bool) {
	toolID := reg.ResolveTool(info)
	if filter != "" && toolID != filter {
		return toolSendPlan{}, false
	}
	profile := reg.Profile(toolID)
	if req.Raw {
		profile = reg.Profile("")
	}
	payload := tool.ApplyProfile(req.Input, profile, req.Raw)
	submit := []byte(nil)
	if req.Submit {
		submit = append([]byte(nil), profile.Submit...)
	}
	plan := toolSendPlan{
		Payload:     payload,
		Submit:      submit,
		SubmitDelay: profile.SubmitDelay,
		Combine:     profile.CombineSubmit,
		ToolID:      toolID,
	}
	if req.SubmitDelayMS != nil {
		plan.SubmitDelay = time.Duration(*req.SubmitDelayMS) * time.Millisecond
		if plan.SubmitDelay < 0 {
			plan.SubmitDelay = 0
		}
	}
	return plan, true
}

func sendToolInput(manager sessionManager, paneID string, plan toolSendPlan) (string, string) {
	return sendToolInputWithContext(manager, context.Background(), paneID, plan)
}

func sendToolInputWithContext(manager sessionManager, ctx context.Context, paneID string, plan toolSendPlan) (string, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	start := time.Time{}
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		start = time.Now()
		slog.Debug(
			"sessiond: send_tool start",
			slog.String("pane_id", paneID),
			slog.String("tool", plan.ToolID),
			slog.Int("bytes", len(plan.Payload)),
			slog.Int("submit_bytes", len(plan.Submit)),
			slog.Bool("combine", plan.Combine),
		)
	}
	if len(plan.Submit) > 0 && plan.Combine {
		combined := make([]byte, 0, len(plan.Payload)+len(plan.Submit))
		combined = append(combined, plan.Payload...)
		combined = append(combined, plan.Submit...)
		if err := manager.SendInput(ctx, paneID, combined); err != nil {
			if !start.IsZero() {
				slog.Debug(
					"sessiond: send_tool done",
					slog.String("pane_id", paneID),
					slog.String("tool", plan.ToolID),
					slog.Duration("dur", time.Since(start)),
					slog.Any("err", err),
				)
			}
			return sendInputError(ctx, err)
		}
		if !start.IsZero() {
			slog.Debug(
				"sessiond: send_tool done",
				slog.String("pane_id", paneID),
				slog.String("tool", plan.ToolID),
				slog.Duration("dur", time.Since(start)),
			)
		}
		return "ok", ""
	}
	if err := manager.SendInput(ctx, paneID, plan.Payload); err != nil {
		if !start.IsZero() {
			slog.Debug(
				"sessiond: send_tool done",
				slog.String("pane_id", paneID),
				slog.String("tool", plan.ToolID),
				slog.Duration("dur", time.Since(start)),
				slog.Any("err", err),
			)
		}
		return sendInputError(ctx, err)
	}
	if len(plan.Submit) == 0 {
		if !start.IsZero() {
			slog.Debug(
				"sessiond: send_tool done",
				slog.String("pane_id", paneID),
				slog.String("tool", plan.ToolID),
				slog.Duration("dur", time.Since(start)),
			)
		}
		return "ok", ""
	}
	if plan.SubmitDelay > 0 {
		if err := sleepWithContext(ctx, plan.SubmitDelay); err != nil {
			if !start.IsZero() {
				slog.Debug(
					"sessiond: send_tool done",
					slog.String("pane_id", paneID),
					slog.String("tool", plan.ToolID),
					slog.Duration("dur", time.Since(start)),
					slog.Any("err", err),
				)
			}
			return "timeout", "send timed out"
		}
	}
	if err := manager.SendInput(ctx, paneID, plan.Submit); err != nil {
		if !start.IsZero() {
			slog.Debug(
				"sessiond: send_tool done",
				slog.String("pane_id", paneID),
				slog.String("tool", plan.ToolID),
				slog.Duration("dur", time.Since(start)),
				slog.Any("err", err),
			)
		}
		return sendInputError(ctx, err)
	}
	if !start.IsZero() {
		slog.Debug(
			"sessiond: send_tool done",
			slog.String("pane_id", paneID),
			slog.String("tool", plan.ToolID),
			slog.Duration("dur", time.Since(start)),
		)
	}
	return "ok", ""
}

func sendToolInputWithTimeout(manager sessionManager, paneID string, plan toolSendPlan) (string, string) {
	timeout := scopeSendTimeout
	if plan.SubmitDelay > 0 && len(plan.Submit) > 0 && !plan.Combine {
		timeout += plan.SubmitDelay
	}
	if timeout <= 0 {
		return sendToolInput(manager, paneID, plan)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return sendToolInputWithContext(manager, ctx, paneID, plan)
}

func sendInputError(ctx context.Context, err error) (string, string) {
	if ctx != nil && ctx.Err() != nil {
		return "timeout", "send timed out"
	}
	return "failed", err.Error()
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	if ctx == nil {
		time.Sleep(delay)
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func detectAndSetPaneTool(manager sessionManager, reg *tool.Registry, paneID string, input []byte) {
	if manager == nil || reg == nil {
		return
	}
	toolID := reg.DetectFromInputBytes(input, limits.PayloadInspectLimit)
	if toolID == "" {
		return
	}
	_ = manager.SetPaneTool(paneID, toolID)
}

func (d *Daemon) lookupPaneInfo(manager sessionManager, paneID string) (tool.PaneInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := manager.Snapshot(ctx, 0)
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == paneID {
				return tool.PaneInfo{
					Tool:         pane.Tool,
					Command:      pane.Command,
					StartCommand: pane.StartCommand,
					Title:        pane.Title,
				}, nil
			}
		}
	}
	return tool.PaneInfo{}, fmt.Errorf("sessiond: pane %q not found", paneID)
}

func snapshotPaneInfo(sessions []native.SessionSnapshot) map[string]tool.PaneInfo {
	out := make(map[string]tool.PaneInfo, len(sessions)*2)
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == "" {
				continue
			}
			out[pane.ID] = tool.PaneInfo{
				Tool:         pane.Tool,
				Command:      pane.Command,
				StartCommand: pane.StartCommand,
				Title:        pane.Title,
			}
		}
	}
	return out
}
