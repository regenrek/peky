package app

import "testing"

func TestLayoutMetricsRects(t *testing.T) {
	m := newTestModelLite()
	m.width = 0
	m.height = 0
	if _, ok := m.dashboardBodyRect(); ok {
		t.Fatalf("expected body rect to be false for zero size")
	}
	if _, ok := m.headerRect(); ok {
		t.Fatalf("expected header rect to be false for zero size")
	}

	m.width = 120
	m.height = 40
	m.settings.ShowThumbnails = true
	body, ok := m.dashboardBodyRect()
	if !ok || body.W <= 0 || body.H <= 0 {
		t.Fatalf("expected body rect, got %#v ok=%v", body, ok)
	}

	header, ok := m.headerRect()
	if !ok || header.W <= 0 || header.H <= 0 {
		t.Fatalf("expected header rect, got %#v ok=%v", header, ok)
	}
}
