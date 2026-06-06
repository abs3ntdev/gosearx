package native

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// QuickAnswers bundles many small keyword-triggered developer/utility tools into
// one answerer (timestamp, hash, encode, color, base conversion, char count,
// percentage, …). Each returns an instant Answer so you don't click a result.
type QuickAnswers struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &QuickAnswers{} })
}

func (p *QuickAnswers) Name() string { return "quick_answers" }

// Keyword-gated so it never fires on ordinary searches.
func (p *QuickAnswers) Keywords() []string {
	return []string{
		"timestamp", "epoch", "unixtime", "now",
		"md5", "sha1", "sha256", "sha512", "hash",
		"urlencode", "urldecode", "base64", "unbase64",
		"hex", "bin", "oct", "dec", "tobinary", "tohex",
		"color", "rgb", "hex2rgb", "rgb2hex",
		"charcount", "wordcount", "len", "length",
		"upper", "lower", "reverse", "rot13",
		"percent", "percentage",
	}
}

func (p *QuickAnswers) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *QuickAnswers) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func answer(s string) result.EngineResults {
	return result.EngineResults{&result.Answer{EngineName: "quick_answers", Answer: s}}
}

func (p *QuickAnswers) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	q := strings.TrimSpace(sc.Query)
	cmd, rest := splitFirst(q)
	cmd = strings.ToLower(cmd)

	switch cmd {
	case "now", "timestamp", "epoch", "unixtime":
		// "timestamp" alone -> current; "timestamp <unix>" -> human; "timestamp <date>" -> unix
		if rest == "" {
			now := time.Now()
			return answer(fmt.Sprintf("Unix: %d · UTC: %s · Local: %s",
				now.Unix(), now.UTC().Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05 MST"))), nil
		}
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil {
			// treat ms if it's very large
			if n > 1e12 {
				n /= 1000
			}
			t := time.Unix(n, 0)
			return answer(fmt.Sprintf("%d → %s UTC (%s local)", n,
				t.UTC().Format("2006-01-02 15:04:05"), t.Format("2006-01-02 15:04:05 MST"))), nil
		}
		// try parsing a date string
		for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339} {
			if t, err := time.Parse(layout, rest); err == nil {
				return answer(fmt.Sprintf("%s → Unix %d", rest, t.Unix())), nil
			}
		}

	case "md5":
		h := md5.Sum([]byte(rest))
		return answer("md5: " + hex.EncodeToString(h[:])), nil
	case "sha1":
		h := sha1.Sum([]byte(rest))
		return answer("sha1: " + hex.EncodeToString(h[:])), nil
	case "sha256", "hash":
		h := sha256.Sum256([]byte(rest))
		return answer("sha256: " + hex.EncodeToString(h[:])), nil
	case "sha512":
		h := sha512.Sum512([]byte(rest))
		return answer("sha512: " + hex.EncodeToString(h[:])), nil

	case "urlencode":
		return answer(url.QueryEscape(rest)), nil
	case "urldecode":
		if d, err := url.QueryUnescape(rest); err == nil {
			return answer(d), nil
		}

	case "charcount", "len", "length":
		return answer(fmt.Sprintf("%d characters, %d bytes", len([]rune(rest)), len(rest))), nil
	case "wordcount":
		return answer(fmt.Sprintf("%d words", len(strings.Fields(rest)))), nil
	case "upper":
		return answer(strings.ToUpper(rest)), nil
	case "lower":
		return answer(strings.ToLower(rest)), nil
	case "reverse":
		return answer(reverseStr(rest)), nil
	case "rot13":
		return answer(rot13(rest)), nil

	case "hex", "tohex":
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil {
			return answer(fmt.Sprintf("%d = 0x%X", n, n)), nil
		}
	case "bin", "tobinary":
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil {
			return answer(fmt.Sprintf("%d = 0b%b", n, n)), nil
		}
	case "oct":
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil {
			return answer(fmt.Sprintf("%d = 0o%o", n, n)), nil
		}
	case "dec":
		// accept 0x.. 0b.. 0o.. or plain
		if n, err := parseAnyInt(rest); err == nil {
			return answer(fmt.Sprintf("%s = %d", rest, n)), nil
		}

	case "hex2rgb", "color":
		h := strings.TrimPrefix(strings.TrimSpace(rest), "#")
		if len(h) == 6 {
			r, _ := strconv.ParseInt(h[0:2], 16, 0)
			g, _ := strconv.ParseInt(h[2:4], 16, 0)
			b, _ := strconv.ParseInt(h[4:6], 16, 0)
			return answer(fmt.Sprintf("#%s → rgb(%d, %d, %d)", h, r, g, b)), nil
		}
	case "rgb2hex":
		parts := strings.FieldsFunc(rest, func(r rune) bool { return r == ',' || r == ' ' })
		if len(parts) == 3 {
			r, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			g, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			b, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			return answer(fmt.Sprintf("rgb(%d,%d,%d) → #%02X%02X%02X", r, g, b, r, g, b)), nil
		}

	case "percent", "percentage":
		// "percent 20 of 150" -> 30 ; "percent 30 150" -> 20%
		f := strings.Fields(rest)
		if len(f) == 3 && f[1] == "of" {
			a, e1 := strconv.ParseFloat(f[0], 64)
			b, e2 := strconv.ParseFloat(f[2], 64)
			if e1 == nil && e2 == nil {
				return answer(fmt.Sprintf("%g%% of %g = %g", a, b, a/100*b)), nil
			}
		}
		if len(f) == 2 {
			a, e1 := strconv.ParseFloat(f[0], 64)
			b, e2 := strconv.ParseFloat(f[1], 64)
			if e1 == nil && e2 == nil && b != 0 {
				return answer(fmt.Sprintf("%g is %g%% of %g", a, a/b*100, b)), nil
			}
		}
	}
	return nil, nil
}

func splitFirst(s string) (first, rest string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

func reverseStr(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func rot13(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		}
		return r
	}, s)
}

func parseAnyInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "0x"), strings.HasPrefix(s, "0X"):
		return strconv.ParseInt(s[2:], 16, 64)
	case strings.HasPrefix(s, "0b"), strings.HasPrefix(s, "0B"):
		return strconv.ParseInt(s[2:], 2, 64)
	case strings.HasPrefix(s, "0o"), strings.HasPrefix(s, "0O"):
		return strconv.ParseInt(s[2:], 8, 64)
	}
	return strconv.ParseInt(s, 10, 64)
}
