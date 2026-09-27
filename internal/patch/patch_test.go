package patch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsert(t *testing.T) {
	got := Insert([]byte("abcdef"), 3, "XYZ")
	if string(got) != "abcXYZdef" {
		t.Errorf("Insert() = %q, want %q", got, "abcXYZdef")
	}
}

func TestRenderExistingFile(t *testing.T) {
	original := []byte("line1\nline2\nline3\n")
	at := len("line1\n") // right before line2
	edit := Edit{File: "a.txt", InsertAt: at, Snippet: "inserted\n"}

	diff := Render(edit, original)
	if !strings.Contains(diff, "+inserted") {
		t.Errorf("diff missing inserted line: %s", diff)
	}
	if !strings.Contains(diff, "--- a/a.txt") || !strings.Contains(diff, "+++ b/a.txt") {
		t.Errorf("diff missing file headers: %s", diff)
	}
}

func TestRenderMidLineNoTrailingNewline(t *testing.T) {
	original := []byte(`<application android:icon="x">
</application>
`)
	at := len(`<application`)
	edit := Edit{File: "AndroidManifest.xml", InsertAt: at, Snippet: ` android:name=".App"`}

	diff := Render(edit, original) // must not panic
	if !strings.Contains(diff, `+<application android:name=".App" android:icon="x">`) {
		t.Errorf("diff missing expected merged line: %s", diff)
	}
	if !strings.Contains(diff, `-<application android:icon="x">`) {
		t.Errorf("diff missing removed original line: %s", diff)
	}
}

func TestRenderNewFile(t *testing.T) {
	edit := Edit{File: "New.kt", NewFile: true, Snippet: "class New {}\n"}
	diff := Render(edit, nil)
	if !strings.Contains(diff, "@@ -0,0 +1,1 @@") {
		t.Errorf("diff missing new-file hunk header: %s", diff)
	}
	if !strings.Contains(diff, "+class New {}") {
		t.Errorf("diff missing content line: %s", diff)
	}
}

func TestApplyNewFile(t *testing.T) {
	dir := t.TempDir()
	edit := Edit{File: "sub/New.kt", NewFile: true, Snippet: "class New {}\n"}
	if err := Apply(dir, edit); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "sub/New.kt"))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(data) != edit.Snippet {
		t.Errorf("written content = %q, want %q", data, edit.Snippet)
	}
}

func TestApplyExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("line1\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	edit := Edit{File: "a.txt", InsertAt: len("line1\n"), Snippet: "line2\n"}
	if err := Apply(dir, edit); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\nline2\nline3\n" {
		t.Errorf("content = %q, want %q", data, "line1\nline2\nline3\n")
	}
}

func TestGitCleanNotARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := GitClean(dir); err == nil {
		t.Error("GitClean() on a non-repo dir expected an error, got nil")
	}
}
