package googledrive2hugo

import (
	"golang.org/x/net/html"
)

var xxx = map[string]map[string]string{
	"table": {
		"class": "table table-sm",
	},
	"blockquote": {
		"class": "pl-3 lines-dense",
	},
	"pre": {
		"class": "p-1 pl-3 lines-dense",
	},
	"h1": {
		// no top margin
		"class": "h2 mb-3",
	},
	"h2": {
		"class": "h4 mt-4 mb-4",
	},
	"h3": {
		"class": "h5 mt-4 mb-4",
	},
}

func AddClassAttr(n *html.Node) {
	if override := xxx[n.Data]; override != nil {
		for k, v := range override {
			n.Attr = append(n.Attr, html.Attribute{Key: k, Val: v})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {

		if c.Type == html.ElementNode {
			AddClassAttr(c)
		}
	}
}
