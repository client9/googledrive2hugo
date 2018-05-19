package googledrive2hugo

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/andybalholm/cascadia"
)

var (
	// <p><code>...</code</p> or
	// <p>foo <code>...</code> bar </p>
	selectorCodeBlock = cascadia.MustCompile(`p>code:only-child`)
)

func GdocCodeBlock(root *html.Node) error {
	var first *html.Node

	for _, code := range selectorCodeBlock.MatchAll(root) {
		// gdoc is <p><code>.. nothing between the <p> and <code>
		// selector will match <p>foo<code> since it doesn't care about
		// text nodes.  Make sure <code> is truly only child
		if code.PrevSibling != nil || code.NextSibling != nil {
			continue
		}

		// get enclosing <p>
		p := code.Parent

		// if this is inside a <td> or <li> just leave alone
		if p.Parent.DataAtom == atom.Td || p.Parent.DataAtom == atom.Li {
			first = nil
			continue
		}

		// merge into previous
		if first != nil && p.PrevSibling == first {
			p.Parent.RemoveChild(p)
			first.FirstChild.AppendChild(newTextNode("\n"))
			reparentChildren(first.FirstChild, code)
			continue
		}

		// convert from <p> to <pre>
		p.DataAtom = atom.Pre
		p.Data = "pre"
		p.Attr = nil
		first = p
	}

	return nil
}
