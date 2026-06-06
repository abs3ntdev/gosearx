package native

import (
	"strings"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// SelfInfo answers "my ip" / "my user agent" with the requester's details.
// Port of SearXNG's self_info plugin.
type SelfInfo struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &SelfInfo{} })
}

func (p *SelfInfo) Name() string                                  { return "self_info" }
func (p *SelfInfo) Keywords() []string                            { return nil }
func (p *SelfInfo) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *SelfInfo) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *SelfInfo) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	q := strings.ToLower(strings.TrimSpace(sc.Query))
	switch q {
	case "ip", "my ip", "what is my ip", "what's my ip":
		if sc.ClientIP != "" {
			return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: "Your IP: " + sc.ClientIP}}, nil
		}
	case "user agent", "my user agent", "what is my user agent", "useragent", "user-agent":
		if sc.UserAgent != "" {
			return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: "Your user agent: " + sc.UserAgent}}, nil
		}
	}
	return nil, nil
}
