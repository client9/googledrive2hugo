package googledrive2hugo

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestTextNodes(t *testing.T) {
	cases := []string{
		"<p><b>1</b><b>2</b><b>3</b></p>",
		"<p>1<b>2</b>3</p>",
		"<p>1<b><i>2</i></b><b>3</b></p>",
	}
	body := newElementNode("body")
	for i, tt := range cases {
		r := strings.NewReader(tt)
		tree, err := html.ParseFragment(r, body)
		if err != nil {
			t.Fatalf("unable to parse %q", tt)
		}
		nodes := getTextNodes(tree[0])
		if len(nodes) != 3 {
			t.Fatalf("case %d: %q expected 3 nodes, got %d", i, tt, len(nodes))
		}
		if nodes[0].Data != "1" {
			t.Errorf("case %d: %q expected %q, got %q", i, tt, "1", nodes[0].Data)
		}
		if nodes[1].Data != "2" {
			t.Errorf("case %d: %q expected %q, got %q", i, tt, "2", nodes[1].Data)
		}
		if nodes[2].Data != "3" {
			t.Errorf("case %d: %q expected %q, got %q", i, tt, "3", nodes[2].Data)
		}
	}
}
