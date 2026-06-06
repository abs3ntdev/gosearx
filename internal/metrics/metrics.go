// Package metrics records per-engine timing and error statistics and tracks
// temporarily suspended engines (after repeated failures). It is the Go
// successor to SearXNG's metrics + the SuspendedStatus / ban_time_on_fail logic.
package metrics

import (
	"sync"
	"time"
)

// EngineStat is the aggregated stats for one engine.
type EngineStat struct {
	Name        string  `json:"name"`
	Requests    int64   `json:"requests"`
	Errors      int64   `json:"errors"`
	AvgTimeMS   float64 `json:"avg_time_ms"`
	Suspended   bool    `json:"suspended"`
	SuspendedTo string  `json:"suspended_until,omitempty"`
}

type engineRec struct {
	requests   int64
	errors     int64
	totalMS    float64
	failStreak int
	suspendTo  time.Time
}

// Metrics aggregates engine statistics and suspension state.
type Metrics struct {
	mu      sync.Mutex
	engines map[string]*engineRec

	// banTime is the base suspension after consecutive failures; maxBan caps it.
	banTime time.Duration
	maxBan  time.Duration
}

// New builds a Metrics tracker. banTime/maxBan mirror SearXNG's
// ban_time_on_fail / max_ban_time_on_fail.
func New(banTime, maxBan time.Duration) *Metrics {
	if banTime == 0 {
		banTime = 5 * time.Second
	}
	if maxBan == 0 {
		maxBan = 120 * time.Second
	}
	return &Metrics{engines: map[string]*engineRec{}, banTime: banTime, maxBan: maxBan}
}

func (m *Metrics) rec(name string) *engineRec {
	r, ok := m.engines[name]
	if !ok {
		r = &engineRec{}
		m.engines[name] = r
	}
	return r
}

// RecordSuccess logs a successful engine call with its elapsed time.
func (m *Metrics) RecordSuccess(name string, elapsed time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.rec(name)
	r.requests++
	r.totalMS += float64(elapsed.Milliseconds())
	r.failStreak = 0
	r.suspendTo = time.Time{}
}

// RecordError logs a failed engine call and may suspend the engine.
func (m *Metrics) RecordError(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.rec(name)
	r.requests++
	r.errors++
	r.failStreak++
	// Exponential backoff capped at maxBan, after 2 consecutive failures.
	if r.failStreak >= 2 {
		dur := m.banTime * time.Duration(1<<uint(min(r.failStreak-2, 6)))
		if dur > m.maxBan {
			dur = m.maxBan
		}
		r.suspendTo = time.Now().Add(dur)
	}
}

// IsSuspended reports whether an engine is currently suspended.
func (m *Metrics) IsSuspended(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.engines[name]
	if !ok {
		return false
	}
	return time.Now().Before(r.suspendTo)
}

// Snapshot returns a copy of all engine stats for the /api/stats endpoint.
func (m *Metrics) Snapshot() []EngineStat {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	out := make([]EngineStat, 0, len(m.engines))
	for name, r := range m.engines {
		avg := 0.0
		if r.requests-r.errors > 0 {
			avg = r.totalMS / float64(r.requests-r.errors)
		}
		st := EngineStat{
			Name: name, Requests: r.requests, Errors: r.errors, AvgTimeMS: avg,
		}
		if now.Before(r.suspendTo) {
			st.Suspended = true
			st.SuspendedTo = r.suspendTo.UTC().Format(time.RFC3339)
		}
		out = append(out, st)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
