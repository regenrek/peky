package terminal

import "errors"

// ErrPaneClosed indicates the pane can no longer accept input.
var ErrPaneClosed = errors.New("pane closed")

// PaneClosedReason describes why a pane stopped accepting input.
type PaneClosedReason int32

const (
	PaneClosedUnknown PaneClosedReason = iota
	PaneClosedProcessExited
	PaneClosedPTYClosed
	PaneClosedWindowClosed
)

// PaneClosedError reports a pane-closed condition without exposing low-level I/O details.
type PaneClosedError struct {
	Reason PaneClosedReason
	Cause  error
}

func (e *PaneClosedError) Error() string {
	switch e.Reason {
	case PaneClosedProcessExited:
		return "pane closed (process exited)"
	case PaneClosedPTYClosed:
		return "pane closed (pty disconnected)"
	case PaneClosedWindowClosed:
		return "pane closed"
	default:
		return "pane closed"
	}
}

func (e *PaneClosedError) Unwrap() error { return e.Cause }

func (e *PaneClosedError) Is(target error) bool { return target == ErrPaneClosed }
