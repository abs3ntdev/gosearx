package native

import (
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// TimeZone answers "time in <city>" using Go's embedded timezone database.
// Port of SearXNG's time_zone plugin (subset: IANA-name + common-city lookup).
type TimeZone struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &TimeZone{} })
}

func (p *TimeZone) Name() string                                  { return "time_zone" }
func (p *TimeZone) Keywords() []string                            { return nil }
func (p *TimeZone) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *TimeZone) OnResult(*result.MainResult) (bool, error)     { return true, nil }

// commonCities maps lowercase city names to IANA timezone identifiers.
var commonCities = map[string]string{
	"new york": "America/New_York", "nyc": "America/New_York",
	"los angeles": "America/Los_Angeles", "la": "America/Los_Angeles",
	"chicago": "America/Chicago", "denver": "America/Denver",
	"london": "Europe/London", "paris": "Europe/Paris",
	"berlin": "Europe/Berlin", "madrid": "Europe/Madrid",
	"rome": "Europe/Rome", "moscow": "Europe/Moscow",
	"tokyo": "Asia/Tokyo", "beijing": "Asia/Shanghai",
	"shanghai": "Asia/Shanghai", "hong kong": "Asia/Hong_Kong",
	"singapore": "Asia/Singapore", "dubai": "Asia/Dubai",
	"sydney": "Australia/Sydney", "melbourne": "Australia/Melbourne",
	"mumbai": "Asia/Kolkata", "delhi": "Asia/Kolkata", "india": "Asia/Kolkata",
	"toronto": "America/Toronto", "vancouver": "America/Vancouver",
	"sao paulo": "America/Sao_Paulo", "mexico city": "America/Mexico_City",
	"utc": "UTC", "gmt": "UTC",
}

// PostSearch returns a time answer when the query asks for the time somewhere.
func (p *TimeZone) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	q := strings.ToLower(strings.TrimSpace(sc.Query))
	place := ""
	switch {
	case strings.HasPrefix(q, "time in "):
		place = strings.TrimPrefix(q, "time in ")
	case strings.HasPrefix(q, "what time is it in "):
		place = strings.TrimPrefix(q, "what time is it in ")
	case strings.HasPrefix(q, "current time in "):
		place = strings.TrimPrefix(q, "current time in ")
	default:
		return nil, nil
	}
	place = strings.TrimSpace(place)

	tzName, ok := commonCities[place]
	if !ok {
		// try the place as a direct IANA name (e.g. "Europe/Berlin")
		tzName = canonicalIANA(place)
	}
	if tzName == "" {
		return nil, nil
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, nil
	}
	now := time.Now().In(loc)
	answer := "Time in " + place + ": " + now.Format("Mon, 15:04 MST (2006-01-02)")
	return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: answer}}, nil
}

// canonicalIANA accepts an IANA-style name (case-insensitive) and returns it if
// LoadLocation recognizes it.
func canonicalIANA(s string) string {
	// Title-case the segments like "europe/berlin" -> "Europe/Berlin".
	parts := strings.Split(s, "/")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	cand := strings.Join(parts, "/")
	if _, err := time.LoadLocation(cand); err == nil {
		return cand
	}
	return ""
}
