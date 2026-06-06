// Package native holds compiled-in Go plugins for features that need
// capabilities beyond the sandboxed Lua tier (e.g. cryptographic hashing or
// network access). eth_checksum ports the user's custom SearXNG plugin:
// EIP-55 checksumming + ENS resolution for Ethereum addresses.
package native

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
	"golang.org/x/crypto/sha3"
)

const (
	ensResolveURL = "https://api.ensideas.com/ens/resolve/"
	ensTimeout    = 2 * time.Second
)

var (
	reAddr = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	reENS  = regexp.MustCompile(`(?i)^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+eth$`)
	reKW   = regexp.MustCompile(`(?i)^eth\s+(\S+)$`)
)

// EthChecksum is the native plugin.
type EthChecksum struct {
	hc *http.Client
}

func init() {
	plugin.Register(func() plugin.Plugin {
		return &EthChecksum{hc: &http.Client{Timeout: ensTimeout}}
	})
}

func (p *EthChecksum) Name() string       { return "eth_checksum" }
func (p *EthChecksum) Keywords() []string { return nil } // runs on every query

func (p *EthChecksum) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *EthChecksum) OnResult(*result.MainResult) (bool, error)     { return true, nil }

// PostSearch detects an address / ENS name (bare or via the "eth" keyword) and
// returns answer results with the EIP-55 checksum and/or ENS resolution.
func (p *EthChecksum) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	kind, value := classify(sc.Query)
	if kind == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), ensTimeout)
	defer cancel()

	var out result.EngineResults
	switch kind {
	case "addr":
		checksummed := toChecksumAddress(value[2:])
		raw := value[2:]
		hadUpper := strings.ToLower(raw) != raw
		hadLower := strings.ToUpper(raw) != raw
		if hadUpper && hadLower {
			status := "valid checksum"
			if value != checksummed {
				status = "INVALID checksum"
			}
			out = append(out, &result.Answer{EngineName: p.Name(),
				Answer: "EIP-55 (" + status + "): " + checksummed})
		} else {
			out = append(out, &result.Answer{EngineName: p.Name(),
				Answer: "EIP-55 checksum: " + checksummed})
		}
		if data := p.resolve(ctx, checksummed); data != nil && data.Name != "" {
			out = append(out, &result.Answer{EngineName: p.Name(),
				Answer: "ENS name: " + data.Name})
		}
	case "ens":
		data := p.resolve(ctx, value)
		if data == nil || data.Address == "" {
			out = append(out, &result.Answer{EngineName: p.Name(),
				Answer: "ENS name not resolved: " + value})
			break
		}
		checksummed := toChecksumAddress(strings.ToLower(strings.TrimPrefix(data.Address, "0x")))
		out = append(out, &result.Answer{EngineName: p.Name(),
			Answer: value + " resolves to: " + checksummed})
	}
	return out, nil
}

func classify(query string) (kind, value string) {
	q := strings.TrimSpace(query)
	if m := reKW.FindStringSubmatch(q); m != nil {
		q = strings.TrimSpace(m[1])
	}
	switch {
	case reAddr.MatchString(q):
		return "addr", q
	case reENS.MatchString(q):
		return "ens", strings.ToLower(q)
	}
	return "", ""
}

type ensData struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

func (p *EthChecksum) resolve(ctx context.Context, q string) *ensData {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ensResolveURL+q, nil)
	if err != nil {
		return nil
	}
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil
	}
	var d ensData
	if json.Unmarshal(body, &d) != nil {
		return nil
	}
	return &d
}

// toChecksumAddress applies EIP-55 mixed-case checksumming using keccak256.
func toChecksumAddress(addrHex string) string {
	addr := strings.ToLower(addrHex)
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(addr))
	digest := h.Sum(nil)
	hexDigest := make([]byte, 0, 64)
	const hexchars = "0123456789abcdef"
	for _, b := range digest {
		hexDigest = append(hexDigest, hexchars[b>>4], hexchars[b&0x0f])
	}
	var out strings.Builder
	out.WriteString("0x")
	for i, ch := range addr {
		if ch >= '0' && ch <= '9' {
			out.WriteRune(ch)
			continue
		}
		// uppercase the letter if the corresponding nibble >= 8
		nibble := hexDigest[i]
		val := nibble - '0'
		if nibble >= 'a' {
			val = nibble - 'a' + 10
		}
		if val >= 8 {
			out.WriteRune(ch - 32) // to uppercase
		} else {
			out.WriteRune(ch)
		}
	}
	return out.String()
}
