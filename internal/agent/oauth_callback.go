package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type callbackResult struct {
	Code  string
	State string
	Err   error
}

type CallbackServer struct {
	server *http.Server
	addr   string
	path   string
	result chan callbackResult
}

func startCallbackServer(addr, path string) (*CallbackServer, error) {
	if addr == "" {
		return nil, errors.New("callback addr is required")
	}
	if path == "" {
		return nil, errors.New("callback path is required")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen callback: %w", err)
	}
	result := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if errStr := r.URL.Query().Get("error"); errStr != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Authentication failed. You can close this window."))
			result <- callbackResult{Err: fmt.Errorf("oauth error: %s", errStr)}
			return
		}
		if code == "" || state == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing code or state. You can close this window."))
			result <- callbackResult{Err: errors.New("missing code or state")}
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Authentication successful. You can close this window."))
		result <- callbackResult{Code: code, State: state}
	})
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(ln)
	}()
	return &CallbackServer{server: server, addr: addr, path: path, result: result}, nil
}

func (s *CallbackServer) Wait(ctx context.Context) (string, string, error) {
	if s == nil {
		return "", "", errors.New("callback server is nil")
	}
	select {
	case res := <-s.result:
		return res.Code, res.State, res.Err
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

func (s *CallbackServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
