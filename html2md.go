package googledrive2hugo

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gohugoio/hugo/parser"
	"golang.org/x/net/html"
	"google.golang.org/api/drive/v3"
)

// TODO: have list of alternate monospace fonts
func isCode(s string) bool {
	if strings.Contains(s, "font-family:\"Consolas\"") {
		return true
	}

	return false
}

func isBold(s string) bool {
	return strings.Contains(s, "font-weight:700")
}

func isItalic(s string) bool {
	return strings.Contains(s, "font-style:italic")
}

func isBlockQuote(s string) bool {
	return strings.Contains(s, "margin-left:36pt")
}

// getHref returns the href attribute
func getHref(z *html.Tokenizer) []byte {
	for {
		key, value, more := z.TagAttr()
		if "href" == string(key) {
			return value
		}
		if !more {
			break
		}
	}
	return nil

}

// getAttr scans the HTML attributes and returns the class and style
func getAttr(z *html.Tokenizer) (string, string) {
	var cname string
	var style string
	for {
		key, value, more := z.TagAttr()
		switch string(key) {
		case "style":
			style = string(value)
		case "class":
			cname = string(value)
		}
		if !more {
			break
		}
	}
	return cname, style

}

// extractHref - extracts the "q" parameter from a full URL with query string.
// TODO: using Index functions instead of string convert/url parsing/byte convert
func extractHref(src []byte) []byte {
	aurl, err := url.Parse(string(src))
	if err != nil {
		return nil
	}
	return []byte(aurl.Query().Get("q"))
}

// cleanupText fixes up various text issues
// * the '--' in <!-- more --> can get converted to a smart-hyphen character
func cleanupText(src []byte) []byte {

	// there is only one <!-- more --> per file
	// TODO: use RegExp to simplify
	src = bytes.Replace(src, []byte("<!—more—>"), []byte("<!-- more -->"), 1)
	src = bytes.Replace(src, []byte("<!—more —>"), []byte("<!-- more -->"), 1)
	src = bytes.Replace(src, []byte("<!— more—>"), []byte("<!-- more -->"), 1)
	src = bytes.Replace(src, []byte("<!— more —>"), []byte("<!-- more -->"), 1)

	// other fix ups here.

	return src
}

func parse(src io.Reader, out io.Writer) error {
	z := html.NewTokenizer(src)
	bold := false
	italic := false
	spanCode := false
	inTable := false
	inTitle := false
	inSubtitle := false
	inStyle := false

	rowCount := 0
	cellCount := 0

	var href []byte
	depth := 0
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			err := z.Err()
			if err == io.EOF {
				return nil
			}
			return err
		case html.TextToken:
			// don't print out style sheet
			if inStyle {
				inStyle = false
				continue
			}
			if inTitle {
				continue
			}
			if inSubtitle {
				continue
			}

			// emitBytes should copy the []byte it receives,
			// if it doesn't process it immediately.
			out.Write(cleanupText(z.Text()))
		case html.StartTagToken:
			tn, _ := z.TagName()
			tns := string(tn)
			switch tns {
			case "span":
				_, style := getAttr(z)
				bold = isBold(style)
				italic = isItalic(style)
				spanCode = isCode(style)
				if bold {
					out.Write([]byte{'*', '*'})
				}
				if italic {
					out.Write([]byte{'_'})
				}
				if spanCode {
					out.Write([]byte{'`'})
				}
			case "p":
				cname, style := getAttr(z)
				if cname == "title" {
					inTitle = true
					continue
				}
				if cname == "subtitle" {
					inSubtitle = true
					continue
				}
				if isBlockQuote(style) {
					out.Write([]byte{'>', ' '})
				}
			case "a":
				href = extractHref(getHref(z))
				if len(href) > 0 {
					out.Write([]byte{'['})
				}
			case "h1":
				out.Write([]byte{'\n', '#', ' '})
			case "h2":
				out.Write([]byte{'\n', '#', '#', ' '})
			case "h3":
				out.Write([]byte{'\n', '#', '#', '#', ' '})
			case "style":
				// don't print out internal style sheet
				inStyle = true
			case "ul":
				depth++
				//out.Write([]byte{'\n'})
			case "li":
				for i := 1; i < depth; i++ {
					out.Write([]byte{' '})
				}
				out.Write([]byte{'*', ' '})
			case "hr":
				out.Write([]byte{'-', '-', '-', '\n'})
			case "table":
				inTable = true
				cellCount = 0
				rowCount = 0
			case "tr":
				rowCount++
				out.Write([]byte{'|'})
			case "td":
				cellCount++
				out.Write([]byte{' '})
			}
		case html.EndTagToken:
			tn, _ := z.TagName()
			tns := string(tn)
			switch tns {
			case "span":
				if spanCode {
					out.Write([]byte{'`'})
					spanCode = false
				}
				if italic {
					out.Write([]byte{'_'})
					italic = false
				}
				if bold {
					out.Write([]byte{'*', '*'})
					bold = false
				}
			case "ul":
				depth--
				//out.Write([]byte{'\n'})
			case "li":
				out.Write([]byte{'\n'})
			case "h1", "h2", "h3", "h4", "h5", "h6":
				out.Write([]byte{'\n', '\n'})
			case "p":
				if !inTable && !inTitle && !inSubtitle {
					out.Write([]byte{'\n'})
				}
				inTitle = false
				inSubtitle = false
			case "a":
				if len(href) > 0 {
					out.Write([]byte{']', '('})
					out.Write(href)
					out.Write([]byte{')'})
					href = nil
				}
			case "style":
				inStyle = false
			case "td":
				out.Write([]byte{' ', '|'})
			case "tr":
				out.Write([]byte{'\n'})
				// make an arbitrary header row
				if rowCount == 1 {
					for i := 0; i < cellCount; i++ {
						out.Write([]byte{'|', '-', '-', '-', '-'})
					}
					out.Write([]byte{'|', '\n'})
				}
			case "table":
				inTable = false
			}
		}
	}
}

