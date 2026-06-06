package traits

// cldr.go holds a compact subset of CLDR territory -> official languages data,
// enough to drive the territory-narrowing rules in getEngineLocale for common
// locales. SearXNG uses babel's full dataset; this can be expanded or replaced
// by loading a generated data file later.

// territoryOfficialLanguages maps an ISO territory to its main official
// language(s), most significant first.
var territoryOfficialLanguages = map[string][]string{
	"US": {"en"},
	"GB": {"en"},
	"AU": {"en"},
	"CA": {"en", "fr"},
	"IE": {"en", "ga"},
	"NZ": {"en"},
	"FR": {"fr"},
	"BE": {"nl", "fr", "de"},
	"CH": {"de", "fr", "it"},
	"DE": {"de"},
	"AT": {"de"},
	"ES": {"es", "ca"},
	"MX": {"es"},
	"AR": {"es"},
	"PT": {"pt"},
	"BR": {"pt"},
	"IT": {"it"},
	"NL": {"nl"},
	"SE": {"sv"},
	"NO": {"no", "nb"},
	"DK": {"da"},
	"FI": {"fi", "sv"},
	"PL": {"pl"},
	"RU": {"ru"},
	"UA": {"uk"},
	"CN": {"zh"},
	"TW": {"zh"},
	"HK": {"zh", "en"},
	"JP": {"ja"},
	"KR": {"ko"},
	"IN": {"hi", "en"},
	"SA": {"ar"},
	"AE": {"ar"},
	"EG": {"ar"},
	"TR": {"tr"},
	"GR": {"el"},
	"CZ": {"cs"},
	"HU": {"hu"},
	"RO": {"ro"},
	"BG": {"bg"},
	"TH": {"th"},
	"VN": {"vi"},
	"ID": {"id"},
	"MY": {"ms"},
	"IL": {"he"},
}

// officialLanguages returns the official languages of a territory.
func officialLanguages(territory string) []string {
	return territoryOfficialLanguages[territory]
}

// languageTerritories is the reverse index, lazily built.
var languageTerritories map[string][]string

func buildReverseIndex() {
	languageTerritories = map[string][]string{}
	for terr, langs := range territoryOfficialLanguages {
		for _, l := range langs {
			languageTerritories[l] = append(languageTerritories[l], terr)
		}
	}
}

// territoriesForLanguage returns territories where lang is official.
func territoriesForLanguage(lang string) []string {
	if languageTerritories == nil {
		buildReverseIndex()
	}
	return languageTerritories[lang]
}
