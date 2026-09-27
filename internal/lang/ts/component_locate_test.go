package ts

import "testing"

func TestLocateComponentInitExistingUseEffect(t *testing.T) {
	src := `function App() {
  useEffect(() => {
    doSomethingElse();
  }, []);
  return <View />;
}
`
	tree := parse(t, src)
	defer tree.Close()

	at, hasUseEffect, found := LocateComponentInit(tree, []byte(src), "App")
	if !found {
		t.Fatal("LocateComponentInit() found = false, want true")
	}
	if !hasUseEffect {
		t.Error("hasUseEffect = false, want true")
	}
	if src[at-1] != '{' {
		t.Errorf("at = %d points at %q, want right after useEffect's '{'", at, string(src[at-1]))
	}
	after := src[at:]
	if len(after) < len("\n    doSomethingElse") || string(after[:len("\n    doSomethingElse")]) != "\n    doSomethingElse" {
		t.Errorf("content after at = %q, want to precede doSomethingElse()", after)
	}
}

func TestLocateComponentInitNoUseEffect(t *testing.T) {
	src := `function App() {
  return <View />;
}
`
	tree := parse(t, src)
	defer tree.Close()

	at, hasUseEffect, found := LocateComponentInit(tree, []byte(src), "App")
	if !found {
		t.Fatal("LocateComponentInit() found = false, want true")
	}
	if hasUseEffect {
		t.Error("hasUseEffect = true, want false")
	}
	if src[at-1] != '{' {
		t.Errorf("at = %d points at %q, want right after App's '{'", at, string(src[at-1]))
	}
}

func TestLocateComponentInitUseEffectWithDeps(t *testing.T) {
	// useEffect with non-empty deps isn't mount-only: should be treated
	// as "no usable useEffect", not matched.
	src := `function App() {
  useEffect(() => {
    onPropsChanged();
  }, [someProp]);
  return <View />;
}
`
	tree := parse(t, src)
	defer tree.Close()

	_, hasUseEffect, found := LocateComponentInit(tree, []byte(src), "App")
	if !found {
		t.Fatal("LocateComponentInit() found = false, want true")
	}
	if hasUseEffect {
		t.Error("hasUseEffect = true, want false (dependency array is non-empty)")
	}
}

func TestLocateComponentInitArrowComponent(t *testing.T) {
	src := `const App = () => {
  return <View />;
};
`
	tree := parse(t, src)
	defer tree.Close()

	_, _, found := LocateComponentInit(tree, []byte(src), "App")
	if !found {
		t.Error("LocateComponentInit() found = false, want true for a block-bodied arrow component")
	}
}

func TestLocateComponentInitNotFound(t *testing.T) {
	src := `function Other() { return null; }`
	tree := parse(t, src)
	defer tree.Close()

	if _, _, found := LocateComponentInit(tree, []byte(src), "App"); found {
		t.Error("LocateComponentInit() found = true, want false for a missing component")
	}
}
