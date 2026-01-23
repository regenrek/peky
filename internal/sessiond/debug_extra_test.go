package sessiond

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

func TestIsConnectionError(t *testing.T) {
	if IsConnectionError(nil) {
		t.Fatalf("expected false for nil")
	}
	if !IsConnectionError(ErrClientClosed) {
		t.Fatalf("expected client closed")
	}
	if !IsConnectionError(net.ErrClosed) {
		t.Fatalf("expected net closed")
	}
	if !IsConnectionError(&net.OpError{Err: errors.New("boom")}) {
		t.Fatalf("expected op error")
	}
	if !IsConnectionError(os.ErrNotExist) {
		t.Fatalf("expected not exist")
	}
	if IsConnectionError(errors.New("other")) {
		t.Fatalf("expected false for other")
	}
}

func TestPaneViewHelpers(t *testing.T) {
	if paneViewAfterOutputReady("", time.Time{}, time.Now(), time.Now(), PaneViewResponse{}) {
		t.Fatalf("expected false for empty pane")
	}
	now := time.Now()
	if !paneViewAfterOutputReady("p1", now.Add(-time.Second), now, now, PaneViewResponse{}) {
		t.Fatalf("expected true for ready")
	}
	if got := normalizePaneViewRequestAt(time.Time{}, now); !got.Equal(now) {
		t.Fatalf("normalize request time mismatch")
	}
	if clampDuration(-time.Second) != 0 {
		t.Fatalf("expected clamp to zero")
	}
	timing := buildPaneViewAfterOutputTiming(now.Add(-time.Second), now.Add(-500*time.Millisecond), now, now.Add(100*time.Millisecond), now.Add(200*time.Millisecond))
	if timing.outputToViewReq < 0 || timing.viewReqToRender < 0 {
		t.Fatalf("timing=%#v", timing)
	}
}

func TestMarkPerfPaneViewOnce(t *testing.T) {
	d := &Daemon{}
	if !d.markPerfPaneViewOnce("p1", perfPaneViewFirst) {
		t.Fatalf("expected first mark")
	}
	if d.markPerfPaneViewOnce("p1", perfPaneViewFirst) {
		t.Fatalf("expected second mark false")
	}
}
