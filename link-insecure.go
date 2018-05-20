package googledrive2hugo

import (
	"fmt"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"github.com/client9/ilog"
	"golang.org/x/net/html"
)

func inWhitelist(whitelist []string, link string) bool {
	for _, w := range whitelist {
		if strings.Contains(link, w) {
			return true
		}
	}
	return false
}

type LinkInsecure struct {
	Whitelist []string
	selector  cascadia.Selector
	init      sync.Once
}

func (n *LinkInsecure) Run(root *html.Node, log ilog.Logger) (err error) {
	n.init.Do(func() {
		n.selector, err = cascadia.Compile(`a[href^="http:"]`)
	})
	if err != nil {
		return err
	}
	insecure := make(map[string]bool)
	for _, node := range n.selector.MatchAll(root) {
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				if inWhitelist(n.Whitelist, attr.Val) {
					log.Debug("whitelisted", "url", attr.Val)
				} else {
					insecure[attr.Val] = true
				}
			}
		}
	}
	if len(insecure) > 0 {
		for k, _ := range insecure {
			log.Debug("insecure", "url", k)
		}
		return fmt.Errorf("Found %d insecure links.  Fix or add to whitelist", len(insecure))
	}
	return nil
}
