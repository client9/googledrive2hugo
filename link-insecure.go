package googledrive2hugo

import (
	"fmt"
	"log"
	"strings"

	"github.com/andybalholm/cascadia"
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

func LinkInsecure(whitelist []string) func(*html.Node) error {
	selector := cascadia.MustCompile(fmt.Sprintf("a[href^=%q]", "http:"))
	return func(root *html.Node) error {
		insecure := make(map[string]bool)
		for _, n := range selector.MatchAll(root) {
			for _, attr := range n.Attr {
				if attr.Key == "href" && !inWhitelist(whitelist, attr.Val) {
					insecure[attr.Val] = true
				}
			}
		}
		if len(insecure) > 0 {
			for k, _ := range insecure {
				log.Printf("Insecure link: %s", k)
			}
			return fmt.Errorf("Found %d insecure links.  Fix or add to whitelist", len(insecure))
		}
		return nil
	}
}
