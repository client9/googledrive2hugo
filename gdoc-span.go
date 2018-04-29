package googledrive2hugo

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	//"github.com/andybalholm/cascadia"
)

// converts span wrappers to a series of <b><i><code> elements
func convertSpan(n *html.Node) *html.Node {
	next := n.NextSibling

	if n.Type == html.TextNode && strings.Contains(n.Data, "\u00a0") {
		n.Data = removeNbsp(n.Data)
		return next
	}

	// useless span tag wrapping an anchor
	// before: <span><a href="...">txt</a></span>
	// after  :<a href="...">txt</a>
	//
	if isLinkWrapper(n) {
		link := n.FirstChild
		fixHrefAttr(link)
		n.RemoveChild(link)
		// promote link to main
		parent := n.Parent
		parent.InsertBefore(link, n)
		parent.RemoveChild(n)
		return next
	}

	// span that encodes some style and only has text or <br> children
	// after : <span style="...">text</span>
	// before: <b><i>text</i></b>
	//
	if isTextWrapper(n) {
		text := getTextContent(n)
		// span tag with style, and no children
		// often seen in line breaks like
		//
		// <p style="..."><span style="..."></span></p>
		//
		// delete the span.  The remaining <p></p> will get
		// zapped later.
		//
		if text == "" {
			n.Parent.RemoveChild(n)
			return next
		}

		// create a new text node
		newNode := newTextNode(text)

		// based on style wrap the text node with appropriate
		// tags
		style := getStyleAttr(n)
		if isStyleItalics(style) {
			wrapper := newElementNode("em")
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleBold(style) {
			wrapper := newElementNode("strong")
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleUnderline(style) {
			wrapper := newElementNode("u")
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleStrikethrough(style) {
			wrapper := newElementNode("del")
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleCode(style) {
			wrapper := newElementNode("code")
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}

		parent := n.Parent
		parent.InsertBefore(newNode, n)
		parent.RemoveChild(n)
		return next
	}

	c := n.FirstChild
	for c != nil {
		c = convertSpan(c)
	}
	return next
}

// must be a <span> and all children are text nodes or <br>
func isTextWrapper(n *html.Node) bool {
	if n.Type != html.ElementNode || n.DataAtom != atom.Span {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			continue
		}
		if c.Type == html.ElementNode && c.DataAtom == atom.Br {
			continue
		}
		return false
	}
	return true
}

func isLinkWrapper(n *html.Node) bool {
	if n.Type != html.ElementNode || n.DataAtom != atom.Span {
		return false
	}
	link := n.FirstChild
	if link == nil {
		return false
	}
	if link.NextSibling != nil {
		return false
	}
	return link.Type == html.ElementNode && link.DataAtom == atom.A
}

func isStyleBold(s string) bool {
	return strings.Contains(s, "font-weight:700")
}
func isStyleItalics(s string) bool {
	return strings.Contains(s, "font-style:italic")
}
func isStyleStrikethrough(s string) bool {
	return strings.Contains(s, "text-decoration:line-through")
}
func isStyleUnderline(s string) bool {
	return strings.Contains(s, "text-decoration:underline")
}

// isStyleCode inspects the CSS Style to see if a monospace font is used
//  This is hte current list of monospace fonts in google docs circa 2018
func isStyleCode(s string) bool {
	// other monospace fonts TBD
	var monospace = []string{
		"Anonymous Pro",
		"Consolas",
		"Courier",
		"Cousine",
		"Cutive Mono",
		"Fira Mono",
		"Inconsolata",
		"Nova Mono",
		"Overpass Mono",
		"Oxygen Mono",
		"PT Mono",
		"Roboto Mono",
		"Share Tech Mono",
		"Source Code Pro",
		"Space Mono",
		"Ubuntu Mono",
		"VT323",
	}
	for _, font := range monospace {
		if strings.Contains(s, font) {
			return true
		}
	}
	return false
}

func fixHrefAttr(n *html.Node) {
	const prefix = "https://www.google.com/url?q="
	const suffix = "&sa="
	if n.Attr == nil {
		return
	}
	for i := 0; i < len(n.Attr); i++ {
		if n.Attr[i].Key != "href" {
			continue
		}
		val := n.Attr[i].Val
		if strings.HasPrefix(val, prefix) {
			val = val[len(prefix):]
		}
		if idx := strings.LastIndex(val, suffix); idx != -1 {
			val = val[0:idx]
		}
		n.Attr[i].Val = val
	}
}
