//go:build profiler
// +build profiler

package main

var daemonHelpTextExtra = `  --pprof               Start a local pprof HTTP server (default: 127.0.0.1:6060)
  --pprof-addr <addr>   Bind address for pprof HTTP server
`
