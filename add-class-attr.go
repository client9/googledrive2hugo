package googledrive2hugo

import (
	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
	"sync"
)

type AddClassAttr struct {
	ClassMap  map[string]string
	classmaps []classmap
	once      sync.Once
}

type classmap struct {
	selector cascadia.Selector
	classes  string
}

func (a *AddClassAttr) Run(n *html.Node, log ilog.Logger) (err error) {
	a.once.Do(func() {
		a.classmaps = make([]classmap, 0, len(a.ClassMap))
		for pattern, classes := range a.ClassMap {
			selector := cascadia.MustCompile(pattern)
			cm := classmap{selector: selector, classes: classes}
			a.classmaps = append(a.classmaps, cm)
		}
	})
	for _, x := range a.classmaps {
		for _, node := range x.selector.MatchAll(n) {
			node.Attr = append(node.Attr, html.Attribute{Key: "class", Val: x.classes})
		}
	}

	return nil
}
