package finance

import "testing"

func TestDetectSymbol(t *testing.T) {
	cases := map[string]string{
		// equities
		"$AAPL":             "AAPL",
		"AAPL stock":        "AAPL",
		"stock TSLA":        "TSLA",
		"msft price":        "MSFT",
		"quote GOOG":        "GOOG",
		"how to cook pasta": "",
		"golang tutorial":   "",
		"$brk.b":            "BRK.B",
		// crypto (resolve to Yahoo "-USD" pairs; names trigger without keyword)
		"bitcoin":          "BTC-USD",
		"ethereum price":   "ETH-USD",
		"$BTC":             "BTC-USD",
		"BTC":              "BTC-USD",
		"dogecoin":         "DOGE-USD",
		"crypto SOL":       "SOL-USD",
		"how much is doge": "DOGE-USD",
	}
	for q, want := range cases {
		if got := DetectSymbol(q); got != want {
			t.Errorf("DetectSymbol(%q) = %q, want %q", q, got, want)
		}
	}
}

func TestSourceRegistry(t *testing.T) {
	for _, name := range []string{"yahoo", "stooq"} {
		src, err := Get(name)
		if err != nil {
			t.Errorf("Get(%q): %v", name, err)
			continue
		}
		if src.Name() != name {
			t.Errorf("source name = %q, want %q", src.Name(), name)
		}
	}
	if _, err := Get("nonexistent"); err == nil {
		t.Error("expected error for unknown source")
	}
}
