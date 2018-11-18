package googledrive2hugo

import (
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

var (
	unsmartReplacer *strings.Replacer
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
}

// pure function
func unsmart(s string) string {
	return unsmartReplacer.Replace(s)
}

type UnsmartCode struct {
	selector cascadia.Selector
}

func (n *UnsmartCode) Init() (err error) {
	const pattern = "code,var,kbd"
	n.selector, err = cascadia.Compile(pattern)
	return err
}

func (n *UnsmartCode) Run(root *html.Node, log ilog.Logger) (err error) {
	for _, node := range n.selector.MatchAll(root) {
		if transformTextNodes(node, unsmart) {
			log.Debug("", "tag", node.Data)
		}
	}
	return nil
}
