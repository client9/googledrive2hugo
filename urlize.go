package googledrive2hugo

import (
	"strings"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"unicode"
)

// URLize takes a path and converts into something nice
// for a URL
func URLize(path string) string {

	path = strings.Replace(strings.TrimSpace(path), " ", "-", -1)
	path = strings.ToLower(path)
	path = UnicodeSanitize(path)
	return path
}

// From https://golang.org/src/net/url/url.go
func ishex(c rune) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// UnicodeSanitize sanitizes string to be used in Hugo URL's, allowing only
// a predefined set of special Unicode characters.
// If RemovePathAccents configuration flag is enabled, Uniccode accents
// are also removed.
// TODO: From Hugo github.com/gohugoio/hugo/helpers/path.go
func UnicodeSanitize(s string) string {
	source := []rune(s)
	target := make([]rune, 0, len(source))

	for i, r := range source {
		if r == '%' && i+2 < len(source) && ishex(source[i+1]) && ishex(source[i+2]) {
			target = append(target, r)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '.' || r == '/' || r == '\\' || r == '_' || r == '-' || r == '#' || r == '+' || r == '~' {
			target = append(target, r)
		}
	}

	var result string

	if true {
		// remove accents - see https://blog.golang.org/normalization
		t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
		result, _, _ = transform.String(t, string(target))
	} else {
		result = string(target)
	}

	return result
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}
