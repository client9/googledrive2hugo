package googledrive2hugo

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	defaultSelectorNarrowTag = "h2,p a[href],p code,p i,p b,p strong,p em,p span"
	defaultSelectorPunc      = "p"
)

// <a>...</a> should never end in these
var badAnchorEndings = []string{
	".",
	",",
	"!",
	"?",
	":",
	";",
}

func getParentBlock(node *html.Node) *html.Node {
	for {
		switch node.DataAtom {
		case atom.P, atom.Div, atom.Li, atom.Th, atom.Td:
			return node
		}
		if node.Parent == nil {
			return node
		}
		node = node.Parent
	}

	// should never happen
	return nil
}

type Runner interface {
	Run(root *html.Node, log ilog.Logger) error
}

type NarrowTag struct {
	selector cascadia.Selector
}

func isBlank(nodes []*html.Node) bool {
	for _, n := range nodes {
		if strings.TrimSpace(n.Data) != "" {
			return false
		}
	}
	return true
}
func (n *NarrowTag) Init() (err error) {
	n.selector, err = cascadia.Compile(defaultSelectorNarrowTag)
	return err
}

func (n *NarrowTag) Run(root *html.Node, log ilog.Logger) (err error) {

	// get tags that shouldn't have leading or trailing whitespace
	for _, p := range n.selector.MatchAll(root) {

		// get textnodes inside these tags
		nodes := getTextNodes(p)

		// remove weird empty text nodes (rarely happens if at all)
		if isBlank(nodes) {
			log.Debug("blank node", "tag", p.Data)
			prev := getPrevTextNode(getParentBlock(p), p)
			if prev != nil && !hasSuffixSpace(prev.Data) {
				prev.Data = prev.Data + " "
			}
			p.Parent.RemoveChild(p)
			continue
		}

		// get first text node
		first := nodes[0]
		linked := first.Data
		tmp := trimLeftSpace(linked)

		// if has leading spaces, remove and possibly add space to previous node
		if linked != tmp {
			first.Data = tmp
			log.Debug("trim left", "tag", p.Data, "text", getTextContent(p))
			prev := getPrevTextNode(getParentBlock(first), first)
			if prev != nil && !hasSuffixSpace(prev.Data) {
				prev.Data = prev.Data + " "
			}
		}

		// get last text node
		last := nodes[len(nodes)-1]
		linked = last.Data
		tmp = trimRightSpace(linked)

		// if has trailing white space, remove and possibly add space to next node
		if linked != tmp {
			log.Debug("trim right", "tag", p.Data, "text", getTextContent(p))
			last.Data = tmp
			next := getNextTextNode(getParentBlock(last), last)
			if next == nil {
				log.Debug("no new text node!!")
			}
			if next != nil && !hasPrefixSpace(next.Data) {
				// add a space to next text node
				next.Data = " " + next.Data
			}
		}

		// check to see <a> ends in punctuation
		if p.DataAtom == atom.A {
			linked = last.Data
			for _, bada := range badAnchorEndings {
				if strings.HasSuffix(linked, bada) {
					return fmt.Errorf("tag <%s> %q has ending %q", p.Data, getTextContent(p), bada)
				}
			}
		}
	}

	return nil
}

type Punc struct {
	selector cascadia.Selector
}

func (n *Punc) Init() (err error) {
	n.selector, err = cascadia.Compile(defaultSelectorPunc)
	return err
}

func (n *Punc) Run(root *html.Node, log ilog.Logger) error {
	for _, p := range n.selector.MatchAll(root) {
		nodes := getTextNodes(p)
		if len(nodes) == 0 {
			continue
		}
		first := nodes[0]
		tmp := first.Data
		//log.Debug("first text node", "text", tmp)
		first.Data = trimLeftSpace(tmp)
		if tmp != first.Data {
			log.Debug("deleting leading whitespace from <p>")
		}

		// checking ending
		last := nodes[len(nodes)-1]
		if err := pEnding(last, log); err != nil {
			body := getTextContent(p)
			return fmt.Errorf("Error in <p> %q: %s", body, err)
		}
	}
	return nil
}

func pEnding(root *html.Node, log ilog.Logger) error {
	if root.Type != html.TextNode {
		panic("expected textnode")
	}
	if len(root.Data) == 0 {
		return fmt.Errorf("Weird: paragraph ended in empty text node")
	}
	/*
		if root.Parent != nil && (root.Parent.DataAtom != atom.P && root.Parent.DataAtom != atom.A) {
			return fmt.Errorf("Last Paragraph text node's parent is not <p>, got <%s>", root.Parent.Data)
		}
	*/
	tmp := trimRightSpace(root.Data)
	if tmp != root.Data {
		if len(tmp) == 0 {
			log.Debug("Weird: paragraph ended in whitespace only text node")
			return nil
		}
		log.Debug("deleting trailing whitespace from <p>")
		root.Data = tmp
	}
	chars := []rune(root.Data)
	last1 := chars[len(chars)-1]
	if !isEndOrShortCode(last1) && !isQuote(last1) {
		return fmt.Errorf("does not end with any punctuation, got %U", last1)
	}
	if len(root.Data) > 1 {
		last2 := chars[len(chars)-2]

		//   foo".   should be foo."
		if isQuote(last2) && isEndOrShortCode(last1) {
			return fmt.Errorf("punctuation %U is outside quote %U", last1, last2)
		}

		// foo"  should be foo."
		if !isEndOrShortCode(last2) && isQuote(last1) {
			return fmt.Errorf("ending quote %U is missing inner punctuation, got %U", last1, last2)
		}
	}
	return nil
}

func hasParentAnchor(root *html.Node, current *html.Node) bool {
	for c := current; c != root; c = c.Parent {
		if c.DataAtom == atom.A {
			return true
		}
	}
	return false
}

func isEnd(r rune) bool {
	// u2026 is HORIZONTAL ELLIPSIS
	switch r {
	case '.', '?', '!', ':', '\u2026':
		return true
	}
	return false
}

func isShortCode(r rune) bool {
	return r == '}'
}

func isEndOrShortCode(r rune) bool {
	return isEnd(r) || isShortCode(r)
}

// Should be named 'isEndingQuote'
func isQuote(r rune) bool {
	// u201d is RIGHT DOUBLE QUOTATION MARK
	switch r {
	case '"', '\u201d':
		return true
	}
	return false
}

func hasPrefixSpace(s string) bool {
	return strings.IndexFunc(s, unicode.IsSpace) == 0
}

func hasSuffixSpace(s string) bool {
	return strings.LastIndexFunc(s, unicode.IsSpace) == len(s)-1
}

func trimLeftSpace(s string) string {
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}
func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}
