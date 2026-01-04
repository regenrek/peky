package app

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sahilm/fuzzy"

	"github.com/regenrek/peakypanes/internal/filelist"
)

type atGroup int

const (
	atGroupNone atGroup = iota
	atGroupPanes
	atGroupSessions
)

type atEntryKind int

const (
	atEntryGroup atEntryKind = iota
	atEntryItem
)

type atEntry struct {
	Text         string
	Desc         string
	Kind         atEntryKind
	IsDir        bool
	MatchIndexes []int
}

type atInputContext struct {
	Active    bool
	Query     string
	Start     int
	End       int
	Group     atGroup
	FileDir   string
	FileQuery string
}

func atInputContextFor(input string, cursor int) atInputContext {
	runes := []rune(input)
	if cursor < 0 || cursor > len(runes) {
		cursor = len(runes)
	}
	start := cursor
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	end := cursor
	for end < len(runes) && !unicode.IsSpace(runes[end]) {
		end++
	}
	if start == end {
		return atInputContext{}
	}
	token := string(runes[start:end])
	if !strings.HasPrefix(token, "@") {
		return atInputContext{}
	}
	query := strings.TrimPrefix(token, "@")
	ctx := atInputContext{
		Active: true,
		Query:  query,
		Start:  start,
		End:    end,
	}
	ctx.Group = atGroupFromQuery(query)
	if ctx.Group == atGroupNone {
		ctx.FileDir, ctx.FileQuery = parseAtFileQuery(query)
	}
	return ctx
}

func atGroupFromQuery(query string) atGroup {
	switch strings.ToLower(strings.TrimSpace(query)) {
	case "panes", "pane":
		return atGroupPanes
	case "sessions", "session":
		return atGroupSessions
	default:
		return atGroupNone
	}
}

func (m *Model) atMenuState() quickReplyMenu {
	ctx := atInputContextFor(m.quickReplyInput.Value(), m.quickReplyInput.Position())
	if !ctx.Active {
		return quickReplyMenu{}
	}
	entries := m.atSuggestionEntries(ctx)
	if len(entries) == 0 {
		return quickReplyMenu{}
	}
	prefix := strings.ToLower(ctx.Query)
	if ctx.Group != atGroupNone {
		prefix = ""
	}
	suggestions := make([]quickReplySuggestion, len(entries))
	for i, entry := range entries {
		suggestions[i] = quickReplySuggestion{
			Text:         entry.Text,
			Desc:         entry.Desc,
			MatchIndexes: entry.MatchIndexes,
		}
	}
	return quickReplyMenu{
		kind:        quickReplyMenuAt,
		prefix:      prefix,
		suggestions: suggestions,
	}
}

func (m *Model) atSuggestionEntries(ctx atInputContext) []atEntry {
	entries := m.atBaseEntries(ctx)
	if len(entries) == 0 {
		return nil
	}
	query := strings.TrimSpace(ctx.FileQuery)
	if ctx.Group != atGroupNone {
		query = ""
	}
	if query == "" {
		return entries
	}
	matches := fuzzy.FindFrom(query, atEntrySource(entries))
	if len(matches) == 0 {
		return nil
	}
	filtered := make([]atEntry, 0, len(matches))
	for _, match := range matches {
		entry := entries[match.Index]
		entry.MatchIndexes = match.MatchedIndexes
		filtered = append(filtered, entry)
	}
	return filtered
}

type atEntrySource []atEntry

func (s atEntrySource) String(i int) string {
	return s[i].Text
}

func (s atEntrySource) Len() int { return len(s) }

func (m *Model) atBaseEntries(ctx atInputContext) []atEntry {
	switch ctx.Group {
	case atGroupPanes:
		return m.atPaneEntries()
	case atGroupSessions:
		return m.atSessionEntries()
	}
	if strings.TrimSpace(ctx.FileDir) != "" {
		entries, err := m.atFileEntries(ctx.FileDir)
		if err != nil {
			return nil
		}
		return entries
	}
	return m.atRootEntries()
}

func (m *Model) atRootEntries() []atEntry {
	entries := make([]atEntry, 0, 16)
	entries = append(entries,
		atEntry{Text: "@panes", Desc: "pane targets", Kind: atEntryGroup},
		atEntry{Text: "@pane", Desc: "pane targets", Kind: atEntryGroup},
		atEntry{Text: "@sessions", Desc: "session targets", Kind: atEntryGroup},
		atEntry{Text: "@session", Desc: "session targets", Kind: atEntryGroup},
	)
	files, err := m.atFileEntries("")
	if err == nil {
		entries = append(entries, files...)
	}
	return entries
}

func (m *Model) atPaneEntries() []atEntry {
	panes := m.sessionPanes()
	if len(panes) == 0 {
		return nil
	}
	entries := make([]atEntry, 0, len(panes)+1)
	entries = append(entries, atEntry{Text: "@allpanes", Desc: "all panes", Kind: atEntryItem})
	for _, pane := range panes {
		text := "@pane-" + pane.ID
		desc := strings.TrimSpace(pane.Title)
		if desc == "" {
			desc = "pane " + pane.Index
		}
		entries = append(entries, atEntry{Text: text, Desc: desc, Kind: atEntryItem})
	}
	return entries
}

