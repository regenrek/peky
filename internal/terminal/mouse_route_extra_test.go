package terminal

import "testing"

func TestMouseRouteValidAndParse(t *testing.T) {
	if !MouseRouteAuto.Valid() || !MouseRouteApp.Valid() || !MouseRouteHostSelection.Valid() {
		t.Fatalf("expected valid routes")
	}
	if route, ok := ParseMouseRoute("HOST"); !ok || route != MouseRouteHostSelection {
		t.Fatalf("route=%v ok=%v", route, ok)
	}
	if route, ok := ParseMouseRoute("hostselection"); !ok || route != MouseRouteHostSelection {
		t.Fatalf("route=%v ok=%v", route, ok)
	}
	if _, ok := ParseMouseRoute("nope"); ok {
		t.Fatalf("expected invalid parse")
	}
	if MouseRoute("bad").Valid() {
		t.Fatalf("expected invalid route")
	}
}
