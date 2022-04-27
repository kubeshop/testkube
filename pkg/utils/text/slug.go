package text

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var lat = []*unicode.RangeTable{unicode.Letter, unicode.Number}
var nop = []*unicode.RangeTable{unicode.Mark, unicode.Sk, unicode.Lm}

// Slug replaces each run of characters which are not unicode letters or
// numbers with a single hyphen, except for leading or trailing runs. Letters
// will be stripped of diacritical marks and lowercased. Letter or number
// codepoints that do not have combining marks or a lower-cased variant will
// be passed through unaltered.
func Slug(s string) string {
	buf := make([]rune, 0, len(s))
	dash := false
	for _, r := range norm.NFKD.String(s) {
		switch {
		// unicode 'letters' like mandarin characters pass through
		case unicode.IsOneOf(lat, r):
			buf = append(buf, unicode.ToLower(r))
			dash = true
		case unicode.IsOneOf(nop, r):
			// skip
		case dash:
			buf = append(buf, '-')
			dash = false
		}
	}
	if i := len(buf) - 1; i >= 0 && buf[i] == '-' {
		buf = buf[:i]
	}

	return strings.Replace(string(buf), "ł", "l", -1)
}

var eventNameFilter = regexp.MustCompile("[^a-zA-Z0-9]+")

func GAEventName(in string) string {
	filtered := eventNameFilter.ReplaceAllString(in, "_")
	filtered = strings.TrimLeft(filtered, "_")
	if len(filtered) > 40 {
		return filtered[:40]
	}
	return filtered
}
