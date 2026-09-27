package ts

import sitter "github.com/tree-sitter/go-tree-sitter"

// LocateComponentInit finds where to insert the AppStorys init call in
// componentName's body: inside its existing mount-only
// `useEffect(() => {...}, [])` if one exists (hasUseEffect=true), or
// the start of the component's own body if not (hasUseEffect=false, so
// the caller inserts a brand new useEffect). found is false if
// componentName can't be resolved to a block-bodied function/arrow-
// function component (e.g. an arrow function with a direct expression
// body, which can't sensibly host a useEffect).
func LocateComponentInit(tree *sitter.Tree, src []byte, componentName string) (insertAt int, hasUseEffect bool, found bool) {
	fnNode := findComponentFn(tree, src, componentName)
	if fnNode == nil {
		return 0, false, false
	}
	body := fnNode.ChildByFieldName("body")
	if body == nil || body.Kind() != "statement_block" {
		return 0, false, false
	}

	for _, stmt := range namedChildren(body) {
		effectBody, ok := mountOnlyUseEffectBody(stmt, src)
		if !ok {
			continue
		}
		return int(effectBody.StartByte()) + 1, true, true
	}
	return int(body.StartByte()) + 1, false, true
}

// mountOnlyUseEffectBody recognizes `useEffect(() => { ... }, [])` as a
// top-level expression statement and returns its callback's block.
func mountOnlyUseEffectBody(stmt *sitter.Node, src []byte) (*sitter.Node, bool) {
	if stmt.Kind() != "expression_statement" {
		return nil, false
	}
	named := namedChildren(stmt)
	if len(named) != 1 || named[0].Kind() != "call_expression" {
		return nil, false
	}
	call := named[0]
	fn := call.ChildByFieldName("function")
	if fn == nil || fn.Kind() != "identifier" || fn.Utf8Text(src) != "useEffect" {
		return nil, false
	}
	argsNode := call.ChildByFieldName("arguments")
	if argsNode == nil {
		return nil, false
	}
	args := namedChildren(argsNode)
	if len(args) < 2 || args[0].Kind() != "arrow_function" || args[1].Kind() != "array" {
		return nil, false
	}
	if len(namedChildren(args[1])) != 0 {
		return nil, false // has dependencies: not a mount-only effect
	}
	effectBody := args[0].ChildByFieldName("body")
	if effectBody == nil || effectBody.Kind() != "statement_block" {
		return nil, false
	}
	return effectBody, true
}

func findComponentFn(tree *sitter.Tree, src []byte, name string) *sitter.Node {
	var result *sitter.Node
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if result != nil {
			return
		}
		switch n.Kind() {
		case "function_declaration":
			if nameNode := n.ChildByFieldName("name"); nameNode != nil && nameNode.Utf8Text(src) == name {
				result = n
				return
			}
		case "variable_declarator":
			nameNode := n.ChildByFieldName("name")
			val := n.ChildByFieldName("value")
			if nameNode != nil && nameNode.Utf8Text(src) == name && val != nil &&
				(val.Kind() == "arrow_function" || val.Kind() == "function_expression") {
				result = val
				return
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return result
}
