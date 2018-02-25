package googledrive2hugo

import (
	"bytes"
	"io"
	"log"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func renderChildren(w io.Writer, root *html.Node) error {
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(w, c); err != nil {
			return err
		}
	}
	return nil
}

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
func isStyleCode(s string) bool {
	// other monospace fonts TBD
	return strings.Contains(s, "Consolas")
}

func getStyleAttr(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "style" {
			return attr.Val
		}
	}
	return ""
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

// remove empty Spans
func stripAttr(n *html.Node) {
	removeStyleAttr(n)
	//	n.Attr = nil
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		stripAttr(c)
	}
}

func reparentChildren(newParent, oldParent *html.Node) {
	for c := oldParent.FirstChild; c != nil; c = oldParent.FirstChild {
		oldParent.RemoveChild(c)
		newParent.AppendChild(c)
	}
}

// Reparent <code> children
func reparentCodeChildren(newParent, oldParent *html.Node) {
	for c := oldParent.FirstChild; c != nil; c = oldParent.FirstChild {
		if c.DataAtom == atom.Code {
			reparentCodeChildren(newParent, c)
			oldParent.RemoveChild(c)
			continue
		}
		oldParent.RemoveChild(c)
		newParent.AppendChild(c)
	}
}

// recursive
func getTextContent(n *html.Node) string {
	out := ""
	if n.Type == html.TextNode {
		return n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out += getTextContent(c)
	}
	return out
}

// non-recursive
func getTextChildren(n *html.Node) string {
	out := ""
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			out += c.Data
		}
	}
	return out
}

func isCodeWrapper(n *html.Node) (bool, string) {
	if n.Type != html.ElementNode || n.DataAtom != atom.P {
		return false, ""
	}

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

func isTextWrapper(n *html.Node) bool {
	if n.Type != html.ElementNode || n.DataAtom != atom.Span {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.TextNode {
			return false
		}
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

func removeAllChildren(n *html.Node) {
	c := n.FirstChild
	for c != nil {
		n.RemoveChild(c)
		c = n.FirstChild
	}
}
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
			pre.AppendChild(&html.Node{
				Type: html.TextNode,
				Data: "\n",
			})
			n.RemoveChild(c)
			reparentCodeChildren(pre, c)
			c = next
			continue
		}

		// we have <p><code>.  Create new <pre><code> block
		bq := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Blockquote,
			Data:     "blockquote",
		}
		pre = &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Pre,
			Data:     "pre",
		}
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

			bq.AppendChild(&html.Node{
				Type:     html.ElementNode,
				DataAtom: atom.Br,
				Data:     "br",
			})
			reparentChildren(bq, c)
			c = next
			continue
		}

		// we have <p margin-left:36pt>
		//  create new blockquote
		bq = &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Blockquote,
			Data:     "blockquote",
		}
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
	if isTextWrapper(n) {
		text := getTextChildren(n)
		if text == "" {
			n.Parent.RemoveChild(n)
			return next
		}
		style := getStyleAttr(n)
		newNode := &html.Node{
			Type: html.TextNode,
			Data: text,
		}
		if isStyleItalics(style) {
			wrapper := &html.Node{
				Type:     html.ElementNode,
				DataAtom: atom.Em,
				Data:     "em",
			}
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleBold(style) {
			wrapper := &html.Node{
				Type:     html.ElementNode,
				DataAtom: atom.Strong,
				Data:     "strong",
			}
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleUnderline(style) {
			wrapper := &html.Node{
				Type:     html.ElementNode,
				DataAtom: atom.U,
				Data:     "u",
			}
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleStrikethrough(style) {
			wrapper := &html.Node{
				Type:     html.ElementNode,
				DataAtom: atom.Del,
				Data:     "del",
			}
			wrapper.AppendChild(newNode)
			newNode = wrapper
		}
		if isStyleCode(style) {
			wrapper := &html.Node{
				Type:     html.ElementNode,
				DataAtom: atom.Code,
				Data:     "code",
			}
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

func fixTableNode(table *html.Node) {
	tbody := table.FirstChild
	if tbody == nil || tbody.DataAtom != atom.Tbody {
		return
	}
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
		td.AppendChild(&html.Node{
			Type: html.TextNode,
			Data: text,
		})
	}

	// move tr from tbody to new thead
	thead := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Thead,
		Data:     "thead",
	}
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
			td.AppendChild(&html.Node{
				Type: html.TextNode,
				Data: text,
			})
		}
	}
}

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

		// we have a <p><code> and we have an existing code block
		if textNode != nil {
			n.RemoveChild(c)
			textNode.Data += code + "\n"
			c = next
			continue
		}

		// we have <p><code>.  Create new <pre><code> block
		textNode = &html.Node{
			Type: html.TextNode,
			Data: code + "\n",
		}
		codeNode := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Code,
			Data:     "code",
		}
		preNode := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Pre,
			Data:     "pre",
		}
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

func getBody1(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == atom.Body {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if body := getBody1(c); body != nil {
			return body
		}
	}
	return nil
}

func getBody(root *html.Node) *html.Node {
	body := getBody1(root)
	if body == nil {
		return root
	}
	return body
}

func getClassAttr(root *html.Node) string {
	for _, attr := range root.Attr {
		if attr.Key == "class" {
			return attr.Val
		}
	}
	return ""
}

func isTitleP(n *html.Node) bool {
	return n.Type == html.ElementNode &&
		n.DataAtom == atom.P &&
		strings.Contains(getClassAttr(n), "title")
}

func extractTitle(n *html.Node) string {
	if isTitleP(n) {
		val := getTextContent(n)
		n.Parent.RemoveChild(n)
		return val
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}
	return ""
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
			text := strings.TrimSpace(getTextChildren(c))
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
		log.Printf("Did not find front matter start")
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
		text := getTextChildren(c)
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

	// insert front matter as first element
	root.InsertBefore(&html.Node{
		Type: html.TextNode,
		Data: front,
	}, root.FirstChild)
}

var xxx = map[string]map[string]string{
	"table": {
		"class": "table table-sm",
	},
	"thead": {
		"class": "thead-light",
	},
	"blockquote": {
		"class": "blockquote border-left border-left-thick pl-3",
	},
	"pre": {
		"class": "p-1 bg-light border",
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

func unescape(buf []byte) []byte {
	return []byte(html.UnescapeString(string(buf)))
}

// unescapeShortcode:
// From:
//  {{&lt; instgram &#34;8203823&#34; &lt;}}
//
// To:
//  {{< instagram "8203823" >}}
func unescapeShortcodes(buf []byte, w io.Writer) error {
	re := regexp.MustCompile(`{{&lt;.*&gt;}}`)
	_, err := w.Write(re.ReplaceAllFunc(buf, unescape))
	return err
}

func ToHTML(r io.Reader, w io.Writer) (map[string]interface{}, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	root = getBody(root)
	meta := make(map[string]interface{})
	if title := extractTitle(root); title != "" {
		meta["title"] = title
	}

	convertSpan(root)
	convertPre(root)
	convertBlockquote(root)
	fixCodeBlock(root)
	fixTables(root)
	stripAttr(root)
	removeEmptyTags(root)
	createFrontMatter(root)

	fixAttr(root)

	buf := bytes.Buffer{}
	if err = renderChildren(&buf, root); err != nil {
		return nil, err
	}
	err = unescapeShortcodes(buf.Bytes(), w)
	return meta, err
}
