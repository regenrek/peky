package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"os"
	"time"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// Styles - using centralized theme for consistency
var (
	appStyle           = theme.App
	dialogStyle        = theme.Dialog
	dialogStyleCompact = theme.DialogCompact
)

// Key bindings

type dashboardKeyMap struct {
	projectLeft     key.Binding
	projectRight    key.Binding
	sessionUp       key.Binding
	sessionDown     key.Binding
	sessionOnlyUp   key.Binding
	sessionOnlyDown key.Binding
	paneNext        key.Binding
	panePrev        key.Binding
	attach          key.Binding
	newSession      key.Binding
	terminalFocus   key.Binding
	resizeMode      key.Binding
	togglePanes     key.Binding
	toggleSidebar   key.Binding
	openProject     key.Binding
	commandPalette  key.Binding
	refresh         key.Binding
	editConfig      key.Binding
	kill            key.Binding
	closeProject    key.Binding
	help            key.Binding
	quit            key.Binding
	filter          key.Binding
	scrollback      key.Binding
	copyMode        key.Binding
}

// Model implements tea.Model for peakypanes TUI.
type Model struct {
	client               *sessiond.Client
	paneViewClient       *sessiond.Client
	daemonVersion        string
	daemonDisconnected   bool
	restartNoticePending bool
	reconnectInFlight    bool
	reconnectBackoff     time.Duration
	reconnectToastAt     time.Time
	state                ViewState
	tab                  DashboardTab
	width                int
	height               int

	configPath string

	data               DashboardData
	selection          selectionState
	selectionByProject map[string]selectionState
	focusPending       bool
	focusSelection     selectionState
	settings           DashboardConfig
	config             *layout.Config
	projectConfigState map[string]projectConfigState
	projectLocalConfig map[string]projectLocalConfigCache
	sidebarOverrides   map[string]bool

	keys *dashboardKeyMap

	layoutEngines       map[string]*layout.Engine
	layoutEngineVersion uint64

	expandedSessions map[string]bool

	filterInput  textinput.Model
	filterActive bool

	quickReplyInput        textinput.Model
	quickReplyHistory      []string
	quickReplyHistoryIndex int
	quickReplyHistoryDraft string
	quickReplyMenuIndex    int
	quickReplyMenuPrefix   string
	quickReplyMenuKind     quickReplyMenuKind
	quickReplyMode         quickReplyMode
	quickReplyFileCache    quickReplyFileCache

	mouse       mouse.Handler
	contextMenu contextMenuState

	oscEmit func(string)

	cursorShape               cursorShape
	cursorShapePending        cursorShape
	cursorShapeLastSentAt     time.Time
	cursorShapeFlushScheduled bool
	cursorShapeNow            func() time.Time

	projectPicker    list.Model
	layoutPicker     list.Model
	paneSwapPicker   list.Model
	commandPalette   list.Model
	settingsMenu     list.Model
	perfMenu         list.Model
	debugMenu        list.Model
	authFlow         authFlowState
	authDialogInput  textinput.Model
	authDialogTitle  string
	authDialogBody   string
	authDialogFooter string
	authDialogKind   authDialogKind
	gitProjects      []picker.ProjectItem
	dialogHelpOpen   bool

	confirmSession     string
	confirmProject     string
	confirmClose       string
	confirmCloseID     string
	confirmPaneSession string
	confirmPaneIndex   string
	confirmPaneID      string
	confirmPaneTitle   string
	confirmPaneRunning bool
	confirmQuitRunning int
	pendingQuit        quitAction

	renameInput     textinput.Model
	renameSession   string
	renamePane      string
	renamePaneIndex string

	swapSourceSession string
	swapSourcePane    string
	swapSourcePaneID  string

	projectRootInput textinput.Model

	toast      toastMessage
	refreshing bool

	selectionVersion uint64
	refreshInFlight  int
	refreshQueued    bool
	refreshSeq       uint64
	lastAppliedSeq   uint64
	refreshStarted   map[uint64]time.Time

	terminalFocus bool
	// terminalMouseDrag tracks an in-progress drag selection in terminal focus.
	terminalMouseDrag bool

	offlineScroll         map[string]int
	offlineScrollViewport map[string]int
	offlineScrollActive   bool
	offlineScrollPane     string

	autoStart *AutoStartSpec

	paneViewProfile   colorprofile.Profile
	paneViews         map[paneViewKey]paneViewEntry
	paneMouseMotion   map[string]bool
	paneInputDisabled map[string]struct{}

	resize resizeState

	paneViewSeq           map[paneViewKey]uint64
	paneViewLastReq       map[paneViewKey]time.Time
	paneViewFirst         map[string]struct{}
	paneViewSkipLog       map[string]struct{}
	paneViewPerfBurstDone bool

	paneViewInFlight       int
	paneViewQueuedIDs      map[string]struct{}
	paneViewInFlightByPane map[string]struct{}
	paneViewPumpScheduled  bool
	paneViewPumpBackoff    time.Duration
	paneLastSize           map[string]paneSize
	paneLastFallback       map[string]time.Time

	paneUpdatePerf    map[string]*paneUpdatePerf
	paneViewQueuedAt  map[string]time.Time
	panePerfLastPrune time.Time

	lastSplitVertical bool
	lastSplitSet      bool

	lastUrgentRefreshAt time.Time

	pekyBusy            bool
	pekySpinnerIndex    int
	pekyMessages        []agent.Message
	pekyDialogTitle     string
	pekyDialogFooter    string
	pekyDialogPrevState ViewState
	pekyDialogIsError   bool
	pekyViewport        viewport.Model
	pekyPromptLine      string
	pekyRunID           int64
	pekyCancel          context.CancelFunc
	pekyPromptLineID    int64
}

