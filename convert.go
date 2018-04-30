package googledrive2hugo

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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

// if you already have a google doc node
func fromNode(root *html.Node) ([]byte, map[string]interface{}, error) {

	// hugo specific
	meta, err := HugoFrontMatter(root)
	if err != nil {
		return nil, nil, err
	}

	// generic transforms
	tx := []func(*html.Node){
		// gdoc specific
		GdocSpan,
		GdocBlockquotePre,
		GdocBlockquote,
		GdocCodeBlock,
		GdocTable,
		GdocAttr,

		// more generic
		RemoveEmptyTags,
		UnsmartCode,
		AddClassAttr,
	}
	for _, fn := range tx {
		fn(root)
	}
	// Render into buffer
	buf := bytes.Buffer{}
	if err := renderChildren(&buf, root); err != nil {
		return nil, nil, err
	}

	// final hugo fixups.. needed to be done outside of tree
	out := buf.Bytes()
	out = unescapeShortcodes(out)
	out = unescapeEntities(out)
	out = bytes.TrimSpace(out)
	return out, meta, nil
}

func parseFragment(src string) (string, error) {
	body := newElementNode("body")
	r := strings.NewReader(src)
	nodes, err := html.ParseFragment(r, body)
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		body.AppendChild(n)
	}
	content, _, err := fromNode(body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func ToHTML(r io.Reader) ([]byte, map[string]interface{}, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, nil, err
	}
	return fromNode(getBody(root))
}
