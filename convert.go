package googledrive2hugo

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/andybalholm/cascadia"
)

var (
	selectorTitle    = cascadia.MustCompile("p[class~=title]")
	selectorSubtitle = cascadia.MustCompile("p[class~=subtitle]")
	selectorCode     = cascadia.MustCompile("code")
)

func isStyleIndent(s string) bool {
	// could be contains margin-left && not margin-left:0
	return strings.Contains(s, "margin-left:36pt")
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

// recursive, with special rules for gDoc
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

// remove non-breaking spaces.  Unclear why google adds them or how
// they get added.
func removeNbsp(src string) string {
	return strings.Replace(src, "\u00a0", " ", -1)
}

// assumes <span style=font-family=monospace> has been convert to <code>
// Must be <p><code>
func isCodeWrapper(n *html.Node) (bool, string) {
	if n.Type != html.ElementNode || n.DataAtom != atom.P {
		return false, ""
	}

	// if written in google-docs, then it's exactly one <code> inside <p>
	// but if cut-n-paste, then it can be
	// exactly one <code> inside <p>
	code := n.FirstChild
	if code == nil || code.NextSibling != nil || code.Type != html.ElementNode || code.DataAtom != atom.Code {
		return false, ""
	}

	// figure out text content
	text := code.FirstChild
	if text == nil {
		return true, "\n"
	}
	return true, text.Data
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

func isIndentedP(n *html.Node) bool {
	return n.Type == html.ElementNode && n.DataAtom == atom.P &&
		isStyleIndent(getStyleAttr(n))
}

func isIndentedCode(n *html.Node) bool {
	if !isIndentedP(n) {
		return false
	}
	code := n.FirstChild
	return code != nil &&
		code.Type == html.ElementNode &&
		code.DataAtom == atom.Code
}

// Reparent <code> children
func reparentCodeChildren(newParent, oldParent *html.Node) {
	nodes := selectorCode.MatchAll(oldParent)
	for _, n := range nodes {
		reparentChildren(newParent, n)
	}
}
func convertPre(n *html.Node) {
	var pre *html.Node
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		if !isIndentedCode(c) {
			pre = nil
			convertPre(c)
			c = next
			continue
		}

		// we have a <p><code> and we have an existing code block
		if pre != nil {
			pre.AppendChild(newTextNode("\n"))
			n.RemoveChild(c)
			reparentCodeChildren(pre, c)
			c = next
			continue
		}

		// we have <p><code>.  Create new <pre><code> block
		bq := newElementNode("blockquote")
		pre = newElementNode("pre")
		bq.AppendChild(pre)

		n.InsertBefore(bq, c)
		n.RemoveChild(c)
		reparentCodeChildren(pre, c)
		c = next
	}
}

//  <p style="margin-left:36pt">
func convertBlockquote(n *html.Node) {
	var bq *html.Node

	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		if !isIndentedP(c) {
			bq = nil
			convertBlockquote(c)
			c = next
			continue
		}

		// we have a <p margin-left:36pt> and we have an existing code block
		if bq != nil {
			n.RemoveChild(c)
			//bq.AppendChild(c)

			bq.AppendChild(newElementNode("br"))
			reparentChildren(bq, c)
			c = next
			continue
		}

		// we have <p margin-left:36pt>
		//  create new blockquote
		bq = newElementNode("blockquote")
		n.InsertBefore(bq, c)
		n.RemoveChild(c)
		//bq.AppendChild(c)
		reparentChildren(bq, c)
		c = next
	}
}

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

func fixCodeBlock(n *html.Node) {
	// scan child and merge adjacent <p><code> lines into one
	var textNode *html.Node
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		isCode, code := isCodeWrapper(c)
		if !isCode {
			if textNode != nil {
				textNode.Data = strings.Trim(textNode.Data, "\n")
				textNode = nil
			}
			fixCodeBlock(c)
			c = next
			continue
		}

		// we have <p><code>
		// we don't support tables or list of codes
		if c.Parent.DataAtom == atom.Td || c.Parent.DataAtom == atom.Li {
			c = next
			continue
		}

		// we have a <p><code> and we have an existing code block
		if textNode != nil {
			n.RemoveChild(c)
			textNode.Data += code + "\n"
			c = next
			continue
		}

		// we have <p><code>.
		// Create new <pre><code> block
		textNode = newTextNode(code + "\n")
		codeNode := newElementNode("code")
		preNode := newElementNode("pre")
		codeNode.AppendChild(textNode)
		preNode.AppendChild(codeNode)
		n.InsertBefore(preNode, c)
		n.RemoveChild(c)

		c = next
	}
	if textNode != nil {
		textNode.Data = strings.Trim(textNode.Data, "\n")
	}
}

func extractTitle(root *html.Node) string {
	n := selectorTitle.MatchFirst(root)
	if n == nil {
		return ""
	}
	val := getTextContent(n)
	n.Parent.RemoveChild(n)
	return val
}
func extractSubtitle(root *html.Node) string {
	n := selectorSubtitle.MatchFirst(root)
	if n == nil {
		return ""
	}
	val := getTextContent(n)
	n.Parent.RemoveChild(n)
	return val
}

// if you already have a google doc node
func fromNode(root *html.Node, w io.Writer) (map[string]interface{}, error) {
	meta := make(map[string]interface{})
	if title := extractTitle(root); title != "" {
		meta["title"] = title
	}

	if desc := extractSubtitle(root); desc != "" {
		meta["description"] = desc
	}

	tx := []func(*html.Node){
		// gdoc specific
		GdocTable,
		GdocAttr,

		// more generic
		RemoveEmptyTags,
		HugoFrontMatter,
		UnsmartCode,
		AddClassAttr,
	}

	// GDoc specific Transformations
	convertSpan(root)
	convertPre(root)
	convertBlockquote(root)
	fixCodeBlock(root)

	for _, fn := range tx {
		fn(root)
	}
	buf := bytes.Buffer{}
	if err := renderChildren(&buf, root); err != nil {
		return nil, err
	}

	// final fixups
	out := buf.Bytes()
	out = unescapeShortcodes(out)
	out = unescapeEntities(out)

	_, err := w.Write(out)

	return meta, err
}

func parseFragment(src string) (string, map[string]interface{}, error) {
	body := newElementNode("body")
	r := strings.NewReader(src)
	buf := bytes.Buffer{}
	nodes, err := html.ParseFragment(r, body)
	if err != nil {
		return "", nil, err
	}
	for _, n := range nodes {
		body.AppendChild(n)
	}
	meta, err := fromNode(body, &buf)
	if err != nil {
		return "", nil, err
	}
	return buf.String(), meta, nil
}

func ToHTML(r io.Reader, w io.Writer) (map[string]interface{}, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	root = getBody(root)
	return fromNode(root, w)
}
