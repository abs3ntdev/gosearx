// Package traits maps a user-selected locale (SearXNG style, e.g. "en-US") to
// the language/region code a given engine expects. It is the Go port of
// searx/enginelib/traits.py + the get_engine_locale matching from
// searx/locales.py.
//
// The full SearXNG implementation leans on babel's CLDR territory/official-
// language tables. This port implements the same matching *structure* (1:1
// mapping → language-only → territory narrowing) over a compact built-in table
// of official languages, which covers the common cases. Engines whose traits
// are loaded from data files supply their own mapping dicts.
package traits

import "strings"

// EngineTraits holds an engine's locale mappings (port of EngineTraits).
type EngineTraits struct {
	// Languages maps SearXNG language (e.g. "fr") -> engine code (e.g. "fr_FR").
	Languages map[string]string `json:"languages"`
	// Regions maps SearXNG region (e.g. "fr-BE") -> engine code.
	Regions map[string]string `json:"regions"`
	// AllLocale is what "all" maps to (engine default), if any.
	AllLocale string `json:"all_locale"`
	// Custom is a free-form bag for engine-specific traits.
	Custom map[string]any `json:"custom"`
}

// NewEngineTraits returns an empty traits struct with initialized maps.
func NewEngineTraits() *EngineTraits {
	return &EngineTraits{
		Languages: map[string]string{},
		Regions:   map[string]string{},
		Custom:    map[string]any{},
	}
}

// Language returns the engine language code best fitting locale (or def).
func (t *EngineTraits) Language(locale, def string) string {
	if locale == "all" && t.AllLocale != "" {
		return t.AllLocale
	}
	return getEngineLocale(locale, t.Languages, def)
}

// Region returns the engine region code best fitting locale (or def).
func (t *EngineTraits) Region(locale, def string) string {
	if locale == "all" && t.AllLocale != "" {
		return t.AllLocale
	}
	return getEngineLocale(locale, t.Regions, def)
}

// getEngineLocale ports searx.locales.get_engine_locale's matching structure.
func getEngineLocale(locale string, m map[string]string, def string) string {
	if locale == "" {
		return def
	}
	// 1) direct 1:1 mapping (region or language).
	if v, ok := m[locale]; ok {
		return v
	}
	lang, territory := splitLocale(locale)

	// 2) language-only 1:1 mapping (e.g. "zh-HK" -> try "zh").
	if v, ok := m[lang]; ok {
		return v
	}

	// 3) territory narrowing: try official languages of the territory.
	if territory != "" {
		for _, off := range officialLanguages(territory) {
			if v, ok := m[off+"-"+territory]; ok {
				return v
			}
		}
		// 3b) language in other territories where it is official.
		for _, terr := range territoriesForLanguage(lang) {
			if v, ok := m[lang+"-"+terr]; ok {
				return v
			}
		}
	} else {
		// 4) language without territory: find a territory where it's official.
		for _, terr := range territoriesForLanguage(lang) {
			if v, ok := m[lang+"-"+terr]; ok {
				return v
			}
		}
	}
	return def
}

func splitLocale(locale string) (lang, territory string) {
	locale = strings.ReplaceAll(locale, "_", "-")
	parts := strings.SplitN(locale, "-", 2)
	lang = strings.ToLower(parts[0])
	if len(parts) == 2 {
		territory = strings.ToUpper(parts[1])
	}
	return lang, territory
}
