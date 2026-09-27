package dart

import sitter "github.com/tree-sitter/go-tree-sitter"

// RunAppWidget finds runApp(X) and resolves X to a widget name, whether
// X is a bare identifier or a constructor call (X()).
func RunAppWidget(tree *sitter.Tree, src []byte) (string, bool) {
	for _, call := range Extract(tree, src) {
		if len(call.Parts) != 1 || call.Parts[0] != "runApp" || len(call.Args) == 0 {
			continue
		}
		if arg := call.Args[0].Node; arg != nil && arg.Kind() == "identifier" {
			return arg.Utf8Text(src), true
		}
	}
	return "", false
}

// Route is one route registration, from either MaterialAppRoutes or
// GoRoutes.
type Route struct {
	Path       string
	WidgetName string // "" if the builder's return isn't a simple `(context[, state]) => Widget()` shape
	Line       int
}

// MaterialAppRoutes finds every route registered via
// MaterialApp(routes: { 'path': (context) => Widget(), ... }).
// onGenerateRoute, MaterialPageRoute(builder:) and Navigator.push aren't
// matched — documented gap, not a silent miss: doctor and integrate
// simply won't see routes registered those ways, rather than guessing.
// go_router's GoRoute is matched separately, by GoRoutes.
func MaterialAppRoutes(tree *sitter.Tree, src []byte) []Route {
	var routes []Route
	for _, call := range Extract(tree, src) {
		if len(call.Parts) != 1 || call.Parts[0] != "MaterialApp" {
			continue
		}
		for _, arg := range call.Args {
			if arg.Name == "routes" {
				routes = append(routes, routesFromMapLiteral(arg.Node, src)...)
			}
		}
	}
	return routes
}

func routesFromMapLiteral(n *sitter.Node, src []byte) []Route {
	if n == nil || n.Kind() != "set_or_map_literal" {
		return nil
	}
	lr := LiteralReader{}
	var routes []Route
	for _, item := range namedChildren(n) {
		if item.Kind() != "pair" {
			continue
		}
		keyNode := item.ChildByFieldName("key")
		valNode := item.ChildByFieldName("value")
		if keyNode == nil || valNode == nil {
			continue
		}
		path, ok := lr.StringLiteral(keyNode, src)
		if !ok {
			continue
		}
		routes = append(routes, Route{
			Path:       path,
			WidgetName: widgetFromBuilder(valNode, src),
			Line:       int(item.StartPosition().Row) + 1,
		})
	}
	return routes
}

// GoRoutes finds every `GoRoute(path: '...', builder: (context, state)
// => Widget())` call, from the go_router package. Extract already walks
// the whole tree and matches call/constructor shapes generically, so a
// GoRoute call is found the same way regardless of how deep it's
// nested — including as a sub-route in another GoRoute's own `routes:`
// list, which is how go_router expresses nested navigation. That means
// nested routes are picked up for free, with no special-casing.
//
// A GoRoute whose builder isn't the simple `(context, state) =>
// Widget()` shape (for example ShellRoute, or a builder with a
// conditional body) resolves to an empty WidgetName and is skipped by
// callers the same way an unresolved MaterialApp route builder is.
func GoRoutes(tree *sitter.Tree, src []byte) []Route {
	var routes []Route
	for _, call := range Extract(tree, src) {
		if len(call.Parts) != 1 || call.Parts[0] != "GoRoute" {
			continue
		}
		var route Route
		var hasPath bool
		lr := LiteralReader{}
		for _, arg := range call.Args {
			switch arg.Name {
			case "path":
				if text, ok := lr.StringLiteral(arg.Node, src); ok {
					route.Path = text
					hasPath = true
				}
			case "builder":
				route.WidgetName = widgetFromBuilder(arg.Node, src)
			}
		}
		if !hasPath {
			continue
		}
		if call.Node != nil {
			route.Line = int(call.Node.StartPosition().Row) + 1
		}
		routes = append(routes, route)
	}
	return routes
}

// StackChildrenInsertPoint finds a `Stack(children: [...])` call
// anywhere in tree and returns the byte offset right before its
// children list's closing "]", so callers can append
// `...AppStorys.overlayElements(),`. This only looks for an
// already-present Stack — per the spec, wrapping a screen's body in a
// new Stack when one is absent is "diff only, never auto-apply", and
// generating that transformation isn't attempted at all here rather
// than half-implementing the "diff but refuse apply" distinction: a
// screen with no Stack simply gets no overlay-host suggestion, left to
// doctor's report instead.
func StackChildrenInsertPoint(tree *sitter.Tree, src []byte) (insertAt int, found bool) {
	for _, call := range Extract(tree, src) {
		if len(call.Parts) != 1 || call.Parts[0] != "Stack" {
			continue
		}
		for _, arg := range call.Args {
			if arg.Name == "children" && arg.Node != nil && arg.Node.Kind() == "list_literal" {
				return int(arg.Node.EndByte()) - 1, true
			}
		}
	}
	return 0, false
}

// widgetFromBuilder reads `(context) => Widget()`'s returned widget
// name from a route map value.
func widgetFromBuilder(n *sitter.Node, src []byte) string {
	if n.Kind() != "function_expression" {
		return ""
	}
	body := n.ChildByFieldName("body")
	if body == nil || body.Kind() != "function_expression_body" {
		return ""
	}
	named := namedChildren(body)
	if len(named) == 0 || named[0].Kind() != "identifier" {
		return ""
	}
	return named[0].Utf8Text(src)
}
