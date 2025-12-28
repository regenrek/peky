package sessiond

import "testing"

type protocolSample struct {
	Name  string
	Count int
}

func TestEncodeDecodePayload(t *testing.T) {
	payload, err := encodePayload(protocolSample{Name: "demo", Count: 3})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	var out protocolSample
	if err := decodePayload(payload, &out); err != nil {
		t.Fatalf("decodePayload: %v", err)
	}
	if out.Name != "demo" || out.Count != 3 {
		t.Fatalf("unexpected decoded payload: %#v", out)
	}
}

func TestEncodeDecodePayloadNil(t *testing.T) {
	payload, err := encodePayload(nil)
	if err != nil {
		t.Fatalf("encodePayload nil: %v", err)
	}
	if payload != nil {
		t.Fatalf("expected nil payload, got %#v", payload)
	}

	if err := decodePayload(nil, nil); err != nil {
		t.Fatalf("decodePayload nil: %v", err)
	}
	if err := decodePayload([]byte{}, &protocolSample{}); err != nil {
		t.Fatalf("decodePayload empty: %v", err)
	}
}

func TestDecodePayloadError(t *testing.T) {
	var out protocolSample
	if err := decodePayload([]byte("not-gob"), &out); err == nil {
		t.Fatalf("expected decode error")
	}
}
