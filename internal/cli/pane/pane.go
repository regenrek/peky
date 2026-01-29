package pane

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/transform"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/termrender"
)

// Register registers pane handlers.
func Register(reg *root.Registry) {
	reg.Register("pane.list", runList)
	reg.Register("pane.rename", runRename)
	reg.Register("pane.add", runAdd)
	reg.Register("pane.split", runSplit)
	reg.Register("pane.close", runClose)
	reg.Register("pane.swap", runSwap)
	reg.Register("pane.resize", runResize)
	reg.Register("pane.reset-sizes", runResetSizes)
	reg.Register("pane.zoom", runZoom)
	reg.Register("pane.send", runSend)
	reg.Register("pane.run", runRun)
	reg.Register("pane.view", runView)
	reg.Register("pane.tail", runTail)
	reg.Register("pane.snapshot", runSnapshot)
	reg.Register("pane.history", runHistory)
	reg.Register("pane.wait", runWait)
	reg.Register("pane.tag.add", runTagAdd)
	reg.Register("pane.tag.remove", runTagRemove)
	reg.Register("pane.tag.list", runTagList)
	reg.Register("pane.action", runAction)
	reg.Register("pane.key", runKey)
	reg.Register("pane.signal", runSignal)
	reg.Register("pane.focus", runFocus)
}

func runList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.list", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.SnapshotState(ctxTimeout, 0)
	if err != nil {
		return err
	}
	ws, err := transform.LoadWorkspace()
	if err != nil {
		return err
	}
	panes := transform.PaneList(resp.Sessions, ws, ctx.Cmd.String("session"))
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Panes []output.PaneSummaryWithContext `json:"panes"`
			Total int                             `json:"total"`
		}{Panes: panes, Total: len(panes)})
	}
	for _, pane := range panes {
		if err := writeLine(ctx.Out, pane.ID); err != nil {
			return err
		}
	}
	return nil
}

func runRename(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.rename", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	sessionName := strings.TrimSpace(ctx.Cmd.String("session"))
	paneIndex := intFlagString(ctx.Cmd, "index")
	newName := strings.TrimSpace(ctx.Cmd.String("name"))
	if newName == "" {
		return fmt.Errorf("pane name is required")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if paneID != "" {
		resolved, err := resolvePaneID(ctxTimeout, client, paneID)
		if err != nil {
			return err
		}
		if err := client.RenamePaneByID(ctxTimeout, resolved, newName); err != nil {
			return err
		}
	} else {
		if err := client.RenamePane(ctxTimeout, sessionName, paneIndex, newName); err != nil {
			return err
		}
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		target := paneID
		if target == "" {
			target = fmt.Sprintf("%s:%s", sessionName, paneIndex)
		} else if resolved, err := resolvePaneID(ctxTimeout, client, paneID); err == nil {
			target = resolved
		}
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.rename",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: target}},
		})
	}
	if paneID != "" {
		if resolved, err := resolvePaneID(ctxTimeout, client, paneID); err == nil {
			paneID = resolved
		}
		return writef(ctx.Out, "Renamed pane %s\n", paneID)
	}
	return writef(ctx.Out, "Renamed pane %s:%s\n", sessionName, paneIndex)
}

