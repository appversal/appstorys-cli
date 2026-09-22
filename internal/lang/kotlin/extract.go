package kotlin

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
)

// Extract walks tree and returns every call_expression as a CallExpr.
// Extraction here is a direct AST walk rather than a tree-sitter Query
// (.scm) — Kotlin's grammar shape (call_expression / navigation_expression
// chains) is simple enough to walk directly, and it keeps this package's
// public surface (Extract + LiteralReader) identical to what a
// query-based implementation would expose, so the two are interchangeable
// later without touching internal/extract.
func Extract(tree *sitter.Tree, src []byte) []extract.CallExpr {
	var calls []extract.CallExpr
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "call_expression" {
			if ce, ok := buildCallExpr(n, src); ok {
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

func buildCallExpr(n *sitter.Node, src []byte) (extract.CallExpr, bool) {
	if n.ChildCount() == 0 {
		return extract.CallExpr{}, false
	}
	parts := resolveParts(n.Child(0), src)
	if parts == nil {
		return extract.CallExpr{}, false
	}
	var args []extract.Arg
	if argsNode := findChildKind(n, "value_arguments"); argsNode != nil {
		args = resolveArgs(argsNode, src)
	}
	return extract.CallExpr{
		Parts:     parts,
		Args:      args,
		Node:      n,
		Enclosing: enclosingName(n, src),
	}, true
}

// resolveParts turns a callee expression into a dot-separated identifier
// chain, dropping any call parens along the way (so
// `AppStorys.getInstance().getScreenCampaigns` and
// `AppStorys.getScreenCampaigns` both resolve their AppStorys/getInstance
// prefix the same way).
func resolveParts(n *sitter.Node, src []byte) []string {
	switch n.Kind() {
	case "identifier", "simple_identifier":
		return []string{n.Utf8Text(src)}
	case "navigation_expression":
		named := namedChildren(n)
		if len(named) != 2 {
			return nil
		}
		receiver := resolveParts(named[0], src)
		member := resolveParts(named[1], src)
		if receiver == nil || member == nil {
			return nil
		}
		return append(receiver, member...)
	case "call_expression":
		if n.ChildCount() == 0 {
			return nil
		}
		return resolveParts(n.Child(0), src)
	default:
		return nil
	}
}

func resolveArgs(n *sitter.Node, src []byte) []extract.Arg {
	var args []extract.Arg
	for _, va := range namedChildren(n) {
		named := namedChildren(va)
		switch len(named) {
		case 1:
			args = append(args, extract.Arg{Node: named[0]})
		case 2:
			args = append(args, extract.Arg{Name: named[0].Utf8Text(src), Node: named[1]})
		}
	}
	return args
}

func enclosingName(n *sitter.Node, src []byte) string {
	var funcName, className string
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		switch cur.Kind() {
		case "function_declaration":
			if funcName == "" {
				if name := cur.ChildByFieldName("name"); name != nil {
					funcName = name.Utf8Text(src)
				}
			}
		case "class_declaration", "object_declaration":
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

func findChildKind(n *sitter.Node, kind string) *sitter.Node {
	for i := range n.ChildCount() {
		if c := n.Child(i); c.Kind() == kind {
			return c
		}
	}
	return nil
}

// LiteralReader implements extract.LiteralReader for Kotlin.
type LiteralReader struct{}

func (LiteralReader) StringLiteral(n *sitter.Node, src []byte) (string, bool) {
	if n == nil || n.Kind() != "string_literal" {
		return "", false
	}
	named := namedChildren(n)
	if len(named) != 1 || named[0].Kind() != "string_content" {
		// Empty string, or interpolation present — not a plain literal
		// we can safely extract.
		if len(named) == 0 {
			return "", true
		}
		return "", false
	}
	return named[0].Utf8Text(src), true
}

func (LiteralReader) PropType(n *sitter.Node) extract.PropType {
	if n == nil {
		return extract.PropDynamic
	}
	k := n.Kind()
	switch {
	case strings.Contains(k, "string"):
		return extract.PropString
	case strings.Contains(k, "boolean") || k == "true" || k == "false":
		return extract.PropBool
	case strings.Contains(k, "integer"), strings.Contains(k, "long"), strings.Contains(k, "float"),
		strings.Contains(k, "double"), strings.Contains(k, "real"), strings.Contains(k, "hex"):
		return extract.PropNumber
	default:
		return extract.PropDynamic
	}
}

func (lr LiteralReader) MapEntries(n *sitter.Node, src []byte) []extract.MapEntry {
	if n == nil || n.Kind() != "call_expression" || n.ChildCount() == 0 {
		return nil
	}
	callee := n.Child(0)
	if callee.Kind() != "identifier" || callee.Utf8Text(src) != "mapOf" {
		return nil
	}
	argsNode := findChildKind(n, "value_arguments")
	if argsNode == nil {
		return nil
	}
	var entries []extract.MapEntry
	for _, va := range namedChildren(argsNode) {
		named := namedChildren(va)
		if len(named) != 1 || named[0].Kind() != "infix_expression" {
			continue
		}
		pair := namedChildren(named[0])
		if len(pair) < 2 {
			continue
		}
		key, ok := lr.StringLiteral(pair[0], src)
		if !ok {
			continue
		}
		entries = append(entries, extract.MapEntry{Key: key, Value: pair[len(pair)-1]})
	}
	return entries
}

func (lr LiteralReader) ListLiteral(n *sitter.Node, src []byte) []string {
	if n == nil || n.Kind() != "call_expression" || n.ChildCount() == 0 {
		return nil
	}
	callee := n.Child(0)
	if callee.Kind() != "identifier" || callee.Utf8Text(src) != "listOf" {
		return nil
	}
	argsNode := findChildKind(n, "value_arguments")
	if argsNode == nil {
		return nil
	}
	var items []string
	for _, va := range namedChildren(argsNode) {
		named := namedChildren(va)
		if len(named) != 1 {
			continue
		}
		if text, ok := lr.StringLiteral(named[0], src); ok {
			items = append(items, text)
		}
	}
	return items
}
