package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/appdirs"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessionrestore"
	"github.com/regenrek/peakypanes/internal/tool"
)

const (
	defaultReadTimeout  = 2 * time.Minute
	defaultWriteTimeout = 5 * time.Second
	defaultOpTimeout    = 5 * time.Second
)

// DaemonConfig configures a session daemon instance.
type DaemonConfig struct {
	Version        string
	SocketPath     string
	PidPath        string
	SessionRestore sessionrestore.Config
	HandleSignals  bool
	PprofAddr      string
}

type pprofServer interface {
	Shutdown(context.Context) error
}

// Daemon owns persistent sessions and serves clients over a local socket.
type Daemon struct {
	manager       sessionManager
	toolRegistry  *tool.Registry
	listener      net.Listener
	listenerMu    sync.RWMutex
	socketPath    string
	pidPath       string
	pprofAddr     string
	pprofServer   pprofServer
	pprofListener net.Listener
	version       string
	restore       *restoreService
	profileStop   func()
	startMu       sync.Mutex
	started       chan struct{}
	startOnce     sync.Once
	spawnMu       sync.Mutex
	shutdownMu    sync.Mutex
	shutdownErr   error
	shutdownOne   sync.Once

	ctx    context.Context
	cancel context.CancelFunc

	clients        map[uint64]*clientConn
	clientsMu      sync.RWMutex
	clientSeq      atomic.Uint64
	debugSnap      atomic.Int64
	perfPaneViewMu sync.Mutex
	perfPaneView   map[string]uint8

	actionMu   sync.RWMutex
	actionLogs map[string]*actionLog

	eventMu  sync.RWMutex
	eventLog *eventLog

	focusMu        sync.RWMutex
	focusedSession string
	focusedPane    string

	relays *relayManager

	eventSeq atomic.Uint64

	closing atomic.Bool
	wg      sync.WaitGroup
}

// NewDaemon creates a daemon instance.
func NewDaemon(cfg DaemonConfig) (*Daemon, error) {
	socketPath := cfg.SocketPath
	if socketPath == "" {
		path, err := DefaultSocketPath()
		if err != nil {
			return nil, err
		}
		socketPath = path
	}
	pidPath := cfg.PidPath
	if pidPath == "" {
		path, err := DefaultPidPath()
		if err != nil {
			return nil, err
		}
		pidPath = path
	}
	ctx, cancel := context.WithCancel(context.Background())
	registry, err := loadToolRegistry()
	if err != nil {
		cancel()
		return nil, err
	}
	nativeMgr, err := native.NewManager()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := nativeMgr.SetToolRegistry(registry); err != nil {
		cancel()
		return nil, err
	}
	var restore *restoreService
	restoreCfg := cfg.SessionRestore.Normalized()
	if restoreCfg.Enabled {
		if strings.TrimSpace(restoreCfg.BaseDir) == "" {
			dataDir, err := appdirs.DataDir()
			if err != nil {
				cancel()
				return nil, err
			}
			restoreCfg.BaseDir = filepath.Join(dataDir, "sessions")
		}
		store, err := sessionrestore.NewStore(restoreCfg)
		if err != nil {
			cancel()
			return nil, err
		}
		restore = newRestoreService(store, restoreCfg)
	}
	d := &Daemon{
		manager:      wrapManager(nativeMgr),
		toolRegistry: registry,
		socketPath:   socketPath,
		pidPath:      pidPath,
		pprofAddr:    strings.TrimSpace(cfg.PprofAddr),
		version:      cfg.Version,
		restore:      restore,
		ctx:          ctx,
		cancel:       cancel,
		clients:      make(map[uint64]*clientConn),
		actionLogs:   make(map[string]*actionLog),
		eventLog:     newEventLog(0),
		relays:       newRelayManager(),
		started:      make(chan struct{}),
	}
	if cfg.HandleSignals {
		d.handleSignals()
	}
	return d, nil
}

