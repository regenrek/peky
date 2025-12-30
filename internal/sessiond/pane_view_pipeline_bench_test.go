package sessiond

import (
	"context"
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

func BenchmarkPaneViewResponse(b *testing.B) {
	view := strings.Repeat("x", 80*24)
	win := &fakeTerminalWindow{
		viewANSI:     view,
		viewLipgloss: view,
		updateSeq:    42,
	}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	baseReq := PaneViewRequest{
		PaneID:       "pane-1",
		Cols:         80,
		Rows:         24,
		Mode:         PaneViewANSI,
		ShowCursor:   false,
		ColorProfile: termenv.TrueColor,
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

	b.Run("ANSI", func(b *testing.B) {
		req := baseReq
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = d.paneViewResponse(context.Background(), nil, "pane-1", req)
		}
	})

	b.Run("Lipgloss", func(b *testing.B) {
		req := baseReq
		req.Mode = PaneViewLipgloss
		req.ShowCursor = true
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = d.paneViewResponse(context.Background(), nil, "pane-1", req)
		}
	})
}
