package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/tui/views"
)

type paneViewSize struct {
	w int
	h int
}

type paneViewSizeSet map[paneViewSize]struct{}

func capturePaneViewSizes(vm views.Model) map[string]paneViewSizeSet {
	sizes := make(map[string]paneViewSizeSet)
	vm.PaneView = func(id string, width, height int, showCursor bool) string {
		set := sizes[id]
		if set == nil {
			set = make(paneViewSizeSet)
			sizes[id] = set
		}
		set[paneViewSize{w: width, h: height}] = struct{}{}
		return ""
	}
	_ = views.Render(vm)
	return sizes
}

func assertPaneViewSizesMatchHits(t *testing.T, m *Model) {
	t.Helper()
	vm := m.viewModel()
	sizes := capturePaneViewSizes(vm)
	hits := m.paneHits()
	if len(hits) == 0 {
		t.Fatalf("expected pane hits")
	}
	for _, hit := range hits {
		if hit.PaneID == "" || hit.Content.Empty() {
			continue
		}
		sizeSet, ok := sizes[hit.PaneID]
		if !ok {
			t.Fatalf("missing pane view request for %s", hit.PaneID)
		}
		if len(sizeSet) != 1 {
			t.Fatalf("pane %s requested with multiple sizes: %#v", hit.PaneID, sizeSet)
		}
		for size := range sizeSet {
			if size.w != hit.Content.W || size.h != hit.Content.H {
				t.Fatalf("pane %s size mismatch: render=%dx%d hit=%dx%d", hit.PaneID, size.w, size.h, hit.Content.W, hit.Content.H)
			}
		}
	}
}

func TestPaneViewRequestMatchesRenderSize(t *testing.T) {
	hFrame, vFrame := appStyle.GetFrameSize()
	smallHeight := 10 + vFrame
	smallWidth := 80 + hFrame
	cases := []struct {
		name           string
		tab            DashboardTab
		pekyPromptLine string
		width          int
		height         int
	}{
		{name: "dashboard_prompt_off", tab: TabDashboard, pekyPromptLine: ""},
		{name: "dashboard_prompt_on", tab: TabDashboard, pekyPromptLine: "ok"},
		{name: "project_prompt_off", tab: TabProject, pekyPromptLine: ""},
		{name: "project_prompt_on", tab: TabProject, pekyPromptLine: "ok"},
		{name: "dashboard_small_prompt_on", tab: TabDashboard, pekyPromptLine: "ok", width: smallWidth, height: smallHeight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModelLite()
			m.tab = tc.tab
			m.state = StateDashboard
			m.pekyPromptLine = tc.pekyPromptLine
			m.settings.PreviewMode = "grid"
			if tc.width > 0 {
				m.width = tc.width
			} else {
				m.width = 120
			}
			if tc.height > 0 {
				m.height = tc.height
			} else {
				m.height = 40
			}
			assertPaneViewSizesMatchHits(t, m)
		})
	}
}
