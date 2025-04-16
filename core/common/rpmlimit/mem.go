package rpmlimit

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type windowCounts struct {
	normal int64
	over   int64
}

type entry struct {
	sync.Mutex
	windows    map[int64]*windowCounts
	lastAccess atomic.Value
}

type InMemoryRateLimiter struct {
	entries sync.Map
}

func newInMemoryRateLimiter() *InMemoryRateLimiter {
	rl := &InMemoryRateLimiter{
		entries: sync.Map{},
	}
	go rl.cleanupInactiveEntries(2*time.Minute, 1*time.Minute)
	return rl
}

var memoryRateLimiter = newInMemoryRateLimiter()

func (m *InMemoryRateLimiter) getEntry(group, model string) *entry {
	key := fmt.Sprintf("%s:%s", group, model)
	actual, _ := m.entries.LoadOrStore(key, &entry{
		windows: make(map[int64]*windowCounts),
	})
	e, _ := actual.(*entry)
	if e.lastAccess.Load() == nil {
		e.lastAccess.CompareAndSwap(nil, time.Now())
	}
	return e
}

func (m *InMemoryRateLimiter) cleanupAndCount(e *entry, cutoff int64) (int64, int64) {
	normalCount := int64(0)
	overCount := int64(0)
	for ts, wc := range e.windows {
		if ts < cutoff {
			delete(e.windows, ts)
		} else {
			normalCount += wc.normal
			overCount += wc.over
		}
	}
	return normalCount, overCount
}

func (m *InMemoryRateLimiter) pushRequest(group, model string, maxReq int64, duration time.Duration) (int64, int64) {
	e := m.getEntry(group, model)

	e.Lock()
	defer e.Unlock()

	now := time.Now()

	e.lastAccess.Store(now)

	windowStart := now.Unix()
	cutoff := windowStart - int64(duration.Seconds())

	normalCount, overCount := m.cleanupAndCount(e, cutoff)

	wc, exists := e.windows[windowStart]
	if !exists {
		wc = &windowCounts{}
		e.windows[windowStart] = wc
	}

	if maxReq == 0 || normalCount <= maxReq {
		wc.normal++
		normalCount++
	} else {
		wc.over++
		overCount++
	}

	return normalCount, overCount
}

func (m *InMemoryRateLimiter) getRPM(group, model string, duration time.Duration) int {
	total := 0
	cutoff := time.Now().Unix() - int64(duration.Seconds())

	m.entries.Range(func(key, value any) bool {
		k, _ := key.(string)
		currentGroup, currentModel := parseKey(k)

		if (group == "*" || group == currentGroup) &&
			(model == "" || model == "*" || model == currentModel) {
			e, _ := value.(*entry)
			e.Lock()
			normalCount, overCount := m.cleanupAndCount(e, cutoff)
			e.Unlock()
			total += int(normalCount + overCount)
		}
		return true
	})

	return total
}

func (m *InMemoryRateLimiter) cleanupInactiveEntries(interval time.Duration, maxInactivity time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		m.entries.Range(func(key, value any) bool {
			e, _ := value.(*entry)
			la := e.lastAccess.Load()
			if la == nil {
				return true
			}
			lastAccess, _ := la.(time.Time)
			if time.Since(lastAccess) > maxInactivity {
				m.entries.CompareAndDelete(key, e)
			}
			return true
		})
	}
}

func parseKey(key string) (group, model string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func MemoryPushRequest(group, model string, maxReq int64, duration time.Duration) (int64, int64) {
	return memoryRateLimiter.pushRequest(group, model, maxReq, duration)
}

func MemoryRateLimit(group, model string, maxReq int64, duration time.Duration) bool {
	current, _ := memoryRateLimiter.pushRequest(group, model, maxReq, duration)
	return current <= maxReq
}

func GetMemoryRPM(group, model string) (int64, error) {
	return int64(memoryRateLimiter.getRPM(group, model, time.Minute)), nil
}
