package googledrive2hugo

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

// remove non-breaking spaces.  Unclear why google adds them or how
// they get added.
func removeNbsp(src string) string {
	return strings.Replace(src, "\u00a0", " ", -1)
}

type Converter struct {
	Logger  ilog.Logger
	Filters []Runner
}

func (c *Converter) ToHTML(src []byte, fileMeta map[string]interface{}) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}

	content, textMeta, err := c.FromNode(getBody(root))
	if err != nil {
		return nil, err
	}

	meta := MetaMerge(textMeta, fileMeta)

	// generate some extra tags for rollup or archives
	date, ok := meta["date"].(time.Time)
	if !ok {
		return nil, fmt.Errorf("unable to get document date metadata")
	}
	meta["date-year"] = fmt.Sprintf("%d", date.Year())
	meta["date-month"] = fmt.Sprintf("%d/%02d", date.Year(), date.Month())
	meta["date-day"] = fmt.Sprintf("%d/%02d/%02d", date.Year(), date.Month(), date.Day())

	return HugoContentWrite(content, meta)
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

	for _, fn := range c.Filters {

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
