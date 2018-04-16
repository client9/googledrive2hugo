package googledrive2hugo

import (
	"regexp"
)

// special handling for shortcodes
// The golang rendering will HTML escape any text nodes
// We need to unescape shortcodes

var (
	reEntities = regexp.MustCompile(`&amp;#[0-9]+;`)
)

// unescapeShortcode:
//
// From:
//  &lt;#037;
//
// To:
//  &#037;
//
func unescapeEntities(buf []byte) []byte {
	buf = reEntities.ReplaceAllFunc(buf, unescape)
	return buf
}
