package googledrive2hugo

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/andybalholm/cascadia"
)

var (
	unsmartReplacer *strings.Replacer
	unsmartSelector cascadia.Selector
)

func init() {
	replacements := []string{
		"\u00a0", " ", // non breaking space
		"\u201c", `"`, // double quote open
		"\u201d", `"`, // double quote close
		"\u2018", "'", // single quote open
		"\u2019", "'", // single quote close
		"\u2026", "...", // ellipis
		"\u2010", "-", // hyphen
		"\u2011", "-", // non-breaking hyphen
		"\u2012", "--", // figure dash
		"\u2013", "--", // en dash
		"\u2014", "---", // em dash
	}
	unsmartReplacer = strings.NewReplacer(replacements...)
	unsmartSelector = cascadia.MustCompile("code,var,kbd")

}

// pure function
func unsmart(s string) string {
	return unsmartReplacer.Replace(s)
}

func UnsmartCode(root *html.Node) error {
	for _, n := range unsmartSelector.MatchAll(root) {
		transformTextNodes(n, unsmart)
	}
	return nil
}
