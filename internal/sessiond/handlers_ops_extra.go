package sessiond

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessionpolicy"
)

func (d *Daemon) handleSessionFocus(payload []byte) ([]byte, error) {
	var req FocusSessionRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	name, err := sessionpolicy.ValidateSessionName(req.Name)
	if err != nil {
		return nil, err
	}
	d.setFocusSession(name)
	return nil, nil
}

func (d *Daemon) handlePaneFocus(payload []byte) ([]byte, error) {
	var req PaneFocusRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	d.setFocusPane(paneID)
	d.recordPaneAction(paneID, "focus", "Focused pane", "", "ok")
	return nil, nil
}

func (d *Daemon) handlePaneOutput(payload []byte) ([]byte, error) {
	var req PaneOutputRequest
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
	if req.Limit < 0 {
		req.Limit = 0
	}
	for {
		var lines []native.OutputLine
		var next uint64
		var truncated bool
		if req.SinceSeq == 0 && req.Limit > 0 && !req.Wait {
			lines, err = manager.OutputSnapshot(paneID, req.Limit)
			if err != nil {
				return nil, err
			}
			if len(lines) > 0 {
				next = lines[len(lines)-1].Seq
			}
		} else {
			lines, next, truncated, err = manager.OutputLinesSince(paneID, req.SinceSeq)
			if err != nil {
				return nil, err
			}
		}
		if req.Limit > 0 && len(lines) > req.Limit {
			lines = lines[len(lines)-req.Limit:]
			truncated = true
		}
		if len(lines) > 0 || !req.Wait {
			resp := PaneOutputResponse{
				PaneID:    paneID,
				Lines:     lines,
				NextSeq:   next,
				Truncated: truncated,
			}
			return encodePayload(resp)
		}
		waitCtx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		ok := manager.WaitForOutput(waitCtx, paneID)
		cancel()
		if !ok {
			resp := PaneOutputResponse{PaneID: paneID, Lines: nil, NextSeq: req.SinceSeq, Truncated: truncated}
			return encodePayload(resp)
		}
	}
}

func (d *Daemon) handlePaneSnapshot(payload []byte) ([]byte, error) {
	var req PaneSnapshotRequest
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
	content, truncated, err := manager.PaneScrollbackSnapshot(paneID, req.Rows)
	if err != nil {
		return nil, err
	}
	resp := PaneSnapshotResponse{
		PaneID:    paneID,
		Rows:      req.Rows,
		Content:   content,
		Truncated: truncated,
	}
	return encodePayload(resp)
}

func (d *Daemon) handlePaneHistory(payload []byte) ([]byte, error) {
	var req PaneHistoryRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	entries := d.paneHistory(paneID, req.Limit, req.Since)
	resp := PaneHistoryResponse{PaneID: paneID, Entries: entries}
	return encodePayload(resp)
}

func (d *Daemon) handlePaneWait(payload []byte) ([]byte, error) {
	var req PaneWaitRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		return nil, errors.New("sessiond: pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("sessiond: invalid regex: %w", err)
	}
	manager, err := d.requireManager()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	ctx := context.Background()
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	_, seq, _, _ := manager.OutputLinesSince(paneID, 0)
	for {
		lines, next, _, err := manager.OutputLinesSince(paneID, seq)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			if re.MatchString(line.Text) {
				resp := PaneWaitResponse{
					PaneID:  paneID,
					Pattern: pattern,
					Matched: true,
					Match:   line.Text,
					Elapsed: time.Since(start),
				}
				return encodePayload(resp)
			}
		}
		seq = next
		if ctx.Err() != nil {
			resp := PaneWaitResponse{
				PaneID:  paneID,
				Pattern: pattern,
				Matched: false,
				Elapsed: time.Since(start),
			}
			return encodePayload(resp)
		}
		if !manager.WaitForOutput(ctx, paneID) {
			resp := PaneWaitResponse{
				PaneID:  paneID,
				Pattern: pattern,
				Matched: false,
				Elapsed: time.Since(start),
			}
			return encodePayload(resp)
		}
	}
}

func (d *Daemon) handlePaneTagAdd(payload []byte) ([]byte, error) {
	var req PaneTagRequest
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
	tags, err := manager.AddPaneTags(paneID, req.Tags)
	if err != nil {
		return nil, err
	}
	d.recordPaneAction(paneID, "tag.add", strings.Join(req.Tags, ","), "", "ok")
	return encodePayload(PaneTagListResponse{PaneID: paneID, Tags: tags})
}

func (d *Daemon) handlePaneTagRemove(payload []byte) ([]byte, error) {
	var req PaneTagRequest
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
	tags, err := manager.RemovePaneTags(paneID, req.Tags)
	if err != nil {
		return nil, err
	}
	d.recordPaneAction(paneID, "tag.remove", strings.Join(req.Tags, ","), "", "ok")
	return encodePayload(PaneTagListResponse{PaneID: paneID, Tags: tags})
}

func (d *Daemon) handlePaneTagList(payload []byte) ([]byte, error) {
	var req PaneTagRequest
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
	tags, err := manager.PaneTags(paneID)
	if err != nil {
		return nil, err
	}
	return encodePayload(PaneTagListResponse{PaneID: paneID, Tags: tags})
}

func (d *Daemon) handlePaneSignal(payload []byte) ([]byte, error) {
	var req PaneSignalRequest
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
	if err := manager.SignalPane(paneID, req.Signal); err != nil {
		return nil, err
	}
	d.recordPaneAction(paneID, "signal", req.Signal, "", "ok")
	return nil, nil
}

func (d *Daemon) handleRelayCreate(payload []byte) ([]byte, error) {
	var req RelayCreateRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	cfg := req.Config
	if cfg.Mode == "" {
		cfg.Mode = RelayModeLine
	}
	if len(cfg.ToPaneIDs) == 0 && strings.TrimSpace(cfg.Scope) != "" {
		targets, err := d.resolveScopeTargets(cfg.Scope)
		if err != nil {
			return nil, err
		}
		cfg.ToPaneIDs = targets
	}
	if d.relays == nil {
		return nil, errors.New("sessiond: relay manager unavailable")
	}
	info, err := d.relays.create(context.Background(), d.manager, cfg)
	if err != nil {
		return nil, err
	}
	return encodePayload(RelayCreateResponse{Relay: info})
}

func (d *Daemon) handleRelayList(_ []byte) ([]byte, error) {
	if d.relays == nil {
		return encodePayload(RelayListResponse{Relays: nil})
	}
	relays := d.relays.list()
	return encodePayload(RelayListResponse{Relays: relays})
}

func (d *Daemon) handleRelayStop(payload []byte) ([]byte, error) {
	var req RelayStopRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return nil, errors.New("sessiond: relay id is required")
	}
	if d.relays == nil {
		return nil, errors.New("sessiond: relay manager unavailable")
	}
	if !d.relays.stop(id) {
		return nil, fmt.Errorf("sessiond: relay %q not found", id)
	}
	return nil, nil
}

func (d *Daemon) handleRelayStopAll(_ []byte) ([]byte, error) {
	if d.relays == nil {
		return nil, nil
	}
	d.relays.stopAll()
	return nil, nil
}

func (d *Daemon) handleEventsReplay(payload []byte) ([]byte, error) {
	var req EventsReplayRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	types := make(map[EventType]struct{})
	for _, t := range req.Types {
		types[t] = struct{}{}
	}
	d.eventMu.RLock()
	events := d.eventLog.list(req.Since, req.Until, req.Limit, types)
	d.eventMu.RUnlock()
	return encodePayload(EventsReplayResponse{Events: events})
}