func runSplit(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.split", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}
	sessionName := ctx.Cmd.String("session")
	paneIndex := intFlagString(ctx.Cmd, "index")
	orientation := strings.ToLower(strings.TrimSpace(ctx.Cmd.String("orientation")))
	vertical := orientation == "vertical"
	percent := ctx.Cmd.Int("percent")
	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	before, err := captureLayoutBefore(snapCtx, client, layoutMode, sessionName)
	cancel()
	if err != nil {
		return err
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	newIndex, _, err := client.SplitPane(ctxTimeout, sessionName, paneIndex, vertical, percent)
	if err != nil {
		return err
	}
	var focusedPaneID string
	if ctx.Cmd.Bool("focus") {
		focusedPaneID, err = focusPaneByIndex(ctxTimeout, client, sessionName, newIndex)
		if err != nil {
			return err
		}
	}
	if ctx.JSON {
		snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		after, err := captureLayoutAfter(snapCtx, client, layoutMode, sessionName)
		cancel()
		if err != nil {
			return err
		}
		meta = output.WithDuration(meta, start)
		details := map[string]any{
			"new_index": newIndex,
		}
		if focusedPaneID != "" {
			details["focused_pane_id"] = focusedPaneID
		}
		result := output.ActionResult{
			Action:  "pane.split",
			Status:  "ok",
			Details: details,
		}
		if layoutMode != layoutOutputNone {
			result.Layout = buildLayoutState(sessionName, "", nil, before, after)
		}
		return output.WriteSuccess(ctx.Out, meta, result)
	}
	return writef(ctx.Out, "Split pane created %s\n", newIndex)
}

func runClose(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.close", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}
	opts, err := parsePaneCloseOptions(ctx)
	if err != nil {
		return err
	}
	if opts.scope != "" {
		if layoutMode != layoutOutputNone {
			return fmt.Errorf("layout output not supported with scope closes")
		}
		return closePaneScope(ctx, client, opts.scope, start, meta)
	}
	sessionName, paneID, err := resolveCloseTarget(ctx, client, layoutMode, opts)
	if err != nil {
		return err
	}
	before, err := captureBeforeLayout(ctx, client, layoutMode, sessionName)
	if err != nil {
		return err
	}
	if err := closePaneByTarget(ctx, client, sessionName, paneID, opts.paneIndex); err != nil {
		return err
	}
	if ctx.JSON {
		return writeCloseJSON(ctx, client, meta, start, layoutMode, sessionName, paneID, opts.paneIndex, before)
	}
	return writeCloseText(ctx, sessionName, paneID, opts.paneIndex)
}

func resolveCloseTarget(ctx root.CommandContext, client *sessiond.Client, layoutMode layoutOutputMode, opts paneCloseOptions) (string, string, error) {
	sessionName := opts.sessionName
	paneID := opts.paneID
	if paneID == "" {
		return sessionName, "", nil
	}
	resolveCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if layoutMode != layoutOutputNone {
		sessionName, paneID, err := resolvePaneSession(resolveCtx, client, paneID)
		return sessionName, paneID, err
	}
	resolved, err := resolvePaneID(resolveCtx, client, paneID)
	return sessionName, resolved, err
}

func captureBeforeLayout(ctx root.CommandContext, client *sessiond.Client, layoutMode layoutOutputMode, sessionName string) (*layout.TreeSnapshot, error) {
	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	return captureLayoutBefore(snapCtx, client, layoutMode, sessionName)
}

func captureAfterLayout(ctx root.CommandContext, client *sessiond.Client, layoutMode layoutOutputMode, sessionName string) (*layout.TreeSnapshot, error) {
	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	return captureLayoutAfter(snapCtx, client, layoutMode, sessionName)
}

func closePaneByTarget(ctx root.CommandContext, client *sessiond.Client, sessionName, paneID, paneIndex string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if paneID != "" {
		return client.ClosePaneByID(ctxTimeout, paneID)
	}
	return client.ClosePane(ctxTimeout, sessionName, paneIndex)
}

func writeCloseJSON(ctx root.CommandContext, client *sessiond.Client, meta output.Meta, start time.Time, layoutMode layoutOutputMode, sessionName, paneID, paneIndex string, before *layout.TreeSnapshot) error {
	after, err := captureAfterLayout(ctx, client, layoutMode, sessionName)
	if err != nil {
		return err
	}
	meta = output.WithDuration(meta, start)
	target := paneID
	if target == "" {
		target = fmt.Sprintf("%s:%s", sessionName, paneIndex)
	}
	result := output.ActionResult{
		Action:  "pane.close",
		Status:  "ok",
		Targets: []output.TargetRef{{Type: "pane", ID: target}},
	}
	if layoutMode != layoutOutputNone {
		result.Layout = buildLayoutState(sessionName, paneID, nil, before, after)
	}
	return output.WriteSuccess(ctx.Out, meta, result)
}

func writeCloseText(ctx root.CommandContext, sessionName, paneID, paneIndex string) error {
	if paneID != "" {
		return writef(ctx.Out, "Closed pane %s\n", paneID)
	}
	return writef(ctx.Out, "Closed pane %s:%s\n", sessionName, paneIndex)
}

type paneCloseOptions struct {
	paneID      string
	sessionName string
	paneIndex   string
	scope       string
}

func parsePaneCloseOptions(ctx root.CommandContext) (paneCloseOptions, error) {
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	sessionName := ctx.Cmd.String("session")
	paneIndex := intFlagString(ctx.Cmd, "index")
	scope := strings.TrimSpace(ctx.Cmd.String("scope"))
	if paneID != "" && (sessionName != "" || paneIndex != "") {
		return paneCloseOptions{}, fmt.Errorf("pane-id cannot be combined with session or index")
	}
	if scope != "" {
		if paneID != "" || sessionName != "" || paneIndex != "" {
			return paneCloseOptions{}, fmt.Errorf("scope cannot be combined with pane-id or session/index")
		}
		if !ctx.Cmd.Bool("all") {
			return paneCloseOptions{}, fmt.Errorf("scope close requires --all")
		}
		return paneCloseOptions{scope: scope}, nil
	}
	if paneID == "" && sessionName == "" {
		return paneCloseOptions{}, fmt.Errorf("pane-id or session is required")
	}
	if sessionName != "" && paneIndex == "" {
		return paneCloseOptions{}, fmt.Errorf("session requires index")
	}
	return paneCloseOptions{
		paneID:      paneID,
		sessionName: sessionName,
		paneIndex:   paneIndex,
	}, nil
}

func runSwap(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.swap", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}
	sessionName := ctx.Cmd.String("session")
	paneA := intFlagString(ctx.Cmd, "a")
	paneB := intFlagString(ctx.Cmd, "b")
	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	before, err := captureLayoutBefore(snapCtx, client, layoutMode, sessionName)
	cancel()
	if err != nil {
		return err
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.SwapPanes(ctxTimeout, sessionName, paneA, paneB); err != nil {
		return err
	}
	if ctx.JSON {
		snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		after, err := captureLayoutAfter(snapCtx, client, layoutMode, sessionName)
		cancel()
		if err != nil {
			return err
		}
		meta = output.WithDuration(meta, start)
		result := output.ActionResult{
			Action: "pane.swap",
			Status: "ok",
		}
		if layoutMode != layoutOutputNone {
			result.Layout = buildLayoutState(sessionName, "", nil, before, after)
		}
		return output.WriteSuccess(ctx.Out, meta, result)
	}
	return writef(ctx.Out, "Swapped panes %s:%s and %s\n", sessionName, paneA, paneB)
}

func runResize(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.resize", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}
	paneID := ctx.Cmd.String("pane-id")
	edge, err := parseResizeEdge(ctx.Cmd.String("edge"))
	if err != nil {
		return err
	}
	delta := ctx.Cmd.Int("delta")
	snap := ctx.Cmd.Bool("snap")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	sessionName, resolved, err := resolvePaneSession(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	paneID = resolved
	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	before, err := captureLayoutBefore(snapCtx, client, layoutMode, sessionName)
	cancel()
	if err != nil {
		return err
	}
	opResp, err := client.ResizePaneEdge(ctxTimeout, sessionName, paneID, edge, delta, snap, sessiond.SnapState{})
	if err != nil {
		return err
	}
	if ctx.JSON {
		snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		after, err := captureLayoutAfter(snapCtx, client, layoutMode, sessionName)
		cancel()
		if err != nil {
			return err
		}
		meta = output.WithDuration(meta, start)
		result := output.ActionResult{
			Action:  "pane.resize",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
		}
		if layoutMode != layoutOutputNone {
			result.Layout = buildLayoutState(sessionName, paneID, &opResp, before, after)
		}
		return output.WriteSuccess(ctx.Out, meta, result)
	}
	return writef(ctx.Out, "Resized pane %s\n", paneID)
}

func parseResizeEdge(value string) (sessiond.ResizeEdge, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(sessiond.ResizeEdgeLeft):
		return sessiond.ResizeEdgeLeft, nil
	case string(sessiond.ResizeEdgeRight):
		return sessiond.ResizeEdgeRight, nil
	case string(sessiond.ResizeEdgeUp):
		return sessiond.ResizeEdgeUp, nil
	case string(sessiond.ResizeEdgeDown):
		return sessiond.ResizeEdgeDown, nil
	default:
		return "", fmt.Errorf("invalid resize edge %q", value)
	}
}

func runResetSizes(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.reset-sizes", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}
	sessionName := strings.TrimSpace(ctx.Cmd.String("session"))
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	sessionName, paneID, err = resolveResetSizesTarget(ctxTimeout, client, sessionName, paneID)
	if err != nil {
		return err
	}
	before, err := captureBeforeLayout(ctx, client, layoutMode, sessionName)
	if err != nil {
		return err
	}
	opResp, err := client.ResetPaneSizes(ctxTimeout, sessionName, paneID)
	if err != nil {
		return err
	}
	if ctx.JSON {
		return writeResetSizesJSON(ctx, client, meta, start, layoutMode, sessionName, paneID, before, opResp)
	}
	return writeResetSizesText(ctx, sessionName, paneID)
}

func resolveResetSizesTarget(ctxTimeout context.Context, client *sessiond.Client, sessionName, paneID string) (string, string, error) {
	if paneID != "" && sessionName != "" {
		return "", "", fmt.Errorf("pane-id cannot be combined with session")
	}
	if paneID == "" {
		if sessionName == "" {
			return "", "", fmt.Errorf("session is required")
		}
		return sessionName, "", nil
	}
	resolvedSession, resolvedPane, err := resolvePaneSession(ctxTimeout, client, paneID)
	if err != nil {
		return "", "", err
	}
	return resolvedSession, resolvedPane, nil
}

func writeResetSizesJSON(ctx root.CommandContext, client *sessiond.Client, meta output.Meta, start time.Time, layoutMode layoutOutputMode, sessionName, paneID string, before *layout.TreeSnapshot, opResp sessiond.LayoutOpResponse) error {
	after, err := captureAfterLayout(ctx, client, layoutMode, sessionName)
	if err != nil {
		return err
	}
	meta = output.WithDuration(meta, start)
	details := map[string]any{"session": sessionName}
	if paneID != "" {
		details["pane_id"] = paneID
	}
	result := output.ActionResult{
		Action:  "pane.reset-sizes",
		Status:  "ok",
		Details: details,
	}
	if layoutMode != layoutOutputNone {
		result.Layout = buildLayoutState(sessionName, paneID, &opResp, before, after)
	}
	return output.WriteSuccess(ctx.Out, meta, result)
}

func writeResetSizesText(ctx root.CommandContext, sessionName, paneID string) error {
	if paneID != "" {
		return writef(ctx.Out, "Reset sizes for pane %s\n", paneID)
	}
	return writef(ctx.Out, "Reset sizes for session %s\n", sessionName)
}

func runZoom(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.zoom", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	sessionName := strings.TrimSpace(ctx.Cmd.String("session"))
	toggle := ctx.Cmd.Bool("toggle")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()

	if paneID == "" {
		return fmt.Errorf("pane id is required")
	}
	if sessionName == "" {
		resolvedSession, resolvedPane, err := resolvePaneSession(ctxTimeout, client, paneID)
		if err != nil {
			return err
		}
		sessionName = resolvedSession
		paneID = resolvedPane
	} else {
		resolved, err := resolvePaneID(ctxTimeout, client, paneID)
		if err != nil {
			return err
		}
		paneID = resolved
	}

	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	before, err := captureLayoutBefore(snapCtx, client, layoutMode, sessionName)
	cancel()
	if err != nil {
		return err
	}
	opResp, err := client.ZoomPane(ctxTimeout, sessionName, paneID, toggle)
	if err != nil {
		return err
	}
	if ctx.JSON {
		snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		after, err := captureLayoutAfter(snapCtx, client, layoutMode, sessionName)
		cancel()
		if err != nil {
			return err
		}
		meta = output.WithDuration(meta, start)
		result := output.ActionResult{
			Action:  "pane.zoom",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
		}
		if layoutMode != layoutOutputNone {
			result.Layout = buildLayoutState(sessionName, paneID, &opResp, before, after)
		}
		return output.WriteSuccess(ctx.Out, meta, result)
	}
	return writef(ctx.Out, "Zoom toggled for pane %s\n", paneID)
}

func runSend(ctx root.CommandContext) error {
	return runSendLike(ctx, false)
}

func runRun(ctx root.CommandContext) error {
	return runSendLike(ctx, true)
}

func runSendLike(ctx root.CommandContext, withNewline bool) error {
	start := time.Now()
	cmd := sendCommandMeta(withNewline)
	meta := output.NewMeta(cmd.ID, ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	payload, err := readPayload(ctx, withNewline)
	if err != nil {
		return err
	}
	proceed, err := confirmSendLike(ctx, withNewline)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}
	applySendDelay(ctx)
	target, err := resolveSendTarget(ctx)
	if err != nil {
		return err
	}
	results, warnings, err := sendPayloadToTarget(ctx, client, target, payload, cmd.Action, withNewline)
	if err != nil {
		return err
	}
	return writeSendLikeOutput(ctx, meta, start, cmd.ID, results, warnings)
}

type sendCommand struct {
	ID     string
	Action string
}

func sendCommandMeta(withNewline bool) sendCommand {
	if withNewline {
		return sendCommand{ID: "pane.run", Action: "run"}
	}
	return sendCommand{ID: "pane.send", Action: "send"}
}

func confirmSendLike(ctx root.CommandContext, withNewline bool) (bool, error) {
	if !withNewline {
		return true, nil
	}
	if ctx.Cmd.Bool("confirm") {
		ok, err := root.PromptConfirm(ctx.Stdin, ctx.ErrOut, "Confirm pane.run")
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	if ctx.Cmd.Bool("require-ack") {
		if err := requireAck(ctx.Stdin, ctx.ErrOut); err != nil {
			return false, err
		}
	}
	return true, nil
}

func applySendDelay(ctx root.CommandContext) {
	if delay := ctx.Cmd.Duration("delay"); delay > 0 {
		time.Sleep(delay)
	}
}

type sendTarget struct {
	PaneID string
	Scope  string
}

func resolveSendTarget(ctx root.CommandContext) (sendTarget, error) {
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	scope := strings.TrimSpace(ctx.Cmd.String("scope"))
	if paneID == "" && scope == "" {
		return sendTarget{}, fmt.Errorf("pane-id or scope is required")
	}
	if paneID != "" {
		return sendTarget{PaneID: paneID}, nil
	}
	return sendTarget{Scope: scope}, nil
}

func sendPayloadToTarget(ctx root.CommandContext, client *sessiond.Client, target sendTarget, payload payloadData, action string, withNewline bool) ([]output.TargetResult, []string, error) {
	if target.PaneID != "" {
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		resolved, err := resolvePaneID(ctxTimeout, client, target.PaneID)
		cancel()
		if err != nil {
			return nil, nil, err
		}
		target.PaneID = resolved
	}
	var submitDelayMS *int
	submitDelay := time.Duration(0)
	if withNewline && ctx.Cmd.IsSet("submit-delay") {
		submitDelay = ctx.Cmd.Duration("submit-delay")
		ms := int(submitDelay / time.Millisecond)
		submitDelayMS = &ms
	}
	req := sessiond.SendInputToolRequest{
		PaneID:        target.PaneID,
		Scope:         target.Scope,
		Input:         payload.Data,
		RecordAction:  true,
		Action:        action,
		Summary:       payload.Summary,
		Submit:        withNewline,
		SubmitDelayMS: submitDelayMS,
		Raw:           ctx.Cmd.Bool("raw"),
		ToolFilter:    ctx.Cmd.String("tool"),
		DetectTool:    withNewline,
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	resp, err := client.SendInputTool(ctxTimeout, req)
	cancel()
	if err != nil {
		return nil, nil, err
	}
	results := mapSendResults(resp)
	return results, sendWarnings(withNewline, submitDelay), nil
}

func sendWarnings(withNewline bool, submitDelay time.Duration) []string {
	if withNewline && submitDelay > 0 {
		return []string{"submit_delay_applied"}
	}
	return nil
}

func writeSendLikeOutput(ctx root.CommandContext, meta output.Meta, start time.Time, cmdID string, results []output.TargetResult, warnings []string) error {
	status := actionStatus(results)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:   cmdID,
			Status:   status,
			Results:  results,
			Warnings: warnings,
		})
	}
	return writef(ctx.Out, "Sent to %d pane(s)\n", len(results))
}

func runView(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.view", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	rows := ctx.Cmd.Int("rows")
	cols := ctx.Cmd.Int("cols")
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	mode := strings.ToLower(strings.TrimSpace(ctx.Cmd.String("mode")))
	if mode == "" {
		mode = "ansi"
	}
	switch mode {
	case "plain", "ansi":
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
	req := sessiond.PaneViewRequest{PaneID: paneID, Rows: rows, Cols: cols}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	req.PaneID = resolved
	resp, err := client.GetPaneView(ctxTimeout, req)
	if err != nil {
		return err
	}
	content := termrender.Render(resp.Frame, termrender.Options{
		Profile:    colorprofile.Detect(ctx.Out, os.Environ()),
		ShowCursor: false,
	})
	if mode == "plain" {
		content = ansi.Strip(content)
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.PaneView{
			PaneID:    resp.PaneID,
			Mode:      mode,
			Rows:      resp.Rows,
			Cols:      resp.Cols,
			Content:   content,
			Truncated: false,
		})
	}
	return write(ctx.Out, content)
}

func runTail(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.tail", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	opts, err := parseTailOptions(ctx)
	if err != nil {
		return err
	}
	if opts.PaneID != "" {
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		resolved, err := resolvePaneID(ctxTimeout, client, opts.PaneID)
		cancel()
		if err != nil {
			return err
		}
		opts.PaneID = resolved
	}
	if err := tailLoop(ctx, client, opts); err != nil {
		return err
	}
	if ctx.JSON {
		_ = output.WithDuration(meta, start)
	}
	return nil
}

type tailOptions struct {
	PaneID string
	Follow bool
	Limit  int
	Regex  *regexp.Regexp
	Since  time.Time
	Until  time.Time
}

func parseTailOptions(ctx root.CommandContext) (tailOptions, error) {
	follow := ctx.Cmd.Bool("follow")
	if !ctx.Cmd.IsSet("follow") {
		follow = true
	}
	re, err := parseTailRegex(ctx.Cmd.String("grep"))
	if err != nil {
		return tailOptions{}, err
	}
	now := time.Now().UTC()
	since, err := parseTimeOrDuration(ctx.Cmd.String("since"), now, false)
	if err != nil {
		return tailOptions{}, err
	}
	until, err := parseTimeOrDuration(ctx.Cmd.String("until"), now, true)
	if err != nil {
		return tailOptions{}, err
	}
	return tailOptions{
		PaneID: ctx.Cmd.String("pane-id"),
		Follow: follow,
		Limit:  ctx.Cmd.Int("lines"),
		Regex:  re,
		Since:  since,
		Until:  until,
	}, nil
}

func parseTailRegex(value string) (*regexp.Regexp, error) {
	grep := strings.TrimSpace(value)
	if grep == "" {
		return nil, nil
	}
	return regexp.Compile(grep)
}

func tailLoop(ctx root.CommandContext, client *sessiond.Client, opts tailOptions) error {
	seq := uint64(0)
	streamSeq := int64(0)
	for {
		opts = stopTailIfUntilReached(opts)
		resp, err := fetchPaneOutput(ctx, client, opts, seq)
		if err != nil {
			return err
		}
		seq = resp.NextSeq
		filtered := filterOutputLines(resp.Lines, opts.Since, opts.Until, opts.Regex)
		if ctx.JSON {
			if err := emitTailJSON(ctx, opts.PaneID, filtered, resp.Truncated, opts.Follow, &streamSeq); err != nil {
				return err
			}
		} else if err := emitTailText(ctx, filtered); err != nil {
			return err
		}
		if !opts.Follow {
			return nil
		}
	}
}

func stopTailIfUntilReached(opts tailOptions) tailOptions {
	if !opts.Until.IsZero() && time.Now().UTC().After(opts.Until) {
		opts.Follow = false
	}
	return opts
}

func fetchPaneOutput(ctx root.CommandContext, client *sessiond.Client, opts tailOptions, seq uint64) (sessiond.PaneOutputResponse, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	resp, err := client.PaneOutput(ctxTimeout, sessiond.PaneOutputRequest{
		PaneID:   opts.PaneID,
		SinceSeq: seq,
		Limit:    opts.Limit,
		Wait:     opts.Follow,
	})
	cancel()
	return resp, err
}

func emitTailJSON(ctx root.CommandContext, paneID string, lines []native.OutputLine, truncated bool, follow bool, streamSeq *int64) error {
	for i, line := range lines {
		frame := output.PaneTailFrame{
			PaneID:    paneID,
			Chunk:     line.Text + "\n",
			Encoding:  "utf-8",
			Truncated: truncated && i == 0,
		}
		*streamSeq++
		metaFrame := output.NewStreamMeta("pane.tail", ctx.Deps.Version, *streamSeq, false)
		if err := output.WriteSuccess(ctx.Out, metaFrame, frame); err != nil {
			return err
		}
	}
	if follow {
		return nil
	}
	*streamSeq++
	metaFrame := output.NewStreamMeta("pane.tail", ctx.Deps.Version, *streamSeq, true)
	return output.WriteSuccess(ctx.Out, metaFrame, output.PaneTailFrame{PaneID: paneID, Chunk: "", Encoding: "utf-8"})
}

func emitTailText(ctx root.CommandContext, lines []native.OutputLine) error {
	for _, line := range lines {
		if err := writeLine(ctx.Out, line.Text); err != nil {
			return err
		}
	}
	return nil
}

func runSnapshot(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.snapshot", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	rows := ctx.Cmd.Int("rows")
	if rows <= 0 {
		rows = 200
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	resp, err := client.PaneSnapshot(ctxTimeout, resolved, rows)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.PaneSnapshotOutput{
			PaneID:    resp.PaneID,
			Rows:      resp.Rows,
			Content:   resp.Content,
			Truncated: resp.Truncated,
		})
	}
	return write(ctx.Out, resp.Content)
}

