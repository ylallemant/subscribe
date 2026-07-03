package project

import (
	"strings"
	"unicode"
)

// latinFold maps common accented Latin letters (and a few ligatures) to their
// ASCII form so slugs stay readable for French, Spanish, German, etc. without
// pulling in golang.org/x/text.
var latinFold = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'č': "c", 'ĉ': "c", 'ċ': "c",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ę': "e", 'ě': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i", 'ı': "i",
	'ñ': "n", 'ń': "n", 'ň': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ő': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u", 'ű': "u",
	'ý': "y", 'ÿ': "y",
	'š': "s", 'ś': "s", 'ş': "s",
	'ž': "z", 'ź': "z", 'ż': "z",
	'ł': "l", 'đ': "d", 'ð': "d", 'þ': "th", 'ß': "ss",
	'œ': "oe", 'æ': "ae",
}

// Slugify normalises a display name to a slug: lowercase ASCII, accents folded,
// every other character collapsed to a single "_", trimmed. Returns "" if
// nothing usable remains (callers substitute a fallback).
func Slugify(name string) string {
	var b strings.Builder
	prevUnderscore := false
	emit := func(s string) {
		b.WriteString(s)
		prevUnderscore = false
	}
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			emit(string(r))
		case latinFold[r] != "":
			emit(latinFold[r])
		case unicode.IsLetter(r) && r < 128:
			emit(string(r))
		default:
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
