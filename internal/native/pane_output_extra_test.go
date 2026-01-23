package native

import (
	"context"
	"strings"
	"testing"
)

func newTestManagerWithPaneOutput(max int) (*Manager, *Pane) {
	manager := &Manager{panes: make(map[string]*Pane)}
	pane := &Pane{ID: "p1", output: newOutputLog(max), Tags: []string{"alpha"}}
	manager.panes[pane.ID] = pane
	return manager, pane
}

func TestPaneOutputErrors(t *testing.T) {
	manager, _ := newTestManagerWithPaneOutput(2)
	if _, err := (*Manager)(nil).OutputSnapshot("p1", 1); err == nil {
		t.Fatalf("expected error for nil manager")
	}
	if _, err := manager.OutputSnapshot("", 1); err == nil {
		t.Fatalf("expected error for empty pane id")
	}
	if _, err := manager.OutputSnapshot("missing", 1); err == nil {
		t.Fatalf("expected error for missing pane")
	}
	if _, _, _, err := manager.OutputLinesSince("", 0); err == nil {
		t.Fatalf("expected error for empty pane id")
	}
}

func TestPaneOutputSnapshotAndLinesSince(t *testing.T) {
	manager, pane := newTestManagerWithPaneOutput(2)
	pane.output.append([]byte("one\n"))
	pane.output.append([]byte("two\n"))
	pane.output.append([]byte("three\n"))

	lines, err := manager.OutputSnapshot("p1", 1)
	if err != nil {
		t.Fatalf("OutputSnapshot error: %v", err)
	}
	if len(lines) != 1 || lines[0].Text != "three" {
		t.Fatalf("unexpected snapshot lines: %#v", lines)
	}

	lines, next, truncated, err := manager.OutputLinesSince("p1", 0)
	if err != nil {
		t.Fatalf("OutputLinesSince error: %v", err)
	}
	if !truncated {
		t.Fatalf("expected truncated output")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if next != lines[len(lines)-1].Seq {
		t.Fatalf("expected next seq %d, got %d", lines[len(lines)-1].Seq, next)
	}
}

func TestPaneOutputWaitAndSubscribe(t *testing.T) {
	manager, pane := newTestManagerWithPaneOutput(4)
	pane.output.append([]byte("ping\n"))

	ok := manager.WaitForOutput(context.Background(), "p1")
	if !ok {
		t.Fatalf("expected wait to succeed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if manager.WaitForOutput(ctx, "p1") {
		t.Fatalf("expected canceled wait to return false")
	}

	ch, stop, err := manager.SubscribeRawOutput("p1", 1)
	if err != nil {
		t.Fatalf("SubscribeRawOutput error: %v", err)
	}
	pane.output.append([]byte("raw\n"))
	select {
	case chunk := <-ch:
		if !strings.Contains(string(chunk.Data), "raw") {
			t.Fatalf("unexpected chunk data: %q", string(chunk.Data))
		}
	default:
		t.Fatalf("expected raw output chunk")
	}
	stop()
}

func TestPaneTagsAddRemove(t *testing.T) {
	manager, _ := newTestManagerWithPaneOutput(2)
	tags, err := manager.PaneTags("p1")
	if err != nil || len(tags) != 1 || tags[0] != "alpha" {
		t.Fatalf("unexpected tags: %#v err=%v", tags, err)
	}

	tags, err = manager.AddPaneTags("p1", []string{"Beta", "alpha", "  ", "BETA"})
	if err != nil {
		t.Fatalf("AddPaneTags error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "alpha" || tags[1] != "beta" {
		t.Fatalf("unexpected tags after add: %#v", tags)
	}

	tags, err = manager.RemovePaneTags("p1", []string{"beta"})
	if err != nil {
		t.Fatalf("RemovePaneTags error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "alpha" {
		t.Fatalf("unexpected tags after remove: %#v", tags)
	}

	if out := normalizeTags([]string{" ", "X", "x"}); len(out) != 1 || out[0] != "x" {
		t.Fatalf("unexpected normalized tags: %#v", out)
	}
	if out := sortedTags(map[string]struct{}{"b": {}, "a": {}}); len(out) != 2 || out[0] != "a" {
		t.Fatalf("unexpected sorted tags: %#v", out)
	}
}
