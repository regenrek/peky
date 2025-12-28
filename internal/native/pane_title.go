package native

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/pathutil"
)

type paneTitleKind int

const (
	paneTitleExplicit paneTitleKind = iota
	paneTitleWindow
	paneTitlePath
	paneTitleFallback
)

type paneTitleEntry struct {
	pane  *Pane
	title string
	kind  paneTitleKind
}

type paneTitleStats struct {
	total    int
	eligible int
	used     int
}

func resolveSessionPaneTitles(session *Session) map[*Pane]string {
	if session == nil || len(session.Panes) == 0 {
		return map[*Pane]string{}
	}

	entries := make([]paneTitleEntry, 0, len(session.Panes))
	for _, pane := range session.Panes {
		if pane == nil {
			continue
		}
		winTitle := ""
		if pane.window != nil {
			winTitle = pane.window.Title()
		}
		title, kind := resolvePaneTitle(session.Path, pane.Title, winTitle, pane.Index)
		entries = append(entries, paneTitleEntry{pane: pane, title: title, kind: kind})
	}

	return dedupePaneTitles(entries)
}

func dedupePaneTitles(entries []paneTitleEntry) map[*Pane]string {
	out := make(map[*Pane]string, len(entries))
	stats := make(map[string]*paneTitleStats, len(entries))
	for _, entry := range entries {
		if entry.title == "" {
			continue
		}
		rec := stats[entry.title]
		if rec == nil {
			rec = &paneTitleStats{}
			stats[entry.title] = rec
		}
		rec.total++
		if entry.kind == paneTitlePath || entry.kind == paneTitleFallback {
			rec.eligible++
		}
	}

	for _, entry := range entries {
		title := entry.title
		if title == "" {
			continue
		}
		rec := stats[title]
		if rec == nil || rec.total <= 1 {
			out[entry.pane] = title
			continue
		}
		if entry.kind != paneTitlePath && entry.kind != paneTitleFallback {
			out[entry.pane] = title
			continue
		}
		rec.used++
		if rec.eligible == rec.total {
			if rec.used == 1 {
				out[entry.pane] = title
			} else {
				out[entry.pane] = fmt.Sprintf("%s #%d", title, rec.used)
			}
			continue
		}
		out[entry.pane] = fmt.Sprintf("%s #%d", title, rec.used+1)
	}

	return out
}

func resolvePaneTitle(sessionPath, paneTitle, windowTitle, paneIndex string) (string, paneTitleKind) {
	paneTitle = strings.TrimSpace(paneTitle)
	windowTitle = strings.TrimSpace(windowTitle)

	if windowTitle != "" {
		if pathTitle, ok := extractPathFromTitle(windowTitle); ok {
			if paneTitle != "" {
				return paneTitle, paneTitleExplicit
			}
			if short := shortenPathTitle(pathTitle, sessionPath); short != "" {
				return short, paneTitlePath
			}
			return windowTitle, paneTitlePath
		}
		return windowTitle, paneTitleWindow
	}
	if paneTitle != "" {
		return paneTitle, paneTitleExplicit
	}
	paneIndex = strings.TrimSpace(paneIndex)
	if paneIndex != "" {
		return fmt.Sprintf("pane %s", paneIndex), paneTitleFallback
	}
	return "pane", paneTitleFallback
}

func extractPathFromTitle(title string) (string, bool) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "", false
	}
	if path := pathCandidate(trimmed); path != "" {
		return path, true
	}
	if idx := strings.LastIndex(trimmed, ":"); idx != -1 {
		if path := pathCandidate(trimmed[idx+1:]); path != "" {
			return path, true
		}
	}
	if idx := strings.LastIndex(trimmed, " - "); idx != -1 {
		if path := pathCandidate(trimmed[idx+3:]); path != "" {
			return path, true
		}
	}
	return "", false
}

func pathCandidate(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return ""
	}
	if looksLikePathPrefix(segment) {
		return segment
	}
	return ""
}

func looksLikePathPrefix(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		return true
	}
	if len(value) >= 2 && isAlpha(value[0]) && value[1] == ':' {
		return true
	}
	return false
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func shortenPathTitle(pathTitle, sessionPath string) string {
	pathTitle = strings.TrimSpace(pathTitle)
	if pathTitle == "" {
		return ""
	}
	sessionPath = strings.TrimSpace(sessionPath)

	expanded := pathutil.ExpandUser(pathTitle)
	expanded = strings.TrimSpace(expanded)
	if expanded != "" {
		expanded = filepath.Clean(expanded)
	}

	if sessionPath != "" {
		sessionPath = filepath.Clean(sessionPath)
		if expanded != "" && filepath.IsAbs(expanded) && filepath.IsAbs(sessionPath) {
			rel, err := filepath.Rel(sessionPath, expanded)
			if err == nil {
				if rel == "." {
					if base := sessionBaseName(sessionPath); base != "" {
						return base
					}
				} else if !isOutsideRel(rel) {
					rel = trimRelPrefix(rel)
					rel = filepath.ToSlash(rel)
					rel = compressPathForDisplay(rel, 2)
					base := sessionBaseName(sessionPath)
					if base == "" {
						return rel
					}
					if rel == "" {
						return base
					}
					return base + ":" + rel
				}
			}
		}
	}

	display := pathTitle
	if expanded != "" {
		display = pathutil.ShortenUser(expanded)
	}
	return compressPathForDisplay(display, 2)
}

func sessionBaseName(sessionPath string) string {
	if sessionPath == "" {
		return ""
	}
	base := filepath.Base(sessionPath)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func isOutsideRel(rel string) bool {
	if rel == ".." {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(rel, ".."+sep)
}

func trimRelPrefix(rel string) string {
	rel = strings.TrimPrefix(rel, "."+string(filepath.Separator))
	rel = strings.TrimPrefix(rel, "./")
	return rel
}

func compressPathForDisplay(path string, maxSegments int) string {
	if maxSegments <= 0 {
		return strings.TrimSpace(path)
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path == "~" {
		return "~"
	}
	path = filepath.ToSlash(path)

	prefix := ""
	rest := path
	if strings.HasPrefix(rest, "~") {
		prefix = "~"
		rest = strings.TrimPrefix(rest, "~")
		rest = strings.TrimPrefix(rest, "/")
	} else if len(rest) >= 2 && rest[1] == ':' {
		prefix = rest[:2]
		rest = rest[2:]
		rest = strings.TrimPrefix(rest, "/")
	} else if strings.HasPrefix(rest, "/") {
		rest = strings.TrimPrefix(rest, "/")
	}

	parts := strings.FieldsFunc(rest, func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		if prefix != "" {
			return prefix
		}
		if strings.HasPrefix(path, "/") {
			return "/"
		}
		return ""
	}
	if len(parts) > maxSegments {
		parts = parts[len(parts)-maxSegments:]
	}
	joined := strings.Join(parts, "/")
	if prefix == "~" {
		return "~/" + joined
	}
	if prefix != "" {
		return prefix + "/" + joined
	}
	return joined
}
