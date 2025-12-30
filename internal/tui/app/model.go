package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

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
	togglePanes     key.Binding
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
	client         *sessiond.Client
	paneViewClient *sessiond.Client
	state          ViewState
	tab            DashboardTab
	width          int
	height         int

	configPath string

	data               DashboardData
	selection          selectionState
	selectionByProject map[string]selectionState
	settings           DashboardConfig
	config             *layout.Config
	projectConfigState map[string]projectConfigState

	keys *dashboardKeyMap

	expandedSessions map[string]bool

	filterInput  textinput.Model
	filterActive bool

	quickReplyInput textinput.Model

	mouse mouse.Handler

	projectPicker  list.Model
	layoutPicker   list.Model
	paneSwapPicker list.Model
	commandPalette list.Model
	settingsMenu   list.Model
	debugMenu      list.Model
	gitProjects    []picker.ProjectItem

	confirmSession     string
	confirmProject     string
	confirmClose       string
	confirmCloseID     string
	confirmPaneSession string
	confirmPaneIndex   string
	confirmPaneID      string
	confirmPaneTitle   string
	confirmPaneRunning bool

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

	terminalFocus bool

	autoStart *AutoStartSpec

	paneViewProfile   termenv.Profile
	paneViews         map[paneViewKey]string
	paneMouseMotion   map[string]bool
	paneInputDisabled map[string]struct{}

	paneViewSeq     map[paneViewKey]uint64
	paneViewLastReq map[paneViewKey]time.Time

	paneViewInFlight  bool
	paneViewQueued    bool
	paneViewQueuedIDs map[string]struct{}
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	paneViewClient, err := client.Clone(ctx)
	if err != nil {
		return nil, fmt.Errorf("pane view connection: %w", err)
	}

	m := &Model{
		client:             client,
		paneViewClient:     paneViewClient,
		state:              StateDashboard,
		tab:                TabDashboard,
		configPath:         configPath,
		expandedSessions:   make(map[string]bool),
		selectionByProject: make(map[string]selectionState),
		paneViews:          make(map[paneViewKey]string),
		paneMouseMotion:    make(map[string]bool),
		paneViewProfile:    detectPaneViewProfile(),
		paneInputDisabled:  make(map[string]struct{}),

		paneViewSeq:     make(map[paneViewKey]uint64),
		paneViewLastReq: make(map[paneViewKey]time.Time),
	}

	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "filter sessions"
	m.filterInput.CharLimit = 80
	m.filterInput.Width = 28

	m.quickReplyInput = textinput.New()
	m.quickReplyInput.Placeholder = "send a quick reply…"
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

	m.setupProjectPicker()
	m.setupLayoutPicker()
	m.setupPaneSwapPicker()
	m.setupCommandPalette()
	m.setupSettingsMenu()
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
