package sessiond

import (
	"context"
	"testing"

	"github.com/regenrek/peakypanes/internal/termframe"
)

func BenchmarkPaneViewResponse(b *testing.B) {
	view := termframe.Frame{Cols: 80, Rows: 24, Cells: make([]termframe.Cell, 80*24)}
	win := &fakeTerminalWindow{
		viewFrame: view,
		updateSeq: 42,
	}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	baseReq := PaneViewRequest{
		PaneID: "pane-1",
		Cols:   80,
		Rows:   24,
	}

	b.Run("NotModified", func(b *testing.B) {
		req := baseReq
		req.KnownSeq = win.UpdateSeq()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = d.paneViewResponse(context.Background(), nil, "pane-1", req)
		}
	})

	b.Run("FrameCached", func(b *testing.B) {
		req := baseReq
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = d.paneViewResponse(context.Background(), nil, "pane-1", req)
		}
	})

	b.Run("FrameDirect", func(b *testing.B) {
		req := baseReq
		req.DirectRender = true
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = d.paneViewResponse(context.Background(), nil, "pane-1", req)
		}
	})
}
