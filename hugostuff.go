package googledrive2hugo

import (
	"fmt"
	"io"

	"github.com/gohugoio/hugo/parser"
)

// merge B into A
func MetaMerge(a, b map[string]interface{}) {
	for k, v := range b {
		// don't over-write
		if _, ok := a[k]; !ok {
			a[k] = v
		}
	}
}

// Reads content stream and returns content and metadata
func HugoContentRead(r io.Reader) ([]byte, map[string]interface{}, error) {
	page, err := parser.ReadFrom(r)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read content: %s", err)
	}
	content := page.Content()    // []bytes
	meta, err := page.Metadata() // interface{} :-|
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse metadata: %s", err)
	}

	// no metadata found, and not an error
	if meta == nil {
		return content, make(map[string]interface{}), nil
	}

	// unclear why page.Metadata() returns an interface{} but it does
	// convert
	metamap, ok := meta.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("unable to convert Hugo metadata: %v", meta)
	}

	// all ok
	return content, metamap, nil
}

// HugoContent takes front-matter data, content and writes it
// to output stream
//
func HugoContentWrite(content []byte, metamap map[string]interface{}, w io.Writer) error {
	// TODO: '-' produces yaml, '+' toml, '{' JSON
	//  should make a flag
	if err := parser.InterfaceToFrontMatter(metamap, '-', w); err != nil {
		return err
	}
	//out := gohtml.NewWriter(w)
	if _, err := w.Write(content); err != nil {
		return err
	}
	return nil
}
