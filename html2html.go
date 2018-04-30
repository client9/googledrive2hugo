package googledrive2hugo

import (
	"fmt"
	"io"

	"github.com/gohugoio/hugo/parser"
)

// Convert Google Doc HTML to Hugo Content HTML
func ConvertHTML(r io.Reader, fileMeta map[string]interface{}, w io.Writer) error {

	content, meta, err := ToHTML(r)

	if err != nil {
		return fmt.Errorf("readerErr: %s", err)
	}

	MetaMerge(meta, fileMeta)
	return HugoContentWrite(content, meta, w)
}

// merge B into A
func MetaMerge(a, b map[string]interface{}) {
	for k, v := range b {
		// don't over-write
		if _, ok := a[k]; !ok {
			a[k] = v
		}
	}
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