func runHistory(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.history", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	limit := ctx.Cmd.Int("limit")
	since := strings.TrimSpace(ctx.Cmd.String("since"))
	var sinceTime time.Time
	if since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return err
		}
		sinceTime = t
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	resp, err := client.PaneHistory(ctxTimeout, sessiond.PaneHistoryRequest{PaneID: resolved, Limit: limit, Since: sinceTime})
	if err != nil {
		return err
	}
	entries := make([]output.PaneHistoryEntry, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		entries = append(entries, output.PaneHistoryEntry{
			TS:      entry.TS,
			Action:  entry.Action,
			Summary: entry.Summary,
			Command: entry.Command,
			Status:  entry.Status,
		})
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.PaneHistory{PaneID: resp.PaneID, Entries: entries})
	}
	for _, entry := range entries {
		if err := writef(ctx.Out, "%s\t%s\t%s\n", entry.TS.Format(time.RFC3339), entry.Action, entry.Summary); err != nil {
			return err
		}
	}
	return nil
}

func runWait(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.wait", ctx.Deps.Version)
	pattern := ctx.Cmd.String("for")
	if pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return err
	}
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	timeout := ctx.Cmd.Duration("timeout")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, ctx.Cmd.String("pane-id"))
	if err != nil {
		return err
	}
	resp, err := client.PaneWait(ctxTimeout, sessiond.PaneWaitRequest{PaneID: resolved, Pattern: pattern, Timeout: timeout})
	if err != nil {
		return err
	}
	result := output.PaneWaitResult{
		PaneID:    resp.PaneID,
		Pattern:   resp.Pattern,
		Matched:   resp.Matched,
		Match:     resp.Match,
		ElapsedMS: resp.Elapsed.Milliseconds(),
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, result)
	}
	if !resp.Matched {
		return fmt.Errorf("pattern not matched")
	}
	return writef(ctx.Out, "%s\n", resp.Match)
}

