// Package finance provides a pluggable market-data layer. The DataSource
// interface abstracts the provider (Yahoo Finance, Stooq, a Google Finance
// scraper, or a paid API), so the source is swappable via configuration — the
// "pluggable finance source" design chosen during planning.
//
// Results are normalized to the result.Quote and result.Chart types, which the
// React frontend renders as interactive widgets.
package finance

import (
	"context"
	"fmt"
	"time"
)

// Range is a chart time range.
type Range string

const (
	Range1D  Range = "1d"
	Range5D  Range = "5d"
	Range1M  Range = "1mo"
	Range6M  Range = "6mo"
	RangeYTD Range = "ytd"
	Range1Y  Range = "1y"
	Range5Y  Range = "5y"
	RangeMax Range = "max"
)

// DefaultRange is the chart range used when none is requested (Google-style 1d).
const DefaultRange = Range1D

// validRanges is the set of accepted range values (also the Yahoo wire values).
var validRanges = map[Range]bool{
	Range1D: true, Range5D: true, Range1M: true, Range6M: true,
	RangeYTD: true, Range1Y: true, Range5Y: true, RangeMax: true,
}

// ParseRange validates a range string, returning DefaultRange for empty/unknown.
func ParseRange(s string) Range {
	r := Range(s)
	if validRanges[r] {
		return r
	}
	return DefaultRange
}

// Quote is the normalized current-price snapshot.
type Quote struct {
	Symbol    string
	Name      string
	Currency  string
	Price     float64
	Change    float64
	ChangePct float64
}

// Bar is one OHLC data point.
type Bar struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// History is a quote plus its time series.
type History struct {
	Quote Quote
	Bars  []Bar
}

// DataSource is the pluggable provider contract.
type DataSource interface {
	// Name identifies the provider (for config + attribution).
	Name() string
	// Quote returns the current snapshot for a symbol.
	Quote(ctx context.Context, symbol string) (*Quote, error)
	// History returns OHLC bars for a symbol over a range.
	History(ctx context.Context, symbol string, r Range) (*History, error)
}

// Registry maps source names to constructors, enabling config-driven selection.
var sources = map[string]func() DataSource{}

// Register adds a named data source constructor.
func Register(name string, ctor func() DataSource) { sources[name] = ctor }

// Get returns a data source by name, or an error listing available sources.
func Get(name string) (DataSource, error) {
	ctor, ok := sources[name]
	if !ok {
		return nil, fmt.Errorf("unknown finance source %q (available: %v)", name, Available())
	}
	return ctor(), nil
}

// Available lists registered source names.
func Available() []string {
	out := make([]string, 0, len(sources))
	for n := range sources {
		out = append(out, n)
	}
	return out
}
