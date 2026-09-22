// Package parse holds the tree-sitter parser pool, an in-memory
// per-command parse-tree cache, and the ignore-aware file walker shared
// by every platform adapter.
package parse

import (
	"context"
	"fmt"
	"runtime"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Pool is a fixed set of *sitter.Parser, all configured for the same
// Language. Tree-sitter parsers are not goroutine-safe, so the pool hands
// out exclusive use of one parser per caller via a buffered channel.
type Pool struct {
	lang    *sitter.Language
	parsers chan *sitter.Parser
}

// NewPool creates a pool of size parsers for lang. size <= 0 defaults to
// runtime.NumCPU().
func NewPool(lang *sitter.Language, size int) (*Pool, error) {
	if lang == nil {
		return nil, fmt.Errorf("parse: NewPool: nil language")
	}
	if size <= 0 {
		size = runtime.NumCPU()
	}

	p := &Pool{
		lang:    lang,
		parsers: make(chan *sitter.Parser, size),
	}
	for i := 0; i < size; i++ {
		parser := sitter.NewParser()
		if err := parser.SetLanguage(lang); err != nil {
			p.Close()
			return nil, fmt.Errorf("parse: NewPool: set language: %w", err)
		}
		p.parsers <- parser
	}
	return p, nil
}

// Parse acquires a parser from the pool, parses src, and returns it. It
// blocks until a parser is available or ctx is done. ctx governs only the
// wait for a free parser; a full cancellable parse is Phase 1+ work (see
// Parser.ParseWithOptions's progress callback).
func (p *Pool) Parse(ctx context.Context, src []byte) (*sitter.Tree, error) {
	select {
	case parser := <-p.parsers:
		defer func() { p.parsers <- parser }()
		tree := parser.Parse(src, nil)
		if tree == nil {
			return nil, fmt.Errorf("parse: parsing failed")
		}
		return tree, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close releases every parser in the pool. The pool must not be used
// after Close.
func (p *Pool) Close() {
	close(p.parsers)
	for parser := range p.parsers {
		parser.Close()
	}
}
