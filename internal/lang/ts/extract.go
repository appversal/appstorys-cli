package ts

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
)

// Extract walks tree and returns every call_expression and JSX element
// as a CallExpr.
func Extract(tree *sitter.Tree, src []byte) []extract.CallExpr {
	var calls []extract.CallExpr
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "call_expression":
			if ce, ok := buildCallFromExpression(n, src); ok {
				calls = append(calls, ce)
			}
		case "jsx_opening_element", "jsx_self_closing_element":
			if ce, ok := buildCallFromJSX(n, src); ok {
				calls = append(calls, ce)
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return calls
}

func buildCallFromExpression(n *sitter.Node, src []byte) (extract.CallExpr, bool) {
	fn := n.ChildByFieldName("function")
	if fn == nil {
		return extract.CallExpr{}, false
	}
	parts := resolveParts(fn, src)
	if parts == nil {
		return extract.CallExpr{}, false
	}
	var args []extract.Arg
	if argsNode := n.ChildByFieldName("arguments"); argsNode != nil {
		for _, a := range namedChildren(argsNode) {
			args = append(args, extract.Arg{Node: a}) // JS/TS call args are always positional
		}
	}
	return extract.CallExpr{
		Parts:     parts,
		Args:      args,
		Node:      n,
		Enclosing: enclosingName(n, src),
	}, true
}

func buildCallFromJSX(n *sitter.Node, src []byte) (extract.CallExpr, bool) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return extract.CallExpr{}, false
	}
	parts := resolveParts(nameNode, src)
	if parts == nil {
		return extract.CallExpr{}, false
	}

	cursor := n.Walk()
	defer cursor.Close()
	attrs := n.ChildrenByFieldName("attribute", cursor)

	var args []extract.Arg
	for i := range attrs {
		attr := &attrs[i]
		if attr.Kind() != "jsx_attribute" {
			continue
		}
		named := namedChildren(attr)
		if len(named) != 2 {
			continue
		}
		args = append(args, extract.Arg{Name: named[0].Utf8Text(src), Node: named[1]})
	}

	return extract.CallExpr{
		Parts:     parts,
		Args:      args,
		Node:      n,
		Enclosing: enclosingName(n, src),
	}, true
}

// resolveParts turns a callee or JSX name expression into a
// dot-separated identifier chain, dropping call parens along the way
// (so `AppStorys.getInstance().foo`-style chains, if RN's SDK ever grew
// one, would resolve the same as Kotlin's).
func resolveParts(n *sitter.Node, src []byte) []string {
	switch n.Kind() {
	case "identifier", "property_identifier":
		return []string{n.Utf8Text(src)}
	case "member_expression":
		obj := n.ChildByFieldName("object")
		prop := n.ChildByFieldName("property")
		if obj == nil || prop == nil {
			return nil
		}
		objParts := resolveParts(obj, src)
		propParts := resolveParts(prop, src)
		if objParts == nil || propParts == nil {
			return nil
		}
		return append(objParts, propParts...)
	case "call_expression":
		fn := n.ChildByFieldName("function")
		if fn == nil {
			return nil
		}
		return resolveParts(fn, src)
	default:
		return nil
	}
}

func enclosingName(n *sitter.Node, src []byte) string {
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		switch cur.Kind() {
		case "function_declaration", "method_definition":
			if name := cur.ChildByFieldName("name"); name != nil {
				return name.Utf8Text(src)
			}
		case "variable_declarator":
			// const Foo = () => { ... } / const Foo = function () { ... }
			if name := cur.ChildByFieldName("name"); name != nil {
				val := cur.ChildByFieldName("value")
				if val != nil && (val.Kind() == "arrow_function" || val.Kind() == "function_expression" || val.Kind() == "function") {
					return name.Utf8Text(src)
				}
			}
		}
	}
	return ""
}

func namedChildren(n *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	for i := range n.ChildCount() {
		if c := n.Child(i); c.IsNamed() {
			out = append(out, c)
		}
	}
	return out
}

// LiteralReader implements extract.LiteralReader for TS/JS/JSX.
type LiteralReader struct{}

// StringLiteral reads either a plain string literal or a JSX
// expression container wrapping one ({"literal"}), since a JSX
// attribute value can be written either way.
func (lr LiteralReader) StringLiteral(n *sitter.Node, src []byte) (string, bool) {
	if n == nil {
		return "", false
	}
	if n.Kind() == "jsx_expression" {
		named := namedChildren(n)
		if len(named) != 1 {
			return "", false
		}
		n = named[0]
	}
	if n.Kind() != "string" {
		return "", false
	}
	named := namedChildren(n)
	switch len(named) {
	case 0:
		return "", true // empty string
	case 1:
		if named[0].Kind() == "string_fragment" {
			return named[0].Utf8Text(src), true
		}
	}
	return "", false // interpolation/escapes present: not a plain literal
}

func (LiteralReader) PropType(n *sitter.Node) extract.PropType {
	if n == nil {
		return extract.PropDynamic
	}
	k := n.Kind()
	switch {
	case k == "string":
		return extract.PropString
	case k == "true" || k == "false":
		return extract.PropBool
	case strings.Contains(k, "number"):
		return extract.PropNumber
	default:
		return extract.PropDynamic
	}
}

// MapEntries reads an object literal's key/value pairs, unwrapping a
// JSX expression container first if present (metadata={{...}} vs. a
// bare {...} call argument use the same object shape either way).
func (lr LiteralReader) MapEntries(n *sitter.Node, src []byte) []extract.MapEntry {
	if n == nil {
		return nil
	}
	if n.Kind() == "jsx_expression" {
		named := namedChildren(n)
		if len(named) != 1 {
			return nil
		}
		n = named[0]
	}
	if n.Kind() != "object" {
		return nil
	}
	var entries []extract.MapEntry
	for _, child := range namedChildren(n) {
		if child.Kind() != "pair" {
			continue
		}
		keyNode := child.ChildByFieldName("key")
		valNode := child.ChildByFieldName("value")
		if keyNode == nil || valNode == nil {
			continue
		}
		key := keyNode.Utf8Text(src)
		if s, ok := lr.StringLiteral(keyNode, src); ok {
			key = s
		}
		entries = append(entries, extract.MapEntry{Key: key, Value: valNode})
	}
	return entries
}

// ListLiteral reads an array literal's items, unwrapping a JSX
// expression container first if present.
func (lr LiteralReader) ListLiteral(n *sitter.Node, src []byte) []string {
	if n == nil {
		return nil
	}
	if n.Kind() == "jsx_expression" {
		named := namedChildren(n)
		if len(named) != 1 {
			return nil
		}
		n = named[0]
	}
	if n.Kind() != "array" {
		return nil
	}
	var items []string
	for _, item := range namedChildren(n) {
		text, ok := lr.StringLiteral(item, src)
		if !ok {
			return nil
		}
		items = append(items, text)
	}
	return items
}
