// Package kotlin is the Kotlin (Compose + XML host) language adapter:
// it parses Kotlin source with tree-sitter and walks the resulting AST
// to produce extract.CallExpr values for internal/extract to match
// against the Android symbol map.
package kotlin

import (
	tskotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Language returns the tree-sitter Kotlin grammar.
func Language() *sitter.Language {
	return sitter.NewLanguage(tskotlin.Language())
}
