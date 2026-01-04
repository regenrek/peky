package app

import (
	"log/slog"
	"strings"

	"github.com/regenrek/peakypanes/internal/filelist"
	"github.com/regenrek/peakypanes/internal/pekyconfig"
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
		return quickReplyMenu{}
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

func (m *Model) loadPekyConfig() pekyconfig.Config {
	if m == nil || m.pekyConfigLoader == nil {
		return pekyconfig.Defaults()
	}
	cfg, err := m.pekyConfigLoader.Load()
	if err != nil {
		if m.pekyConfigErr == nil || m.pekyConfigErr.Error() != err.Error() {
			slog.Warn("peky config reload failed", "err", err)
		}
		m.pekyConfigErr = err
		return pekyconfig.Defaults()
	}
	m.pekyConfigErr = nil
	m.pekyConfig = cfg
	return cfg
}

func quickReplyFilesConfig(cfg pekyconfig.Config) filelist.Config {
	files := cfg.QuickReply.Files
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
