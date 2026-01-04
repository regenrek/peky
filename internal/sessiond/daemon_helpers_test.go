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
	if _, _, ok := mousePayloadToEvent(MouseEventPayload{X: -1, Y: 0}); ok {
		t.Fatalf("expected invalid payload")
	}

	evt, route, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Wheel: true})
	if !ok || evt == nil {
		t.Fatalf("expected wheel event")
	}
	if route != MouseRouteAuto {
		t.Fatalf("expected auto route for wheel, got %q", route)
	}

	evt, route, ok = mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress})
	if !ok || evt == nil {
		t.Fatalf("expected press event")
	}
	if route != MouseRouteAuto {
		t.Fatalf("expected auto route for press, got %q", route)
	}
	evt, route, ok = mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionRelease})
	if !ok || evt == nil {
		t.Fatalf("expected release event")
	}
	if route != MouseRouteAuto {
		t.Fatalf("expected auto route for release, got %q", route)
	}
	evt, route, ok = mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionMotion})
	if !ok || evt == nil {
		t.Fatalf("expected motion event")
	}
	if route != MouseRouteAuto {
		t.Fatalf("expected auto route for motion, got %q", route)
	}
	if _, _, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionUnknown}); ok {
		t.Fatalf("expected unknown action to be invalid")
	}
	if _, _, ok := mousePayloadToEvent(MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress, Route: MouseRoute("bogus")}); ok {
		t.Fatalf("expected invalid route to be rejected")
	}
}