// fixBlocks converts sequential independent lines of code into a true
// code block.
//
// before:
// `line 1`
// `line 2`
//
// after:
// ```
// line 1
// line 2
// ```
//
func fixBlocks(src []byte, w io.Writer) error {
	lines := bytes.Split(src, []byte{'\n'})
	inCode := false
	codefence := false
	for _, line := range lines {
		codeline := false
		if len(line) >= 2 && line[0] == '`' && line[len(line)-1] == '`' {
			codeline = true
			if bytes.HasPrefix(line, []byte{'`', '`', '`', '`'}) {
				codefence = true // got "```go" or something
			}
		}

		// in code and got ``` --> end of fence, end of code
		if inCode && codefence {
			w.Write([]byte{'`', '`', '`', '\n'})
			inCode = false
			codefence = false
			codeline = false
			continue
		}

		// had a block of code, and then not --> end of code
		if inCode && !codeline {
			w.Write([]byte{'`', '`', '`', '\n'})
			inCode = false
			continue
		}

		if inCode && codeline {
			w.Write(line[1 : len(line)-1])
			w.Write([]byte{'\n'})
			continue
		}

		// stray "``"
		if !inCode && codeline && len(line) == 2 {
			codeline = false
			w.Write([]byte{'\n'})
			continue
		}

		if !inCode && codeline {
			if !codefence {
				w.Write([]byte{'`', '`', '`', '\n'})
			}
			w.Write(line[1 : len(line)-1])
			w.Write([]byte{'\n'})
			inCode = true
			codefence = false
			codeline = false
			continue
		}

		w.Write(line)
		w.Write([]byte{'\n'})
	}

	return nil
}

// Convert Google Doc HTML to Hugo Markdown
func Convert(src []byte, fileInfo *drive.File, w io.Writer) error {
	// will contain gDrive meta/ Hugo front matter
	metamap := make(map[string]interface{})

	r := bytes.NewReader(src)

	// produce a basic markdown output
	// it should have built-in front matter, and mostly correct markdown
	phase1 := bytes.Buffer{}
	err := parse(r, &phase1)
	if err != nil {
		return err
	}

	// Gross!  Do in one pass with a pipe
	phase2 := bytes.NewReader(phase1.Bytes())
	// phase2 augments the frontmatter, and does fixups for mutli-line code blocks and blockquotes
	page, err := parser.ReadFrom(phase2)
	if err != nil {
		return err
	}
	content := page.Content()    // []bytes
	meta, err := page.Metadata() // interface{} :-|
	if err != nil {
		// Due to bad format of metadata?
		return fmt.Errorf("Metadata format: %s", err)
	}

	// case err == nil && meta == nil: front matter doesn't exist
	if meta != nil {
		var ok bool
		if metamap, ok = meta.(map[string]interface{}); !ok {
			return fmt.Errorf("Unable to convert Hugo metadata", meta)
		}
	}

	// Google File Metadata:
	// https://godoc.org/google.golang.org/api/drive/v3#File
	//
	// Note: description can only be set via gDrive, not gDocs
	//
	// Hugo Front Matter:
	// https://gohugo.io/content-management/front-matter/
	//
	metamap["date"] = fileInfo.CreatedTime
	metamap["lastmod"] = fileInfo.ModifiedTime
	if fileInfo.Description != "" {
		metamap["description"] = fileInfo.Description
	}

	// TODO: '-' produces yaml, '+' toml, '{' JSON
	//  should make a flag
	err = parser.InterfaceToFrontMatter(metamap, '-', w)
	if err != nil {
		return err
	}

	// do fix up of code blocks line by line
	return fixBlocks(content, w)
	//_, err = w.Write(content)
}