func runTagAdd(ctx root.CommandContext) error {
	return runTagMutation(ctx, "pane.tag.add", true)
}

func runTagRemove(ctx root.CommandContext) error {
	return runTagMutation(ctx, "pane.tag.remove", false)
}

func runTagMutation(ctx root.CommandContext, action string, add bool) error {
	start := time.Now()
	meta := output.NewMeta(action, ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	tags := ctx.Cmd.StringSlice("tag")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	if add {
		_, err = client.AddPaneTags(ctxTimeout, resolved, tags)
	} else {
		_, err = client.RemovePaneTags(ctxTimeout, resolved, tags)
	}
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  action,
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: resolved}},
		})
	}
	return nil
}

func runTagList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.tag.list", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	tags, err := client.PaneTags(ctxTimeout, resolved)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.PaneTagList{PaneID: resolved, Tags: tags})
	}
	for _, tag := range tags {
		if err := writeLine(ctx.Out, tag); err != nil {
			return err
		}
	}
	return nil
}

func runAction(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.action", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	actionName := ctx.Cmd.String("action")
	action, err := parseTerminalAction(actionName)
	if err != nil {
		return err
	}
	input := parseTerminalActionInput(ctx, action)
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	resp, err := client.TerminalAction(ctxTimeout, sessiond.TerminalActionRequest{
		PaneID: resolved,
		Action: action,
		Lines:  input.Lines,
		DeltaX: input.DeltaX,
		DeltaY: input.DeltaY,
	})
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.action",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: resolved}},
			Details: buildTerminalActionDetails(actionName, input, resp.Text),
		})
	}
	if resp.Text != "" {
		if err := writeLine(ctx.Out, resp.Text); err != nil {
			return err
		}
	}
	return nil
}

