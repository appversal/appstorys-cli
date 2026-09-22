package dart

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
)

// Extract walks tree and returns every call/constructor invocation as a
// CallExpr.
//
// This grammar has no single "call expression" node: a call is an
// identifier immediately followed, among its own siblings, by one or
// more `selector` nodes (member access, then a final argument_part).
// `AppStorys.trackScreen('Home', ctx)` inside an expression_statement is
// literally the flat sibling sequence [identifier, selector, selector],
// and the same shape recurs unwrapped inside lists, spreads, arguments,
// and so on — so extraction scans every node's direct children for that
// pattern rather than matching one node kind.
func Extract(tree *sitter.Tree, src []byte) []extract.CallExpr {
	var calls []extract.CallExpr
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		scanForCalls(n, src, &calls)
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return calls
}

func scanForCalls(parent *sitter.Node, src []byte, out *[]extract.CallExpr) {
	for i := range parent.ChildCount() {
		if parent.Child(i).Kind() != "identifier" {
			continue
		}
		if call, ok := matchCallRun(parent, i, src); ok {
			*out = append(*out, call)
		}
	}
}

func matchCallRun(parent *sitter.Node, start uint, src []byte) (extract.CallExpr, bool) {
	base := parent.Child(start)
	parts := []string{base.Utf8Text(src)}
	var argsNode *sitter.Node

	j := start + 1
loop:
	for j < parent.ChildCount() {
		sel := parent.Child(j)
		if sel.Kind() != "selector" {
			break
		}
		named := namedChildren(sel)
		if len(named) != 1 {
			return extract.CallExpr{}, false
		}
		switch inner := named[0]; inner.Kind() {
		case "unconditional_assignable_selector", "conditional_assignable_selector":
			member := namedChildren(inner)
			if len(member) != 1 {
				return extract.CallExpr{}, false
			}
			parts = append(parts, member[0].Utf8Text(src))
			j++
		case "argument_part":
			for _, c := range namedChildren(inner) {
				if c.Kind() == "arguments" {
					argsNode = c
				}
			}
			break loop
		default:
			return extract.CallExpr{}, false
		}
	}

	if argsNode == nil {
		return extract.CallExpr{}, false
	}
	return extract.CallExpr{
		Parts:     parts,
		Args:      resolveArgs(argsNode, src),
		Node:      base,
		Enclosing: enclosingName(base, src),
	}, true
}

func resolveArgs(n *sitter.Node, src []byte) []extract.Arg {
	var args []extract.Arg
	for _, a := range namedChildren(n) {
		switch a.Kind() {
		case "argument":
			if named := namedChildren(a); len(named) == 1 {
				args = append(args, extract.Arg{Node: named[0]})
			}
		case "named_argument":
			named := namedChildren(a)
			if len(named) != 2 || named[0].Kind() != "label" {
				continue
			}
			labelName := namedChildren(named[0])
			if len(labelName) != 1 {
				continue
			}
			args = append(args, extract.Arg{Name: labelName[0].Utf8Text(src), Node: named[1]})
		}
	}
	return args
}

func enclosingName(n *sitter.Node, src []byte) string {
	var funcName, className string
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		switch cur.Kind() {
		case "function_body":
			// A function/method's signature (with its name) is this
			// body's previous sibling, not its ancestor: the grammar
			// keeps `method_signature`/`function_signature` and
			// `function_body` as siblings rather than nesting the body
			// inside the signature.
			if funcName == "" {
				if sig := cur.PrevNamedSibling(); sig != nil {
					if sig.Kind() == "method_signature" {
						if nc := namedChildren(sig); len(nc) == 1 {
							sig = nc[0]
						}
					}
					if name := sig.ChildByFieldName("name"); name != nil {
						funcName = name.Utf8Text(src)
					}
				}
			}
		case "class_definition":
			if className == "" {
				if name := cur.ChildByFieldName("name"); name != nil {
					className = name.Utf8Text(src)
				}
			}
		}
	}
	switch {
	case className != "" && funcName != "":
		return className + "." + funcName
	case funcName != "":
		return funcName
	default:
		return className
	}
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

// LiteralReader implements extract.LiteralReader for Dart.
type LiteralReader struct{}

func (LiteralReader) StringLiteral(n *sitter.Node, src []byte) (string, bool) {
	if n == nil || n.Kind() != "string_literal" {
		return "", false
	}
	// This grammar doesn't tokenize a plain string's body into a child
	// node: a non-interpolated 'literal' has exactly two children, the
	// open and close quote tokens, with the text living in the byte gap
	// between them. More children means interpolation (or an
	// unrecognized shape) is present, which isn't a safe literal.
	if n.ChildCount() != 2 {
		return "", false
	}
	text := n.Utf8Text(src)
	if len(text) < 2 {
		return "", false
	}
	return text[1 : len(text)-1], true
}

func (LiteralReader) PropType(n *sitter.Node) extract.PropType {
	if n == nil {
		return extract.PropDynamic
	}
	k := n.Kind()
	switch {
	case strings.Contains(k, "string"):
		return extract.PropString
	case k == "true" || k == "false" || strings.Contains(k, "boolean"):
		return extract.PropBool
	case strings.Contains(k, "integer"), strings.Contains(k, "double"), strings.Contains(k, "decimal"),
		strings.Contains(k, "number"), strings.Contains(k, "hex"):
		return extract.PropNumber
	default:
		return extract.PropDynamic
	}
}

func (lr LiteralReader) MapEntries(n *sitter.Node, src []byte) []extract.MapEntry {
	if n == nil || n.Kind() != "set_or_map_literal" {
		return nil
	}
	var entries []extract.MapEntry
	for _, item := range namedChildren(n) {
		if item.Kind() != "pair" {
			continue
		}
		keyNode := item.ChildByFieldName("key")
		valNode := item.ChildByFieldName("value")
		if keyNode == nil || valNode == nil {
			continue
		}
		key, ok := lr.StringLiteral(keyNode, src)
		if !ok {
			continue
		}
		entries = append(entries, extract.MapEntry{Key: key, Value: valNode})
	}
	return entries
}

func (lr LiteralReader) ListLiteral(n *sitter.Node, src []byte) []string {
	if n == nil || n.Kind() != "list_literal" {
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
