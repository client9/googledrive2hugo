package googledrive2hugo

import (
	"fmt"
	"sync"

	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

type LinkRelative struct {
	Pattern  string
	selector cascadia.Selector
	init     sync.Once
}

func (n *LinkRelative) Run(root *html.Node, log ilog.Logger) (err error) {
	n.init.Do(func() {
		if n.Pattern == "" {
			err = fmt.Errorf("prefix not entered")
			return
		}
		n.selector, err = cascadia.Compile(fmt.Sprintf("a[href^=%q]", n.Pattern))
	})
	if err != nil {
		return err
	}
	for _, node := range n.selector.MatchAll(root) {
		for i, attr := range node.Attr {
			if attr.Key == "href" {
				log.Debug("", "url", node.Attr[i].Val)
				node.Attr[i].Val = attr.Val[len(n.Pattern):]
			}
		}
	}
	return nil
}
