package agent

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type errReadCloser struct {
	reader   io.Reader
	readErr  error
	closeErr error
}

func (e *errReadCloser) Read(p []byte) (int, error) {
	if e.readErr != nil {
		return 0, e.readErr
	}
	return e.reader.Read(p)
}

func (e *errReadCloser) Close() error {
	return e.closeErr
}

func TestReadHTTPResponseSuccess(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}
	data, err := readHTTPResponse(resp, "test")
	if err != nil {
		t.Fatalf("readHTTPResponse error: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("data=%q", string(data))
	}
}

func TestReadHTTPResponseStatusError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader("boom")),
	}
	if _, err := readHTTPResponse(resp, "test"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReadHTTPResponseReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       &errReadCloser{reader: strings.NewReader(""), readErr: errors.New("read failed")},
	}
	if _, err := readHTTPResponse(resp, "test"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReadHTTPResponseCloseError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       &errReadCloser{reader: strings.NewReader("ok"), closeErr: errors.New("close failed")},
	}
	if _, err := readHTTPResponse(resp, "test"); err == nil {
		t.Fatalf("expected close error")
	}
}
