package native

import (
	"context"
	"testing"
)

func TestOutputLogSnapshotAndLinesSince(t *testing.T) {
	log := newOutputLog(3)
	log.append([]byte("a\n"))
	log.append([]byte("b\n"))
	log.append([]byte("c\n"))
	snapshot := log.snapshot(10)
	if len(snapshot) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(snapshot))
	}
	if snapshot[0].Text != "a" || snapshot[2].Text != "c" {
		t.Fatalf("unexpected snapshot contents: %+v", snapshot)
	}
	log.append([]byte("d\n"))
	snapshot = log.snapshot(3)
	if len(snapshot) != 3 || snapshot[0].Text != "b" || snapshot[2].Text != "d" {
		t.Fatalf("unexpected rollover snapshot: %+v", snapshot)
	}
	lines, next, truncated := log.linesSince(0)
	if !truncated {
		t.Fatalf("expected truncated true")
	}
	if next != snapshot[len(snapshot)-1].Seq {
		t.Fatalf("unexpected next seq: %d", next)
	}
	if len(lines) == 0 {
		t.Fatalf("expected lines")
	}
}

func TestOutputLogLinesSinceTruncate(t *testing.T) {
	log := newOutputLog(2)
	log.append([]byte("a\n"))
	log.append([]byte("b\n"))
	log.append([]byte("c\n"))
	lines, next, truncated := log.linesSince(0)
	if !truncated {
		t.Fatalf("expected truncated")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].Text != "b" || lines[1].Text != "c" {
		t.Fatalf("unexpected lines: %+v", lines)
	}
	if next != lines[1].Seq {
		t.Fatalf("unexpected next seq: %d", next)
	}
}

func TestOutputLogRawSubscribeAndTruncate(t *testing.T) {
	log := newOutputLog(2)
	_, ch, cancel := log.subscribeRaw(1)
	defer cancel()

	log.append([]byte("first\n"))
	first := <-ch
	if string(first.Data) != "first\n" {
		t.Fatalf("unexpected first data: %q", string(first.Data))
	}

	log.append([]byte("second\n"))
	log.append([]byte("third\n"))
	second := <-ch
	if second.Truncated {
		t.Fatalf("unexpected truncated flag on second")
	}

	log.append([]byte("fourth\n"))
	third := <-ch
	if !third.Truncated {
		t.Fatalf("expected truncated flag on third payload")
	}
}

func TestOutputLogWaitAndDisable(t *testing.T) {
	log := newOutputLog(2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if log.wait(ctx) {
		t.Fatalf("expected wait to return false on canceled context")
	}
	log.append([]byte("hello\n"))
	if !log.wait(context.Background()) {
		t.Fatalf("expected wait to return true after append")
	}
	lines, next, _ := log.linesSince(0)
	if len(lines) != 1 {
		t.Fatalf("expected one line, got %d", len(lines))
	}
	log.disable()
	log.append([]byte("ignored\n"))
	lines, _, _ = log.linesSince(next)
	if len(lines) != 0 {
		t.Fatalf("expected no new lines after disable")
	}
}
