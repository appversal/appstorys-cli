package kotlin

import sitter "github.com/tree-sitter/go-tree-sitter"

// NavHostInfo is one Compose Navigation NavHost(...) { ... } block: the
// composable function that encloses it (where an overlay host belongs,
// "once in the root composable around NavHost") and its destinations.
type NavHostInfo struct {
	EnclosingFunc         string
	EnclosingBodyInsertAt int // byte offset right after the enclosing composable's "{"
	StartLine, EndLine    int // line range of the whole NavHost call
	Destinations          []NavDestination
}

// NavDestination is one composable("route") { ... } destination inside
// a NavHost.
type NavDestination struct {
	Route        string
	Dynamic      bool
	StartLine    int
	EndLine      int
	BodyInsertAt int // byte offset right after the destination's lambda "{"
}

// ComposeNavHosts finds every NavHost(...) { composable("route") { ... }
// ... } block in tree.
func ComposeNavHosts(tree *sitter.Tree, src []byte) []NavHostInfo {
	var hosts []NavHostInfo
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if name, _, lambda, ok := callWithTrailingLambda(n, src); ok && name == "NavHost" {
			hosts = append(hosts, buildNavHostInfo(n, lambda, src))
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return hosts
}

func buildNavHostInfo(navCall, lambda *sitter.Node, src []byte) NavHostInfo {
	info := NavHostInfo{
		StartLine: int(navCall.StartPosition().Row) + 1,
		EndLine:   int(navCall.EndPosition().Row) + 1,
	}

	for cur := navCall.Parent(); cur != nil; cur = cur.Parent() {
		if cur.Kind() != "function_declaration" {
			continue
		}
		if name := cur.ChildByFieldName("name"); name != nil {
			info.EnclosingFunc = name.Utf8Text(src)
		}
		if fnBody := findChildKind(cur, "function_body"); fnBody != nil {
			if block := findChildKind(fnBody, "block"); block != nil {
				info.EnclosingBodyInsertAt = int(block.StartByte()) + 1
			}
		}
		break
	}

	lr := LiteralReader{}
	for _, child := range namedChildren(lambda) {
		name, argsNode, destLambda, ok := callWithTrailingLambda(child, src)
		if !ok || name != "composable" {
			continue
		}
		dest := NavDestination{
			StartLine:    int(child.StartPosition().Row) + 1,
			EndLine:      int(child.EndPosition().Row) + 1,
			BodyInsertAt: int(destLambda.StartByte()) + 1,
		}
		if argsNode != nil {
			if args := resolveArgs(argsNode, src); len(args) > 0 {
				if text, isLiteral := lr.StringLiteral(args[0].Node, src); isLiteral {
					dest.Route = text
				} else {
					dest.Dynamic = true
				}
			}
		}
		info.Destinations = append(info.Destinations, dest)
	}
	return info
}

// callWithTrailingLambda decomposes a call_expression that has a
// trailing lambda, with or without parenthesized arguments: either a
// bare `name { ... }` (children: [identifier, annotated_lambda]) or
// `name(args) { ... }` (children: [innerCall(identifier, value_arguments),
// annotated_lambda]).
func callWithTrailingLambda(n *sitter.Node, src []byte) (name string, argsNode, lambda *sitter.Node, ok bool) {
	if n.Kind() != "call_expression" {
		return "", nil, nil, false
	}
	named := namedChildren(n)
	if len(named) != 2 {
		return "", nil, nil, false
	}
	lambda = lambdaLiteral(named[1])
	if lambda == nil {
		return "", nil, nil, false
	}

	switch callee := named[0]; callee.Kind() {
	case "identifier":
		return callee.Utf8Text(src), nil, lambda, true
	case "call_expression":
		inner := namedChildren(callee)
		if len(inner) == 0 || inner[0].Kind() != "identifier" {
			return "", nil, nil, false
		}
		return inner[0].Utf8Text(src), findChildKind(callee, "value_arguments"), lambda, true
	default:
		return "", nil, nil, false
	}
}
