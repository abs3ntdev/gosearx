package finance

import (
	"regexp"
	"strings"
)

// DetectSymbol inspects a query and returns a ticker symbol if the query looks
// like a finance lookup, else "". Recognizes:
//
//	$AAPL                 -> AAPL
//	AAPL stock            -> AAPL
//	AAPL price / quote    -> AAPL
//	stock AAPL            -> AAPL
//	!stock TSLA           -> TSLA   (bang already stripped by query parser)
var (
	reDollar = regexp.MustCompile(`^\$([A-Za-z][A-Za-z.\-]{0,9})$`)
	// A bare ticker candidate: 1-5 letters, optionally with a .CLASS suffix
	// (e.g. BRK.B). Kept short to avoid matching ordinary words like "market".
	reTicker = regexp.MustCompile(`^[A-Za-z]{1,5}(\.[A-Za-z]{1,2})?$`)
)

// common English words that pass the short-ticker shape but shouldn't trigger.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "of": true, "to": true,
	"in": true, "on": true, "for": true, "and": true, "or": true, "how": true,
	"what": true, "why": true, "buy": true, "sell": true, "today": true,
	"market": true, "news": true, "best": true, "top": true,
}

var financeKeywords = map[string]bool{
	"stock": true, "stocks": true, "price": true, "quote": true,
	"share": true, "shares": true, "ticker": true,
}

// DetectSymbol returns the detected symbol (Yahoo format), or "". Crypto is
// checked first so "BTC"/"bitcoin"/"$ETH" resolve to "<TICKER>-USD"; bare
// equity tickers still require a finance keyword to avoid false positives.
func DetectSymbol(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}

	// Crypto coins/tickers take priority (e.g. BTC -> BTC-USD).
	if sym := detectCrypto(q); sym != "" {
		return sym
	}

	// $TICKER anywhere (equity).
	for _, tok := range strings.Fields(q) {
		if m := reDollar.FindStringSubmatch(tok); m != nil {
			return strings.ToUpper(m[1])
		}
	}

	fields := strings.Fields(q)
	// Need a finance keyword present to avoid hijacking normal queries.
	hasKeyword := false
	var candidate string
	for _, f := range fields {
		lf := strings.ToLower(f)
		if financeKeywords[lf] {
			hasKeyword = true
			continue
		}
		if candidate == "" && reTicker.MatchString(f) && !stopWords[lf] {
			candidate = strings.ToUpper(f)
		}
	}
	if hasKeyword && candidate != "" {
		return candidate
	}
	return ""
}
