package dart

import sitter "github.com/tree-sitter/go-tree-sitter"

// ClassInfo is one top-level class_definition, with just enough
// structure for doctor/init/integrate: its superclass (to recognize
// StatelessWidget/StatefulWidget/State) and, for a State<X> subclass,
// which widget X it belongs to.
type ClassInfo struct {
	Name      string
	Supertype string // simple name, e.g. "StatelessWidget", "StatefulWidget", "State"
	StateOf   string // if Supertype == "State", the generic type argument (its widget)
	File      string
	Line      int
}

// Classes walks tree and returns every class_definition.
func Classes(tree *sitter.Tree, src []byte, file string) []ClassInfo {
	var out []ClassInfo
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "class_definition" {
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

	if super := n.ChildByFieldName("superclass"); super != nil {
		for _, c := range namedChildren(super) {
			switch c.Kind() {
			case "type_identifier":
				if ci.Supertype == "" {
					ci.Supertype = c.Utf8Text(src)
				}
			case "type_arguments":
				if inner := namedChildren(c); len(inner) == 1 && inner[0].Kind() == "type_identifier" {
					ci.StateOf = inner[0].Utf8Text(src)
				}
			}
		}
	}
	return ci, true
}

func findChildKind(n *sitter.Node, kind string) *sitter.Node {
	for i := range n.ChildCount() {
		if c := n.Child(i); c.Kind() == kind {
			return c
		}
	}
	return nil
}

func findClassNode(tree *sitter.Tree, src []byte, name string) *sitter.Node {
	var result *sitter.Node
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if result != nil {
			return
		}
		if n.Kind() == "class_definition" {
			if nameNode := n.ChildByFieldName("name"); nameNode != nil && nameNode.Utf8Text(src) == name {
				result = n
				return
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return result
}
