package terminal

import (
	"errors"
	"testing"
)

func TestPaneClosedErrorMessages(t *testing.T) {
	cases := []struct {
		reason PaneClosedReason
		want   string
	}{
		{PaneClosedUnknown, "pane closed"},
		{PaneClosedProcessExited, "pane closed (process exited)"},
		{PaneClosedPTYClosed, "pane closed (pty disconnected)"},
		{PaneClosedWindowClosed, "pane closed"},
	}
	for _, tc := range cases {
		err := &PaneClosedError{Reason: tc.reason}
		if err.Error() != tc.want {
			t.Fatalf("reason %v error=%q want %q", tc.reason, err.Error(), tc.want)
		}
		if !errors.Is(err, ErrPaneClosed) {
			t.Fatalf("reason %v should match ErrPaneClosed", tc.reason)
		}
	}
}

func TestWindowDeadFlags(t *testing.T) {
	w := &Window{}
	if w.Dead() {
		t.Fatalf("expected new window to be alive")
	}
	w.exited.Store(true)
	if !w.Dead() {
		t.Fatalf("expected exited window to be dead")
	}
	w = &Window{}
	w.closed.Store(true)
	if !w.Dead() {
		t.Fatalf("expected closed window to be dead")
	}
	w = &Window{}
	w.inputClosed.Store(true)
	if !w.Dead() {
		t.Fatalf("expected input-closed window to be dead")
	}
}
