package googledrive2hugo

// generic routines to deal with HTML (or XML) nodes

import (
	"io"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/andybalholm/cascadia"
	"github.com/client9/htmlfmt"
)

var selectorBody = cascadia.MustCompile("body")

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

func getBody(root *html.Node) *html.Node {
	out := selectorBody.MatchFirst(root)
	if out == nil {
		out = root
	}
	return out
}

func getStyleAttr(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "style" {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	// somehow gdoc occassionally inserts a
	// <span></span> which indicates a space
	// it has no style or attributes
	if n.DataAtom == atom.Span && n.FirstChild == nil && len(n.Attr) == 0 {
		return " "
	}

	if n.Type == html.ElementNode && n.DataAtom == atom.Br {
		return "\n"
	}

	out := ""
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out += getTextContent(c)
	}
	return removeNbsp(out)
}

func getTextNodes(root *html.Node) []*html.Node {
	var out []*html.Node
	if root.Type == html.TextNode {
		out = append(out, root)
		return out
	}
	getChildTextNodes(root, &out)
	return out
}

func getChildTextNodes(root *html.Node, out *[]*html.Node) {
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			*out = append(*out, c)
			continue
		}
		getChildTextNodes(c, out)
	}
}

func getNextTextNode(root *html.Node, current *html.Node) *html.Node {
	for current != root {
		if next := getNextTextNodeSibling(current.NextSibling); next != nil {
			return next
		}
		current = current.Parent
	}
	return nil
}

func getPrevTextNode(root *html.Node, current *html.Node) *html.Node {
	for current != root {
		if prev := getPrevTextNodeSibling(current.PrevSibling); prev != nil {
			return prev
		}
		current = current.Parent
	}
	return nil
}

func getNextTextNodeSibling(current *html.Node) *html.Node {
	for c := current; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			return c
		}
		return getNextTextNodeSibling(c.FirstChild)
	}
	return nil
}
func getPrevTextNodeSibling(current *html.Node) *html.Node {
	for c := current; c != nil; c = c.PrevSibling {
		if c.Type == html.TextNode {
			return c
		}
		return getPrevTextNodeSibling(c.LastChild)
	}
	return nil
}
