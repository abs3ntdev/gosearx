package finance

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// stooq implements DataSource against stooq.com's CSV endpoints (no API key).
// It demonstrates that the finance source is genuinely pluggable: a completely
// different provider and wire format (CSV) behind the same interface.
type stooq struct {
	hc *http.Client
}

func init() {
	Register("stooq", func() DataSource {
		return &stooq{hc: &http.Client{Timeout: 8 * time.Second}}
	})
}

func (s *stooq) Name() string { return "stooq" }

func (s *stooq) Quote(ctx context.Context, symbol string) (*Quote, error) {
	// l/?s=aapl.us&f=sd2t2ohlcv&e=csv -> symbol,date,time,open,high,low,close,volume
	u := "https://stooq.com/q/l/?" + url.Values{
		"s": {stooqSymbol(symbol)},
		"f": {"sd2t2ohlcv"},
		"e": {"csv"},
	}.Encode()
	rows, err := s.fetchCSV(ctx, u)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("stooq: no quote for %q", symbol)
	}
	r := rows[1]
	if len(r) < 8 {
		return nil, fmt.Errorf("stooq: malformed quote row")
	}
	closeP := parseF(r[6])
	openP := parseF(r[3])
	q := &Quote{Symbol: symbol, Currency: "USD", Price: closeP}
	if openP != 0 {
		q.Change = closeP - openP
		q.ChangePct = q.Change / openP * 100
	}
	return q, nil
}

func (s *stooq) History(ctx context.Context, symbol string, r Range) (*History, error) {
	interval := "d"
	if r == Range1D || r == Range5D {
		interval = "d" // stooq daily granularity (intraday needs auth)
	}
	u := "https://stooq.com/q/d/l/?" + url.Values{
		"s": {stooqSymbol(symbol)},
		"i": {interval},
		"e": {"csv"},
	}.Encode()
	rows, err := s.fetchCSV(ctx, u)
	if err != nil {
		return nil, err
	}
	h := &History{Quote: Quote{Symbol: symbol, Currency: "USD"}}
	// rows[0] is header: Date,Open,High,Low,Close,Volume
	cutoff := time.Now().AddDate(0, 0, -daysFor(r))
	for _, row := range rows[1:] {
		if len(row) < 5 {
			continue
		}
		t, err := time.Parse("2006-01-02", row[0])
		if err != nil || t.Before(cutoff) {
			continue
		}
		h.Bars = append(h.Bars, Bar{
			Time: t, Open: parseF(row[1]), High: parseF(row[2]),
			Low: parseF(row[3]), Close: parseF(row[4]),
		})
	}
	if n := len(h.Bars); n > 0 {
		h.Quote.Price = h.Bars[n-1].Close
		if n > 1 {
			prev := h.Bars[n-2].Close
			h.Quote.Change = h.Quote.Price - prev
			if prev != 0 {
				h.Quote.ChangePct = h.Quote.Change / prev * 100
			}
		}
	}
	return h, nil
}

func (s *stooq) fetchCSV(ctx context.Context, u string) ([][]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx finance)")
	resp, err := s.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stooq: status %d", resp.StatusCode)
	}
	return csv.NewReader(resp.Body).ReadAll()
}

// stooqSymbol adds the .us suffix for bare US tickers (stooq convention).
func stooqSymbol(symbol string) string {
	if strings.Contains(symbol, ".") {
		return strings.ToLower(symbol)
	}
	return strings.ToLower(symbol) + ".us"
}

func parseF(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func daysFor(r Range) int {
	switch r {
	case Range1D, Range5D:
		return 7
	case Range1M:
		return 31
	case Range6M:
		return 183
	case Range1Y:
		return 366
	default:
		return 183
	}
}
