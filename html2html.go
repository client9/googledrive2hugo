package googledrive2hugo

import (
	"bytes"

	"github.com/gohugoio/hugo/parser"
	"github.com/gohugoio/hugo/parser/metadecoders"
)

// merge B into A
func MetaMerge(a, b map[string]interface{}) map[string]interface{} {
	for k, v := range b {
		// don't over-write
		if _, ok := a[k]; !ok {
			a[k] = v
		}
	}
	return a
}

// HugoContent takes front-matter data, content and writes it
// to output stream
//
func HugoContentWrite(content []byte, metamap map[string]interface{}) ([]byte, error) {
	w := &bytes.Buffer{}
	// TODO: '-' produces yaml, '+' toml, '{' JSON
	//  should make a flag
	if err := parser.InterfaceToFrontMatter(metamap, metadecoders.YAML, w); err != nil {
		return nil, err
	}
	if _, err := w.Write(content); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
