package native

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestBuildPaneSendQueues(t *testing.T) {
	delayZero := 0
	delayDirect := 100
	submitDelay := 250
	cfg := &layout.LayoutConfig{
		BroadcastSend: []layout.SendAction{
			{Text: "first", WaitForOutput: true},
			{Text: "second", SendDelayMS: &delayZero},
		},
		Panes: []layout.PaneDef{
			{DirectSend: []layout.SendAction{{Text: "pane0", SendDelayMS: &delayDirect, Submit: true, SubmitDelayMS: &submitDelay, WaitForOutput: true}}},
			{DirectSend: []layout.SendAction{{Text: "pane1"}}},
		},
	}
	panes := []*Pane{
		{ID: "p-1", Index: "0"},
		{ID: "p-2", Index: "1"},
	}

	queues := buildPaneSendQueues(cfg, panes)
	if len(queues) != 2 {
		t.Fatalf("expected 2 queues, got %d", len(queues))
	}
	queue0 := queues["p-1"]
	if len(queue0) != 3 {
		t.Fatalf("expected 3 actions for pane 0, got %d", len(queue0))
	}
	if queue0[0].text != "first" || queue0[0].delay != defaultSendDelay || !queue0[0].waitOutput {
		t.Fatalf("pane0 action0 = %#v", queue0[0])
	}
	if queue0[1].text != "second" || queue0[1].delay != 0 {
		t.Fatalf("pane0 action1 = %#v", queue0[1])
	}
	if queue0[2].text != "pane0" || queue0[2].delay != 100*time.Millisecond || !queue0[2].submit || queue0[2].submitDelay != 250*time.Millisecond || !queue0[2].waitOutput {
		t.Fatalf("pane0 action2 = %#v", queue0[2])
	}
	queue1 := queues["p-2"]
	if len(queue1) != 3 {
		t.Fatalf("expected 3 actions for pane 1, got %d", len(queue1))
	}
	if queue1[2].text != "pane1" {
		t.Fatalf("pane1 action2 = %#v", queue1[2])
	}
}

func TestBuildSendPayloadAddsNewline(t *testing.T) {
	payload, ok := buildSendPayload("echo hi", true)
	if !ok {
		t.Fatalf("expected payload")
	}
	if string(payload) != "echo hi\n" {
		t.Fatalf("payload = %q", string(payload))
	}
	payload, ok = buildSendPayload("already\n", true)
	if !ok {
		t.Fatalf("expected payload with newline")
	}
	if string(payload) != "already\n" {
		t.Fatalf("payload = %q", string(payload))
	}
}

func TestBuildSendPayloadRaw(t *testing.T) {
	payload, ok := buildSendPayload("echo hi\n", false)
	if !ok {
		t.Fatalf("expected payload")
	}
	if string(payload) != "echo hi" {
		t.Fatalf("payload = %q", string(payload))
	}
}
