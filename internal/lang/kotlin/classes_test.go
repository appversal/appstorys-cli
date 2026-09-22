package kotlin

import "testing"

func TestClasses(t *testing.T) {
	src := `
class App : android.app.Application() {
    override fun onCreate() {}
}

class HomeActivity : AppCompatActivity() {
}

class PlainClass {
}
`
	tree := parse(t, src)
	defer tree.Close()

	classes := Classes(tree, []byte(src), "App.kt")
	if len(classes) != 3 {
		t.Fatalf("Classes() = %d, want 3: %+v", len(classes), classes)
	}

	byName := map[string]ClassInfo{}
	for _, c := range classes {
		byName[c.Name] = c
	}

	if got := byName["App"].Supertypes; len(got) != 1 || got[0] != "Application" {
		t.Errorf("App supertypes = %v, want [Application]", got)
	}
	if got := byName["HomeActivity"].Supertypes; len(got) != 1 || got[0] != "AppCompatActivity" {
		t.Errorf("HomeActivity supertypes = %v, want [AppCompatActivity]", got)
	}
	if got := byName["PlainClass"].Supertypes; len(got) != 0 {
		t.Errorf("PlainClass supertypes = %v, want none", got)
	}
}
