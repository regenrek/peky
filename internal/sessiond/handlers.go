package sessiond

import "fmt"

func (d *Daemon) handleRequest(env Envelope) Envelope {
	resp := Envelope{
		Kind: EnvelopeResponse,
		Op:   env.Op,
		ID:   env.ID,
	}
	payload, err := d.handleRequestPayload(env)
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	resp.Payload = payload
	return resp
}

func (d *Daemon) handleRequestPayload(env Envelope) ([]byte, error) {
	handler, ok := requestHandlers[env.Op]
	if !ok {
		return nil, fmt.Errorf("sessiond: unknown op %q", env.Op)
	}
	return handler(d, env.Payload)
}

type requestHandler func(d *Daemon, payload []byte) ([]byte, error)

var requestHandlers = map[Op]requestHandler{
	OpHello: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleHello(payload)
	},
	OpSessionNames: func(d *Daemon, _ []byte) ([]byte, error) {
		return d.handleSessionNames()
	},
	OpSnapshot: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSnapshot(payload)
	},
	OpStartSession: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleStartSession(payload)
	},
	OpKillSession: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleKillSession(payload)
	},
	OpRenameSession: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleRenameSession(payload)
	},
	OpSessionFocus: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSessionFocus(payload)
	},
	OpRenamePane: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleRenamePane(payload)
	},
	OpSplitPane: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSplitPane(payload)
	},
	OpClosePane: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleClosePane(payload)
	},
	OpSwapPanes: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSwapPanes(payload)
	},
	OpSetPaneTool: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSetPaneTool(payload)
	},
	OpSetPaneBackground: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSetPaneBackground(payload)
	},
	OpSendInput: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSendInput(payload)
	},
	OpSendInputTool: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSendInputTool(payload)
	},
	OpSendMouse: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleSendMouse(payload)
	},
	OpResizePane: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleResizePane(payload)
	},
	OpResetPaneSizes: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleResetPaneSizes(payload)
	},
	OpZoomPane: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleZoomPane(payload)
	},
	OpPaneView: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneView(payload)
	},
	OpPaneOutput: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneOutput(payload)
	},
	OpPaneSnapshot: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneSnapshot(payload)
	},
	OpPaneHistory: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneHistory(payload)
	},
	OpPaneWait: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneWait(payload)
	},
	OpPaneTagAdd: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneTagAdd(payload)
	},
	OpPaneTagRemove: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneTagRemove(payload)
	},
	OpPaneTagList: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneTagList(payload)
	},
	OpPaneFocus: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneFocus(payload)
	},
	OpPaneSignal: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handlePaneSignal(payload)
	},
	OpRelayCreate: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleRelayCreate(payload)
	},
	OpRelayList: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleRelayList(payload)
	},
	OpRelayStop: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleRelayStop(payload)
	},
	OpRelayStopAll: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleRelayStopAll(payload)
	},
	OpEventsReplay: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleEventsReplay(payload)
	},
	OpTerminalAction: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleTerminalActionPayload(payload)
	},
	OpHandleKey: func(d *Daemon, payload []byte) ([]byte, error) {
		return d.handleHandleKey(payload)
	},
}
