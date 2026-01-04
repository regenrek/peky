package sessiond

import (
	"context"
	"sync"
	"testing"

	"github.com/regenrek/peakypanes/internal/native"
)

type recordManager struct {
	fakeManager
	mu     sync.Mutex
	inputs map[string]int
}

func (m *recordManager) SendInput(_ context.Context, paneID string, input []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inputs == nil {
		m.inputs = make(map[string]int)
	}
	m.inputs[paneID]++
	return nil
}

func TestHandleSendInputToolCodexWrapsAndSubmits(t *testing.T) {
	manager := &fakeManager{
		snapshot: []native.SessionSnapshot{
			{
				Name: "s1",
				Panes: []native.PaneSnapshot{{
					ID:           "pane-1",
					StartCommand: "codex",
				}},
			},
		},
	}
	d := &Daemon{manager: manager, toolRegistry: defaultToolRegistry(t)}

	payload, err := encodePayload(SendInputToolRequest{
		PaneID: "pane-1",
		Input:  []byte("hello"),
		Submit: true,
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendInputTool(payload); err != nil {
		t.Fatalf("handleSendInputTool: %v", err)
	}
	if len(manager.inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(manager.inputs))
	}
	if got := string(manager.inputs[0]); got != "\x1b[200~hello\x1b[201~" {
		t.Fatalf("payload = %q", got)
	}
	if got := string(manager.inputs[1]); got != "\r" {
		t.Fatalf("submit = %q", got)
	}
}

func TestHandleSendInputToolScopeFilterSkips(t *testing.T) {
	manager := &fakeManager{
		snapshot: []native.SessionSnapshot{
			{
				Name:  "s1",
				Panes: []native.PaneSnapshot{{ID: "pane-1", StartCommand: "bash"}},
			},
		},
	}
	d := &Daemon{manager: manager, toolRegistry: defaultToolRegistry(t)}

	payload, err := encodePayload(SendInputToolRequest{
		Scope:      "all",
		Input:      []byte("hello"),
		ToolFilter: "codex",
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleSendInputTool(payload)
	if err != nil {
		t.Fatalf("handleSendInputTool: %v", err)
	}
	var resp SendInputResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decodePayload: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Status != "skipped" {
		t.Fatalf("unexpected results: %#v", resp.Results)
	}
	if len(manager.inputs) != 0 {
		t.Fatalf("expected no sends, got %d", len(manager.inputs))
	}
}

func TestHandleSendInputToolScopeSends(t *testing.T) {
	manager := &recordManager{
		fakeManager: fakeManager{
			snapshot: []native.SessionSnapshot{
				{
					Name: "s1",
					Panes: []native.PaneSnapshot{
						{ID: "p-1", StartCommand: "bash"},
						{ID: "p-2", StartCommand: "bash"},
					},
				},
			},
		},
	}
	d := &Daemon{manager: manager, toolRegistry: defaultToolRegistry(t)}

	payload, err := encodePayload(SendInputToolRequest{
		Scope: "all",
		Input: []byte("hello"),
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleSendInputTool(payload)
	if err != nil {
		t.Fatalf("handleSendInputTool: %v", err)
	}
	var resp SendInputResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decodePayload: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if manager.inputs["p-1"] != 1 || manager.inputs["p-2"] != 1 {
		t.Fatalf("expected sends to both panes, got %#v", manager.inputs)
	}
}

func TestHandleSendInputToolDetectsToolFromInput(t *testing.T) {
	manager := &fakeManager{
		snapshot: []native.SessionSnapshot{
			{
				Name:  "s1",
				Panes: []native.PaneSnapshot{{ID: "pane-1", StartCommand: "bash"}},
			},
		},
	}
	d := &Daemon{manager: manager, toolRegistry: defaultToolRegistry(t)}

	payload, err := encodePayload(SendInputToolRequest{
		PaneID:     "pane-1",
		Input:      []byte("codex"),
		Submit:     true,
		DetectTool: true,
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendInputTool(payload); err != nil {
		t.Fatalf("handleSendInputTool: %v", err)
	}
	if got := manager.lastTool; got[0] != "pane-1" || got[1] != "codex" {
		t.Fatalf("expected tool set, got %#v", got)
	}
}
