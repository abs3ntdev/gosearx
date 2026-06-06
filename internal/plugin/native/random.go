package native

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strings"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// Random is a keyword answerer: "random <type>" generates a random value.
// Port of SearXNG's random answerer (string/int/float/sha256/uuid/color).
type Random struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &Random{} })
}

func (p *Random) Name() string                                  { return "random" }
func (p *Random) Keywords() []string                            { return []string{"random"} }
func (p *Random) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *Random) OnResult(*result.MainResult) (bool, error)     { return true, nil }

const randChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randString() string {
	n := 8 + mrand.Intn(25) // 8..32
	b := make([]byte, n)
	for i := range b {
		b[i] = randChars[mrand.Intn(len(randChars))]
	}
	return string(b)
}

func randUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (p *Random) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(sc.Query)))
	if len(parts) != 2 || parts[0] != "random" {
		return nil, nil
	}
	var val string
	switch parts[1] {
	case "string":
		val = randString()
	case "int":
		max := big.NewInt(1 << 31)
		n, _ := rand.Int(rand.Reader, new(big.Int).Mul(max, big.NewInt(2)))
		val = new(big.Int).Sub(n, max).String()
	case "float":
		val = fmt.Sprintf("%g", mrand.Float64())
	case "sha256":
		h := sha256.Sum256([]byte(randString()))
		val = hex.EncodeToString(h[:])
	case "uuid":
		val = randUUID()
	case "color":
		val = fmt.Sprintf("#%06x", mrand.Intn(0x1000000))
	default:
		return nil, nil
	}
	return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: val}}, nil
}
