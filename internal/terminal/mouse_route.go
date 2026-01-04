package terminal

import "strings"

// MouseRoute controls how mouse events are routed inside a pane.
type MouseRoute string

const (
	// MouseRouteAuto preserves legacy behavior (selection only when allowed).
	MouseRouteAuto MouseRoute = ""
	// MouseRouteApp forwards mouse input to the application (term-capture).
	MouseRouteApp MouseRoute = "app"
	// MouseRouteHostSelection forces host-side selection (pane-selection).
	MouseRouteHostSelection MouseRoute = "host"
)

func (r MouseRoute) Valid() bool {
	switch r {
	case MouseRouteAuto, MouseRouteApp, MouseRouteHostSelection:
		return true
	default:
		return false
	}
}

// ParseMouseRoute normalizes a route string to a known route.
func ParseMouseRoute(raw string) (MouseRoute, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return MouseRouteAuto, true
	case "auto":
		return MouseRouteAuto, true
	case "app":
		return MouseRouteApp, true
	case "host", "host-selection", "hostselection":
		return MouseRouteHostSelection, true
	default:
		return MouseRouteAuto, false
	}
}