type terminalActionInput struct {
	Lines  int
	DeltaX int
	DeltaY int
}

func parseTerminalActionInput(ctx root.CommandContext, action sessiond.TerminalAction) terminalActionInput {
	lines := ctx.Cmd.Int("lines")
	if lines == 0 {
		lines = ctx.Cmd.Int("count")
	}
	deltaX := ctx.Cmd.Int("delta-x")
	deltaY := ctx.Cmd.Int("delta-y")
	if deltaX == 0 && deltaY == 0 && action == sessiond.TerminalCopyMove && lines != 0 {
		deltaY = lines
	}
	lines = defaultActionLines(action, lines)
	return terminalActionInput{
		Lines:  lines,
		DeltaX: deltaX,
		DeltaY: deltaY,
	}
}

func buildTerminalActionDetails(actionName string, input terminalActionInput, text string) map[string]any {
	details := map[string]any{
		"action": actionName,
	}
	if input.Lines != 0 {
		details["lines"] = input.Lines
	}
	if input.DeltaX != 0 {
		details["delta_x"] = input.DeltaX
	}
	if input.DeltaY != 0 {
		details["delta_y"] = input.DeltaY
	}
	if text != "" {
		details["text"] = text
	}
	return details
}

func runKey(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.key", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	key := strings.TrimSpace(ctx.Cmd.String("key"))
	mods := ctx.Cmd.StringSlice("mods")
	fullKey, err := buildKeyWithMods(key, mods)
	if err != nil {
		return err
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	resp, err := client.HandleTerminalKey(ctxTimeout, sessiond.TerminalKeyRequest{
		PaneID:           resolved,
		Key:              fullKey,
		ScrollbackToggle: ctx.Cmd.Bool("scrollback-toggle"),
		CopyToggle:       ctx.Cmd.Bool("copy-toggle"),
	})
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		details := map[string]any{
			"handled": resp.Handled,
		}
		if resp.Toast != "" {
			details["toast"] = resp.Toast
			details["toast_kind"] = toastKindString(resp.ToastKind)
		}
		if resp.YankText != "" {
			details["yank_text"] = resp.YankText
		}
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.key",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: resolved}},
			Details: details,
		})
	}
	if resp.Toast != "" {
		if err := writeLine(ctx.Out, resp.Toast); err != nil {
			return err
		}
	}
	if resp.YankText != "" {
		if err := writeLine(ctx.Out, resp.YankText); err != nil {
			return err
		}
	}
	return nil
}

