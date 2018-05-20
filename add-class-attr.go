package googledrive2hugo

import (
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

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
