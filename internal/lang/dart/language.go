// Package dart is the Dart language adapter: it parses Dart source with
// tree-sitter and walks the resulting AST to produce extract.CallExpr
// values for internal/extract to match against the Flutter symbol map.
package dart

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	tsdart "github.com/tree-sitter/tree-sitter-dart/bindings/go"
)

// Language returns the tree-sitter Dart grammar. The module is fetched
// from github.com/UserNobody14/tree-sitter-dart via a go.mod replace
// directive: its own go.mod self-declares the canonical module path
// github.com/tree-sitter/tree-sitter-dart (which doesn't exist as a repo),
// so importers must import it under that declared path. It's a
// community-maintained grammar rather than an org-maintained one —
// there is no tree-sitter-grammars/tree-sitter-dart — and its handling
// of plain string literals is unusually raw (see LiteralReader).
func Language() *sitter.Language {
	return sitter.NewLanguage(tsdart.Language())
}
