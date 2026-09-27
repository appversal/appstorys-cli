package dart

import sitter "github.com/tree-sitter/go-tree-sitter"

// LocateClassBody returns the byte offset right after className's
// class_body's opening "{", for inserting a new member.
func LocateClassBody(tree *sitter.Tree, src []byte, className string) (insertAt int, found bool) {
	classNode := findClassNode(tree, src, className)
	if classNode == nil {
		return 0, false
	}
	body := findChildKind(classNode, "class_body")
	if body == nil {
		return 0, false
	}
	return int(body.StartByte()) + 1, true
}

// LocateClassMethod finds className's method named methodName and
// returns the byte offset right after its block body's opening "{", for
// inserting a first statement. found is false if the class or method
// doesn't exist, or the method has no block body (an arrow "=>" method
// like createState() can't host an inserted statement this way).
//
// This grammar splits a method into two sibling nodes within class_body
// — method_signature (with the name) and a separate function_body (the
// block) — rather than nesting the body inside the signature, the same
// split internal/lang/dart's own enclosingName already accounts for at
// the call-site level; this is the class-body-level counterpart.
func LocateClassMethod(tree *sitter.Tree, src []byte, className, methodName string) (insertAt int, found bool) {
	classNode := findClassNode(tree, src, className)
	if classNode == nil {
		return 0, false
	}
	body := findChildKind(classNode, "class_body")
	if body == nil {
		return 0, false
	}

	pendingName := ""
	for _, child := range namedChildren(body) {
		switch child.Kind() {
		case "method_signature":
			pendingName = methodSignatureName(child, src)
		case "function_body":
			if pendingName == methodName {
				if block := findChildKind(child, "block"); block != nil {
					return int(block.StartByte()) + 1, true
				}
				return 0, false
			}
			pendingName = ""
		}
	}
	return 0, false
}

// LocateTopLevelFunction finds a top-level function (e.g. main) by name
// and returns the byte offset right after its block body's opening
// "{". Top-level functions have the same signature/body sibling split
// as class methods, just directly under the source file's root instead
// of a class_body.
func LocateTopLevelFunction(tree *sitter.Tree, src []byte, name string) (insertAt int, found bool) {
	pendingName := ""
	for _, child := range namedChildren(tree.RootNode()) {
		switch child.Kind() {
		case "function_signature":
			if n := child.ChildByFieldName("name"); n != nil {
				pendingName = n.Utf8Text(src)
			}
		case "function_body":
			if pendingName == name {
				if block := findChildKind(child, "block"); block != nil {
					return int(block.StartByte()) + 1, true
				}
				return 0, false
			}
			pendingName = ""
		}
	}
	return 0, false
}

func methodSignatureName(methodSig *sitter.Node, src []byte) string {
	fnSig := findChildKind(methodSig, "function_signature")
	if fnSig == nil {
		return ""
	}
	if name := fnSig.ChildByFieldName("name"); name != nil {
		return name.Utf8Text(src)
	}
	return ""
}
