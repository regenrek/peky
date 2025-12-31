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

	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/transform"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers pane handlers.
func Register(reg *root.Registry) {
	reg.Register("pane.list", runList)
	reg.Register("pane.rename", runRename)
	reg.Register("pane.split", runSplit)
	reg.Register("pane.close", runClose)
	reg.Register("pane.swap", runSwap)
	reg.Register("pane.resize", runResize)
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
	paneIndex := strings.TrimSpace(ctx.Cmd.String("index"))
	newName := strings.TrimSpace(ctx.Cmd.String("name"))
	if newName == "" {
		return fmt.Errorf("pane name is required")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if paneID != "" {
		if err := client.RenamePaneByID(ctxTimeout, paneID, newName); err != nil {
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
		}
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.rename",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: target}},
		})
	}
	if paneID != "" {
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
	sessionName := ctx.Cmd.String("session")
	paneIndex := ctx.Cmd.String("index")
	orientation := strings.ToLower(strings.TrimSpace(ctx.Cmd.String("orientation")))
	vertical := orientation == "vertical"
	percent := ctx.Cmd.Int("percent")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	newIndex, err := client.SplitPane(ctxTimeout, sessionName, paneIndex, vertical, percent)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "pane.split",
			Status: "ok",
			Details: map[string]any{
				"new_index": newIndex,
			},
		})
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
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	sessionName := ctx.Cmd.String("session")
	paneIndex := ctx.Cmd.String("index")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if paneID != "" {
		if err := client.ClosePaneByID(ctxTimeout, paneID); err != nil {
			return err
		}
	} else {
		if err := client.ClosePane(ctxTimeout, sessionName, paneIndex); err != nil {
			return err
		}
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		target := paneID
		if target == "" {
			target = fmt.Sprintf("%s:%s", sessionName, paneIndex)
		}
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.close",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: target}},
		})
	}
	if paneID != "" {
		return writef(ctx.Out, "Closed pane %s\n", paneID)
	}
	return writef(ctx.Out, "Closed pane %s:%s\n", sessionName, paneIndex)
}

func runSwap(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.swap", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	sessionName := ctx.Cmd.String("session")
	paneA := ctx.Cmd.String("a")
	paneB := ctx.Cmd.String("b")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.SwapPanes(ctxTimeout, sessionName, paneA, paneB); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "pane.swap",
			Status: "ok",
		})
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
	paneID := ctx.Cmd.String("pane-id")
	cols := ctx.Cmd.Int("cols")
	rows := ctx.Cmd.Int("rows")
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.ResizePane(ctxTimeout, paneID, cols, rows); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.resize",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
		})
	}
	return writef(ctx.Out, "Resized pane %s\n", paneID)
}

func runSend(ctx root.CommandContext) error {
	return runSendLike(ctx, false)
}

func runRun(ctx root.CommandContext) error {
	return runSendLike(ctx, true)
}

