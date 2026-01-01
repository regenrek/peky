//go:build profiler
// +build profiler

package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
)

func (d *Daemon) startPprofServer() error {
	if d == nil {
		return nil
	}
	addr := strings.TrimSpace(d.pprofAddr)
	if addr == "" {
		return nil
	}
	if d.pprofServer != nil || d.pprofListener != nil {
		return fmt.Errorf("sessiond: pprof server already running")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("sessiond: pprof listen on %s: %w", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{Handler: mux}
	d.pprofListener = listener
	d.pprofServer = server
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("sessiond: pprof server error: %v", err)
		}
	}()
	log.Printf("sessiond: pprof server listening on %s", addr)
	return nil
}

func (d *Daemon) stopPprofServer() {
	if d == nil || d.pprofServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	if err := d.pprofServer.Shutdown(ctx); err != nil {
		log.Printf("sessiond: pprof shutdown: %v", err)
	}
	cancel()
	if d.pprofListener != nil {
		_ = d.pprofListener.Close()
	}
	d.pprofListener = nil
	d.pprofServer = nil
}
