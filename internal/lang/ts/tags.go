package ts

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/project"
)

// ScanTags finds every `appstorys="id"` attribute in tree, on any JSX
// element — the tag concept can't be expressed as a call/constructor
// match against one component name (see the note in
// internal/symbols/reactnative.yaml), so it's matched directly here
// instead of going through extract.Match.
func ScanTags(tree *sitter.Tree, src []byte, file string) []extract.CallSite {
	lr := LiteralReader{}
	var sites []extract.CallSite
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "jsx_opening_element", "jsx_self_closing_element":
			cursor := n.Walk()
			attrs := n.ChildrenByFieldName("attribute", cursor)
			cursor.Close()

			for i := range attrs {
				attr := &attrs[i]
				if attr.Kind() != "jsx_attribute" {
					continue
				}
				named := namedChildren(attr)
				if len(named) != 2 || named[0].Utf8Text(src) != "appstorys" {
					continue
				}
				pos := n.StartPosition()
				site := extract.CallSite{
					Kind:      "tag",
					File:      file,
					Line:      int(pos.Row) + 1,
					Col:       int(pos.Column) + 1,
					Via:       "appstorys",
					Enclosing: enclosingName(n, src),
					Platform:  project.ReactNative,
				}
				if text, ok := lr.StringLiteral(named[1], src); ok {
					site.Name = text
				} else {
					site.Dynamic = true
				}
				sites = append(sites, site)
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return sites
}
