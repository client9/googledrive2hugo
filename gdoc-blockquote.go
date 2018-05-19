package googledrive2hugo

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/andybalholm/cascadia"
)

var (
	selectorBlockquote = cascadia.MustCompile(`p[style*="margin-left:36pt"]`)
)

// GdocBlockquote converts a sequence of <p style="margin-left:36pt"> to a blockquote
//
func GdocBlockquote(root *html.Node) error {
	var first *html.Node
	nodes := selectorBlockquote.MatchAll(root)
	for _, n := range nodes {
		// merge into previous
		if first != nil && n.PrevSibling == first {
			n.Parent.RemoveChild(n)
			first.AppendChild(newElementNode("br"))
			reparentChildren(first, n)
			continue
		}

		// transform into blockquote
		first = n
		first.DataAtom = atom.Blockquote
		first.Data = "blockquote"
		first.Attr = nil
	}

	return nil
}