func runSignal(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.signal", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	signal := strings.TrimSpace(ctx.Cmd.String("signal"))
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	if err := client.SignalPane(ctxTimeout, resolved, signal); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.signal",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: resolved}},
			Details: map[string]any{"signal": signal},
		})
	}
	return nil
}

func runFocus(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.focus", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	paneID := ctx.Cmd.String("pane-id")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resolved, err := resolvePaneID(ctxTimeout, client, paneID)
	if err != nil {
		return err
	}
	if err := client.FocusPane(ctxTimeout, resolved); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.focus",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: resolved}},
		})
	}
	return nil
}

func connect(ctx root.CommandContext) (*sessiond.Client, func(), error) {
	connect := ctx.Deps.Connect
	if connect == nil {
		return nil, func() {}, fmt.Errorf("daemon connection not configured")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	client, err := connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	cleanup := func() {
		cancel()
		_ = client.Close()
	}
	return client, cleanup, nil
}

func commandTimeout(ctx root.CommandContext) time.Duration {
	if ctx.Cmd.IsSet("timeout") {
		return ctx.Cmd.Duration("timeout")
	}
	return 10 * time.Second
}

type payloadData struct {
	Data    []byte
	Summary string
}

func readPayload(ctx root.CommandContext, withNewline bool) (payloadData, error) {
	if ctx.Cmd.Bool("stdin") {
		data, err := io.ReadAll(ctx.Stdin)
		if err != nil {
			return payloadData{}, err
		}
		return payloadData{Data: trimTrailingNewline(data), Summary: summarizePayload(data)}, nil
	}
	if path := strings.TrimSpace(ctx.Cmd.String("file")); path != "" {
		clean, err := safePath(path)
		if err != nil {
			return payloadData{}, err
		}
		data, err := os.ReadFile(clean)
		if err != nil {
			return payloadData{}, err
		}
		return payloadData{Data: trimTrailingNewline(data), Summary: summarizePayload(data)}, nil
	}
	text := strings.TrimSpace(ctx.Cmd.String("text"))
	if text == "" && withNewline {
		text = strings.TrimSpace(ctx.Cmd.String("command"))
	}
	if text == "" {
		return payloadData{}, fmt.Errorf("payload is required")
	}
	data := []byte(text)
	return payloadData{Data: data, Summary: summarizePayload(data)}, nil
}

func trimTrailingNewline(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	if data[len(data)-1] == '\n' {
		return data[:len(data)-1]
	}
	return data
}

func summarizePayload(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	limit := limits.PayloadInspectLimit
	if limit <= 0 {
		return ""
	}
	if limit > len(data) {
		limit = len(data)
	}
	trimmed := strings.TrimSpace(string(data[:limit]))
	if trimmed == "" {
		return ""
	}
	if len(trimmed) > 120 {
		return trimmed[:117] + "..."
	}
	return trimmed
}

func safePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func mapSendResults(resp sessiond.SendInputResponse) []output.TargetResult {
	results := make([]output.TargetResult, 0, len(resp.Results))
	for _, res := range resp.Results {
		status := res.Status
		if status == "" {
			status = "ok"
		}
		results = append(results, output.TargetResult{
			Target:  output.TargetRef{Type: "pane", ID: res.PaneID},
			Status:  status,
			Message: res.Message,
		})
	}
	return results
}

func actionStatus(results []output.TargetResult) string {
	if len(results) == 0 {
		return "ok"
	}
	okCount := 0
	for _, res := range results {
		if res.Status == "ok" || res.Status == "skipped" {
			okCount++
		}
	}
	if okCount == len(results) {
		return "ok"
	}
	if okCount == 0 {
		return "failed"
	}
	return "partial"
}

func parseTimeOrDuration(value string, now time.Time, future bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		if future {
			return now.Add(d), nil
		}
		return now.Add(-d), nil
	}
	return time.Parse(time.RFC3339, value)
}

