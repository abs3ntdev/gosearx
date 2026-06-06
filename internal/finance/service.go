package finance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/searxng/gosearx/internal/cache"
	"github.com/searxng/gosearx/internal/result"
)

// Service wraps a DataSource and produces normalized result types.
type Service struct {
	src   DataSource
	cache cache.Cache
	ttl   time.Duration
}

// NewService builds a finance Service over the named data source.
func NewService(sourceName string) (*Service, error) {
	src, err := Get(sourceName)
	if err != nil {
		return nil, err
	}
	return &Service{src: src}, nil
}

// WithCache enables caching of fetched charts (keyed by source+symbol+range)
// so repeated stock/crypto lookups skip the upstream (Yahoo/Stooq) request.
func (s *Service) WithCache(c cache.Cache, ttl time.Duration) *Service {
	s.cache = c
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	s.ttl = ttl
	return s
}

// SourceName returns the active provider name.
func (s *Service) SourceName() string { return s.src.Name() }

// ChartResult fetches history for a symbol and returns a result.Chart (with an
// embedded quote) ready for the frontend. Cached when a cache is configured.
func (s *Service) ChartResult(ctx context.Context, symbol string, r Range) (*result.Chart, error) {
	cacheKey := "fin:chart:" + s.src.Name() + ":" + symbol + ":" + string(r)
	if s.cache != nil {
		if blob, ok := s.cache.Get(ctx, cacheKey); ok {
			var c result.Chart
			if json.Unmarshal([]byte(blob), &c) == nil {
				return &c, nil
			}
		}
	}
	h, err := s.src.History(ctx, symbol, r)
	if err != nil {
		return nil, err
	}
	chart := &result.Chart{
		Type:       result.TypeChart,
		EngineName: "finance:" + s.src.Name(),
		Title:      h.Quote.Name,
		Symbol:     h.Quote.Symbol,
		Currency:   h.Quote.Currency,
		ChartKind:  chartKindFor(r),
		Range:      string(r),
		Quote:      toQuoteResult(s.src.Name(), &h.Quote),
	}
	if chart.Title == "" {
		chart.Title = symbol
	}
	for _, b := range h.Bars {
		chart.Series = append(chart.Series, result.Candle{
			T: b.Time.Unix(), O: b.Open, H: b.High, L: b.Low, C: b.Close, V: b.Volume,
		})
	}
	if s.cache != nil && len(chart.Series) > 0 {
		if blob, err := json.Marshal(chart); err == nil {
			s.cache.Set(ctx, cacheKey, string(blob), s.ttl)
		}
	}
	return chart, nil
}

// QuoteResult fetches a current quote as a result.Quote.
func (s *Service) QuoteResult(ctx context.Context, symbol string) (*result.Quote, error) {
	q, err := s.src.Quote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return toQuoteResult(s.src.Name(), q), nil
}

// chartKindFor uses a line chart for intraday ranges (Google-style) and
// candlesticks for daily-or-longer ranges where OHLC is meaningful.
func chartKindFor(r Range) string {
	switch r {
	case Range1D, Range5D:
		return "line"
	default:
		return "candlestick"
	}
}

func toQuoteResult(source string, q *Quote) *result.Quote {
	return &result.Quote{
		Type:       result.TypeQuote,
		EngineName: "finance:" + source,
		Symbol:     q.Symbol,
		Name:       q.Name,
		Currency:   q.Currency,
		Price:      q.Price,
		Change:     q.Change,
		ChangePct:  q.ChangePct,
	}
}
