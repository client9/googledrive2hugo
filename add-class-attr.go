package googledrive2hugo

import (
	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

type AddClassAttr struct {
	selector cascadia.Selector
	newcss   string
}

func (a *AddClassAttr) Init(pattern string, newcss string) (err error) {
	selector, err := cascadia.Compile(pattern)
	if err != nil {
		return err
	}
	a.selector = selector
	a.newcss = newcss
	return nil
}
func (a *AddClassAttr) Run(n *html.Node, log ilog.Logger) (err error) {
	for _, node := range a.selector.MatchAll(n) {
		node.Attr = append(node.Attr, html.Attribute{Key: "class", Val: a.newcss})
	}
	return nil
}
