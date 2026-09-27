package ts

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ComponentInfo is one top-level function or arrow-function component
// declaration: `function Foo() {...}`, `const Foo = () => {...}`, or
// `const Foo = function () {...}`.
type ComponentInfo struct {
	Name string
	File string
	Line int
	// ReturnJSXStart/End is the byte range of the single top-level JSX
	// expression this component returns (unwrapping a surrounding
	// parenthesized_expression), when its return has that simple shape
	// — a direct `return <X/>` / `() => <X/>`, not a conditional,
	// fragment, or multiple return statements. Both zero otherwise:
	// wrapping that JSX (for integrate) isn't attempted when the shape
	// isn't simple enough to be confident about.
	ReturnJSXStart int
	ReturnJSXEnd   int
}

// Components finds every top-level function/arrow-function component
// declaration in tree, regardless of export wrapping (an
// `export default function Foo() {...}` is still found, since the walk
// descends into export_statement like any other node).
func Components(tree *sitter.Tree, src []byte, file string) []ComponentInfo {
	var out []ComponentInfo
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "function_declaration":
			if name := n.ChildByFieldName("name"); name != nil {
				out = append(out, buildComponentInfo(name.Utf8Text(src), n, file))
			}
		case "variable_declarator":
			name := n.ChildByFieldName("name")
			val := n.ChildByFieldName("value")
			if name != nil && val != nil && (val.Kind() == "arrow_function" || val.Kind() == "function_expression") {
				out = append(out, buildComponentInfo(name.Utf8Text(src), val, file))
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return out
}

func buildComponentInfo(name string, fnNode *sitter.Node, file string) ComponentInfo {
	pos := fnNode.StartPosition()
	ci := ComponentInfo{Name: name, File: file, Line: int(pos.Row) + 1}

	body := fnNode.ChildByFieldName("body")
	if body == nil {
		return ci
	}

	var expr *sitter.Node
	if body.Kind() == "statement_block" {
		var returns []*sitter.Node
		for _, c := range namedChildren(body) {
			if c.Kind() == "return_statement" {
				returns = append(returns, c)
			}
		}
		if len(returns) != 1 {
			return ci
		}
		named := namedChildren(returns[0])
		if len(named) != 1 {
			return ci
		}
		expr = named[0]
	} else {
		expr = body // arrow function with a direct expression body
	}

	for expr.Kind() == "parenthesized_expression" {
		named := namedChildren(expr)
		if len(named) != 1 {
			return ci
		}
		expr = named[0]
	}

	if expr.Kind() != "jsx_element" && expr.Kind() != "jsx_self_closing_element" {
		return ci
	}
	ci.ReturnJSXStart = int(expr.StartByte())
	ci.ReturnJSXEnd = int(expr.EndByte())
	return ci
}

// AppRegistryComponent finds AppRegistry.registerComponent(appName, X)
// and resolves X to a component name, whether X is a bare identifier
// reference or an arrow function returning one (() => App).
func AppRegistryComponent(tree *sitter.Tree, src []byte) (string, bool) {
	var result string
	var found bool
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if found || n.Kind() != "call_expression" {
			if !found {
				for i := range n.ChildCount() {
					walk(n.Child(i))
				}
			}
			return
		}
		fn := n.ChildByFieldName("function")
		if fn != nil {
			if parts := resolveParts(fn, src); len(parts) == 2 && parts[0] == "AppRegistry" && parts[1] == "registerComponent" {
				if argsNode := n.ChildByFieldName("arguments"); argsNode != nil {
					args := namedChildren(argsNode)
					if len(args) >= 2 {
						switch second := args[1]; second.Kind() {
						case "identifier":
							result, found = second.Utf8Text(src), true
						case "arrow_function":
							if body := second.ChildByFieldName("body"); body != nil && body.Kind() == "identifier" {
								result, found = body.Utf8Text(src), true
							}
						}
					}
				}
			}
		}
		if !found {
			for i := range n.ChildCount() {
				walk(n.Child(i))
			}
		}
	}
	walk(tree.RootNode())
	return result, found
}

// NavRoute is one `<X.Screen name="..." component={Y} />` route
// registration (React Navigation's JSX form). The object-config form
// (createXNavigator({ screens: {...} })) and the render-prop children
// form aren't matched — documented gap, not a silent miss: doctor
// treats an unresolved screen as heuristic rather than guessing.
type NavRoute struct {
	RouteName     string // from the "name" attribute; "" if not a literal
	ComponentName string // from the "component" attribute; "" if not a simple identifier
	Line          int
}

