package parse

import (
	"context"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsts "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

func TestPoolParseTypeScript(t *testing.T) {
	lang := sitter.NewLanguage(tsts.LanguageTypescript())

	pool, err := NewPool(lang, 2)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	tree, err := pool.Parse(context.Background(), []byte("const x = 1;\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("Parse() root node is nil")
	}
	if root.HasError() {
		t.Errorf("Parse() root node reports a syntax error for valid input")
	}
}
