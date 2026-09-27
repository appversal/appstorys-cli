// Package ts is the TypeScript/TSX (and plain JavaScript/JSX, which the
// TSX grammar also parses) language adapter for React Native: it parses
// source with tree-sitter and walks the resulting AST to produce
// extract.CallExpr values for internal/extract to match against the
// React Native symbol map.
//
// Unlike Kotlin/Dart, React Native's SDK surface is JSX-element-shaped
// as much as call-shaped (<AppStorys.Screen name="Home">, not just
// AppStorys.trackScreen("Home")), so Extract walks both call_expression
// and JSX element nodes into the same CallExpr representation — a JSX
// element's attributes become named Args, the same as a call's keyword
// arguments would if TS/JS call syntax had them.
package ts

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	tstsx "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Language returns the tree-sitter TSX grammar. TSX is used for every
// extension (.ts, .tsx, .js, .jsx): it's a superset that parses
// type-free JS and JSX fine, so one grammar covers all four.
func Language() *sitter.Language {
	return sitter.NewLanguage(tstsx.LanguageTSX())
}
