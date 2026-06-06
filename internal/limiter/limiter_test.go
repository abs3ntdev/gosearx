package limiter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/searxng/gosearx/internal/config"
)

func newReq(remote, xff string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/search?q=x", nil)
	r.RemoteAddr = remote
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func TestClientIPTrustedProxy(t *testing.T) {
	l := New(config.LimiterConfig{
		Enabled:        true,
		TrustedProxies: []string{"172.19.0.0/16"},
	}, nil)
	// From a trusted proxy: XFF is honored.
	ip := l.clientIP(newReq("172.19.0.5:1234", "203.0.113.9"))
	if ip.String() != "203.0.113.9" {
		t.Errorf("trusted proxy XFF not honored: %s", ip)
	}
	// From an untrusted peer: XFF is ignored (anti-spoof).
	ip = l.clientIP(newReq("8.8.8.8:1234", "203.0.113.9"))
	if ip.String() != "8.8.8.8" {
		t.Errorf("untrusted XFF should be ignored, got %s", ip)
	}
}

func TestPassAndBlockLists(t *testing.T) {
	l := New(config.LimiterConfig{
		Enabled: true,
		PassIP:  []string{"192.168.0.0/24"},
		BlockIP: []string{"10.0.0.5"},
	}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := l.Middleware(next)

	// blocked IP -> 403
	w := httptest.NewRecorder()
	h.ServeHTTP(w, newReq("10.0.0.5:1", ""))
	if w.Code != http.StatusForbidden {
		t.Errorf("block list: want 403, got %d", w.Code)
	}

	// pass IP -> never throttled (many requests all 200)
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, newReq("192.168.0.10:1", ""))
		if w.Code != http.StatusOK {
			t.Fatalf("pass list req %d: want 200, got %d", i, w.Code)
			break
		}
	}
}

func TestRateLimit(t *testing.T) {
	l := New(config.LimiterConfig{Enabled: true}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
	h := l.Middleware(next)

	got429 := false
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, newReq("203.0.113.50:1", ""))
		if w.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Error("expected rate limit to trigger after burst")
	}
}

func TestDisabledLimiterAllowsAll(t *testing.T) {
	l := New(config.LimiterConfig{Enabled: false}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
	h := l.Middleware(next)
	for i := 0; i < 200; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, newReq("203.0.113.99:1", ""))
		if w.Code == http.StatusTooManyRequests {
			t.Fatal("disabled limiter should never throttle")
		}
	}
}
