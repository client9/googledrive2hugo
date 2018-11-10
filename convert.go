package googledrive2hugo

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

// remove non-breaking spaces.  Unclear why google adds them or how
// they get added.
func removeNbsp(src string) string {
	return strings.Replace(src, "\u00a0", " ", -1)
}

type Converter struct {
	Logger ilog.Logger
}

func (c *Converter) ToHTML(src []byte, fileMeta map[string]interface{}) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}

	content, meta, err := c.FromNode(getBody(root))
	if err != nil {
		return nil, err
	}
	return HugoContentWrite(content, MetaMerge(meta, fileMeta))
}

func (c *Converter) parseFragment(src string) (string, error) {
	body := newElementNode("body")
	nodes, err := html.ParseFragment(strings.NewReader(src), body)
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		body.AppendChild(n)
	}
	content, _, err := c.FromNode(body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// if you already have a google doc node
func (c *Converter) FromNode(root *html.Node) ([]byte, map[string]interface{}, error) {
	// hugo specific
	meta, err := HugoFrontMatter(root)
	if err != nil {
		return nil, nil, err
	}

	// generic transforms
	tx := []func(*html.Node) error{
		// gdoc specific
		GdocImg,
		GdocSpan,
		GdocBlockquotePre,
		GdocBlockquote,
		GdocCodeBlock,
		GdocTable,
		GdocAttr,
	}

	for _, fn := range tx {
		if err := fn(root); err != nil {
			return nil, nil, err
		}
	}

	tx2 := []Runner{
		&AddClassAttr{
			ClassMap: map[string]string{
				"table":        "table table-sm",
				"blockquote":   "pl-3 lines-dense",
				"pre":          "p-1 pl-3 lines-dense",
				"h1":           "h2 mb-3", // no top margin
				"h2":           "h4 mt-4 mb-4",
				"h3":           "h5 mt-4 mb-4",
				"img":          "img-fluid",
				"div:has(img)": "container pl-0",
			},
		},
		&LinkRelative{
			Pattern: "https://www.client9.com",
		},
		&LinkInsecure{
			Whitelist: []string{
				"www.lafite.com",
				"www.donki.com",
				"www.nakano-group.co.jp",
				"www.e-shouchu.com",
				"www.satasouji-shouten.co.jp",
				"www.nakano-group.co.jp",
				"ogp.me",
				"www.elliotdahl.com",
				"z12t.com",
				"markdotto.com",
				"montserrat.zkysky.com.ar",
			},
		},
		&RemoveEmptyTag{},
		&UnsmartCode{},
		&NarrowTag{},
		&Punc{},
	}
	for _, fn := range tx2 {

		// get name of function
		fname := fmt.Sprintf("%T", fn)
		if idx := strings.LastIndexByte(fname, '.'); idx != -1 {
			fname = fname[idx+1:]
		}

		mlog := c.Logger.With("fn", fname)
		if err := fn.Run(root, mlog); err != nil {
			return nil, nil, err
		}
	}
	// Render into buffer
	buf := bytes.Buffer{}
	if err := renderChildren(&buf, root); err != nil {
		return nil, nil, err
	}
	out := buf.Bytes()

	// final hugo fixups.. needed to be done outside of tree
	out = unescapeShortcodes(out)
	out = unescapeEntities(out)
	out = bytes.TrimSpace(out)
	return out, meta, nil
}
