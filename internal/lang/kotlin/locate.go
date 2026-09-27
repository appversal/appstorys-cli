package kotlin

import sitter "github.com/tree-sitter/go-tree-sitter"

// LocateClass finds className's class_body and returns the byte offset
// right after its opening "{" (bodyInsertAt, for inserting a new
// member), plus, for every method already declared directly in that
// body, the byte offset right after ITS opening "{" (methodInsertAt, for
// inserting a first statement into an existing method). found is false
// if no class named className exists in tree.
func LocateClass(tree *sitter.Tree, src []byte, className string) (bodyInsertAt int, methodInsertAt map[string]int, found bool) {
	target := findClassNode(tree, src, className)
	if target == nil {
		return 0, nil, false
	}

	body := findChildKind(target, "class_body")
	if body == nil {
		return 0, nil, false
	}
	bodyInsertAt = int(body.StartByte()) + 1

	methodInsertAt = map[string]int{}
	for _, child := range namedChildren(body) {
		if child.Kind() != "function_declaration" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		fnBody := findChildKind(child, "function_body")
		if fnBody == nil {
			continue
		}
		block := findChildKind(fnBody, "block")
		if block == nil {
			continue
		}
		methodInsertAt[nameNode.Utf8Text(src)] = int(block.StartByte()) + 1
	}
	return bodyInsertAt, methodInsertAt, true
}

// LocateBlock finds a nested chain of Gradle-DSL-style trailing-lambda
// blocks — `dependencies { ... }`, `android { defaultConfig { ... } }`
// — and returns the byte offset right before the innermost block's
// closing "}", so callers can append a new line to it. path is the
// chain of block names from outermost to innermost, e.g.
// LocateBlock(tree, src, "android", "defaultConfig").
//
// A trailing-lambda call like `dependencies { ... }` is a call_expression
// whose two named children are an identifier and an annotated_lambda
// wrapping a lambda_literal — distinct from a parenthesized call, which
// LocateClass/the extractor's resolveArgs don't need to understand.
func LocateBlock(tree *sitter.Tree, src []byte, path ...string) (insertAt int, found bool) {
	node, ok := FindBlockNode(tree, src, path...)
	if !ok {
		return 0, false
	}
	return int(node.EndByte()) - 1, true // right before "}"
}

// FindBlockNode is like LocateBlock but returns the block's own
// lambda_literal node instead of an insertion point, so callers can
// inspect its contents — e.g. FindStringAssignment to read an existing
// `namespace = "..."` out of an `android { ... }` block.
func FindBlockNode(tree *sitter.Tree, src []byte, path ...string) (*sitter.Node, bool) {
	if len(path) == 0 {
		return nil, false
	}
	return findBlockNodeIn(namedChildren(tree.RootNode()), src, path)
}

// LocateClassMethodBlock finds a trailing-lambda block (e.g.
// `setContent { ... }`) nested inside one specific method of one
// specific class — as opposed to LocateBlock, which searches the whole
// file — and returns the byte offset right before its closing "}".
// Needed because a generic file-wide LocateBlock("setContent") could
// match a different class's (or a different method's) setContent call
// in a multi-Activity file.
func LocateClassMethodBlock(tree *sitter.Tree, src []byte, className, methodName string, path ...string) (insertAt int, found bool) {
	block, ok := methodBlock(tree, src, className, methodName)
	if !ok {
		return 0, false
	}
	node, ok := findBlockNodeIn(namedChildren(block), src, path)
	if !ok {
		return 0, false
	}
	return int(node.EndByte()) - 1, true
}

// LocateClassAnyMethodBlock is like LocateClassMethodBlock but doesn't
// require the block to sit in one specific method — it searches every
// method declared directly in className's body, in declaration order,
// and returns the first match. Needed because setContent { ... } is
// often called from a helper method (e.g. a private setupContent())
// rather than directly inside onCreate.
func LocateClassAnyMethodBlock(tree *sitter.Tree, src []byte, className string, path ...string) (insertAt int, found bool) {
	classNode := findClassNode(tree, src, className)
	if classNode == nil {
		return 0, false
	}
	body := findChildKind(classNode, "class_body")
	if body == nil {
		return 0, false
	}
	for _, member := range namedChildren(body) {
		if member.Kind() != "function_declaration" {
			continue
		}
		fnBody := findChildKind(member, "function_body")
		if fnBody == nil {
			continue
		}
		block := findChildKind(fnBody, "block")
		if block == nil {
			continue
		}
		if node, ok := findBlockNodeIn(namedChildren(block), src, path); ok {
			return int(node.EndByte()) - 1, true
		}
	}
	return 0, false
}

// FindStringAssignment reads a top-level `name = "value"` assignment
// directly inside block (as FindBlockNode returns it), e.g. `namespace`
// inside an `android { ... }` block.
func FindStringAssignment(block *sitter.Node, src []byte, name string) (string, bool) {
	if block == nil {
		return "", false
	}
	lr := LiteralReader{}
	for _, child := range namedChildren(block) {
		if child.Kind() != "assignment" {
			continue
		}
		left := child.ChildByFieldName("left")
		right := child.ChildByFieldName("right")
		if left == nil || right == nil || left.Utf8Text(src) != name {
			continue
		}
		if text, ok := lr.StringLiteral(right, src); ok {
			return text, true
		}
	}
	return "", false
}

// findClassNode returns the class_declaration node named className
// anywhere in tree, or nil if none exists.
func findClassNode(tree *sitter.Tree, src []byte, className string) *sitter.Node {
	var target *sitter.Node
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if target != nil {
			return
		}
		if n.Kind() == "class_declaration" {
			if name := n.ChildByFieldName("name"); name != nil && name.Utf8Text(src) == className {
				target = n
				return
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return target
}

func methodBlock(tree *sitter.Tree, src []byte, className, methodName string) (*sitter.Node, bool) {
	classNode := findClassNode(tree, src, className)
	if classNode == nil {
		return nil, false
	}
	body := findChildKind(classNode, "class_body")
	if body == nil {
		return nil, false
	}
	for _, member := range namedChildren(body) {
		if member.Kind() != "function_declaration" {
			continue
		}
		nameNode := member.ChildByFieldName("name")
		if nameNode == nil || nameNode.Utf8Text(src) != methodName {
			continue
		}
		fnBody := findChildKind(member, "function_body")
		if fnBody == nil {
			continue
		}
		return findChildKind(fnBody, "block"), true
	}
	return nil, false
}

func findBlockNodeIn(scope []*sitter.Node, src []byte, path []string) (*sitter.Node, bool) {
	for _, n := range scope {
		if n.Kind() != "call_expression" {
			continue
		}
		named := namedChildren(n)
		if len(named) != 2 || named[0].Kind() != "identifier" || named[0].Utf8Text(src) != path[0] {
			continue
		}
		lambda := lambdaLiteral(named[1])
		if lambda == nil {
			continue
		}
		if len(path) == 1 {
			return lambda, true
		}
		if node, ok := findBlockNodeIn(namedChildren(lambda), src, path[1:]); ok {
			return node, true
		}
	}
	return nil, false
}

func lambdaLiteral(n *sitter.Node) *sitter.Node {
	switch n.Kind() {
	case "lambda_literal":
		return n
	case "annotated_lambda":
		return findChildKind(n, "lambda_literal")
	default:
		return nil
	}
}