func runSendLike(ctx root.CommandContext, withNewline bool) error {
	start := time.Now()
	cmdID := "pane.send"
	action := "send"
	if withNewline {
		cmdID = "pane.run"
		action = "run"
	}
	meta := output.NewMeta(cmdID, ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	payload, err := readPayload(ctx, withNewline)
	if err != nil {
		return err
	}
	if withNewline {
		if ctx.Cmd.Bool("confirm") {
			ok, err := root.PromptConfirm(ctx.Stdin, ctx.ErrOut, "Confirm pane.run")
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
		if ctx.Cmd.Bool("require-ack") {
			if err := requireAck(ctx.Stdin, ctx.ErrOut); err != nil {
				return err
			}
		}
	}
	if delay := ctx.Cmd.Duration("delay"); delay > 0 {
		time.Sleep(delay)
	}
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	scope := strings.TrimSpace(ctx.Cmd.String("scope"))
	submitDelay := ctx.Cmd.Duration("submit-delay")
	var results []output.TargetResult
	warnings := []string{}
	if paneID == "" && scope == "" {
		return fmt.Errorf("pane-id or scope is required")
	}
	if paneID != "" {
		results = []output.TargetResult{{Target: output.TargetRef{Type: "pane", ID: paneID}, Status: "ok"}}
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		err := sendPayload(ctxTimeout, client, paneID, payload.Data, payload.Summary, action, withNewline, submitDelay)
		cancel()
		if err != nil {
			return err
		}
		if withNewline && submitDelay > 0 {
			warnings = append(warnings, "submit_delay_applied")
		}
	} else {
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		resp, err := sendPayloadScope(ctxTimeout, client, scope, payload.Data, payload.Summary, action, withNewline, submitDelay)
		cancel()
		if err != nil {
			return err
		}
		results = mapSendResults(resp)
		if withNewline && submitDelay > 0 {
			warnings = append(warnings, "submit_delay_applied")
		}
	}
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
	var viewMode sessiond.PaneViewMode
	switch mode {
	case "lipgloss":
		viewMode = sessiond.PaneViewLipgloss
	case "plain", "ansi":
		viewMode = sessiond.PaneViewANSI
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
	req := sessiond.PaneViewRequest{PaneID: paneID, Rows: rows, Cols: cols, Mode: viewMode}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.GetPaneView(ctxTimeout, req)
	if err != nil {
		return err
	}
	content := resp.View
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
	paneID := ctx.Cmd.String("pane-id")
	follow := ctx.Cmd.Bool("follow")
	if !ctx.Cmd.IsSet("follow") {
		follow = true
	}
	limit := ctx.Cmd.Int("lines")
	grep := strings.TrimSpace(ctx.Cmd.String("grep"))
	var re *regexp.Regexp
	if grep != "" {
		re, err = regexp.Compile(grep)
		if err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	since, err := parseTimeOrDuration(ctx.Cmd.String("since"), now, false)
	if err != nil {
		return err
	}
	until, err := parseTimeOrDuration(ctx.Cmd.String("until"), now, true)
	if err != nil {
		return err
	}
	seq := uint64(0)
	streamSeq := int64(0)
	for {
		if !until.IsZero() && time.Now().UTC().After(until) {
			follow = false
		}
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		resp, err := client.PaneOutput(ctxTimeout, sessiond.PaneOutputRequest{PaneID: paneID, SinceSeq: seq, Limit: limit, Wait: follow})
		cancel()
		if err != nil {
			return err
		}
		seq = resp.NextSeq
		filtered := filterOutputLines(resp.Lines, since, until, re)
		if ctx.JSON {
			emitted := false
			for i, line := range filtered {
				frame := output.PaneTailFrame{
					PaneID:    paneID,
					Chunk:     line.Text + "\n",
					Encoding:  "utf-8",
					Truncated: resp.Truncated && i == 0,
				}
				streamSeq++
				metaFrame := output.NewStreamMeta("pane.tail", ctx.Deps.Version, streamSeq, false)
				if err := output.WriteSuccess(ctx.Out, metaFrame, frame); err != nil {
					return err
				}
				emitted = true
			}
			if !follow {
				if !emitted {
					streamSeq++
					metaFrame := output.NewStreamMeta("pane.tail", ctx.Deps.Version, streamSeq, true)
					if err := output.WriteSuccess(ctx.Out, metaFrame, output.PaneTailFrame{PaneID: paneID, Chunk: "", Encoding: "utf-8"}); err != nil {
						return err
					}
				} else {
					streamSeq++
					metaFrame := output.NewStreamMeta("pane.tail", ctx.Deps.Version, streamSeq, true)
					if err := output.WriteSuccess(ctx.Out, metaFrame, output.PaneTailFrame{PaneID: paneID, Chunk: "", Encoding: "utf-8"}); err != nil {
						return err
					}
				}
				break
			}
		} else {
			for _, line := range filtered {
				if err := writeLine(ctx.Out, line.Text); err != nil {
					return err
				}
			}
			if !follow {
				break
			}
		}
	}
	if ctx.JSON {
		_ = output.WithDuration(meta, start)
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
	resp, err := client.PaneSnapshot(ctxTimeout, paneID, rows)
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
	resp, err := client.PaneHistory(ctxTimeout, sessiond.PaneHistoryRequest{PaneID: paneID, Limit: limit, Since: sinceTime})
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
	resp, err := client.PaneWait(ctxTimeout, sessiond.PaneWaitRequest{PaneID: ctx.Cmd.String("pane-id"), Pattern: pattern, Timeout: timeout})
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
	if add {
		_, err = client.AddPaneTags(ctxTimeout, paneID, tags)
	} else {
		_, err = client.RemovePaneTags(ctxTimeout, paneID, tags)
	}
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  action,
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
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
	tags, err := client.PaneTags(ctxTimeout, paneID)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.PaneTagList{PaneID: paneID, Tags: tags})
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
	lines := ctx.Cmd.Int("lines")
	if lines == 0 {
		lines = ctx.Cmd.Int("count")
	}
	deltaX := ctx.Cmd.Int("delta-x")
	deltaY := ctx.Cmd.Int("delta-y")
	if deltaX == 0 && deltaY == 0 && action == sessiond.TerminalCopyMove && lines != 0 {
		deltaY = lines
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	lines = defaultActionLines(action, lines)
	resp, err := client.TerminalAction(ctxTimeout, sessiond.TerminalActionRequest{
		PaneID: paneID,
		Action: action,
		Lines:  lines,
		DeltaX: deltaX,
		DeltaY: deltaY,
	})
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		details := map[string]any{
			"action": actionName,
		}
		if lines != 0 {
			details["lines"] = lines
		}
		if deltaX != 0 {
			details["delta_x"] = deltaX
		}
		if deltaY != 0 {
			details["delta_y"] = deltaY
		}
		if resp.Text != "" {
			details["text"] = resp.Text
		}
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.action",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
			Details: details,
		})
	}
	if resp.Text != "" {
		if err := writeLine(ctx.Out, resp.Text); err != nil {
			return err
		}
	}
	return nil
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
	resp, err := client.HandleTerminalKey(ctxTimeout, sessiond.TerminalKeyRequest{
		PaneID:           paneID,
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
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
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
	if err := client.SignalPane(ctxTimeout, paneID, signal); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.signal",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
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
	if err := client.FocusPane(ctxTimeout, paneID); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.focus",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "pane", ID: paneID}},
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
	trimmed := strings.TrimSpace(string(data))
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

func sendPayload(ctx context.Context, client *sessiond.Client, paneID string, data []byte, summary, action string, withNewline bool, submitDelay time.Duration) error {
	payload := append([]byte(nil), data...)
	if withNewline && submitDelay <= 0 {
		payload = append(payload, '\n')
		return client.SendInputAction(ctx, paneID, payload, action, summary)
	}
	if err := client.SendInputAction(ctx, paneID, payload, action, summary); err != nil {
		return err
	}
	if withNewline && submitDelay > 0 {
		time.Sleep(submitDelay)
		return client.SendInput(ctx, paneID, []byte("\n"))
	}
	return nil
}

func sendPayloadScope(ctx context.Context, client *sessiond.Client, scope string, data []byte, summary, action string, withNewline bool, submitDelay time.Duration) (sessiond.SendInputResponse, error) {
	payload := append([]byte(nil), data...)
	if withNewline && submitDelay <= 0 {
		payload = append(payload, '\n')
		return client.SendInputScopeAction(ctx, scope, payload, action, summary)
	}
	resp, err := client.SendInputScopeAction(ctx, scope, payload, action, summary)
	if err != nil {
		return sessiond.SendInputResponse{}, err
	}
	if withNewline && submitDelay > 0 {
		time.Sleep(submitDelay)
		_, err = client.SendInputScope(ctx, scope, []byte("\n"))
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
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
		if res.Status == "ok" {
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
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	if normalized == "" {
		return sessiond.TerminalActionUnknown, errors.New("action is required")
	}
	switch normalized {
	case "enter_scrollback", "enter_scrollback_mode":
		return sessiond.TerminalEnterScrollback, nil
	case "exit_scrollback", "exit_scrollback_mode":
		return sessiond.TerminalExitScrollback, nil
	case "scroll_up", "scrollup":
		return sessiond.TerminalScrollUp, nil
	case "scroll_down", "scrolldown":
		return sessiond.TerminalScrollDown, nil
	case "page_up", "pageup":
		return sessiond.TerminalPageUp, nil
	case "page_down", "pagedown":
		return sessiond.TerminalPageDown, nil
	case "scroll_top", "scrolltop":
		return sessiond.TerminalScrollTop, nil
	case "scroll_bottom", "scrollbottom":
		return sessiond.TerminalScrollBottom, nil
	case "enter_copy", "enter_copy_mode", "copy_mode":
		return sessiond.TerminalEnterCopyMode, nil
	case "exit_copy", "exit_copy_mode":
		return sessiond.TerminalExitCopyMode, nil
	case "copy_move", "copy_move_cursor":
		return sessiond.TerminalCopyMove, nil
	case "copy_page_up", "copy_pageup":
		return sessiond.TerminalCopyPageUp, nil
	case "copy_page_down", "copy_pagedown":
		return sessiond.TerminalCopyPageDown, nil
	case "copy_toggle_select", "copy_toggle":
		return sessiond.TerminalCopyToggleSelect, nil
	case "copy_yank", "copy":
		return sessiond.TerminalCopyYank, nil
	default:
		return sessiond.TerminalActionUnknown, fmt.Errorf("unknown action %q", value)
	}
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
