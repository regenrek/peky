package sessiond

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

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
	assertMousePayloadInvalid(t, "negative", MouseEventPayload{X: -1, Y: 0})
	assertMousePayloadWheel(t, "wheel", MouseEventPayload{X: 1, Y: 2, Button: 1, Wheel: true}, MouseRouteAuto)
	assertMousePayloadPress(t, "press", MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress}, MouseRouteAuto)
	assertMousePayloadRelease(t, "release", MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionRelease}, MouseRouteAuto)
	assertMousePayloadMotion(t, "motion", MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionMotion}, MouseRouteAuto)
	assertMousePayloadInvalid(t, "unknown-action", MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionUnknown})
	assertMousePayloadInvalid(t, "invalid-route", MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress, Route: MouseRoute("bogus")})
}

func assertMousePayloadInvalid(t *testing.T, name string, payload MouseEventPayload) {
	t.Helper()
	if _, _, ok := mousePayloadToEvent(payload); ok {
		t.Fatalf("%s: expected invalid payload", name)
	}
}

func assertMousePayloadWheel(t *testing.T, name string, payload MouseEventPayload, wantRoute MouseRoute) {
	t.Helper()
	evt, route, ok := mousePayloadToEvent(payload)
	if !ok || evt == nil {
		t.Fatalf("%s: expected wheel event", name)
	}
	if route != wantRoute {
		t.Fatalf("%s: expected route %q, got %q", name, wantRoute, route)
	}
	if _, ok := evt.(uv.MouseWheelEvent); !ok {
		t.Fatalf("%s: expected wheel event type", name)
	}
}

func assertMousePayloadPress(t *testing.T, name string, payload MouseEventPayload, wantRoute MouseRoute) {
	t.Helper()
	evt, route, ok := mousePayloadToEvent(payload)
	if !ok || evt == nil {
		t.Fatalf("%s: expected press event", name)
	}
	if route != wantRoute {
		t.Fatalf("%s: expected route %q, got %q", name, wantRoute, route)
	}
	if _, ok := evt.(uv.MouseClickEvent); !ok {
		t.Fatalf("%s: expected click event type", name)
	}
}

func assertMousePayloadRelease(t *testing.T, name string, payload MouseEventPayload, wantRoute MouseRoute) {
	t.Helper()
	evt, route, ok := mousePayloadToEvent(payload)
	if !ok || evt == nil {
		t.Fatalf("%s: expected release event", name)
	}
	if route != wantRoute {
		t.Fatalf("%s: expected route %q, got %q", name, wantRoute, route)
	}
	if _, ok := evt.(uv.MouseReleaseEvent); !ok {
		t.Fatalf("%s: expected release event type", name)
	}
}

func assertMousePayloadMotion(t *testing.T, name string, payload MouseEventPayload, wantRoute MouseRoute) {
	t.Helper()
	evt, route, ok := mousePayloadToEvent(payload)
	if !ok || evt == nil {
		t.Fatalf("%s: expected motion event", name)
	}
	if route != wantRoute {
		t.Fatalf("%s: expected route %q, got %q", name, wantRoute, route)
	}
	if _, ok := evt.(uv.MouseMotionEvent); !ok {
		t.Fatalf("%s: expected motion event type", name)
	}
}
