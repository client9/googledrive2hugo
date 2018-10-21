package googledrive2hugo

import (
	"sync"

	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

var (
	defaultSelectorEmpty = "a:empty,p:empty,em:empty,i:empty,strong:empty,b:empty,span:empty,div:empty"
)

// remove some empty tags
// <p></p> is used a typically unintended line spaces
// <a></a> is in docs for unknown reasons
//
type RemoveEmptyTag struct {
	Pattern  string
	init     sync.Once
	selector cascadia.Selector
}

func (n *RemoveEmptyTag) Run(root *html.Node, logger ilog.Logger) error {
	var err error
	n.init.Do(func() {
		if n.Pattern == "" {
			n.Pattern = defaultSelectorEmpty
		}
		n.selector, err = cascadia.Compile(n.Pattern)
	})
	if err != nil {
		return err
	}
	for _, empty := range n.selector.MatchAll(root) {
		logger.Debug("", "tag", empty.Data)
		empty.Parent.RemoveChild(empty)
	}
	return nil
}
