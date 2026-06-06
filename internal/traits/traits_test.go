package traits

import "testing"

func TestEngineLocaleMatching(t *testing.T) {
	tr := NewEngineTraits()
	tr.Languages = map[string]string{
		"fr":    "fr_FR",
		"en":    "en_US",
		"zh":    "zh",
		"zh-TW": "zh_Hant",
	}
	tr.Regions = map[string]string{
		"fr-BE": "fr_BE",
		"fr-CA": "fr_CA",
		"en-US": "en_US",
		"en-GB": "en_GB",
	}
	tr.AllLocale = "en_US"

	cases := []struct {
		name   string
		fn     func(string, string) string
		locale string
		want   string
	}{
		{"region 1:1", tr.Region, "fr-BE", "fr_BE"},
		{"region 1:1 ca", tr.Region, "fr-CA", "fr_CA"},
		{"lang 1:1", tr.Language, "fr", "fr_FR"},
		{"all -> all_locale", tr.Language, "all", "en_US"},
		{"lang fallback from region", tr.Language, "fr-CH", "fr_FR"}, // no fr-CH lang, falls to fr
		{"territory narrowing", tr.Region, "und-US", "en_US"},        // US official lang en -> en-US
		{"default when unknown", tr.Language, "xx-YY", "DEF"},
	}
	for _, c := range cases {
		got := c.fn(c.locale, "DEF")
		if got != c.want {
			t.Errorf("%s: %s -> %q, want %q", c.name, c.locale, got, c.want)
		}
	}
}

func TestSplitLocale(t *testing.T) {
	cases := map[string][2]string{
		"en-US":   {"en", "US"},
		"fr":      {"fr", ""},
		"zh_Hant": {"zh", "HANT"},
	}
	for in, want := range cases {
		l, terr := splitLocale(in)
		if l != want[0] || terr != want[1] {
			t.Errorf("splitLocale(%q) = (%q,%q), want (%q,%q)", in, l, terr, want[0], want[1])
		}
	}
}
