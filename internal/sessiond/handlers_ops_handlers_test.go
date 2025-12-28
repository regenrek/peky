package sessiond

import "testing"

func TestHandlerErrorPaths(t *testing.T) {
	d := &Daemon{}

	cases := []struct {
		name string
		call func() error
	}{
		{
			name: "handleKillSession invalid name",
			call: func() error {
				payload, _ := encodePayload(KillSessionRequest{Name: ""})
				_, err := d.handleKillSession(payload)
				return err
			},
		},
		{
			name: "handleKillSession manager missing",
			call: func() error {
				payload, _ := encodePayload(KillSessionRequest{Name: "demo"})
				_, err := d.handleKillSession(payload)
				return err
			},
		},
		{
			name: "handleRenameSession invalid",
			call: func() error {
				payload, _ := encodePayload(RenameSessionRequest{OldName: "", NewName: "new"})
				_, err := d.handleRenameSession(payload)
				return err
			},
		},
		{
			name: "handleRenameSession manager missing",
			call: func() error {
				payload, _ := encodePayload(RenameSessionRequest{OldName: "old", NewName: "new"})
				_, err := d.handleRenameSession(payload)
				return err
			},
		},
		{
			name: "handleRenamePane empty title",
			call: func() error {
				payload, _ := encodePayload(RenamePaneRequest{SessionName: "session", PaneIndex: "1", NewTitle: " "})
				_, err := d.handleRenamePane(payload)
				return err
			},
		},
		{
			name: "handleRenamePane manager missing",
			call: func() error {
				payload, _ := encodePayload(RenamePaneRequest{SessionName: "session", PaneIndex: "1", NewTitle: "title"})
				_, err := d.handleRenamePane(payload)
				return err
			},
		},
		{
			name: "handleSplitPane invalid pane",
			call: func() error {
				payload, _ := encodePayload(SplitPaneRequest{SessionName: "session", PaneIndex: ""})
				_, err := d.handleSplitPane(payload)
				return err
			},
		},
		{
			name: "handleSplitPane manager missing",
			call: func() error {
				payload, _ := encodePayload(SplitPaneRequest{SessionName: "session", PaneIndex: "1"})
				_, err := d.handleSplitPane(payload)
				return err
			},
		},
		{
			name: "handleClosePane invalid pane",
			call: func() error {
				payload, _ := encodePayload(ClosePaneRequest{SessionName: "session", PaneIndex: ""})
				_, err := d.handleClosePane(payload)
				return err
			},
		},
		{
			name: "handleClosePane manager missing",
			call: func() error {
				payload, _ := encodePayload(ClosePaneRequest{SessionName: "session", PaneIndex: "1"})
				_, err := d.handleClosePane(payload)
				return err
			},
		},
		{
			name: "handleSwapPanes invalid pane",
			call: func() error {
				payload, _ := encodePayload(SwapPanesRequest{SessionName: "session", PaneA: "", PaneB: "1"})
				_, err := d.handleSwapPanes(payload)
				return err
			},
		},
		{
			name: "handleSwapPanes manager missing",
			call: func() error {
				payload, _ := encodePayload(SwapPanesRequest{SessionName: "session", PaneA: "1", PaneB: "2"})
				_, err := d.handleSwapPanes(payload)
				return err
			},
		},
		{
			name: "handleSendInput missing pane",
			call: func() error {
				payload, _ := encodePayload(SendInputRequest{PaneID: ""})
				_, err := d.handleSendInput(payload)
				return err
			},
		},
		{
			name: "handleSendInput manager missing",
			call: func() error {
				payload, _ := encodePayload(SendInputRequest{PaneID: "pane", Input: []byte("hi")})
				_, err := d.handleSendInput(payload)
				return err
			},
		},
		{
			name: "handleSendMouse manager missing",
			call: func() error {
				payload, _ := encodePayload(SendMouseRequest{PaneID: "pane", Event: MouseEventPayload{X: 1, Y: 1, Button: 1, Action: MouseActionPress}})
				_, err := d.handleSendMouse(payload)
				return err
			},
		},
		{
			name: "handleResizePane missing pane",
			call: func() error {
				payload, _ := encodePayload(ResizePaneRequest{PaneID: ""})
				_, err := d.handleResizePane(payload)
				return err
			},
		},
		{
			name: "handleResizePane manager missing",
			call: func() error {
				payload, _ := encodePayload(ResizePaneRequest{PaneID: "pane", Cols: 10, Rows: 10})
				_, err := d.handleResizePane(payload)
				return err
			},
		},
		{
			name: "handlePaneView missing pane",
			call: func() error {
				payload, _ := encodePayload(PaneViewRequest{PaneID: ""})
				_, err := d.handlePaneView(payload)
				return err
			},
		},
		{
			name: "handlePaneView manager missing",
			call: func() error {
				payload, _ := encodePayload(PaneViewRequest{PaneID: "pane", Cols: 10, Rows: 10})
				_, err := d.handlePaneView(payload)
				return err
			},
		},
		{
			name: "handleTerminalActionPayload missing pane",
			call: func() error {
				payload, _ := encodePayload(TerminalActionRequest{PaneID: ""})
				_, err := d.handleTerminalActionPayload(payload)
				return err
			},
		},
		{
			name: "handleHandleKey missing pane",
			call: func() error {
				payload, _ := encodePayload(TerminalKeyRequest{PaneID: ""})
				_, err := d.handleHandleKey(payload)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
