package sessiond

import (
	"context"
	"testing"

	"github.com/muesli/termenv"
)

type fakePaneViewWin struct {
	copyMode bool

	lipglossCalls  int
	ansiCalls      int
	lastShowCursor bool
}

func (w *fakePaneViewWin) UpdateSeq() uint64 { return 0 }
func (w *fakePaneViewWin) ANSICacheSeq() uint64 {
	return 0
}

func (w *fakePaneViewWin) ViewLipglossCtx(ctx context.Context, showCursor bool, profile termenv.Profile) (string, error) {
	w.lipglossCalls++
	w.lastShowCursor = showCursor
	return "LIP", nil
}

func (w *fakePaneViewWin) ViewANSICtx(ctx context.Context) (string, error) {
	w.ansiCalls++
	return "ANSI", nil
}

func (w *fakePaneViewWin) ViewANSIDirectCtx(ctx context.Context) (string, error) {
	w.ansiCalls++
	return "ANSI", nil
}

func (w *fakePaneViewWin) ViewLipgloss(showCursor bool, profile termenv.Profile) string { return "LIP" }
func (w *fakePaneViewWin) ViewANSI() string                                             { return "ANSI" }

// CopyModeActive is required by the paneViewWindow interface for render policy decisions.
func (w *fakePaneViewWin) CopyModeActive() bool { return w.copyMode }

func TestPaneViewString_RenderPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		req             PaneViewRequest
		win             fakePaneViewWin
		wantMode        PaneViewMode
		want            string
		wantLipgloss    int
		wantANSI        int
		wantShowCursor  bool
		checkShowCursor bool
	}{
		{
			name: "lipgloss_requested_with_cursor_overlay_uses_lipgloss",
			req: PaneViewRequest{
				Mode:         PaneViewLipgloss,
				ShowCursor:   true,
				ColorProfile: termenv.TrueColor,
			},
			wantMode:        PaneViewLipgloss,
			want:            "LIP",
			wantLipgloss:    1,
			wantANSI:        0,
			wantShowCursor:  true,
			checkShowCursor: true,
		},
		{
			name: "lipgloss_requested_without_cursor_uses_lipgloss",
			req: PaneViewRequest{
				Mode:         PaneViewLipgloss,
				ShowCursor:   false,
				ColorProfile: termenv.TrueColor,
			},
			wantMode:        PaneViewLipgloss,
			want:            "LIP",
			wantLipgloss:    1,
			wantANSI:        0,
			wantShowCursor:  false,
			checkShowCursor: true,
		},
		{
			name: "lipgloss_requested_without_cursor_but_copy_mode_active_uses_lipgloss",
			req: PaneViewRequest{
				Mode:         PaneViewLipgloss,
				ShowCursor:   false,
				ColorProfile: termenv.TrueColor,
			},
			win: fakePaneViewWin{copyMode: true},

			wantMode:        PaneViewLipgloss,
			want:            "LIP",
			wantLipgloss:    1,
			wantANSI:        0,
			wantShowCursor:  false,
			checkShowCursor: true,
		},
		{
			name: "ansi_requested_without_overlays_uses_ansi",
			req: PaneViewRequest{
				Mode:         PaneViewANSI,
				ShowCursor:   false,
				ColorProfile: termenv.TrueColor,
			},
			wantMode:     PaneViewANSI,
			want:         "ANSI",
			wantLipgloss: 0,
			wantANSI:     1,
		},
		{
			name: "ansi_requested_with_copy_mode_active_keeps_ansi",
			req: PaneViewRequest{
				Mode:         PaneViewANSI,
				ShowCursor:   false,
				ColorProfile: termenv.TrueColor,
			},
			win: fakePaneViewWin{copyMode: true},

			wantMode:     PaneViewANSI,
			want:         "ANSI",
			wantLipgloss: 0,
			wantANSI:     1,
		},
		{
			name: "ansi_requested_with_cursor_overlay_uses_lipgloss",
			req: PaneViewRequest{
				Mode:         PaneViewANSI,
				ShowCursor:   true,
				ColorProfile: termenv.TrueColor,
			},
			wantMode:        PaneViewLipgloss,
			want:            "LIP",
			wantLipgloss:    1,
			wantANSI:        0,
			wantShowCursor:  true,
			checkShowCursor: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			win := tt.win
			mode := paneViewRenderMode(&win, tt.req)
			if mode != tt.wantMode {
				t.Fatalf("render mode mismatch: got %v, want %v", mode, tt.wantMode)
			}
			req := tt.req
			req.Mode = mode
			got, err := paneViewString(context.Background(), &win, req)
			if err != nil {
				t.Fatalf("paneViewString returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("paneViewString output mismatch: got %q, want %q", got, tt.want)
			}
			if win.lipglossCalls != tt.wantLipgloss {
				t.Fatalf("lipgloss call count mismatch: got %d, want %d", win.lipglossCalls, tt.wantLipgloss)
			}
			if win.ansiCalls != tt.wantANSI {
				t.Fatalf("ansi call count mismatch: got %d, want %d", win.ansiCalls, tt.wantANSI)
			}
			if tt.checkShowCursor && win.lastShowCursor != tt.wantShowCursor {
				t.Fatalf("lipgloss showCursor mismatch: got %v, want %v", win.lastShowCursor, tt.wantShowCursor)
			}
		})
	}
}
