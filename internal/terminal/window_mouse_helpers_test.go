package terminal

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func assertSendMouseHandled(t *testing.T, w *Window, event uv.MouseEvent, msg string) {
	t.Helper()
	if !w.SendMouse(event) {
		t.Fatalf("%s", msg)
	}
}

func assertCopyModeInactive(t *testing.T, w *Window) {
	t.Helper()
	if w.CopyModeActive() {
		t.Fatalf("expected copy mode inactive")
	}
}

func assertCopyModeActive(t *testing.T, w *Window) {
	t.Helper()
	if !w.CopyModeActive() || w.CopyMode == nil {
		t.Fatalf("expected copy mode active")
	}
}

func assertCopySelecting(t *testing.T, w *Window) {
	t.Helper()
	if w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected selecting true")
	}
}

func assertMouseSelectionActive(t *testing.T, w *Window) {
	t.Helper()
	if !w.CopySelectionFromMouseActive() {
		t.Fatalf("expected mouse selection active")
	}
}

func assertSelectionCursor(t *testing.T, w *Window, x, absY int) {
	t.Helper()
	if w.CopyMode == nil || w.CopyMode.CursorX != x || w.CopyMode.CursorAbsY != absY {
		t.Fatalf("expected cursor at (%d,%d), got (%d,%d)", x, absY, w.CopyMode.CursorX, w.CopyMode.CursorAbsY)
	}
}

func assertSelectionStartEnd(t *testing.T, w *Window, sx, sy, ex, ey int) {
	t.Helper()
	if w.CopyMode == nil || w.CopyMode.SelStartX != sx || w.CopyMode.SelStartAbsY != sy || w.CopyMode.SelEndX != ex || w.CopyMode.SelEndAbsY != ey {
		t.Fatalf("expected selection start(%d,%d) end(%d,%d), got start(%d,%d) end(%d,%d)",
			sx, sy, ex, ey, w.CopyMode.SelStartX, w.CopyMode.SelStartAbsY, w.CopyMode.SelEndX, w.CopyMode.SelEndAbsY)
	}
}

func assertMouseNotForwarded(t *testing.T, emu *fakeEmu) {
	t.Helper()
	if emu.sentMouse != nil {
		t.Fatalf("expected mouse not forwarded to app")
	}
}
