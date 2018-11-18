package googledrive2hugo

import (
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
	selector cascadia.Selector
}

func (n *RemoveEmptyTag) Init() (err error) {
	n.selector, err = cascadia.Compile(defaultSelectorEmpty)
	return err
}

func (n *RemoveEmptyTag) Run(root *html.Node, logger ilog.Logger) error {
	for _, empty := range n.selector.MatchAll(root) {
		logger.Debug("", "tag", empty.Data)
		empty.Parent.RemoveChild(empty)
	}
	return nil
}
