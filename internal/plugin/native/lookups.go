package native

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

const lookupTimeout = 4 * time.Second

func httpGetJSON(ctx context.Context, hc *http.Client, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "gosearx/1.0")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// ---------- Dictionary: "define <word>" ----------

type Dictionary struct{ hc *http.Client }

func init() {
	plugin.Register(func() plugin.Plugin { return &Dictionary{hc: &http.Client{Timeout: lookupTimeout}} })
}

func (p *Dictionary) Name() string                                  { return "dictionary" }
func (p *Dictionary) Keywords() []string                            { return []string{"define", "definition", "dict"} }
func (p *Dictionary) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *Dictionary) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *Dictionary) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	_, word := splitFirst(sc.Query)
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	var entries []struct {
		Word     string `json:"word"`
		Phonetic string `json:"phonetic"`
		Meanings []struct {
			PartOfSpeech string `json:"partOfSpeech"`
			Definitions  []struct {
				Definition string `json:"definition"`
				Example    string `json:"example"`
			} `json:"definitions"`
		} `json:"meanings"`
	}
	u := "https://api.dictionaryapi.dev/api/v2/entries/en/" + url.PathEscape(word)
	if err := httpGetJSON(ctx, p.hc, u, &entries); err != nil || len(entries) == 0 {
		return nil, nil
	}
	e := entries[0]
	var b strings.Builder
	b.WriteString(e.Word)
	if e.Phonetic != "" {
		b.WriteString(" " + e.Phonetic)
	}
	for _, m := range e.Meanings {
		if len(m.Definitions) == 0 {
			continue
		}
		b.WriteString("\n• (" + m.PartOfSpeech + ") " + m.Definitions[0].Definition)
		if ex := m.Definitions[0].Example; ex != "" {
			b.WriteString(" — \"" + ex + "\"")
		}
	}
	return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: b.String()}}, nil
}

// ---------- Currency conversion: "100 usd to eur" / "convert 5 gbp to jpy" ----------

type Currency struct{ hc *http.Client }

func init() {
	plugin.Register(func() plugin.Plugin { return &Currency{hc: &http.Client{Timeout: lookupTimeout}} })
}

func (p *Currency) Name() string { return "currency" }

// Gated on common currency-ish first tokens; the parser validates the full form.
func (p *Currency) Keywords() []string { return []string{"convert"} }

func (p *Currency) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *Currency) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *Currency) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	amount, from, to, ok := parseCurrency(sc.Query)
	if !ok {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	var data struct {
		Rates map[string]float64 `json:"rates"`
	}
	// Open, key-less FX API.
	u := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", url.PathEscape(from))
	if err := httpGetJSON(ctx, p.hc, u, &data); err != nil {
		return nil, nil
	}
	rate, ok := data.Rates[to]
	if !ok {
		return nil, nil
	}
	converted := amount * rate
	ans := fmt.Sprintf("%s %s = %s %s  (1 %s = %.4f %s)",
		trimFloat(amount), from, trimFloat(converted), to, from, rate, to)
	return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: ans}}, nil
}

// parseCurrency accepts "<amt> <FROM> to <TO>" optionally prefixed with "convert".
func parseCurrency(q string) (amount float64, from, to string, ok bool) {
	f := strings.Fields(strings.TrimSpace(q))
	if len(f) > 0 && strings.EqualFold(f[0], "convert") {
		f = f[1:]
	}
	// forms: [amt FROM to TO] or [amt FROM TO]
	if len(f) == 4 && strings.EqualFold(f[2], "to") {
		f = []string{f[0], f[1], f[3]}
	} else if len(f) != 3 {
		return 0, "", "", false
	}
	amt, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0, "", "", false
	}
	from = strings.ToUpper(f[1])
	to = strings.ToUpper(f[2])
	if len(from) != 3 || len(to) != 3 || !isAlpha(from) || !isAlpha(to) {
		return 0, "", "", false
	}
	return amt, from, to, true
}

func isAlpha(s string) bool {
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// ---------- IP info: "ip", "myip", "ip 8.8.8.8" ----------

type IPInfo struct{ hc *http.Client }

func init() {
	plugin.Register(func() plugin.Plugin { return &IPInfo{hc: &http.Client{Timeout: lookupTimeout}} })
}

func (p *IPInfo) Name() string                                  { return "ipinfo" }
func (p *IPInfo) Keywords() []string                            { return []string{"ip", "myip", "ipinfo"} }
func (p *IPInfo) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *IPInfo) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *IPInfo) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	cmd, rest := splitFirst(sc.Query)
	rest = strings.TrimSpace(rest)
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	target := rest
	if strings.EqualFold(cmd, "myip") || rest == "" {
		// Caller's own IP (best-effort via the request context).
		if sc.ClientIP != "" && rest == "" {
			target = sc.ClientIP
		}
	}

	var data struct {
		IP       string `json:"ip"`
		City     string `json:"city"`
		Region   string `json:"region"`
		Country  string `json:"country_name"`
		Org      string `json:"org"`
		Postal   string `json:"postal"`
		Timezone string `json:"timezone"`
		Error    bool   `json:"error"`
	}
	u := "https://ipapi.co/json/"
	if target != "" {
		u = "https://ipapi.co/" + url.PathEscape(target) + "/json/"
	}
	if err := httpGetJSON(ctx, p.hc, u, &data); err != nil || data.Error || data.IP == "" {
		return nil, nil
	}
	loc := strings.Trim(strings.Join([]string{data.City, data.Region, data.Country}, ", "), ", ")
	parts := []string{data.IP}
	if loc != "" {
		parts = append(parts, loc)
	}
	if data.Org != "" {
		parts = append(parts, data.Org)
	}
	if data.Timezone != "" {
		parts = append(parts, data.Timezone)
	}
	return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: strings.Join(parts, " · ")}}, nil
}
