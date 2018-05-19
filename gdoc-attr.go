package googledrive2hugo

import (
	"golang.org/x/net/html"
)

// GDocAttr remove all unnessecary attributes from a GDoc HTML tree
// in particular removes all attributes except
//
//  * id
//  * href
//  * colspan,rowspan if not "1"
//
// TODO: probably can optimize this by skipping recusion on text-nodes
func GdocAttr(root *html.Node) error {
	removeAttr(root)
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		GdocAttr(c)
	}
	return nil
}

func removeAttr(n *html.Node) {
	if n.Attr == nil {
		return
	}
	idx := 0
	for i := 0; i < len(n.Attr); i++ {
		switch n.Attr[i].Key {
		case "href":
			// needed for <a> and others
			n.Attr[idx] = n.Attr[i]
			idx++
		case "colspan", "rowspan":
			// gdoc does a lot of <td rowspan=1 colspan=1
			// which is not needed
			if n.Attr[i].Val != "1" {
				n.Attr[idx] = n.Attr[i]
				idx++
			}
		case "id":
			// preserve ID for headings.  Used for internal linking
			// hack for h1..h6.  includes hr too but
			//  thats ok
			if len(n.Data) == 2 && n.Data[0] == 'h' {
				n.Attr[idx] = n.Attr[i]
				idx++
			}
		default:
			continue
		}
	}
	// remove any junk at end
	n.Attr = n.Attr[:idx]
}
