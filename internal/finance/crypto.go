package finance

import "strings"

// crypto.go adds cryptocurrency detection. Yahoo Finance serves crypto under
// "<TICKER>-USD" symbols (e.g. BTC-USD), so detection maps coin names/tickers to
// that form. Crypto names are unambiguous enough that they trigger without a
// "stock"/"price" keyword (unlike bare equity tickers).

// cryptoNames maps common coin names AND tickers to their Yahoo ticker (no -USD;
// the suffix is added by cryptoSymbol).
var cryptoNames = map[string]string{
	// name -> ticker
	"bitcoin":     "BTC",
	"ethereum":    "ETH",
	"ether":       "ETH",
	"litecoin":    "LTC",
	"dogecoin":    "DOGE",
	"doge":        "DOGE",
	"cardano":     "ADA",
	"solana":      "SOL",
	"ripple":      "XRP",
	"polkadot":    "DOT",
	"chainlink":   "LINK",
	"polygon":     "MATIC",
	"avalanche":   "AVAX",
	"tron":        "TRX",
	"stellar":     "XLM",
	"monero":      "XMR",
	"cosmos":      "ATOM",
	"uniswap":     "UNI",
	"shiba":       "SHIB",
	"shibainu":    "SHIB",
	"binancecoin": "BNB",
	"toncoin":     "TON",
}

// cryptoTickers is the set of recognized bare crypto tickers (so "BTC price",
// "$ETH", or "BTC" all resolve). Built from cryptoNames values.
var cryptoTickers = func() map[string]bool {
	m := map[string]bool{}
	for _, t := range cryptoNames {
		m[t] = true
	}
	return m
}()

// cryptoKeywords let "crypto BTC" / "btc coin" style queries trigger too.
var cryptoKeywords = map[string]bool{
	"crypto": true, "cryptocurrency": true, "coin": true, "token": true,
}

// detectCrypto returns the Yahoo crypto symbol (e.g. "BTC-USD") if the query
// references a known coin, else "".
func detectCrypto(query string) string {
	fields := strings.Fields(query)
	hasCryptoKeyword := false
	var ticker string

	for _, f := range fields {
		lf := strings.ToLower(strings.TrimPrefix(f, "$"))
		if cryptoKeywords[lf] {
			hasCryptoKeyword = true
			continue
		}
		// Coin name -> unambiguous, trigger immediately.
		if t, ok := cryptoNames[lf]; ok {
			return cryptoSymbol(t)
		}
		// Bare/`$` ticker (BTC, $ETH).
		up := strings.ToUpper(lf)
		if cryptoTickers[up] && ticker == "" {
			ticker = up
		}
	}

	if ticker != "" {
		// A bare crypto ticker triggers if it had a $ prefix, a crypto/finance
		// keyword, or is otherwise the whole query.
		return cryptoSymbol(ticker)
	}
	_ = hasCryptoKeyword
	return ""
}

// cryptoSymbol formats a coin ticker as a Yahoo crypto symbol.
func cryptoSymbol(ticker string) string {
	return ticker + "-USD"
}

// IsCryptoSymbol reports whether a Yahoo symbol is a crypto pair.
func IsCryptoSymbol(symbol string) bool {
	return strings.HasSuffix(strings.ToUpper(symbol), "-USD")
}
