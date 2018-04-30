package googledrive2hugo

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// this tests what google's HTML5 parser does with empty tags
// in particular with <span></span>
// Conclusion: parses and renders them
func TestEmptyTagParsing(t *testing.T) {
	context := &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	}
	doc := "<p><span></span></p>"
	// list of nodes
	nodes, err := html.ParseFragment(strings.NewReader(doc), context)
	if err != nil {
		t.Fatalf("unable to parse: %s", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected a single node root: got %d", len(nodes))
	}
	root := nodes[0]
	first := root.FirstChild
	if first == nil {
		t.Fatalf("expected a child!")
	}
	if first.Type != html.ElementNode && first.DataAtom != atom.Span {
		t.Fatalf("got a non-span child")
	}

	// ok it looks like go parses empty span tags.
	// but how does it render?
	outbuf := bytes.Buffer{}
	html.Render(&outbuf, root)
	out := outbuf.String()

	if out != doc {
		t.Errorf("want %s got %s", doc, out)
	}
}

func TestIsTextWrapper(t *testing.T) {

	// <span></span> should be a text wrapper even if empty

	span := &html.Node{
		Type:     html.ElementNode,
		Data:     "span",
		DataAtom: atom.Span,
	}
	if !isTextWrapper(span) {
		t.Errorf("empty span was not detected as text")
	}
}

func TestGetTextChildren(t *testing.T) {
	span := &html.Node{
		Type:     html.ElementNode,
		Data:     "span",
		DataAtom: atom.Span,
	}
	text := getTextContent(span)
	if text != " " {
		t.Errorf("expected empty span to be convert to a space")
	}
}

func TestXXX(t *testing.T) {
	doc := `<p style=""><span style=""><a href="something" style="">hello</a></span><span></span><span>world</span></p>`
	want := `<p><a href="something">hello</a> world</p>`

	got, err := parseFragment(doc)
	if err != nil {
		t.Fatalf("unable to parse %s", err)
	}

	// hack on fragment parser
	got = strings.TrimSpace(got)
	if got != want {
		t.Errorf("Got %s vs %s", got, want)
	}
}
