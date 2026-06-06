// Package cache provides a key/value + counter store backed by Valkey/Redis,
// with an in-memory fallback when no server is configured. It is the Go
// successor to SearXNG's valkeydb usage: it backs the rate limiter, suspended
// engine tracking, and stats counters, and can cache search results.
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is the storage interface used across the app.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, val string, ttl time.Duration)
	Incr(ctx context.Context, key string, ttl time.Duration) int64
	Del(ctx context.Context, key string)
	// Backend reports "valkey" or "memory" for diagnostics.
	Backend() string
}

// New returns a Valkey-backed cache if url is set and reachable, otherwise an
// in-memory cache. The valkey URL accepts the valkey:// or redis:// scheme.
func New(url string) Cache {
	if url == "" {
		return NewMemory()
	}
	opt, err := redis.ParseURL(normalizeScheme(url))
	if err != nil {
		return NewMemory()
	}
	// Fail fast so an unreachable/unresolvable host falls back to memory at
	// startup instead of blocking on retries.
	opt.DialTimeout = 1 * time.Second
	opt.MaxRetries = -1 // no retries during the connectivity probe
	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return NewMemory()
	}
	return &valkeyCache{rdb: rdb}
}

// normalizeScheme maps valkey:// to redis:// (go-redis understands redis://).
func normalizeScheme(u string) string {
	if len(u) > 9 && u[:9] == "valkey://" {
		return "redis://" + u[9:]
	}
	if len(u) > 10 && u[:10] == "valkeys://" {
		return "rediss://" + u[10:]
	}
	return u
}

// valkeyCache is the Valkey/Redis-backed implementation.
type valkeyCache struct {
	rdb *redis.Client
}

func (c *valkeyCache) Backend() string { return "valkey" }

func (c *valkeyCache) Get(ctx context.Context, key string) (string, bool) {
	v, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (c *valkeyCache) Set(ctx context.Context, key, val string, ttl time.Duration) {
	c.rdb.Set(ctx, key, val, ttl)
}

func (c *valkeyCache) Incr(ctx context.Context, key string, ttl time.Duration) int64 {
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0
	}
	if n == 1 && ttl > 0 {
		c.rdb.Expire(ctx, key, ttl)
	}
	return n
}

func (c *valkeyCache) Del(ctx context.Context, key string) { c.rdb.Del(ctx, key) }

// memoryCache is a process-local fallback with TTL expiry.
type memoryCache struct {
	mu   sync.Mutex
	data map[string]memEntry
}

type memEntry struct {
	val    string
	count  int64
	expire time.Time
}

// NewMemory returns an in-memory cache.
func NewMemory() Cache {
	m := &memoryCache{data: map[string]memEntry{}}
	go m.reaper()
	return m
}

func (m *memoryCache) Backend() string { return "memory" }

func (m *memoryCache) Get(_ context.Context, key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.data[key]
	if !ok || (!e.expire.IsZero() && time.Now().After(e.expire)) {
		return "", false
	}
	return e.val, true
}

func (m *memoryCache) Set(_ context.Context, key, val string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.data[key]
	e.val = val
	if ttl > 0 {
		e.expire = time.Now().Add(ttl)
	}
	m.data[key] = e
}

func (m *memoryCache) Incr(_ context.Context, key string, ttl time.Duration) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.data[key]
	if !ok || (!e.expire.IsZero() && time.Now().After(e.expire)) {
		e = memEntry{}
		if ttl > 0 {
			e.expire = time.Now().Add(ttl)
		}
	}
	e.count++
	m.data[key] = e
	return e.count
}

func (m *memoryCache) Del(_ context.Context, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *memoryCache) reaper() {
	for range time.Tick(time.Minute) {
		now := time.Now()
		m.mu.Lock()
		for k, e := range m.data {
			if !e.expire.IsZero() && now.After(e.expire) {
				delete(m.data, k)
			}
		}
		m.mu.Unlock()
	}
}
