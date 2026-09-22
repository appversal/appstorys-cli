// Package extract turns language-specific tree-sitter ASTs into
// CallSite values, matched against a platform's symbols.Map. The
// AST-walking is language-specific (internal/lang/kotlin,
// internal/lang/dart, ...); the matching against the symbol map — the
// part that has to agree with the CLI's data-driven SDK facts — lives
// here, shared across languages.
package extract

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// PropType is the inferred literal type of a property value. Property
// values themselves are never extracted, only their type.
type PropType string

const (
	PropString  PropType = "string"
	PropNumber  PropType = "number"
	PropBool    PropType = "bool"
	PropDynamic PropType = "dynamic"
)

// CallSite is one recognized AppStorys SDK call, matched against a
// concept in the platform's symbol map.
type CallSite struct {
	Kind       string // init | screen | event | overlay-host | placement | tag
	Name       string // literal value, empty if Dynamic
	Dynamic    bool
	Properties map[string]PropType
	File       string
	Line, Col  int
	Enclosing  string
	Via        string // the SDK symbol actually matched, e.g. "AppStorys.trackEvents"
	Position   string // widget position(s), comma-joined if more than one
	Platform   project.Platform
}

// CallExpr is a language-agnostic view of one call or constructor
// invocation, produced by a language adapter's AST walk. Parts is the
// dot-separated identifier chain with call parens dropped, e.g.
// ["AppStorys", "trackEvents"] for `AppStorys.trackEvents(...)`, or
// ["ValueKey"] for a bare `ValueKey(...)`.
type CallExpr struct {
	Parts     []string
	Args      []Arg
	Node      *sitter.Node
	Enclosing string
}

// Method returns the last segment of Parts (the function/constructor
// name actually being invoked).
func (c CallExpr) Method() string {
	if len(c.Parts) == 0 {
		return ""
	}
	return c.Parts[len(c.Parts)-1]
}

// ReceiverPath returns Parts without the trailing method, dot-joined.
func (c CallExpr) ReceiverPath() string {
	if len(c.Parts) <= 1 {
		return ""
	}
	return strings.Join(c.Parts[:len(c.Parts)-1], ".")
}

// Arg is one call argument: Name is "" for a positional argument.
type Arg struct {
	Name string
	Node *sitter.Node
}

// MapEntry is one key/value pair inside a matched map/dict literal
// argument.
type MapEntry struct {
	Key   string
	Value *sitter.Node
}

// LiteralReader lets the matching engine read language-specific literal
// nodes without knowing each grammar's node kinds.
type LiteralReader interface {
	// StringLiteral returns a string literal's content and true, or
	// ok=false if node isn't (or doesn't resolve to) a literal string.
	StringLiteral(node *sitter.Node, src []byte) (text string, ok bool)
	// PropType infers the literal type of a value node.
	PropType(node *sitter.Node) PropType
	// MapEntries returns the key/value pairs of a map/dict literal
	// argument (unwrapping constructs like Kotlin's `mapOf(...)` where
	// needed), or nil if node isn't a recognized map literal.
	MapEntries(node *sitter.Node, src []byte) []MapEntry
	// ListLiteral returns the literal string items of a list/array
	// argument (e.g. a widget position list), or nil if node isn't a
	// recognized list literal or contains non-literal items.
	ListLiteral(node *sitter.Node, src []byte) []string
}

// FindArg resolves an ArgRef against a call's arguments: Named is tried
// first (if set), then Positional among the remaining positional
// arguments, in source order. This lets one ArgRef describe an SDK call
// that accepts an argument either named or positional, which the spec
// notes Android's trackEvents does.
func FindArg(args []Arg, ref *symbols.ArgRef) (Arg, bool) {
	if ref == nil {
		return Arg{}, false
	}
	if ref.Named != "" {
		for _, a := range args {
			if a.Name == ref.Named {
				return a, true
			}
		}
	}
	if ref.Positional != nil {
		idx := 0
		for _, a := range args {
			if a.Name != "" {
				continue
			}
			if idx == *ref.Positional {
				return a, true
			}
			idx++
		}
	}
	return Arg{}, false
}
