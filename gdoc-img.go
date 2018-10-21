package googledrive2hugo

import (
	"log"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	selectorImg     = cascadia.MustCompile(`p>span>img`)
	selectorComment = cascadia.MustCompile(`sup>a`)
)

// converts <p><span><img/></span><sup><a></a></sup></p> into a nice <p><img>
// comment is in form of
//  <sup><a href="#cmnt1" id="cmnt_ref1">[a]</a></sup>
func GdocImg(root *html.Node) error {
	for _, img := range selectorImg.MatchAll(root) {
		// remove useless span
		span := img.Parent
		if span.DataAtom != atom.Span {
			log.Printf("PARENT NOT A SPAN")
		}
		p := span.Parent
		if p.DataAtom != atom.P {
			log.Printf("PARENT NOT A P")
		}
		reparentChildren(p, span)

		// any comments?
		for _, sup := range selectorComment.MatchAll(p) {
			id := getHrefAttr(sup)
			if id == "" {
				continue
			}
			// for now just remove it
			anchor := cascadia.MustCompile(id).MatchFirst(root)
			if anchor == nil {
				log.Printf("didnt find %q", id)
			}
			text := getTextContent(anchor.NextSibling)
			setAltAttr(img, text)

			// remove everything related to this comment
			ap := anchor.Parent
			ap.Parent.RemoveChild(ap)

			p.RemoveChild(sup.Parent)
		}
	}

	return nil
}
