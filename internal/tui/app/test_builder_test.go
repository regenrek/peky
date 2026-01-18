package app

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/colorprofile"

	"github.com/regenrek/peakypanes/internal/layout"
)

func newTestModelLite() *Model {
	m := &Model{
		state:                  StateDashboard,
		tab:                    TabProject,
		data:                   DashboardData{Projects: sampleProjects()},
		selection:              selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"},
		expandedSessions:       map[string]bool{"alpha-1": true, "alpha-2": false, "beta-1": true},
		selectionByProject:     make(map[string]selectionState),
		paneViews:              make(map[paneViewKey]paneViewEntry),
		paneMouseMotion:        make(map[string]bool),
		paneViewProfile:        colorprofile.TrueColor,
		paneInputDisabled:      make(map[string]struct{}),
		paneViewSeq:            make(map[paneViewKey]uint64),
		paneViewLastReq:        make(map[paneViewKey]time.Time),
		paneViewFirst:          make(map[string]struct{}),
		quickReplyHistoryIndex: -1,
		quickReplyMenuIndex:    -1,
		settings: DashboardConfig{
			PreviewLines: 12,
			Resize: DashboardResizeSettings{
				MouseApply:              ResizeMouseApplyLive,
				MouseThrottle:           16 * time.Millisecond,
				FreezeContentDuringDrag: true,
			},
			Performance: DashboardPerformance{
				Preset:        PerfPresetMax,
				RenderPolicy:  RenderPolicyVisible,
				PreviewRender: PreviewRenderSettings{Mode: PreviewRenderDirect},
				PaneViews:     paneViewPerfMax,
			},
		},
		width:  120,
		height: 40,
		config: &layout.Config{
			QuickReply: layout.QuickReplyConfig{Enabled: true},
		},
	}
	m.resize.snap = true
	m.filterInput = textinput.New()
	m.quickReplyInput = textinput.New()
	m.setupProjectPicker()
	m.setupLayoutPicker()
	m.setupPaneSwapPicker()
	m.setupCommandPalette()
	m.setupSettingsMenu()
	m.setupPerformanceMenu()
	m.setupDebugMenu()
	m.keys = testKeyMap()
	return m
}

func sampleProjects() []ProjectGroup {
	return []ProjectGroup{
		{
			ID:   projectKey("/alpha", "Alpha"),
			Name: "Alpha",
			Path: "/alpha",
			Sessions: []SessionItem{
				{
					Name:   "alpha-1",
					Path:   "/alpha",
					Status: StatusRunning,
					Panes: []PaneItem{
						{ID: "p1", Index: "1", Title: "one", Command: "vim", Active: true, Left: 0, Top: 0, Width: 80, Height: 24},
						{ID: "p2", Index: "2", Title: "two", Command: "bash", Left: 80, Top: 0, Width: 80, Height: 24},
					},
				},
				{
					Name:   "alpha-2",
					Path:   "/alpha",
					Status: StatusRunning,
					Panes: []PaneItem{
						{ID: "p3", Index: "1", Title: "three", Command: "bash", Active: true, Left: 0, Top: 0, Width: 100, Height: 30},
					},
				},
			},
		},
		{
			ID:   projectKey("/beta", "Beta"),
			Name: "Beta",
			Path: "/beta",
			Sessions: []SessionItem{
				{
					Name:   "beta-1",
					Path:   "/beta",
					Status: StatusRunning,
					Panes: []PaneItem{
						{ID: "p4", Index: "1", Title: "four", Command: "bash", Active: true, Left: 0, Top: 0, Width: 50, Height: 20},
					},
				},
			},
		},
	}
}

func testKeyMap() *dashboardKeyMap {
	return &dashboardKeyMap{
		projectLeft:     key.NewBinding(key.WithKeys("h")),
		projectRight:    key.NewBinding(key.WithKeys("l")),
		sessionUp:       key.NewBinding(key.WithKeys("k")),
		sessionDown:     key.NewBinding(key.WithKeys("j")),
		sessionOnlyUp:   key.NewBinding(key.WithKeys("K")),
		sessionOnlyDown: key.NewBinding(key.WithKeys("J")),
		paneNext:        key.NewBinding(key.WithKeys("n")),
		panePrev:        key.NewBinding(key.WithKeys("p")),
		toggleLastPane:  key.NewBinding(key.WithKeys("space")),
		attach:          key.NewBinding(key.WithKeys("enter")),
		newSession:      key.NewBinding(key.WithKeys("s")),
		hardRaw:         key.NewBinding(key.WithKeys("t")),
		resizeMode:      key.NewBinding(key.WithKeys("ctrl+r")),
		togglePanes:     key.NewBinding(key.WithKeys("g")),
		toggleSidebar:   key.NewBinding(key.WithKeys("b")),
		openProject:     key.NewBinding(key.WithKeys("o")),
		commandPalette:  key.NewBinding(key.WithKeys("c")),
		refresh:         key.NewBinding(key.WithKeys("f5")),
		editConfig:      key.NewBinding(key.WithKeys("e")),
		kill:            key.NewBinding(key.WithKeys("x")),
		closeProject:    key.NewBinding(key.WithKeys("z")),
		help:            key.NewBinding(key.WithKeys("?")),
		quit:            key.NewBinding(key.WithKeys("q")),
		filter:          key.NewBinding(key.WithKeys("f")),
		scrollback:      key.NewBinding(key.WithKeys("v")),
		copyMode:        key.NewBinding(key.WithKeys("y")),
	}
}
