package googledrive2hugo

import (
	"log"
	"fmt"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/gohugoio/hugo/parser/metadecoders"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	selectorTitle    = cascadia.MustCompile("p[class~=title]")
	selectorSubtitle = cascadia.MustCompile("p[class~=subtitle]")
)

func HugoFrontMatter(root *html.Node) (map[string]interface{}, error) {
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
		return make(map[string]interface{}), nil
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
		return make(map[string]interface{}), nil
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

	imeta, err := metadecoders.Unmarshal([]byte(front), metadecoders.YAML)
	if err != nil {
		log.Printf("FRONT:\n%s", front)
		return nil, err
	}
	meta, ok := imeta.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("hugo config problem, got unknown type back")
	}
	if title := extractTitle(root); title != "" {
		if _, ok := meta["title"]; !ok {
			meta["title"] = title
		}
	}

	if desc := extractSubtitle(root); desc != "" {
		if _, ok := meta["description"]; !ok {
			meta["description"] = desc
		}
	}
	return meta, nil
}

func extractTitle(root *html.Node) string {
	n := selectorTitle.MatchFirst(root)
	if n == nil {
		return ""
	}
	val := getTextContent(n)
	n.Parent.RemoveChild(n)
	return val
}
func extractSubtitle(root *html.Node) string {
	n := selectorSubtitle.MatchFirst(root)
	if n == nil {
		return ""
	}
	val := getTextContent(n)
	n.Parent.RemoveChild(n)
	return val
}
