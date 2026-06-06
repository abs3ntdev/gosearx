// Package preferences models user settings persisted in a cookie (base64 JSON),
// the Go successor to SearXNG's Preferences/cookie system. Preferences override
// instance defaults per request: language, safesearch, default category,
// autocomplete on/off, results-in-new-tab, and per-engine disable list.
package preferences

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

const cookieName = "prefs"
const cookieMaxAge = 365 * 24 * time.Hour

// Preferences holds the user-configurable settings.
type Preferences struct {
	Language        string `json:"language,omitempty"`
	SafeSearch      *int   `json:"safesearch,omitempty"`
	DefaultCategory string `json:"category,omitempty"`
	Autocomplete    *bool  `json:"autocomplete,omitempty"`
	ResultsNewTab   bool   `json:"results_new_tab,omitempty"`
	// DisabledEngines is the set of engine names the user turned off.
	DisabledEngines []string `json:"disabled_engines,omitempty"`
}

// FromRequest decodes preferences from the request cookie (empty if absent).
func FromRequest(r *http.Request) *Preferences {
	p := &Preferences{}
	c, err := r.Cookie(cookieName)
	if err != nil {
		return p
	}
	raw, err := base64.URLEncoding.DecodeString(c.Value)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(raw, p)
	return p
}

// Encode serializes preferences to the cookie value.
func (p *Preferences) Encode() string {
	b, _ := json.Marshal(p)
	return base64.URLEncoding.EncodeToString(b)
}

// Write sets the preferences cookie on the response.
func (p *Preferences) Write(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    p.Encode(),
		Path:     "/",
		MaxAge:   int(cookieMaxAge.Seconds()),
		HttpOnly: false, // the SPA reads it via /api/preferences, not JS cookie
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear removes the preferences cookie.
func Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
	})
}

// IsEngineDisabled reports whether the user disabled the given engine.
func (p *Preferences) IsEngineDisabled(name string) bool {
	for _, e := range p.DisabledEngines {
		if e == name {
			return true
		}
	}
	return false
}