// NavigatorScreens finds every `<*.Screen ...>` JSX element (matching
// React Navigation's Stack.Screen/Tab.Screen/Drawer.Screen convention:
// any member expression whose property is "Screen") with a `component`
// attribute resolving to a plain identifier.
func NavigatorScreens(tree *sitter.Tree, src []byte) []NavRoute {
	lr := LiteralReader{}
	var routes []NavRoute
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "jsx_opening_element", "jsx_self_closing_element":
			if route, ok := buildNavRoute(n, src, lr); ok {
				routes = append(routes, route)
			}
		}
		for i := range n.ChildCount() {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return routes
}

func buildNavRoute(n *sitter.Node, src []byte, lr LiteralReader) (NavRoute, bool) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil || nameNode.Kind() != "member_expression" {
		return NavRoute{}, false
	}
	prop := nameNode.ChildByFieldName("property")
	if prop == nil || prop.Utf8Text(src) != "Screen" {
		return NavRoute{}, false
	}

	route := NavRoute{Line: int(n.StartPosition().Row) + 1}
	cursor := n.Walk()
	attrs := n.ChildrenByFieldName("attribute", cursor)
	cursor.Close()

	for i := range attrs {
		attr := &attrs[i]
		named := namedChildren(attr)
		if len(named) != 2 {
			continue
		}
		switch named[0].Utf8Text(src) {
		case "name":
			if text, ok := lr.StringLiteral(named[1], src); ok {
				route.RouteName = text
			}
		case "component":
			val := named[1]
			if val.Kind() == "jsx_expression" {
				if inner := namedChildren(val); len(inner) == 1 && inner[0].Kind() == "identifier" {
					route.ComponentName = inner[0].Utf8Text(src)
				}
			}
		}
	}
	if route.ComponentName == "" {
		return NavRoute{}, false
	}
	return route, true
}

// StaticNavigatorScreens finds every React Navigation static-API
// screen registration — createNativeStackNavigator({ screens: { Home:
// HomeScreen, ... } }), createBottomTabNavigator, createDrawerNavigator
// and so on — matched by naming convention (any call starting with
// "create" and ending in "Navigator"), not an exhaustive list of
// factory names, so a new one from the same family is picked up without
// a code change.
//
// Only a direct `Name: Component` (or `'Name': Component`) entry
// resolves to a route: a screen whose config is itself an object (for
// options like `screen: Component, options: {...}`) isn't matched —
// documented gap, not a silent miss.
func StaticNavigatorScreens(tree *sitter.Tree, src []byte) []NavRoute {
	var routes []NavRoute
	for _, call := range Extract(tree, src) {
		if len(call.Parts) != 1 || !isNavigatorFactory(call.Parts[0]) {
			continue
		}
		if len(call.Args) == 0 || call.Args[0].Node == nil || call.Args[0].Node.Kind() != "object" {
			continue
		}
		screens := findScreensProperty(call.Args[0].Node, src)
		if screens == nil {
			continue
		}
		for _, pair := range namedChildren(screens) {
			if pair.Kind() != "pair" {
				continue
			}
			keyNode := pair.ChildByFieldName("key")
			valNode := pair.ChildByFieldName("value")
			if keyNode == nil || valNode == nil || valNode.Kind() != "identifier" {
				continue
			}
			routeName := propertyKeyText(keyNode, src)
			if routeName == "" {
				continue
			}
			routes = append(routes, NavRoute{
				RouteName:     routeName,
				ComponentName: valNode.Utf8Text(src),
				Line:          int(pair.StartPosition().Row) + 1,
			})
		}
	}
	return routes
}

func isNavigatorFactory(name string) bool {
	return strings.HasPrefix(name, "create") && strings.HasSuffix(name, "Navigator")
}

func findScreensProperty(obj *sitter.Node, src []byte) *sitter.Node {
	for _, pair := range namedChildren(obj) {
		if pair.Kind() != "pair" {
			continue
		}
		keyNode := pair.ChildByFieldName("key")
		valNode := pair.ChildByFieldName("value")
		if keyNode == nil || valNode == nil {
			continue
		}
		if propertyKeyText(keyNode, src) == "screens" && valNode.Kind() == "object" {
			return valNode
		}
	}
	return nil
}

func propertyKeyText(keyNode *sitter.Node, src []byte) string {
	switch keyNode.Kind() {
	case "property_identifier":
		return keyNode.Utf8Text(src)
	case "string":
		if text, ok := (LiteralReader{}).StringLiteral(keyNode, src); ok {
			return text
		}
	}
	return ""
}
