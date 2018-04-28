package googledrive2hugo

// generic routines to deal with HTML (or XML) nodes

import (
	"io"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/client9/htmlfmt"
)

func newTextNode(data string) *html.Node {
	return &html.Node{
		Type: html.TextNode,
		Data: data,
	}
}

func newElementNode(name string) *html.Node {
	a := atom.Lookup([]byte(name))
	if a == 0 {
		return nil
	}
	return &html.Node{
		Type:     html.ElementNode,
		Data:     name,
		DataAtom: a,
	}
}

func renderChildren(w io.Writer, root *html.Node) error {
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := htmlfmt.Render(w, c, "", ""); err != nil {
			return err
		}
	}
	return nil
}

func reparentChildren(newParent, oldParent *html.Node) {
	for c := oldParent.FirstChild; c != nil; c = oldParent.FirstChild {
		oldParent.RemoveChild(c)
		newParent.AppendChild(c)
	}
}

func removeAllChildren(n *html.Node) {
	c := n.FirstChild
	for c != nil {
		n.RemoveChild(c)
		c = n.FirstChild
	}
}

func getClassAttr(root *html.Node) string {
	for _, attr := range root.Attr {
		if attr.Key == "class" {
			return attr.Val
		}
	}
	return ""
}

// provides some function to edit every text node
func transformTextNodes(n *html.Node, fn func(string) string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			c.Data = fn(c.Data)
			continue
		}
		transformTextNodes(c, fn)
	}
}
