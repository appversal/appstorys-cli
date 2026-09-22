package kotlin

import sitter "github.com/tree-sitter/go-tree-sitter"

// ClassInfo is one top-level or nested class/object declaration, with
// just enough structure for doctor's checks: which classes extend
// Application (for the init-location check) or an Activity/Fragment
// base (for screen discovery).
type ClassInfo struct {
	Name       string
	Supertypes []string // simple names only, e.g. "Application", "AppCompatActivity"
	File       string
	Line       int
}

// Classes walks tree and returns every class_declaration.
func Classes(tree *sitter.Tree, src []byte, file string) []ClassInfo {
	var out []ClassInfo
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "class_declaration" {
			if ci, ok := buildClassInfo(n, src, file); ok {
				out = append(out, ci)
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return out
}

func buildClassInfo(n *sitter.Node, src []byte, file string) (ClassInfo, bool) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return ClassInfo{}, false
	}
	pos := n.StartPosition()
	ci := ClassInfo{Name: nameNode.Utf8Text(src), File: file, Line: int(pos.Row) + 1}

	specs := findChildKind(n, "delegation_specifiers")
	if specs == nil {
		return ci, true
	}
	for _, spec := range namedChildren(specs) {
		userType := spec
		if len(namedChildren(spec)) == 1 {
			userType = namedChildren(spec)[0]
		}
		if userType.Kind() == "constructor_invocation" {
			if ut := findChildKind(userType, "user_type"); ut != nil {
				userType = ut
			}
		}
		if userType.Kind() != "user_type" {
			continue
		}
		var last *sitter.Node
		for _, c := range namedChildren(userType) {
			if c.Kind() == "identifier" {
				last = c
			}
		}
		if last != nil {
			ci.Supertypes = append(ci.Supertypes, last.Utf8Text(src))
		}
	}
	return ci, true
}