func filterOutputLines(lines []native.OutputLine, since, until time.Time, re *regexp.Regexp) []native.OutputLine {
	if len(lines) == 0 {
		return nil
	}
	filtered := make([]native.OutputLine, 0, len(lines))
	for _, line := range lines {
		if !since.IsZero() && line.TS.Before(since) {
			continue
		}
		if !until.IsZero() && line.TS.After(until) {
			continue
		}
		if re != nil && !re.MatchString(line.Text) {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func parseTerminalAction(value string) (sessiond.TerminalAction, error) {
	normalized := normalizeTerminalAction(value)
	if normalized == "" {
		return sessiond.TerminalActionUnknown, errors.New("action is required")
	}
	if action, ok := terminalActionAliases[normalized]; ok {
		return action, nil
	}
	return sessiond.TerminalActionUnknown, fmt.Errorf("unknown action %q", value)
}

func normalizeTerminalAction(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}

var terminalActionAliases = map[string]sessiond.TerminalAction{
	"enter_scrollback":      sessiond.TerminalEnterScrollback,
	"enter_scrollback_mode": sessiond.TerminalEnterScrollback,
	"exit_scrollback":       sessiond.TerminalExitScrollback,
	"exit_scrollback_mode":  sessiond.TerminalExitScrollback,
	"scroll_up":             sessiond.TerminalScrollUp,
	"scrollup":              sessiond.TerminalScrollUp,
	"scroll_down":           sessiond.TerminalScrollDown,
	"scrolldown":            sessiond.TerminalScrollDown,
	"page_up":               sessiond.TerminalPageUp,
	"pageup":                sessiond.TerminalPageUp,
	"page_down":             sessiond.TerminalPageDown,
	"pagedown":              sessiond.TerminalPageDown,
	"scroll_top":            sessiond.TerminalScrollTop,
	"scrolltop":             sessiond.TerminalScrollTop,
	"scroll_bottom":         sessiond.TerminalScrollBottom,
	"scrollbottom":          sessiond.TerminalScrollBottom,
	"enter_copy":            sessiond.TerminalEnterCopyMode,
	"enter_copy_mode":       sessiond.TerminalEnterCopyMode,
	"copy_mode":             sessiond.TerminalEnterCopyMode,
	"exit_copy":             sessiond.TerminalExitCopyMode,
	"exit_copy_mode":        sessiond.TerminalExitCopyMode,
	"copy_move":             sessiond.TerminalCopyMove,
	"copy_move_cursor":      sessiond.TerminalCopyMove,
	"copy_page_up":          sessiond.TerminalCopyPageUp,
	"copy_pageup":           sessiond.TerminalCopyPageUp,
	"copy_page_down":        sessiond.TerminalCopyPageDown,
	"copy_pagedown":         sessiond.TerminalCopyPageDown,
	"copy_toggle_select":    sessiond.TerminalCopyToggleSelect,
	"copy_toggle":           sessiond.TerminalCopyToggleSelect,
	"copy_yank":             sessiond.TerminalCopyYank,
	"copy":                  sessiond.TerminalCopyYank,
}

func defaultActionLines(action sessiond.TerminalAction, value int) int {
	if value != 0 {
		return value
	}
	switch action {
	case sessiond.TerminalScrollUp, sessiond.TerminalScrollDown:
		return 1
	default:
		return 0
	}
}

func buildKeyWithMods(key string, mods []string) (string, error) {
	if key == "" {
		return "", errors.New("key is required")
	}
	if len(mods) == 0 {
		return key, nil
	}
	normalized := make([]string, 0, len(mods))
	for _, mod := range mods {
		mod = strings.ToLower(strings.TrimSpace(mod))
		if mod == "" {
			continue
		}
		switch mod {
		case "ctrl", "control":
			mod = "ctrl"
		case "shift":
		case "alt", "option":
			mod = "alt"
		case "meta", "cmd", "command":
			mod = "meta"
		default:
			return "", fmt.Errorf("unknown modifier %q", mod)
		}
		normalized = append(normalized, mod)
	}
	if len(normalized) == 0 {
		return key, nil
	}
	normalized = append(normalized, key)
	return strings.Join(normalized, "+"), nil
}

func writeLine(w io.Writer, line string) error {
	_, err := fmt.Fprintln(w, line)
	return err
}

func writef(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func write(w io.Writer, content string) error {
	_, err := fmt.Fprint(w, content)
	return err
}

func toastKindString(level sessiond.ToastLevel) string {
	switch level {
	case sessiond.ToastInfo:
		return "info"
	case sessiond.ToastSuccess:
		return "success"
	case sessiond.ToastWarning:
		return "warning"
	default:
		return "unknown"
	}
}

func requireAck(in io.Reader, out io.Writer) error {
	if _, err := fmt.Fprintln(out, "Type ACK to continue:"); err != nil {
		return err
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if strings.TrimSpace(line) != "ACK" {
		return fmt.Errorf("acknowledgement required")
	}
	return nil
}
