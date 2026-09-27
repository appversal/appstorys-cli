package cli

import (
	"path/filepath"
	"testing"

	"github.com/appversal/appstorys-cli/internal/lang/androidxml"
)

func TestPlanInitAndroidAGPNamespace(t *testing.T) {
	got := planFor(t, "../../testdata/fixtures/android/init-agp-namespace")

	var newAppFile string
	for _, s := range got {
		if s.Kind == "init" {
			// The first suggestion should be the new App.kt, whose path
			// must be derived from the build.gradle.kts namespace
			// ("com.example.agpns"), not left empty or wrong, since the
			// manifest has no package attribute at all.
			newAppFile = s.File
			break
		}
	}
	want := "app/src/main/java/com/example/agpns/App.kt"
	if newAppFile != want {
		t.Errorf("new Application class File = %q, want %q (all suggestions: %+v)", newAppFile, want, got)
	}
}

func TestDerivePackageNeitherManifestNorNamespace(t *testing.T) {
	// No manifest package AND no app/build.gradle.kts at all: should
	// fail clearly rather than guess a package.
	root := t.TempDir()
	manifestPath := filepath.Join("app", "src", "main", "AndroidManifest.xml")

	_, err := derivePackage(root, manifestPath, androidxml.ManifestInfo{})
	if err == nil {
		t.Fatal("derivePackage() error = nil, want an error when neither source is available")
	}
}
