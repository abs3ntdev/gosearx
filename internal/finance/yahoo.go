package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// yahoo implements DataSource against Yahoo Finance's unofficial chart API,
// which returns both meta (current price) and OHLC series in one call — no API
// key required. This is the default source: reliable for charts/quotes.
type yahoo struct {
	hc *http.Client
}

func init() {
	Register("yahoo", func() DataSource {
		return &yahoo{hc: &http.Client{Timeout: 8 * time.Second}}
	})
}

func (y *yahoo) Name() string { return "yahoo" }

// chartResponse models the subset of Yahoo's chart API we consume.
type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				Symbol             string  `json:"symbol"`
				LongName           string  `json:"longName"`
				ShortName          string  `json:"shortName"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

func (y *yahoo) fetchChart(ctx context.Context, symbol, rng, interval string) (*chartResponse, error) {
	u := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?%s",
		url.PathEscape(symbol), url.Values{
			"range":    {rng},
			"interval": {interval},
		}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx finance)")
	resp, err := y.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var cr chartResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, err
	}
	if len(cr.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo: no data for %q", symbol)
	}
	return &cr, nil
}

func (y *yahoo) Quote(ctx context.Context, symbol string) (*Quote, error) {
	cr, err := y.fetchChart(ctx, symbol, "1d", "1d")
	if err != nil {
		return nil, err
	}
	return quoteFromMeta(cr), nil
}

func (y *yahoo) History(ctx context.Context, symbol string, r Range) (*History, error) {
	interval := intervalFor(r)
	cr, err := y.fetchChart(ctx, symbol, string(r), interval)
	if err != nil {
		return nil, err
	}
	res := cr.Chart.Result[0]
	h := &History{Quote: *quoteFromMeta(cr)}
	if len(res.Indicators.Quote) == 0 {
		return h, nil
	}
	q := res.Indicators.Quote[0]
	for i, ts := range res.Timestamp {
		if i >= len(q.Close) || q.Close[i] == 0 {
			continue
		}
		bar := Bar{Time: time.Unix(ts, 0), Close: q.Close[i]}
		if i < len(q.Open) {
			bar.Open = q.Open[i]
		}
		if i < len(q.High) {
			bar.High = q.High[i]
		}
		if i < len(q.Low) {
			bar.Low = q.Low[i]
		}
		if i < len(q.Volume) {
			bar.Volume = q.Volume[i]
		}
		h.Bars = append(h.Bars, bar)
	}
	return h, nil
}

func quoteFromMeta(cr *chartResponse) *Quote {
	m := cr.Chart.Result[0].Meta
	name := m.LongName
	if name == "" {
		name = m.ShortName
	}
	q := &Quote{
		Symbol: m.Symbol, Name: name, Currency: m.Currency,
		Price: m.RegularMarketPrice,
	}
	if m.ChartPreviousClose != 0 {
		q.Change = m.RegularMarketPrice - m.ChartPreviousClose
		q.ChangePct = q.Change / m.ChartPreviousClose * 100
	}
	return q
}

// intervalFor picks the candle granularity per range, Google-style: fine
// intraday candles for short ranges, coarser for long ones. Values are Yahoo's
// supported intervals; finer intervals are only valid for shorter ranges.
func intervalFor(r Range) string {
	switch r {
	case Range1D:
		return "2m" // fine intraday candles
	case Range5D:
		return "15m"
	case Range1M:
		return "1h"
	case Range6M, RangeYTD:
		return "1d"
	case Range1Y:
		return "1d"
	case Range5Y:
		return "1wk"
	case RangeMax:
		return "1mo"
	default:
		return "1d"
	}
}
