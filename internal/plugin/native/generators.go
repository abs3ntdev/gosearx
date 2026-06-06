package native

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// Generators bundles small local generator/answer tools that need no network:
// password, uuid, lorem, roman numerals. Keyword-gated.
type Generators struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &Generators{} })
}

func (p *Generators) Name() string { return "generators" }
func (p *Generators) Keywords() []string {
	return []string{"password", "passwd", "uuid", "guid", "lorem", "roman", "unroman"}
}
func (p *Generators) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *Generators) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *Generators) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	cmd, rest := splitFirst(sc.Query)
	cmd = strings.ToLower(cmd)
	rest = strings.TrimSpace(rest)

	switch cmd {
	case "password", "passwd":
		n := 20
		if v, err := atoiSafe(rest); err == nil && v >= 4 && v <= 256 {
			n = v
		}
		return answer("password: " + genPassword(n)), nil
	case "uuid", "guid":
		return answer("uuid: " + genUUIDv4()), nil
	case "lorem":
		words := 40
		if v, err := atoiSafe(rest); err == nil && v >= 1 && v <= 500 {
			words = v
		}
		return answer(genLorem(words)), nil
	case "roman":
		if v, err := atoiSafe(rest); err == nil && v > 0 && v < 4000 {
			return answer(fmt.Sprintf("%d = %s", v, toRoman(v))), nil
		}
	case "unroman":
		if v := fromRoman(strings.ToUpper(rest)); v > 0 {
			return answer(fmt.Sprintf("%s = %d", strings.ToUpper(rest), v)), nil
		}
	}
	return nil, nil
}

func atoiSafe(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("nan")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

const pwChars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*-_=+"

func genPassword(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(pwChars)))
	for i := range b {
		idx, _ := rand.Int(rand.Reader, max)
		b[i] = pwChars[idx.Int64()]
	}
	return string(b)
}

func genUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var loremWords = strings.Fields(
	"lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor " +
		"incididunt ut labore et dolore magna aliqua enim ad minim veniam quis nostrud " +
		"exercitation ullamco laboris nisi aliquip ex ea commodo consequat duis aute irure")

func genLorem(n int) string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, loremWords[i%len(loremWords)])
	}
	s := strings.Join(out, " ")
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:] + "."
	}
	return s
}

var romanVals = []struct {
	v int
	s string
}{
	{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"}, {100, "C"}, {90, "XC"},
	{50, "L"}, {40, "XL"}, {10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
}

func toRoman(n int) string {
	var b strings.Builder
	for _, rv := range romanVals {
		for n >= rv.v {
			b.WriteString(rv.s)
			n -= rv.v
		}
	}
	return b.String()
}

func fromRoman(s string) int {
	vals := map[byte]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	total, prev := 0, 0
	for i := len(s) - 1; i >= 0; i-- {
		v, ok := vals[s[i]]
		if !ok {
			return 0
		}
		if v < prev {
			total -= v
		} else {
			total += v
			prev = v
		}
	}
	if toRoman(total) != s {
		return 0 // reject malformed numerals
	}
	return total
}
