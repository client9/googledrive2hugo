package googledrive2hugo

// indented blocks in a monospace font are <blockquote><pre>
// or rather, preformatted blockquotes.

// the tree operations previous leave the tree in <p style='margin-left:36pt><code>
// but it's not code.

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/andybalholm/cascadia"
)

var (
	selectorCode      = cascadia.MustCompile("code")
	selectorBlockCode = cascadia.MustCompile(`p[style="margin-left:36pt"]>code:only-child`)
)

func GdocBlockquotePre(root *html.Node) error {
	var first *html.Node

	for _, code := range selectorBlockCode.MatchAll(root) {
		// gdoc is <p><code>.. nothing between the <p> and <code>
		// selector will match <p>foo<code> since it doesn't care about
		// text nodes.  Make sure <code> is truly only child
		if code.PrevSibling != nil || code.NextSibling != nil {
			continue
		}

		// get enclosing <p>
		p := code.Parent

		// merge into previous
		if first != nil && p.PrevSibling == first {
			p.Parent.RemoveChild(p)
			first.FirstChild.AppendChild(newTextNode("\n"))
			reparentChildren(first.FirstChild, code)
			continue
		}

		// convert from <p> to <pre>
		p.DataAtom = atom.Blockquote
		p.Data = "blockquote"
		p.Attr = nil
		first = p

	}
	return nil
}

// Reparent <code> children
func reparentCodeChildren(newParent, oldParent *html.Node) {
	nodes := selectorCode.MatchAll(oldParent)
	for _, n := range nodes {
		reparentChildren(newParent, n)
	}
}
