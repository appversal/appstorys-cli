package kotlin

import (
	"strings"
	"testing"
)

func TestLocateBlockTopLevel(t *testing.T) {
	src := `dependencies {
    implementation("com.example:foo:1.0")
}
`
	tree := parse(t, src)
	defer tree.Close()

	at, found := LocateBlock(tree, []byte(src), "dependencies")
	if !found {
		t.Fatal("LocateBlock() found = false, want true")
	}
	if src[at] != '}' {
		t.Errorf("at = %d points at %q, want '}'", at, string(src[at]))
	}
}

func TestLocateBlockNested(t *testing.T) {
	src := `android {
    defaultConfig {
        minSdk = 22
    }
    buildFeatures {
        compose = true
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	at, found := LocateBlock(tree, []byte(src), "android", "defaultConfig")
	if !found {
		t.Fatal("LocateBlock(android, defaultConfig) found = false, want true")
	}
	// Must land inside defaultConfig's braces, not android's or
	// buildFeatures'.
	before := string(src[:at])
	if !strings.Contains(before, "minSdk = 22") {
		t.Errorf("insertion point is before minSdk assignment: %q", before)
	}
	if strings.Contains(before, "compose = true") {
		t.Errorf("insertion point is after buildFeatures' content, should be inside defaultConfig only: %q", before)
	}
}

func TestLocateBlockMissing(t *testing.T) {
	src := `dependencies {
    implementation("x")
}
`
	tree := parse(t, src)
	defer tree.Close()

	if _, found := LocateBlock(tree, []byte(src), "repositories"); found {
		t.Error("LocateBlock(repositories) found = true, want false")
	}
	if _, found := LocateBlock(tree, []byte(src), "dependencies", "nope"); found {
		t.Error("LocateBlock(dependencies, nope) found = true, want false")
	}
}
