package kotlin

import (
	"strings"
	"testing"
)

func TestLocateClassMethodBlock(t *testing.T) {
	src := `class HomeActivity {
    fun onCreate() {
        setContent {
            HomeScreen()
        }
    }
}

class SettingsActivity {
    fun onCreate() {
        setContent {
            SettingsScreen()
        }
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	at, found := LocateClassMethodBlock(tree, []byte(src), "SettingsActivity", "onCreate", "setContent")
	if !found {
		t.Fatal("LocateClassMethodBlock() found = false, want true")
	}
	if src[at] != '}' {
		t.Errorf("at = %d points at %q, want '}'", at, string(src[at]))
	}
	before := string(src[:at])
	if !strings.Contains(before, "SettingsScreen()") {
		t.Errorf("insertion point is after SettingsActivity's own content: %q", before)
	}
	if !strings.Contains(before, "HomeScreen()") {
		t.Errorf("sanity check failed: HomeActivity (earlier in the file) should precede the insertion point: %q", before)
	}
}

func TestLocateClassMethodBlockWrongClass(t *testing.T) {
	src := `class Other {
    fun onCreate() {
        setContent { Foo() }
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	if _, found := LocateClassMethodBlock(tree, []byte(src), "NotThere", "onCreate", "setContent"); found {
		t.Error("LocateClassMethodBlock() found = true, want false for a missing class")
	}
	if _, found := LocateClassMethodBlock(tree, []byte(src), "Other", "onResume", "setContent"); found {
		t.Error("LocateClassMethodBlock() found = true, want false for a missing method")
	}
}

func TestLocateClassAnyMethodBlock(t *testing.T) {
	src := `class HomeActivity {
    fun onCreate() {
        super.onCreate()
        setupContent()
    }

    private fun setupContent() {
        setContent {
            HomeScreen()
        }
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	at, found := LocateClassAnyMethodBlock(tree, []byte(src), "HomeActivity", "setContent")
	if !found {
		t.Fatal("LocateClassAnyMethodBlock() found = false, want true (setContent is in a helper method)")
	}
	if src[at] != '}' {
		t.Errorf("at = %d points at %q, want '}'", at, string(src[at]))
	}
	before := string(src[:at])
	if !strings.Contains(before, "HomeScreen()") {
		t.Errorf("insertion point is before setContent's own content: %q", before)
	}
}

func TestLocateClassAnyMethodBlockMissing(t *testing.T) {
	src := `class Other {
    fun onCreate() {
        super.onCreate()
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	if _, found := LocateClassAnyMethodBlock(tree, []byte(src), "NotThere", "setContent"); found {
		t.Error("LocateClassAnyMethodBlock() found = true, want false for a missing class")
	}
	if _, found := LocateClassAnyMethodBlock(tree, []byte(src), "Other", "setContent"); found {
		t.Error("LocateClassAnyMethodBlock() found = true, want false when no method calls setContent")
	}
}

func TestFindStringAssignment(t *testing.T) {
	src := `android {
    namespace = "com.example.app"
    compileSdk = 34
}
`
	tree := parse(t, src)
	defer tree.Close()

	block, ok := FindBlockNode(tree, []byte(src), "android")
	if !ok {
		t.Fatal("FindBlockNode(android) found = false, want true")
	}
	ns, ok := FindStringAssignment(block, []byte(src), "namespace")
	if !ok || ns != "com.example.app" {
		t.Errorf("FindStringAssignment(namespace) = (%q, %v), want (\"com.example.app\", true)", ns, ok)
	}
	if _, ok := FindStringAssignment(block, []byte(src), "missing"); ok {
		t.Error("FindStringAssignment(missing) found = true, want false")
	}
}
