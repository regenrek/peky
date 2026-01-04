package sessiond

import "testing"

func TestHandleSendInputDebugLogging(t *testing.T) {
	d := &Daemon{manager: &fakeManager{}}
	payload, err := encodePayload(SendInputRequest{PaneID: "pane-1", Input: []byte("hi")})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendInput(payload); err != nil {
		t.Fatalf("handleSendInput: %v", err)
	}
}
