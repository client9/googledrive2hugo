package googledrive2hugo

import (
	"log"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func HugoFrontMatter(root *html.Node) {
	var front string
	var fStart *html.Node
	var fEnd *html.Node

	// find opening front matter
	count := 0
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		count++
		if count > 5 {
			break
		}
		if c.DataAtom == atom.P {
			text := strings.TrimSpace(getTextContent(c))
			if text == "" {
				continue
			}
			if text == "---" || text == "{" || text == "+++" {
				front = text + "\n"
				fStart = c
				break
			}
			break
		}
		if c.DataAtom == atom.Hr {
			c.Data = "---"
			c.DataAtom = 0
			c.Type = html.TextNode
			fStart = c
			front = "---\n"
			break
		}
	}

	// no front matter
	if fStart == nil {
		log.Printf("OK - did not find front matter start")
		return
	}

	// find ending
	for c := fStart.NextSibling; c != nil; c = c.NextSibling {
		if c.DataAtom != atom.Hr && c.DataAtom != atom.P {
			break
		}
		if c.DataAtom == atom.Hr {
			front += "---\n"
			fEnd = c
			break
		}
		text := getTextContent(c)
		front += text + "\n"
		if text == "---" || text == "}" || text == "+++" {
			fEnd = c
			break
		}
	}

	// didn't find end
	if fEnd == nil {
		log.Printf("did not find front matter end")
		return
	}

	// delete all the nodes up to and including fEnd
	c := root.FirstChild
	for {
		next := c.NextSibling
		root.RemoveChild(c)
		if c == fEnd {
			break
		}
		c = next
	}

	// remove any special typography that might have been used
	// the front matter is code!
	front = unsmart(front)

	// insert front matter as first element
	root.InsertBefore(newTextNode(front), root.FirstChild)
}
