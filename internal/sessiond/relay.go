package sessiond

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/terminal"
)

type relayManager struct {
	mu     sync.RWMutex
	relays map[string]*relay
	nextID atomic.Uint64
}

type relay struct {
	id        string
	cfg       RelayConfig
	createdAt time.Time

	mu     sync.RWMutex
	status RelayStatus
	stats  RelayStats

	cancel context.CancelFunc
	done   chan struct{}
}

func newRelayManager() *relayManager {
	return &relayManager{
		relays: make(map[string]*relay),
	}
}

func (m *relayManager) create(ctx context.Context, mgr sessionManager, cfg RelayConfig) (RelayInfo, error) {
	if m == nil {
		return RelayInfo{}, errors.New("sessiond: relay manager unavailable")
	}
	if mgr == nil {
		return RelayInfo{}, errors.New("sessiond: manager unavailable")
	}
	if strings.TrimSpace(cfg.FromPaneID) == "" {
		return RelayInfo{}, errors.New("sessiond: relay source is required")
	}
	if len(cfg.ToPaneIDs) == 0 {
		return RelayInfo{}, errors.New("sessiond: relay targets are required")
	}
	id := fmt.Sprintf("relay-%d", m.nextID.Add(1))
	created := time.Now().UTC()
	relay := &relay{
		id:        id,
		cfg:       cfg,
		createdAt: created,
		status:    RelayStatusRunning,
		done:      make(chan struct{}),
	}
	runCtx := ctx
	if runCtx == nil {
		runCtx = context.Background()
	}
	if cfg.TTL > 0 {
		runCtx, relay.cancel = context.WithTimeout(runCtx, cfg.TTL)
	} else {
		runCtx, relay.cancel = context.WithCancel(runCtx)
	}
	m.mu.Lock()
	m.relays[id] = relay
	m.mu.Unlock()
	go relay.run(runCtx, mgr, m)
	return relay.info(), nil
}

func (m *relayManager) list() []RelayInfo {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]RelayInfo, 0, len(m.relays))
	for _, relay := range m.relays {
		out = append(out, relay.info())
	}
	return out
}

func (m *relayManager) stop(id string) bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	relay := m.relays[id]
	m.mu.RUnlock()
	if relay == nil {
		return false
	}
	relay.stop()
	return true
}

func (m *relayManager) stopAll() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	relays := make([]*relay, 0, len(m.relays))
	for _, relay := range m.relays {
		relays = append(relays, relay)
	}
	m.mu.RUnlock()
	for _, relay := range relays {
		relay.stop()
	}
	return len(relays)
}

func (r *relay) info() RelayInfo {
	if r == nil {
		return RelayInfo{}
	}
	r.mu.RLock()
	status := r.status
	stats := r.stats
	r.mu.RUnlock()
	return RelayInfo{
		ID:        r.id,
		FromPane:  r.cfg.FromPaneID,
		ToPanes:   append([]string(nil), r.cfg.ToPaneIDs...),
		Scope:     r.cfg.Scope,
		Mode:      r.cfg.Mode,
		Status:    status,
		Delay:     r.cfg.Delay,
		Prefix:    r.cfg.Prefix,
		TTL:       r.cfg.TTL,
		CreatedAt: r.createdAt,
		Stats:     stats,
	}
}

func (r *relay) stop() {
	if r == nil {
		return
	}
	if r.cancel != nil {
		r.cancel()
	}
	<-r.done
}

func (r *relay) run(ctx context.Context, mgr sessionManager, mgrRef *relayManager) {
	defer close(r.done)
	var err error
	if r.cfg.Mode == RelayModeRaw {
		err = r.runRaw(ctx, mgr)
	} else {
		err = r.runLine(ctx, mgr)
	}
	r.mu.Lock()
	if err != nil {
		r.status = RelayStatusFailed
	} else {
		r.status = RelayStatusStopped
	}
	r.mu.Unlock()
	if mgrRef != nil {
		mgrRef.mu.Lock()
		delete(mgrRef.relays, r.id)
		mgrRef.mu.Unlock()
	}
}

func (r *relay) runLine(ctx context.Context, mgr sessionManager) error {
	seq := uint64(0)
	_, seq, _, _ = mgr.OutputLinesSince(r.cfg.FromPaneID, seq)
	for {
		if ctx.Err() != nil {
			return nil
		}
		lines, next, _, err := mgr.OutputLinesSince(r.cfg.FromPaneID, seq)
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			if !mgr.WaitForOutput(ctx, r.cfg.FromPaneID) {
				return nil
			}
			continue
		}
		for _, line := range lines {
			if ctx.Err() != nil {
				return nil
			}
			payload := []byte(r.cfg.Prefix + line.Text + "\n")
			if err := r.sendToTargets(ctx, mgr, payload); err != nil {
				return err
			}
			r.bumpStats(true, uint64(len(payload)))
			if r.cfg.Delay > 0 {
				timer := time.NewTimer(r.cfg.Delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil
				case <-timer.C:
				}
			}
		}
		seq = next
	}
}

func (r *relay) runRaw(ctx context.Context, mgr sessionManager) error {
	ch, cancel, err := mgr.SubscribeRawOutput(r.cfg.FromPaneID, 128)
	if err != nil {
		return err
	}
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, ok := <-ch:
			if !ok {
				return nil
			}
			if len(chunk.Data) == 0 {
				continue
			}
			if err := r.sendToTargets(ctx, mgr, chunk.Data); err != nil {
				return err
			}
			r.bumpStats(false, uint64(len(chunk.Data)))
			if r.cfg.Delay > 0 {
				timer := time.NewTimer(r.cfg.Delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil
				case <-timer.C:
				}
			}
		}
	}
}

func (r *relay) sendToTargets(ctx context.Context, mgr sessionManager, payload []byte) error {
	if mgr == nil {
		return errors.New("sessiond: manager unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	targets := r.cfg.ToPaneIDs
	if len(targets) == 0 {
		return errors.New("sessiond: relay targets empty")
	}
	remaining := targets[:0]
	for _, paneID := range targets {
		sendCtx, cancel := context.WithTimeout(ctx, defaultOpTimeout)
		err := mgr.SendInput(sendCtx, paneID, payload)
		cancel()
		if err == nil {
			remaining = append(remaining, paneID)
			continue
		}
		if errors.Is(err, terminal.ErrPaneClosed) {
			continue
		}
		return err
	}
	if len(remaining) == 0 {
		return errors.New("sessiond: relay targets closed")
	}
	r.cfg.ToPaneIDs = append([]string(nil), remaining...)
	return nil
}

func (r *relay) bumpStats(line bool, bytes uint64) {
	r.mu.Lock()
	if line {
		r.stats.Lines++
	}
	r.stats.Bytes += bytes
	r.stats.LastActivity = time.Now().UTC()
	r.mu.Unlock()
}
