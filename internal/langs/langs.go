// Package langs holds a curated set of major world languages, keyed by their
// ISO 639-3 code. It is intentionally a hand-picked subset (not the full ~7,900
// codes); extend the list below as needed.
package langs

// Language is one selectable translation language.
type Language struct {
	Code string `json:"code"` // ISO 639-3, e.g. "fra"
	Name string `json:"name"` // English display name
}

// All is the curated list, roughly ordered by number of speakers / usefulness.
var All = []Language{
	{"eng", "English"},
	{"cmn", "Mandarin Chinese"},
	{"hin", "Hindi"},
	{"spa", "Spanish"},
	{"fra", "French"},
	{"ara", "Arabic (Standard)"},
	{"ben", "Bengali"},
	{"por", "Portuguese"},
	{"rus", "Russian"},
	{"urd", "Urdu"},
	{"ind", "Indonesian"},
	{"deu", "German"},
	{"jpn", "Japanese"},
	{"pcm", "Nigerian Pidgin"},
	{"mar", "Marathi"},
	{"tel", "Telugu"},
	{"tur", "Turkish"},
	{"tam", "Tamil"},
	{"yue", "Cantonese"},
	{"vie", "Vietnamese"},
	{"kor", "Korean"},
	{"ita", "Italian"},
	{"hau", "Hausa"},
	{"tha", "Thai"},
	{"guj", "Gujarati"},
	{"pol", "Polish"},
	{"ukr", "Ukrainian"},
	{"fas", "Persian (Farsi)"},
	{"pan", "Punjabi"},
	{"swa", "Swahili"},
	{"nld", "Dutch"},
	{"ron", "Romanian"},
	{"ell", "Greek"},
	{"ces", "Czech"},
	{"swe", "Swedish"},
	{"heb", "Hebrew"},
	{"hun", "Hungarian"},
	{"fin", "Finnish"},
	{"dan", "Danish"},
	{"nor", "Norwegian"},
	{"nob", "Norwegian Bokmål"},
	{"cat", "Catalan"},
	{"bul", "Bulgarian"},
	{"srp", "Serbian"},
	{"hrv", "Croatian"},
	{"slk", "Slovak"},
	{"slv", "Slovenian"},
	{"lit", "Lithuanian"},
	{"lav", "Latvian"},
	{"est", "Estonian"},
	{"msa", "Malay"},
	{"fil", "Filipino"},
	{"afr", "Afrikaans"},
	{"amh", "Amharic"},
	{"zul", "Zulu"},
	{"isl", "Icelandic"},
	{"gle", "Irish"},
	{"eus", "Basque"},
	{"glg", "Galician"},
}

var byCode = func() map[string]Language {
	m := make(map[string]Language, len(All))
	for _, l := range All {
		m[l.Code] = l
	}
	return m
}()

// Valid reports whether code is in the curated list.
func Valid(code string) bool {
	_, ok := byCode[code]
	return ok
}

// Name returns the English name for a code, or the code itself if unknown.
func Name(code string) string {
	if l, ok := byCode[code]; ok {
		return l.Name
	}
	return code
}