func (m *Model) atSessionEntries() []atEntry {
	project := m.selectedProject()
	if project == nil {
		return nil
	}
	entries := make([]atEntry, 0, len(project.Sessions))
	for _, session := range project.Sessions {
		text := "@session-" + session.Name
		entries = append(entries, atEntry{Text: text, Desc: session.Path, Kind: atEntryItem})
	}
	return entries
}

func parseAtFileQuery(query string) (string, string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", ""
	}
	query = strings.ReplaceAll(query, "\\", "/")
	if strings.HasPrefix(query, "/") {
		return "", ""
	}
	lastSlash := strings.LastIndex(query, "/")
	if lastSlash < 0 {
		return "", query
	}
	dir := strings.TrimSuffix(query[:lastSlash], "/")
	filter := strings.TrimPrefix(query[lastSlash+1:], "/")
	return dir, filter
}

func (m *Model) atFileEntries(relDir string) ([]atEntry, error) {
	pane := m.selectedPane()
	if pane == nil {
		return nil, errors.New("no pane selected")
	}
	cfg := quickReplyFilesConfig(m.loadPekyConfig())
	cwd := strings.TrimSpace(pane.Cwd)
	if cwd == "" {
		return nil, errors.New("pane cwd unavailable")
	}
	root, relDir, err := resolveAtFileRoot(cwd, relDir)
	if err != nil {
		return nil, err
	}
	if m.quickReplyFileCache.paneID != pane.ID || m.quickReplyFileCache.cwd != cwd ||
		m.quickReplyFileCache.dir != relDir || m.quickReplyFileCache.cfg != cfg {
		entries, _, err := filelist.List(root, cfg)
		m.quickReplyFileCache = quickReplyFileCache{
			paneID:  pane.ID,
			cwd:     cwd,
			dir:     relDir,
			cfg:     cfg,
			entries: entries,
			err:     err,
		}
	}
	if m.quickReplyFileCache.err != nil {
		return nil, m.quickReplyFileCache.err
	}
	out := make([]atEntry, 0, len(m.quickReplyFileCache.entries))
	for _, entry := range m.quickReplyFileCache.entries {
		display := entry.Path
		if relDir != "" {
			display = filepath.ToSlash(filepath.Join(relDir, entry.Path))
		}
		if entry.IsDir && !strings.HasSuffix(display, "/") {
			display += "/"
		}
		out = append(out, atEntry{Text: "@" + display, Kind: atEntryItem, IsDir: entry.IsDir})
	}
	return out, nil
}

func (m *Model) sessionPanes() []PaneItem {
	project := m.selectedProject()
	if project == nil {
		return nil
	}
	var panes []PaneItem
	for _, session := range project.Sessions {
		if len(session.Panes) == 0 {
			continue
		}
		panes = append(panes, session.Panes...)
	}
	return panes
}

func (m *Model) applyAtCompletion() bool {
	ctx := atInputContextFor(m.quickReplyInput.Value(), m.quickReplyInput.Position())
	if !ctx.Active {
		return false
	}
	entries := m.atSuggestionEntries(ctx)
	if len(entries) == 0 {
		return false
	}
	if m.quickReplyMenuIndex < 0 || m.quickReplyMenuIndex >= len(entries) {
		return false
	}
	entry := entries[m.quickReplyMenuIndex]
	replacement := entry.Text
	if entry.IsDir && !strings.HasSuffix(replacement, "/") {
		replacement += "/"
	}
	applySpace := entry.Kind == atEntryItem && !entry.IsDir
	m.replaceQuickReplyToken(ctx.Start, ctx.End, replacement, applySpace)
	return true
}

func (m *Model) replaceQuickReplyToken(start, end int, replacement string, addSpace bool) {
	runes := []rune(m.quickReplyInput.Value())
	if start < 0 || end < start || start > len(runes) || end > len(runes) {
		return
	}
	out := make([]rune, 0, len(runes)+len(replacement)+1)
	out = append(out, runes[:start]...)
	out = append(out, []rune(replacement)...)
	cursor := len(out)
	if addSpace {
		if end >= len(runes) || (end < len(runes) && unicode.IsSpace(runes[end])) {
			out = append(out, ' ')
			cursor++
		}
	}
	out = append(out, runes[end:]...)
	m.quickReplyInput.SetValue(string(out))
	m.quickReplyInput.SetCursor(cursor)
}

func resolveAtFileRoot(cwd, relDir string) (string, string, error) {
	base := strings.TrimSpace(cwd)
	if base == "" {
		return "", "", errors.New("empty cwd")
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", "", err
	}
	relDir = strings.TrimSpace(relDir)
	if relDir == "" {
		return baseAbs, "", nil
	}
	relDir = filepath.Clean(filepath.FromSlash(relDir))
	if relDir == "." {
		return baseAbs, "", nil
	}
	if filepath.IsAbs(relDir) || strings.HasPrefix(relDir, "..") {
		return "", "", errors.New("invalid directory")
	}
	target := filepath.Join(baseAbs, relDir)
	rel, err := filepath.Rel(baseAbs, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", errors.New("invalid directory")
	}
	return target, filepath.ToSlash(relDir), nil
}
