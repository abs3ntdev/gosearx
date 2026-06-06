package native

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// DNSLookup is a keyword answerer: "dns <host>" / "ip <host>" resolves A/AAAA
// records; "reverse <ip>" does a reverse lookup.
type DNSLookup struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &DNSLookup{} })
}

func (p *DNSLookup) Name() string                                  { return "dns" }
func (p *DNSLookup) Keywords() []string                            { return []string{"dns", "nslookup", "resolve", "reverse"} }
func (p *DNSLookup) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *DNSLookup) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *DNSLookup) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	cmd, host := splitFirst(strings.TrimSpace(sc.Query))
	cmd = strings.ToLower(cmd)
	if host == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r := net.Resolver{}

	if cmd == "reverse" {
		names, err := r.LookupAddr(ctx, host)
		if err != nil || len(names) == 0 {
			return answer("reverse " + host + ": no PTR record"), nil
		}
		return answer(host + " → " + strings.Join(names, ", ")), nil
	}

	// dns / nslookup / resolve -> A/AAAA
	ips, err := r.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return answer("dns " + host + ": no records"), nil
	}
	var v4, v6 []string
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			v4 = append(v4, ip.IP.String())
		} else {
			v6 = append(v6, ip.IP.String())
		}
	}
	parts := []string{host + ":"}
	if len(v4) > 0 {
		parts = append(parts, "A "+strings.Join(v4, ", "))
	}
	if len(v6) > 0 {
		parts = append(parts, "AAAA "+strings.Join(v6, ", "))
	}
	return answer(strings.Join(parts, "  ")), nil
}
