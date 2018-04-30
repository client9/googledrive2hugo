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
		GdocSpan,
		GdocBlockquotePre,
		GdocBlockquote,
		GdocCodeBlock,
		GdocTable,
		GdocAttr,

		// more generic
		RemoveEmptyTags,
		HugoFrontMatter,
		UnsmartCode,
		AddClassAttr,
	}

	for _, fn := range tx {
		fn(root)
	}

	// Render into buffer
	buf := bytes.Buffer{}
	if err := renderChildren(&buf, root); err != nil {
		return nil, err
	}

	// final fixups.. needed to be done outside of tree
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
