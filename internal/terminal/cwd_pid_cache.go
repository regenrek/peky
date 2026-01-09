package terminal

import (
	"strconv"
	"sync"
	"time"
)

const pidCwdCacheTTL = 750 * time.Millisecond

type pidCwdCache struct {
	mu   sync.Mutex
	byID map[int]pidCwdEntry
	ttl  time.Duration
}

type pidCwdEntry struct {
	cwd     string
	expires time.Time
}

var globalPidCwdCache = pidCwdCache{
	byID: make(map[int]pidCwdEntry),
	ttl:  pidCwdCacheTTL,
}

func cachedPIDCwd(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	now := time.Now()
	globalPidCwdCache.mu.Lock()
	entry, ok := globalPidCwdCache.byID[pid]
	if ok && now.Before(entry.expires) {
		globalPidCwdCache.mu.Unlock()
		if entry.cwd == "" {
			return "", false
		}
		return entry.cwd, true
	}
	globalPidCwdCache.mu.Unlock()

	cwd, ok := pidCwd(pid)
	globalPidCwdCache.mu.Lock()
	if len(globalPidCwdCache.byID) > 4096 {
		globalPidCwdCache.pruneLocked(now)
	}
	globalPidCwdCache.byID[pid] = pidCwdEntry{cwd: cwd, expires: now.Add(globalPidCwdCache.ttl)}
	globalPidCwdCache.mu.Unlock()

	if !ok || cwd == "" {
		return "", false
	}
	return cwd, true
}

func (c *pidCwdCache) pruneLocked(now time.Time) {
	for pid, entry := range c.byID {
		if now.After(entry.expires) {
			delete(c.byID, pid)
		}
	}
	if len(c.byID) <= 4096 {
		return
	}
	for pid := range c.byID {
		delete(c.byID, pid)
		if len(c.byID) <= 4096 {
			return
		}
	}
}

func pidString(pid int) string {
	if pid <= 0 {
		return ""
	}
	return strconv.Itoa(pid)
}
