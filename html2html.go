package googledrive2hugo

import (
	"io"
)

// Convert Google Doc HTML to Hugo Content HTML
func ConvertHTML(r io.Reader, fileMeta map[string]interface{}, w io.Writer) error {
	var contentMeta map[string]interface{}
	var readerErr error

	rp, wp := io.Pipe()

	go func() {
		// parse html and re-render
		contentMeta, readerErr = ToHTML(r, wp)
		wp.Close()
	}()

	content, meta, err := HugoContentRead(rp)
	if readerErr != nil {
		return readerErr
	}

	if err != nil {
		return err
	}

	MetaMerge(meta, contentMeta)
	MetaMerge(meta, fileMeta)
	return HugoContentWrite(content, meta, w)
}
