package googledrive2hugo

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/andybalholm/cascadia"
)

var (
	selectorBold  = cascadia.MustCompile("b,strong")
	selectorTable = cascadia.MustCompile("table")
	selectorTdP   = cascadia.MustCompile("td > p:only-child")
)

// ConvertGdocTables cleans up tables found in gdocs
//  Removes redundant <p> inside each <td>
//  If first row had bold <td> elements, convert the row into
//    a <thead> with <th> elements
//
func GdocTable(root *html.Node) {

	// gdoc puts a <p> inside each <td>.  Remove the unnecessary <p> tag.
	for _, p := range selectorTdP.MatchAll(root) {
		td := p.Parent
		td.RemoveChild(p)
		reparentChildren(td, p)
	}

	// may turn first <tr> into a <thead><tr> and turn the <td> into <th>
	for _, table := range selectorTable.MatchAll(root) {
		fixTableNode(table)
	}
}
func hasBoldChildren(n *html.Node) bool {
	return selectorBold.MatchFirst(n) != nil
}

// looks at first row to see if it is a header row, and if so
// move it to a new thead, and change <td> to <th>
func fixTableNode(table *html.Node) {
	tbody := table.FirstChild

	// probably should do a warning here.
	// this table isn't what we expected.
	if tbody == nil || tbody.DataAtom != atom.Tbody {
		return
	}

	// we expect the first child to be a <tr>
	//  if it's not, or if none of the subquent <td> are bold
	//  then nothing to do.
	tr := tbody.FirstChild
	if tr == nil || tr.DataAtom != atom.Tr || !hasBoldChildren(tr) {
		return
	}

	// convert TD to TH
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		text := getTextContent(td)
		td.Data = "th"
		td.DataAtom = atom.Th
		removeAllChildren(td)
		td.AppendChild(newTextNode(text))
	}

	// move tr from tbody to new thead
	thead := newElementNode("thead")
	tbody.RemoveChild(tr)
	thead.AppendChild(tr)
	table.InsertBefore(thead, tbody)

	// DO FOOT

	// get Last TR
	// are TD bold?
	// remove, create TFOOT, add TR

	// how iterate  over remaining rows, checking first entry
	for tr := tbody.FirstChild; tr != nil; tr = tr.NextSibling {
		td := tr.FirstChild
		if td != nil && hasBoldChildren(td) {
			text := getTextContent(td)
			td.DataAtom = atom.Th
			td.Data = "th"
			removeAllChildren(td)
			td.AppendChild(newTextNode(text))
		}
	}
}
