package kotlin

import "testing"

func TestLocateClassExistingMethod(t *testing.T) {
	src := `class App : Application() {
    override fun onCreate() {
        super.onCreate()
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	bodyAt, methods, found := LocateClass(tree, []byte(src), "App")
	if !found {
		t.Fatal("LocateClass() found = false, want true")
	}
	if src[bodyAt-1] != '{' {
		t.Errorf("bodyAt = %d points at %q, want right after class '{'", bodyAt, string(src[bodyAt-1]))
	}
	onCreateAt, ok := methods["onCreate"]
	if !ok {
		t.Fatal("onCreate not found in methods map")
	}
	if src[onCreateAt-1] != '{' {
		t.Errorf("onCreateAt = %d points at %q, want right after onCreate's '{'", onCreateAt, string(src[onCreateAt-1]))
	}

	// Sanity: inserting right at onCreateAt lands before super.onCreate().
	after := src[onCreateAt:]
	want := "\n        super.onCreate()"
	if len(after) < len(want) || after[:len(want)] != want {
		t.Errorf("content after onCreateAt = %q, want prefix %q", after, want)
	}
}

func TestLocateClassEmptyBody(t *testing.T) {
	src := `class App : Application() {
}
`
	tree := parse(t, src)
	defer tree.Close()

	bodyAt, methods, found := LocateClass(tree, []byte(src), "App")
	if !found {
		t.Fatal("LocateClass() found = false, want true")
	}
	if len(methods) != 0 {
		t.Errorf("methods = %v, want none", methods)
	}
	if src[bodyAt-1] != '{' {
		t.Errorf("bodyAt = %d points at %q, want right after class '{'", bodyAt, string(src[bodyAt-1]))
	}
}

func TestLocateClassNotFound(t *testing.T) {
	src := `class Other {}`
	tree := parse(t, src)
	defer tree.Close()

	_, _, found := LocateClass(tree, []byte(src), "App")
	if found {
		t.Error("LocateClass() found = true, want false for a missing class")
	}
}
