package native

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/profiling"
)

const (
	defaultSendDelay   = 750 * time.Millisecond
	defaultSendTimeout = 10 * time.Second
	maxSendDelay       = 5 * time.Minute
	maxSendBytes       = 64 * 1024
)

type paneSendAction struct {
	text        string
	delay       time.Duration
	submit      bool
	submitDelay time.Duration
	waitOutput  bool
}

func (m *Manager) dispatchLayoutSends(session *Session, layoutCfg *layout.LayoutConfig) {
	if m == nil || session == nil || layoutCfg == nil {
		return
	}
	queues := buildPaneSendQueues(layoutCfg, session.Panes)
	if len(queues) == 0 {
		return
	}
	for paneID, actions := range queues {
		if paneID == "" || len(actions) == 0 {
			continue
		}
		paneID := paneID
		actions := append([]paneSendAction(nil), actions...)
		go m.runPaneSendQueue(paneID, actions)
	}
}

func buildPaneSendQueues(layoutCfg *layout.LayoutConfig, panes []*Pane) map[string][]paneSendAction {
	if layoutCfg == nil || len(panes) == 0 {
		return nil
	}
	queues := make(map[string][]paneSendAction)
	appendBroadcastSendQueues(queues, layoutCfg.BroadcastSend, panes)
	indexToID := paneIndexToID(panes)
	appendDirectSendQueues(queues, layoutCfg.Panes, indexToID)
	if len(queues) == 0 {
		return nil
	}
	return queues
}

func appendBroadcastSendQueues(queues map[string][]paneSendAction, actions []layout.SendAction, panes []*Pane) {
	if len(actions) == 0 || len(panes) == 0 {
		return
	}
	for _, action := range actions {
		normalized, ok := normalizeSendAction(action)
		if !ok {
			continue
		}
		for _, pane := range panes {
			if pane == nil || strings.TrimSpace(pane.ID) == "" {
				continue
			}
			queues[pane.ID] = append(queues[pane.ID], normalized)
		}
	}
}

func paneIndexToID(panes []*Pane) map[string]string {
	indexToID := make(map[string]string, len(panes))
	for _, pane := range panes {
		if pane == nil {
			continue
		}
		index := strings.TrimSpace(pane.Index)
		if index == "" {
			continue
		}
		indexToID[index] = pane.ID
	}
	return indexToID
}

func appendDirectSendQueues(queues map[string][]paneSendAction, paneDefs []layout.PaneDef, indexToID map[string]string) {
	if len(paneDefs) == 0 || len(indexToID) == 0 {
		return
	}
	for i, paneDef := range paneDefs {
		if len(paneDef.DirectSend) == 0 {
			continue
		}
		paneID := indexToID[strconv.Itoa(i)]
		if strings.TrimSpace(paneID) == "" {
			continue
		}
		for _, action := range paneDef.DirectSend {
			normalized, ok := normalizeSendAction(action)
			if !ok {
				continue
			}
			queues[paneID] = append(queues[paneID], normalized)
		}
	}
}

func normalizeSendAction(action layout.SendAction) (paneSendAction, bool) {
	if strings.TrimSpace(action.Text) == "" {
		return paneSendAction{}, false
	}
	explicitDelay := action.SendDelayMS != nil
	delay := defaultSendDelay
	if explicitDelay {
		delay = normalizeSendDelay(*action.SendDelayMS)
	}
	if delay > maxSendDelay {
		delay = maxSendDelay
	}
	if delay < 0 {
		delay = 0
	}
	submitDelay := time.Duration(0)
	if action.SubmitDelayMS != nil {
		submitDelay = normalizeSendDelay(*action.SubmitDelayMS)
	}
	if submitDelay > maxSendDelay {
		submitDelay = maxSendDelay
	}
	if submitDelay < 0 {
		submitDelay = 0
	}
	waitOutput := action.WaitForOutput
	if !explicitDelay {
		waitOutput = true
	}
	return paneSendAction{
		text:        action.Text,
		delay:       delay,
		submit:      action.Submit,
		submitDelay: submitDelay,
		waitOutput:  waitOutput,
	}, true
}

func normalizeSendDelay(ms int) time.Duration {
	if ms <= 0 {
		return 0
	}
	maxMillis := int(maxSendDelay / time.Millisecond)
	if ms > maxMillis {
		return maxSendDelay
	}
	return time.Duration(ms) * time.Millisecond
}

func (m *Manager) runPaneSendQueue(paneID string, actions []paneSendAction) {
	if m == nil || len(actions) == 0 {
		return
	}
	for _, action := range actions {
		if m.closed.Load() {
			return
		}
		if action.waitOutput {
			m.waitForPaneOutput(paneID, action.delay)
		} else if action.delay > 0 {
			time.Sleep(action.delay)
		}
		payload, ok := buildSendPayload(action.text, !action.submit)
		if !ok {
			continue
		}
		if len(payload) > maxSendBytes {
			m.notifyToast(paneID, "Automation send skipped: payload too large")
			continue
		}
		profiling.Trigger("layout-send")
		sendCtx, cancel := context.WithTimeout(context.Background(), defaultSendTimeout)
		err := m.SendInput(sendCtx, paneID, payload)
		cancel()
		if err != nil {
			m.notifyToast(paneID, "Automation send failed: "+err.Error())
			continue
		}
		m.maybeSetPaneToolFromSend(paneID, action)
		if action.submit {
			if action.submitDelay > 0 {
				time.Sleep(action.submitDelay)
			}
			submitCtx, cancel := context.WithTimeout(context.Background(), defaultSendTimeout)
			err := m.SendInput(submitCtx, paneID, []byte{'\r'})
			cancel()
			if err != nil {
				m.notifyToast(paneID, "Automation submit failed: "+err.Error())
			}
		}
	}
}

func (m *Manager) maybeSetPaneToolFromSend(paneID string, action paneSendAction) {
	if m == nil || paneID == "" || !action.submit {
		return
	}
	reg := m.toolRegistryRef()
	if reg == nil {
		return
	}
	toolID := reg.DetectFromCommand(action.text)
	if toolID == "" {
		return
	}
	_ = m.SetPaneTool(paneID, toolID)
}

func buildSendPayload(text string, appendNewline bool) ([]byte, bool) {
	if strings.TrimSpace(text) == "" {
		return nil, false
	}
	payload := text
	if appendNewline {
		if !strings.HasSuffix(payload, "\n") && !strings.HasSuffix(payload, "\r") {
			payload += "\n"
		}
	} else {
		payload = strings.TrimRight(payload, "\r\n")
		if strings.TrimSpace(payload) == "" {
			return nil, false
		}
	}
	return []byte(payload), true
}
