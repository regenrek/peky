package native

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// OutputSnapshot returns the last N output lines for a pane.
func (m *Manager) OutputSnapshot(paneID string, limit int) ([]OutputLine, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil, errors.New("native: pane id is required")
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil || pane.output == nil {
		return nil, fmt.Errorf("native: pane %q not found", paneID)
	}
	if limit <= 0 {
		limit = defaultOutputLineCap
	}
	return pane.output.snapshot(limit), nil
}

// OutputLinesSince returns output lines since the given sequence.
func (m *Manager) OutputLinesSince(paneID string, seq uint64) ([]OutputLine, uint64, bool, error) {
	if m == nil {
		return nil, seq, false, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil, seq, false, errors.New("native: pane id is required")
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil || pane.output == nil {
		return nil, seq, false, fmt.Errorf("native: pane %q not found", paneID)
	}
	lines, next, truncated := pane.output.linesSince(seq)
	return lines, next, truncated, nil
}

// WaitForOutput blocks until new output is available or the context is done.
func (m *Manager) WaitForOutput(ctx context.Context, paneID string) bool {
	if m == nil {
		return false
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return false
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil || pane.output == nil {
		return false
	}
	return pane.output.wait(ctx)
}

// SubscribeRawOutput subscribes to raw output chunks for a pane.
func (m *Manager) SubscribeRawOutput(paneID string, buffer int) (<-chan OutputChunk, func(), error) {
	if m == nil {
		return nil, func() {}, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil, func() {}, errors.New("native: pane id is required")
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil || pane.output == nil {
		return nil, func() {}, fmt.Errorf("native: pane %q not found", paneID)
	}
	_, ch, cancel := pane.output.subscribeRaw(buffer)
	return ch, cancel, nil
}

// PaneTags returns tags for a pane.
func (m *Manager) PaneTags(paneID string) ([]string, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil, errors.New("native: pane id is required")
	}
	m.mu.RLock()
	pane := m.panes[paneID]
	m.mu.RUnlock()
	if pane == nil {
		return nil, fmt.Errorf("native: pane %q not found", paneID)
	}
	return append([]string(nil), pane.Tags...), nil
}

// AddPaneTags adds tags to a pane.
func (m *Manager) AddPaneTags(paneID string, tags []string) ([]string, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil, errors.New("native: pane id is required")
	}
	normalized := normalizeTags(tags)
	if len(normalized) == 0 {
		return nil, errors.New("native: tags are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	pane := m.panes[paneID]
	if pane == nil {
		return nil, fmt.Errorf("native: pane %q not found", paneID)
	}
	existing := make(map[string]struct{})
	for _, tag := range pane.Tags {
		existing[tag] = struct{}{}
	}
	for _, tag := range normalized {
		existing[tag] = struct{}{}
	}
	pane.Tags = sortedTags(existing)
	return append([]string(nil), pane.Tags...), nil
}

// RemovePaneTags removes tags from a pane.
func (m *Manager) RemovePaneTags(paneID string, tags []string) ([]string, error) {
	if m == nil {
		return nil, errors.New("native: manager is nil")
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil, errors.New("native: pane id is required")
	}
	normalized := normalizeTags(tags)
	if len(normalized) == 0 {
		return nil, errors.New("native: tags are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	pane := m.panes[paneID]
	if pane == nil {
		return nil, fmt.Errorf("native: pane %q not found", paneID)
	}
	if len(pane.Tags) == 0 {
		return nil, nil
	}
	remove := make(map[string]struct{})
	for _, tag := range normalized {
		remove[tag] = struct{}{}
	}
	kept := make([]string, 0, len(pane.Tags))
	for _, tag := range pane.Tags {
		if _, ok := remove[tag]; !ok {
			kept = append(kept, tag)
		}
	}
	pane.Tags = kept
	return append([]string(nil), pane.Tags...), nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag)
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func sortedTags(tags map[string]struct{}) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	for tag := range tags {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}
