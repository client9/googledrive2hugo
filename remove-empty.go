package googledrive2hugo

import (
	"golang.org/x/net/html"

	"github.com/andybalholm/cascadia"
)

var (
	selectorEmpty = cascadia.MustCompile("a:empty,p:empty")
)

// remove some empty tags
// <p></p> is used a typically unintended line spaces
// <a></a> is in docs for unknown reasons
//
func RemoveEmptyTags(root *html.Node) error {
	for _, empty := range selectorEmpty.MatchAll(root) {
		empty.Parent.RemoveChild(empty)
	}
	return nil
}