// NewModel creates a new peakypanes TUI model.
func NewModel(client *sessiond.Client) (*Model, error) {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("session client is required")
	}
	version := client.Version()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	paneViewClient, err := client.Clone(ctx)
	if err != nil {
		return nil, fmt.Errorf("pane view connection: %w", err)
	}

	m := &Model{
		client:             client,
		paneViewClient:     paneViewClient,
		daemonVersion:      version,
		state:              StateDashboard,
		tab:                TabDashboard,
		configPath:         configPath,
		expandedSessions:   make(map[string]bool),
		selectionByProject: make(map[string]selectionState),
		layoutEngines:      make(map[string]*layout.Engine),
		paneViews:          make(map[paneViewKey]paneViewEntry),
		paneMouseMotion:    make(map[string]bool),
		paneViewProfile:    detectPaneViewProfile(),
		paneInputDisabled:  make(map[string]struct{}),

		paneViewSeq:            make(map[paneViewKey]uint64),
		paneViewLastReq:        make(map[paneViewKey]time.Time),
		paneViewFirst:          make(map[string]struct{}),
		paneViewSkipLog:        make(map[string]struct{}),
		paneViewInFlightByPane: make(map[string]struct{}),
		paneLastSize:           make(map[string]paneSize),
		paneLastFallback:       make(map[string]time.Time),
		cursorShapeNow:         time.Now,
		pekyViewport:           viewport.New(0, 0),
	}
	m.resize.snap = true

	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "filter sessions"
	m.filterInput.CharLimit = 80
	m.filterInput.Width = 28

	m.quickReplyInput = textinput.New()
	m.quickReplyInput.Placeholder = "talk to your panes"
	m.quickReplyInput.CharLimit = 400
	m.quickReplyInput.Prompt = ""
	qrStyle := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)
	m.quickReplyInput.TextStyle = qrStyle
	m.quickReplyInput.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Background(theme.QuickReplyBg)
	m.quickReplyInput.PromptStyle = qrStyle
	m.quickReplyInput.Cursor.Style = qrStyle.Reverse(true)
	m.quickReplyInput.Focus()
	m.quickReplyHistoryIndex = -1
	m.quickReplyMenuIndex = -1
	m.quickReplyMode = quickReplyModePane

	m.setupProjectPicker()
	m.setupLayoutPicker()
	m.setupPaneSwapPicker()
	m.setupCommandPalette()
	m.setupSettingsMenu()
	m.setupPerformanceMenu()
	m.setupDebugMenu()

	configExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configExists = false
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		_ = paneViewClient.Close()
		return nil, err
	}
	m.config = cfg
	settings, err := defaultDashboardConfig(cfg.Dashboard)
	if err != nil {
		_ = paneViewClient.Close()
		return nil, err
	}
	m.settings = settings
	keys, err := buildDashboardKeyMap(cfg.Dashboard.Keymap)
	if err != nil {
		_ = paneViewClient.Close()
		return nil, err
	}
	m.keys = keys

	if needsProjectRootSetup(cfg, configExists) {
		m.openProjectRootSetup()
	}

	return m, nil
}