// Start begins listening for client connections.
func (d *Daemon) Start() error {
	if d == nil {
		return errors.New("sessiond: daemon is nil")
	}
	d.startMu.Lock()
	defer d.startMu.Unlock()
	defer d.signalStarted()
	if d.closing.Load() {
		return errors.New("sessiond: daemon is shutting down")
	}
	if d.listenerValue() != nil {
		return errors.New("sessiond: daemon already started")
	}
	if err := ensureSocketDir(d.socketPath); err != nil {
		return err
	}
	if err := d.removeStaleSocket(); err != nil {
		return err
	}
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("sessiond: listen on %s: %w", d.socketPath, err)
	}
	d.setListener(listener)
	if err := os.Chmod(d.socketPath, 0o700); err != nil {
		_ = listener.Close()
		return fmt.Errorf("sessiond: chmod socket: %w", err)
	}
	if err := d.writePidFile(); err != nil {
		_ = listener.Close()
		return err
	}
	if err := d.startPprofServer(); err != nil {
		if l := d.clearListener(); l != nil {
			_ = l.Close()
		}
		_ = os.Remove(d.pidPath)
		return err
	}
	d.wg.Add(2)
	go d.acceptLoop()
	go d.eventLoop()

	if d.restore != nil {
		if err := d.restore.Load(d.ctx); err != nil {
			slog.Warn("sessiond: restore load failed", slog.Any("err", err))
		}
		d.wg.Add(1)
		go d.restoreLoop()
	}

	d.startProfiler()

	slog.Info("sessiond: daemon listening", slog.String("socket", d.socketPath))
	return nil
}

// Run starts the daemon and blocks until it is stopped.
func (d *Daemon) Run() error {
	if err := d.Start(); err != nil {
		return err
	}
	<-d.ctx.Done()
	return d.shutdownOnce()
}

// Stop signals the daemon to shut down.
func (d *Daemon) Stop() error {
	if d == nil {
		return nil
	}
	if d.closing.Swap(true) {
		return nil
	}
	d.cancel()
	d.startMu.Lock()
	_ = d.listenerValue()
	d.startMu.Unlock()
	return d.shutdownOnce()
}

func (d *Daemon) shutdown() error {
	d.closing.Store(true)
	if listener := d.clearListener(); listener != nil {
		_ = listener.Close()
	}
	d.stopPprofServer()

	d.spawnMu.Lock()
	_ = d.closing.Load()
	d.spawnMu.Unlock()

	d.clientsMu.Lock()
	for _, client := range d.clients {
		closeClient(client)
	}
	d.clients = make(map[uint64]*clientConn)
	d.clientsMu.Unlock()

	if d.restore != nil {
		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		if err := d.restore.Flush(ctx, d.manager); err != nil {
			slog.Warn("sessiond: flush restore failed", slog.Any("err", err))
		}
		cancel()
	}

	if d.manager != nil {
		d.manager.Close()
	}

	d.wg.Wait()

	_ = os.Remove(d.socketPath)
	_ = os.Remove(d.pidPath)
	d.stopProfiler()
	return nil
}

func (d *Daemon) shutdownOnce() error {
	if d == nil {
		return nil
	}
	d.shutdownOne.Do(func() {
		err := d.shutdown()
		d.shutdownMu.Lock()
		d.shutdownErr = err
		d.shutdownMu.Unlock()
	})
	d.shutdownMu.Lock()
	err := d.shutdownErr
	d.shutdownMu.Unlock()
	return err
}

func (d *Daemon) setListener(listener net.Listener) {
	d.listenerMu.Lock()
	d.listener = listener
	d.listenerMu.Unlock()
}

func (d *Daemon) signalStarted() {
	if d == nil {
		return
	}
	d.startOnce.Do(func() {
		if d.started != nil {
			close(d.started)
		}
	})
}

func (d *Daemon) listenerValue() net.Listener {
	d.listenerMu.RLock()
	listener := d.listener
	d.listenerMu.RUnlock()
	return listener
}

func (d *Daemon) clearListener() net.Listener {
	d.listenerMu.Lock()
	listener := d.listener
	d.listener = nil
	d.listenerMu.Unlock()
	return listener
}

func (d *Daemon) writePidFile() error {
	if d.pidPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(d.pidPath), 0o755); err != nil {
		return fmt.Errorf("sessiond: create pid dir: %w", err)
	}
	pid := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(d.pidPath, []byte(pid), 0o600); err != nil {
		return fmt.Errorf("sessiond: write pid file: %w", err)
	}
	return nil
}

func (d *Daemon) removeStaleSocket() error {
	if _, err := os.Stat(d.socketPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("sessiond: stat socket: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := probeDaemon(ctx, d.socketPath, d.version); err == nil {
		return fmt.Errorf("sessiond: daemon already running on %s", d.socketPath)
	} else if errors.Is(err, ErrDaemonProbeTimeout) {
		return err
	}
	if err := os.Remove(d.socketPath); err != nil {
		return fmt.Errorf("sessiond: remove stale socket: %w", err)
	}
	return nil
}

func ensureSocketDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sessiond: create socket dir: %w", err)
	}
	return nil
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok {
		return ne.Timeout()
	}
	return false
}
