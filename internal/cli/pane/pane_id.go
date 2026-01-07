package pane

import (
	"context"
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

const focusedPaneToken = "@focused"

func resolvePaneID(ctx context.Context, client *sessiond.Client, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !isFocusedPaneToken(value) {
		return value, nil
	}
	if client == nil {
		return "", fmt.Errorf("session client unavailable")
	}
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		return "", err
	}
	return resolvePaneIDFromSnapshot(value, resp)
}

func isFocusedPaneToken(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), focusedPaneToken)
}

func resolvePaneIDFromSnapshot(value string, resp sessiond.SnapshotResponse) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !isFocusedPaneToken(value) {
		return value, nil
	}
	paneID := strings.TrimSpace(resp.FocusedPaneID)
	if paneID == "" {
		return "", fmt.Errorf("focused pane unavailable; run pane focus first")
	}
	return paneID, nil
}

func resolvePaneSession(ctx context.Context, client *sessiond.Client, value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", fmt.Errorf("pane id is required")
	}
	if client == nil {
		return "", "", fmt.Errorf("session client unavailable")
	}
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		return "", "", err
	}
	paneID, err := resolvePaneIDFromSnapshot(value, resp)
	if err != nil {
		return "", "", err
	}
	sessionName, _, ok := findPaneByID(resp.Sessions, paneID)
	if !ok {
		return "", "", fmt.Errorf("pane id %q not found", paneID)
	}
	return sessionName, paneID, nil
}

func findPaneByID(sessions []native.SessionSnapshot, paneID string) (string, string, bool) {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return "", "", false
	}
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == paneID {
				return session.Name, pane.Index, true
			}
		}
	}
	return "", "", false
}
