package googledrive2hugo

import (
	"bytes"
	"io"
	"log"
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

func removeStyleAttr(n *html.Node) {
	if n.Attr == nil {
		return
	}
	idx := 0
	for i := 0; i < len(n.Attr); i++ {
		switch n.Attr[i].Key {
		case "href":
			// needed for <a> and others
			n.Attr[idx] = n.Attr[i]
			idx++
		case "colspan", "rowspan":
			// gdoc does a lot of <td rowspan=1 colspan=1
			// which is not needed
			if n.Attr[i].Val != "1" {
				n.Attr[idx] = n.Attr[i]
				idx++
			}
		case "id":
			// preserver ID is headings.  Used for internal linking
			// hack for h1..h6.  includes hr too but
			//  thats ok
			if len(n.Data) == 2 && n.Data[0] == 'h' {
				n.Attr[idx] = n.Attr[i]
				idx++
			}
		default:
			continue
		}
	}
	// remove any junk at end
	n.Attr = n.Attr[:idx]
}

// remove all attributes
func stripAttr(n *html.Node) {
	removeStyleAttr(n)
	//	n.Attr = nil
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		stripAttr(c)
	}
}

// Reparent <code> children
func reparentCodeChildren(newParent, oldParent *html.Node) {
	nodes := selectorCode.MatchAll(oldParent)
	for _, n := range nodes {
		reparentChildren(newParent, n)
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

// isEmpty returns true if <p></p> or <a></a> is found
// <p></p> is used a typically unintended line spaces
// <a></a> is in docs for unknown reasons
//
func isEmpty(n *html.Node) bool {
	if n.Type != html.ElementNode || n.FirstChild != nil {
		return false
	}

	switch n.DataAtom {
	case atom.P, atom.A:
		return true
	}
	return false
}

func removeEmptyTags(n *html.Node) *html.Node {
	next := n.NextSibling
	if isEmpty(n) {
		n.Parent.RemoveChild(n)
		return next
	}
	c := n.FirstChild
	for c != nil {
		c = removeEmptyTags(c)
	}
	return next
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

		// we have a <p><code> and we have an existing code block
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

func hasBoldChildren(n *html.Node) bool {
	if n.DataAtom == atom.B || n.DataAtom == atom.Strong {
		return true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if hasBoldChildren(c) {
			return true
		}
	}
	return false
}

// looks at first row to see if it is a header row, and if so
// move it to a new thead, and change <td> to <th>
func fixTableNode(table *html.Node) {
	tbody := table.FirstChild

	// probably should do a warning here.
	// this table isn't what we expected.
	if tbody == nil || tbody.DataAtom != atom.Tbody {
		return
	}

	// we expect the first child to be a <tr>
	//  if it's not, or if none of the subquent <td> are bold
	//  then nothing to do.
	tr := tbody.FirstChild
	if tr == nil || tr.DataAtom != atom.Tr || !hasBoldChildren(tr) {
		return
	}

	// convert TD to TH
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		text := getTextContent(td)
		td.Data = "th"
		td.DataAtom = atom.Th
		removeAllChildren(td)
		td.AppendChild(newTextNode(text))
	}

	// move tr from tbody to new thead
	thead := newElementNode("thead")
	tbody.RemoveChild(tr)
	thead.AppendChild(tr)
	table.InsertBefore(thead, tbody)

	// DO FOOT

	// get Last TR
	// are TD bold?
	// remove, create TFOOT, add TR

	// how iterate  over remaining rows, checking first entry
	for tr := tbody.FirstChild; tr != nil; tr = tr.NextSibling {
		td := tr.FirstChild
		if td != nil && hasBoldChildren(td) {
			text := getTextContent(td)
			td.DataAtom = atom.Th
			td.Data = "th"
			removeAllChildren(td)
			td.AppendChild(newTextNode(text))
		}
	}
}

// gdoc puts a <p> inside each <td>.  Remove the unnecessary <p> tag.
func fixTableCells(n *html.Node) {
	child := n.FirstChild
	if n.DataAtom == atom.Td && child.DataAtom == atom.P && child.NextSibling == nil {
		n.RemoveChild(child)
		reparentChildren(n, child)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		fixTableCells(c)
	}
}

// may turn first <tr> into a <thead><tr> and turn the <td> into <th>
func fixTables(n *html.Node) {
	if n.DataAtom == atom.Table {
		fixTableNode(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		fixTableNode(c)
	}
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

func createFrontMatter(root *html.Node) {
	var front string
	var fStart *html.Node
	var fEnd *html.Node

	// find opening front matter
	count := 0
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		count++
		if count > 5 {
			break
		}
		if c.DataAtom == atom.P {
			text := strings.TrimSpace(getTextContent(c))
			if text == "" {
				continue
			}
			if text == "---" || text == "{" || text == "+++" {
				front = text + "\n"
				fStart = c
				break
			}
			break
		}
		if c.DataAtom == atom.Hr {
			c.Data = "---"
			c.DataAtom = 0
			c.Type = html.TextNode
			fStart = c
			front = "---\n"
			break
		}
	}

	// no front matter
	if fStart == nil {
		log.Printf("OK - did not find front matter start")
		return
	}

	// find ending
	for c := fStart.NextSibling; c != nil; c = c.NextSibling {
		if c.DataAtom != atom.Hr && c.DataAtom != atom.P {
			break
		}
		if c.DataAtom == atom.Hr {
			front += "---\n"
			fEnd = c
			break
		}
		text := getTextContent(c)
		front += text + "\n"
		if text == "---" || text == "}" || text == "+++" {
			fEnd = c
			break
		}
	}

	// didn't find end
	if fEnd == nil {
		log.Printf("did not find front matter end")
		return
	}

	// delete all the nodes up to and including fEnd
	c := root.FirstChild
	for {
		next := c.NextSibling
		root.RemoveChild(c)
		if c == fEnd {
			break
		}
		c = next
	}

	// remove any special typography that might have been used
	// the front matter is code!
	front = unsmart(front)

	// insert front matter as first element
	root.InsertBefore(newTextNode(front), root.FirstChild)
}

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

func fixAttr(n *html.Node) {
	if override := xxx[n.Data]; override != nil {
		for k, v := range override {
			n.Attr = append(n.Attr, html.Attribute{Key: k, Val: v})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {

		if c.Type == html.ElementNode {
			fixAttr(c)
		}
	}
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

	convertSpan(root)
	convertPre(root)
	convertBlockquote(root)
	fixCodeBlock(root)
	fixTableCells(root)
	fixTables(root)
	stripAttr(root)
	removeEmptyTags(root)
	createFrontMatter(root)
	unsmartCode(root)
	fixAttr(root)
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
