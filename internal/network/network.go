// Package network is the HTTP client layer for executing engine requests.
// It is the Go successor to searx/network, simplified for Phase 1: a shared
// http.Client with sane defaults, context-driven timeouts, and the Fetcher
// interface so engines/tests can mock the network.
package network

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/searxng/gosearx/internal/engine"
)

// Fetcher executes an engine.HTTPRequest and returns an engine.HTTPResponse.
// The orchestrator depends on this interface so it can be mocked in tests.
type Fetcher interface {
	Fetch(ctx context.Context, req *engine.HTTPRequest) (*engine.HTTPResponse, error)
}

// DefaultUserAgent is sent when an engine doesn't set its own.
const DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

// Client is the default Fetcher backed by net/http.
type Client struct {
	hc        *http.Client
	userAgent string
	maxBody   int64
}

// Option configures a Client.
type Option func(*Client)

// WithUserAgent overrides the default User-Agent.
func WithUserAgent(ua string) Option { return func(c *Client) { c.userAgent = ua } }

// WithMaxBodyBytes caps the response body read (defense against huge responses).
func WithMaxBodyBytes(n int64) Option { return func(c *Client) { c.maxBody = n } }

// New returns a Client. The per-request deadline is taken from the context, so
// the http.Client.Timeout is left unset; transport-level timeouts guard dials.
func New(opts ...Option) *Client {
	c := &Client{
		userAgent: DefaultUserAgent,
		maxBody:   8 << 20, // 8 MiB
		hc: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
				ForceAttemptHTTP2:   true,
			},
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Fetch executes req, honoring ctx's deadline.
func (c *Client) Fetch(ctx context.Context, req *engine.HTTPRequest) (*engine.HTTPResponse, error) {
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if len(req.Body) > 0 {
		body = byteReader(req.Body)
	}
	hreq, err := http.NewRequestWithContext(ctx, method, req.URL, body)
	if err != nil {
		return nil, err
	}
	if _, ok := req.Headers["User-Agent"]; !ok {
		hreq.Header.Set("User-Agent", c.userAgent)
	}
	for k, v := range req.Headers {
		hreq.Header.Set(k, v)
	}
	for k, v := range req.Cookies {
		hreq.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	resp, err := c.hc.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody))
	if err != nil {
		return nil, err
	}
	return &engine.HTTPResponse{
		StatusCode: resp.StatusCode,
		URL:        resp.Request.URL.String(),
		Body:       data,
	}, nil
}

type byteReaderT struct {
	b []byte
	i int
}

func byteReader(b []byte) *byteReaderT { return &byteReaderT{b: b} }

func (r *byteReaderT) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
