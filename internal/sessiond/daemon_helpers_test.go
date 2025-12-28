package sessiond

import "testing"

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func TestIsTimeout(t *testing.T) {
	if !isTimeout(timeoutErr{}) {
		t.Fatalf("expected timeout true")
	}
	if isTimeout(nil) {
		t.Fatalf("expected timeout false for nil")
	}
	if isTimeout(assertError("not-timeout")) {
		t.Fatalf("expected timeout false for non-timeout error")
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }

func TestMousePayloadToEvent(t *testing.T) {
	if _, ok := mousePayloadToEvent(MouseEventPayload{X: -1, Y: 0}); ok {
		t.Fatalf("expected invalid payload")
	}

	if evt, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Wheel: true}); !ok || evt == nil {
		t.Fatalf("expected wheel event")
	}

	if evt, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress}); !ok || evt == nil {
		t.Fatalf("expected press event")
	}
	if evt, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionRelease}); !ok || evt == nil {
		t.Fatalf("expected release event")
	}
	if evt, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionMotion}); !ok || evt == nil {
		t.Fatalf("expected motion event")
	}
	if _, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionUnknown}); ok {
		t.Fatalf("expected unknown action to be invalid")
	}
}
