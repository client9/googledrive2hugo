package googledrive2hugo

import (
	"strings"
	"testing"

	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

func TestPunc(t *testing.T) {
	cases := []string{
		"<p>foo</p>",         // no ending punc
		"<p><b>foo.</b></p>", // ending punc not plain text
		"<p>foo\u201d.</p>",  // ending punc outside quote
		"<p>foo\"</p>",       // ending punc outside quote
		"<p>(foo.)</p>",      // ending punc inside parens
	}
	body := newElementNode("body")
	for _, tt := range cases {
		r := strings.NewReader(tt)
		nodes, err := html.ParseFragment(r, body)
		if err != nil {
			t.Fatalf("unable to parse %q", tt)
		}
		p := Punc{}
		err = p.Run(nodes[0], &ilog.NopLogger{})
		if err == nil {
			t.Errorf("expected an error with %q", tt)
		}
	}
}