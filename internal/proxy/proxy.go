// Package proxy implements the privacy-preserving image proxy and favicon
// resolver. Ports searx/webapp.py's /image_proxy and the favicon subsystem:
// result thumbnails/favicons are fetched server-side so the client never
// contacts third-party hosts directly.
package proxy

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ImageProxy fetches and re-serves remote images.
type ImageProxy struct {
	hc      *http.Client
	maxSize int64
}

// NewImageProxy returns an image proxy.
func NewImageProxy() *ImageProxy {
	return &ImageProxy{
		hc:      &http.Client{Timeout: 10 * time.Second},
		maxSize: 16 << 20, // 16 MiB
	}
}

// Serve proxies the image at the ?url= param. Only image content types are
// passed through (defense against using the proxy as an open relay).
func (p *ImageProxy) Serve(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	if raw == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx image proxy)")
	resp, err := p.hc.Do(req)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		http.Error(w, "not an image", http.StatusUnsupportedMediaType)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, io.LimitReader(resp.Body, p.maxSize))
}

// FaviconResolver builds favicon URLs via a configured backend and proxies the
// bytes through this server (so the browser never contacts a third party — the
// privacy point of self-hosting, and it avoids adblock/CORS hiding icons).
type FaviconResolver struct {
	backend string
	hc      *http.Client
}

// NewFaviconResolver returns a resolver for the named backend (duckduckgo,
// google, allesedv, yandex), or nil if empty/unknown.
func NewFaviconResolver(backend string) *FaviconResolver {
	switch backend {
	case "duckduckgo", "google", "allesedv", "yandex":
		return &FaviconResolver{backend: backend, hc: &http.Client{Timeout: 6 * time.Second}}
	}
	return nil
}

// URL returns the favicon URL for a given hostname.
func (f *FaviconResolver) URL(host string) string {
	switch f.backend {
	case "duckduckgo":
		return "https://icons.duckduckgo.com/ip3/" + host + ".ico"
	case "google":
		return "https://www.google.com/s2/favicons?domain=" + host + "&sz=32"
	case "allesedv":
		return "https://f1.allesedv.com/32/" + host
	case "yandex":
		return "https://favicon.yandex.net/favicon/" + host
	}
	return ""
}

// Serve resolves a favicon and redirects/proxies it. Uses the ?host= param.
func (f *FaviconResolver) Serve(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		http.Error(w, "missing host", http.StatusBadRequest)
		return
	}
	target := f.URL(host)
	if target == "" {
		http.Error(w, "no resolver", http.StatusNotFound)
		return
	}
	// Proxy the bytes through this server (avoids third-party requests being
	// blocked by adblock/privacy tools, matching SearXNG's favicon proxy).
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx favicon)")
	resp, err := f.hc.Do(req)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/x-icon"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 1<<20))
}
