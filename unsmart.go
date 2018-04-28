package googledrive2hugo

import (
	"strings"
)

var unsmartReplacer *strings.Replacer

func init() {
	replacements := []string{

		"\u8220", `"`, // double quote open
		"\u8221", `"`, // double quote close
		"\u8216", "'", // single quote open
		"\u8217", "'", // single quote close
		"\u8230", "...", // ellipis
		"\u8212", "--", // em dash
		"\u8211", "--", // en dash
	}
	unsmartReplacer = strings.NewReplacer(replacements...)
}
func unsmart(s string) string {
	return unsmartReplacer.Replace(s)
}
