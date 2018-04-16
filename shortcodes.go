package googledrive2hugo

import (
	"regexp"

	"golang.org/x/net/html"
)

// special handling for shortcodes
// The golang rendering will HTML escape any text nodes
// We need to unescape shortcodes

var (
	reShortCode1 = regexp.MustCompile(`{{&lt;.*&gt;}}`)
	reShortCode2 = regexp.MustCompile(`{{%.*%}}`)
)

func unescape(buf []byte) []byte {
	return []byte(html.UnescapeString(string(buf)))
}

// unescapeShortcode:
//
// From:
//  {{&lt; instgram &#34;8203823&#34; &lt;}}
//  {{% instgram &#34;8203823&#34; %}}

// To:
//  {{< instagram "8203823" >}}
//  {{% instagram "8203823" %}}
//
func unescapeShortcodes(buf []byte) []byte {
	buf = reShortCode1.ReplaceAllFunc(buf, unescape)
	buf = reShortCode2.ReplaceAllFunc(buf, unescape)
	return buf
}
