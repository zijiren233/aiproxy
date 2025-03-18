package rpmlimit

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type windowCounts struct {
	normal int64
	over   int64
}

type entry struct {
	sync.Mutex
	windows map[int64]*windowCounts
}

type InMemoryRateLimiter struct {
	entries sync.Map
}

var memoryRateLimiter = &InMemoryRateLimiter{}

func (m *InMemoryRateLimiter) getEntry(group, model string) *entry {
	key := fmt.Sprintf("%s:%s", group, model)
	actual, _ := m.entries.LoadOrStore(key, &entry{
		windows: make(map[int64]*windowCounts),
	})
	return actual.(*entry)
}

func (m *InMemoryRateLimiter) cleanup(e *entry, cutoff int64) {
	for ts := range e.windows {
		if ts < cutoff {
			delete(e.windows, ts)
		}
	}
}

func (m *InMemoryRateLimiter) pushRequest(group, model string, maxReq int64, duration time.Duration) (int64, int64) {
	e := m.getEntry(group, model)

	e.Lock()
	defer e.Unlock()

	now := time.Now().Unix()
	windowStart := now
	cutoff := windowStart - int64(duration.Seconds())

	m.cleanup(e, cutoff)

	var currentCount, overCount int64
	for _, wc := range e.windows {
		currentCount += wc.normal
		overCount += wc.over
	}

	wc, exists := e.windows[windowStart]
	if !exists {
		wc = &windowCounts{}
		e.windows[windowStart] = wc
	}

	if maxReq == 0 || currentCount <= maxReq {
		wc.normal++
		currentCount++
	} else {
		wc.over++
		overCount++
	}

	return currentCount, overCount
}

func (m *InMemoryRateLimiter) getRPM(group, model string, duration time.Duration) int {
	total := 0
	now := time.Now().Unix()
	cutoff := now - int64(duration.Seconds())

	m.entries.Range(func(key, value interface{}) bool {
		k := key.(string)
		currentGroup, currentModel := parseKey(k)

		if (group == "" || group == currentGroup) && (model == "" || model == currentModel) {
			e := value.(*entry)
			e.Lock()
			m.cleanup(e, cutoff)
			var count int
			for _, wc := range e.windows {
				count += int(wc.normal + wc.over)
			}
			total += count
			e.Unlock()
		}
		return true
	})

	return total
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
