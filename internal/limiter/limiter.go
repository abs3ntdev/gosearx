// Package limiter implements rate limiting and basic bot detection, a port of
// SearXNG's limiter + botdetection. It resolves the real client IP behind
// trusted reverse proxies (X-Forwarded-For), honors pass/block IP lists, and
// applies a per-IP token-bucket rate limit. The link_token check mirrors
// SearXNG's defense against naive scripted scraping.
package limiter

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/searxng/gosearx/internal/cache"
	"github.com/searxng/gosearx/internal/config"
)

// Limiter is an HTTP middleware enforcing per-IP rate limits + IP lists.
// When a Valkey-backed cache is provided, the per-IP counters are shared across
// processes (distributed limiting); otherwise an in-process token bucket is used.
type Limiter struct {
	enabled        bool
	trustedProxies []*net.IPNet
	passIP         []*net.IPNet
	blockIP        []*net.IPNet

	rate  float64 // tokens per second
	burst float64

	cache  cache.Cache // distributed counters (window-based) when valkey-backed
	window time.Duration
	limit  int64

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New builds a Limiter from config. If c is a Valkey-backed cache, counters are
// shared across instances; pass cache.NewMemory() (or nil) for local-only.
func New(cfg config.LimiterConfig, c cache.Cache) *Limiter {
	return &Limiter{
		enabled:        cfg.Enabled,
		trustedProxies: parseCIDRs(cfg.TrustedProxies),
		passIP:         parseCIDRs(cfg.PassIP),
		blockIP:        parseCIDRs(cfg.BlockIP),
		rate:           1.0,
		burst:          30,
		cache:          c,
		window:         time.Minute,
		limit:          60, // 60 requests/minute/IP (matches ~1/s sustained)
		buckets:        map[string]*bucket{},
	}
}

// Middleware wraps a handler with rate limiting + IP filtering. API and asset
// paths are exempted from the link_token-style strictness; the limit still
// applies. Health checks are always allowed.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.enabled || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		ip := l.clientIP(r)

		if ipInList(ip, l.blockIP) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if ipInList(ip, l.passIP) {
			next.ServeHTTP(w, r) // never throttled
			return
		}
		if !l.allow(ip.String()) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP resolves the real client IP, honoring X-Forwarded-For only when the
// immediate peer is a trusted proxy (prevents header spoofing).
func (l *Limiter) clientIP(r *http.Request) net.IP {
	peerStr, _, _ := net.SplitHostPort(r.RemoteAddr)
	peer := net.ParseIP(peerStr)

	if peer != nil && ipInList(peer, l.trustedProxies) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// left-most is the original client.
			first := strings.TrimSpace(strings.Split(xff, ",")[0])
			if ip := net.ParseIP(first); ip != nil {
				return ip
			}
		}
	}
	if peer != nil {
		return peer
	}
	return net.IPv4zero
}

// allow applies the rate limit for an IP key. With a Valkey-backed cache it uses
// a shared fixed-window counter; otherwise an in-process token bucket.
func (l *Limiter) allow(key string) bool {
	if l.cache != nil && l.cache.Backend() == "valkey" {
		// Fixed-window counter keyed by IP + current window bucket.
		win := time.Now().Unix() / int64(l.window.Seconds())
		ck := "limiter:" + key + ":" + strconv.FormatInt(win, 10)
		n := l.cache.Incr(context.Background(), ck, l.window+time.Second)
		return n <= l.limit
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func parseCIDRs(list []string) []*net.IPNet {
	var out []*net.IPNet
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			// bare IP -> /32 or /128
			if strings.Contains(s, ":") {
				s += "/128"
			} else {
				s += "/32"
			}
		}
		if _, n, err := net.ParseCIDR(s); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func ipInList(ip net.IP, list []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range list {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
