package app

import "testing"

func TestLayoutMetricsRectsZeroSize(t *testing.T) {
	m := newTestModelLite()
	m.width = 0
	m.height = 0
	if _, ok := m.dashboardBodyRect(); ok {
		t.Fatalf("expected body rect to be false for zero size")
	}
	if _, ok := m.headerRect(); ok {
		t.Fatalf("expected header rect to be false for zero size")
	}
	if _, ok := m.footerRect(); ok {
		t.Fatalf("expected footer rect to be false for zero size")
	}
	if _, ok := m.serverStatusRect(); ok {
		t.Fatalf("expected server status rect to be false for zero size")
	}
}

func TestLayoutMetricsRectsNonZero(t *testing.T) {
	m := newTestModelLite()
	m.width = 120
	m.height = 40
	body, ok := m.dashboardBodyRect()
	if !ok || body.W <= 0 || body.H <= 0 {
		t.Fatalf("expected body rect, got %#v ok=%v", body, ok)
	}

	header, ok := m.headerRect()
	if !ok || header.W <= 0 || header.H <= 0 {
		t.Fatalf("expected header rect, got %#v ok=%v", header, ok)
	}

	footer, ok := m.footerRect()
	if !ok || footer.W <= 0 || footer.H <= 0 {
		t.Fatalf("expected footer rect, got %#v ok=%v", footer, ok)
	}

	status, ok := m.serverStatusRect()
	if !ok || status.W <= 0 || status.H <= 0 {
		t.Fatalf("expected server status rect, got %#v ok=%v", status, ok)
	}
}
