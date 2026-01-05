package app

import (
	"strings"

	"github.com/regenrek/peakypanes/internal/filelist"
	"github.com/regenrek/peakypanes/internal/layout"
)

type quickReplyMode int

const (
	quickReplyModePane quickReplyMode = iota
	quickReplyModePeky
)

type quickReplyFileCache struct {
	paneID string
	cwd    string
	dir    string
	cfg    filelist.Config

	entries []filelist.Entry
	err     error
}

func (m *Model) quickReplyMenuState() quickReplyMenu {
	if m == nil || m.state != StateDashboard || m.filterActive || m.terminalFocus {
		return quickReplyMenu{}
	}
	if menu := m.authMenuState(); menu.kind != quickReplyMenuNone {
		return menu
	}
	if menu := m.modelMenuState(); menu.kind != quickReplyMenuNone {
		return menu
	}
	if menu := m.slashMenuState(); menu.kind != quickReplyMenuNone {
		return menu
	}
	if menu := m.atMenuState(); menu.kind != quickReplyMenuNone {
		return menu
	}
	return quickReplyMenu{}
}

func (m *Model) slashMenuState() quickReplyMenu {
	if m.quickReplyMode == quickReplyModePeky {
		value := strings.ToLower(strings.TrimSpace(m.quickReplyInput.Value()))
		if !strings.HasPrefix(value, "/") || (!strings.HasPrefix(value, "/auth") && !strings.HasPrefix(value, "/model")) {
			return quickReplyMenu{}
		}
	}
	ctx := slashInputForSuggestions(m.quickReplyInput.Value())
	if !ctx.Active || ctx.HasArgs {
		return quickReplyMenu{}
	}
	suggestions := m.slashSuggestions()
	if len(suggestions) == 0 {
		return quickReplyMenu{}
	}
	return quickReplyMenu{
		kind:        quickReplyMenuSlash,
		prefix:      strings.ToLower(ctx.Prefix),
		suggestions: suggestions,
	}
}

func (m *Model) updateQuickReplyMenuSelection() {
	menu := m.quickReplyMenuState()
	if menu.kind == quickReplyMenuNone {
		m.resetQuickReplyMenu()
		return
	}
	if menu.kind != m.quickReplyMenuKind || menu.prefix != m.quickReplyMenuPrefix {
		m.quickReplyMenuKind = menu.kind
		m.quickReplyMenuPrefix = menu.prefix
		m.quickReplyMenuIndex = -1
	}
	if len(menu.suggestions) == 0 {
		m.quickReplyMenuIndex = -1
		return
	}
	if m.quickReplyMenuIndex < 0 || m.quickReplyMenuIndex >= len(menu.suggestions) {
		m.quickReplyMenuIndex = 0
	}
}

func (m *Model) moveQuickReplyMenuSelection(delta int) bool {
	menu := m.quickReplyMenuState()
	if menu.kind == quickReplyMenuNone || len(menu.suggestions) == 0 {
		return false
	}
	if m.quickReplyMenuIndex < 0 || m.quickReplyMenuIndex >= len(menu.suggestions) {
		m.quickReplyMenuIndex = 0
	}
	count := len(menu.suggestions)
	next := (m.quickReplyMenuIndex + delta) % count
	if next < 0 {
		next += count
	}
	m.quickReplyMenuIndex = next
	return true
}

func (m *Model) applyQuickReplyMenuCompletion() bool {
	menu := m.quickReplyMenuState()
	applied := false
	switch menu.kind {
	case quickReplyMenuSlash:
		applied = m.applySlashCompletion()
	case quickReplyMenuAt:
		applied = m.applyAtCompletion()
	case quickReplyMenuAuthProvider:
		applied = m.applyAuthProviderCompletion()
	case quickReplyMenuAuthMethod:
		applied = m.applyAuthMethodCompletion()
	case quickReplyMenuModel:
		applied = m.applyModelCompletion()
	}
	if applied {
		m.updateQuickReplyMenuSelection()
	}
	return applied
}

func (m *Model) resetQuickReplyMenu() {
	m.quickReplyMenuKind = quickReplyMenuNone
	m.quickReplyMenuIndex = -1
	m.quickReplyMenuPrefix = ""
}

func (m *Model) pekyConfig() layout.Config {
	if m == nil || m.config == nil {
		cfg := layout.Config{}
		layout.ApplyDefaults(&cfg)
		return cfg
	}
	cfg := *m.config
	layout.ApplyDefaults(&cfg)
	return cfg
}

func quickReplyFilesConfig(cfg layout.QuickReplyConfig) filelist.Config {
	files := cfg.Files
	includeHidden := false
	if files.ShowHidden != nil {
		includeHidden = *files.ShowHidden
	}
	return filelist.Config{
		MaxDepth:      files.MaxDepth,
		MaxItems:      files.MaxItems,
		IncludeHidden: includeHidden,
	}
}
