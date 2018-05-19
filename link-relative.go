package googledrive2hugo

import (
	"fmt"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

func LinkRelative(prefix string) func(*html.Node) error {
	selector := cascadia.MustCompile(fmt.Sprintf("a[href^=%q]", prefix))
	return func(root *html.Node) error {
		for _, n := range selector.MatchAll(root) {
			for i, attr := range n.Attr {
				if attr.Key == "href" {
					n.Attr[i].Val = attr.Val[len(prefix):]
				}
			}
		}
		return nil
	}
}
