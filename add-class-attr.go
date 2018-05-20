package googledrive2hugo

import (
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

var xxx = map[string]string{
	"table":      "table table-sm",
	"blockquote": "pl-3 lines-dense",
	"pre":        "p-1 pl-3 lines-dense",
	"h1":         "h2 mb-3", // no top margin
	"h2":         "h4 mt-4 mb-4",
	"h3":         "h5 mt-4 mb-4",
}

type AddClassAttr struct {
	ClassMap map[string]string
}

func (a *AddClassAttr) Run(n *html.Node, log ilog.Logger) (err error) {
	if override := a.ClassMap[n.Data]; override != "" {
		n.Attr = append(n.Attr, html.Attribute{Key: "class", Val: override})
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {

		if c.Type == html.ElementNode {
			a.Run(c, log)
		}
	}

	return nil
}
