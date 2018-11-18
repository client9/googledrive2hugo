package googledrive2hugo

import (
	"fmt"

	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

type LinkRelative struct {
	pattern  string
	selector cascadia.Selector
}

func (n *LinkRelative) Init(host string) (err error) {
	n.pattern = host
	n.selector, err = cascadia.Compile(fmt.Sprintf("a[href^=%q]", host))
	return err
}

func (n *LinkRelative) Run(root *html.Node, log ilog.Logger) (err error) {
	for _, node := range n.selector.MatchAll(root) {
		for i, attr := range node.Attr {
			if attr.Key == "href" {
				log.Debug("", "url", node.Attr[i].Val)
				node.Attr[i].Val = attr.Val[len(n.pattern):]
			}
		}
	}
	return nil
}
