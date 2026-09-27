package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/suggest"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// blockSnippet builds a snippet to insert right before a LocateBlock
// result (the position of a block's closing "}"): body, then a newline
// and whitespace matching that "}"'s own indentation, so the brace ends
// up back on its own line instead of glued onto the inserted content.
func blockSnippet(data []byte, at int, body string) string {
	lineStart := bytes.LastIndexByte(data[:at], '\n') + 1
	end := lineStart
	for end < len(data) && (data[end] == ' ' || data[end] == '\t') {
		end++
	}
	return body + "\n" + string(data[lineStart:end])
}

// planGradleAndroid figures out the Gradle-side wiring init needs: the
// JitPack repo in the root settings.gradle.kts, and the SDK dependency +
// buildConfigField + buildFeatures.buildConfig in the app module's
// build.gradle.kts. manifestPath locates the app module (its
// grandparent directory, by the src/main/AndroidManifest.xml
// convention).
//
// Only .gradle.kts files are touched: a plain .gradle (Groovy DSL) file
// isn't parsed by the Kotlin grammar and would silently misdetect
// blocks if we tried, so it's left as a manual step instead — a wrong
// Gradle edit breaks the build, which is a much worse failure mode than
// not proposing one.
func planGradleAndroid(app *App, manifestPath string) ([]suggest.Suggestion, error) {
	sdkVersion := "5.0.2"
	if m, err := symbols.LoadPlatform("android"); err == nil {
		sdkVersion = m.SDKVersion
	}

	var suggestions []suggest.Suggestion

	if s, err := planJitpackRepo(app); err != nil {
		return nil, err
	} else if s != nil {
		suggestions = append(suggestions, *s)
	}

	moduleDir := filepath.Dir(filepath.Dir(filepath.Dir(manifestPath))) // .../src/main/AndroidManifest.xml -> module dir
	moduleGradle := filepath.Join(moduleDir, "build.gradle.kts")
	if !fileExistsAt(app.Root, moduleGradle) {
		return suggestions, nil // Groovy build.gradle, or none at all: manual step
	}

	more, err := planAppModuleGradle(app, moduleGradle, sdkVersion)
	if err != nil {
		return nil, err
	}
	return append(suggestions, more...), nil
}

func planJitpackRepo(app *App) (*suggest.Suggestion, error) {
	const settingsFile = "settings.gradle.kts"
	if !fileExistsAt(app.Root, settingsFile) {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(app.Root, settingsFile))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", settingsFile, err)
	}
	if strings.Contains(strings.ToLower(string(data)), "jitpack") {
		return nil, nil // already there
	}

	tree, closeFn, err := parseKotlinFile(data)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	const jitpackLine = "\n        maven { url = uri(\"https://jitpack.io\") }"
	if at, ok := kotlin.LocateBlock(tree, data, "dependencyResolutionManagement", "repositories"); ok {
		return &suggest.Suggestion{
			ID:         suggest.StableID("init", settingsFile, "jitpack-repo"),
			Kind:       "init",
			File:       settingsFile,
			InsertAt:   at,
			Snippet:    blockSnippet(data, at, jitpackLine),
			Reason:     "Add the JitPack repository (dependencyResolutionManagement.repositories)",
			Confidence: 0.85,
		}, nil
	}
	if at, ok := kotlin.LocateBlock(tree, data, "repositories"); ok {
		return &suggest.Suggestion{
			ID:         suggest.StableID("init", settingsFile, "jitpack-repo"),
			Kind:       "init",
			File:       settingsFile,
			InsertAt:   at,
			Snippet:    blockSnippet(data, at, jitpackLine),
			Reason:     "Add the JitPack repository (repositories)",
			Confidence: 0.85,
		}, nil
	}
	return nil, nil // no repositories block found: leave it as a manual step
}

func planAppModuleGradle(app *App, moduleGradle, sdkVersion string) ([]suggest.Suggestion, error) {
	data, err := os.ReadFile(filepath.Join(app.Root, moduleGradle))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", moduleGradle, err)
	}
	content := string(data)

	tree, closeFn, err := parseKotlinFile(data)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	var suggestions []suggest.Suggestion

	if !strings.Contains(strings.ToLower(content), "appstorys") {
		if at, ok := kotlin.LocateBlock(tree, data, "dependencies"); ok {
			body := fmt.Sprintf("\n    implementation(\"com.github.appversal:AppStorys-Android-SDK-Downgraded:%s\")", sdkVersion)
			suggestions = append(suggestions, suggest.Suggestion{
				ID:         suggest.StableID("init", moduleGradle, "sdk-dependency"),
				Kind:       "init",
				File:       moduleGradle,
				InsertAt:   at,
				Snippet:    blockSnippet(data, at, body),
				Reason:     "Add the AppStorys SDK dependency",
				Confidence: 0.85,
			})
		}
	}

	if !strings.Contains(content, "buildConfig = true") && !strings.Contains(content, "buildConfig=true") {
		if at, ok := kotlin.LocateBlock(tree, data, "android", "buildFeatures"); ok {
			suggestions = append(suggestions, suggest.Suggestion{
				ID:         suggest.StableID("init", moduleGradle, "buildconfig-flag-existing"),
				Kind:       "init",
				File:       moduleGradle,
				InsertAt:   at,
				Snippet:    blockSnippet(data, at, "\n        buildConfig = true"),
				Reason:     "Enable buildConfig generation (android.buildFeatures)",
				Confidence: 0.85,
			})
		} else if at, ok := kotlin.LocateBlock(tree, data, "android"); ok {
			suggestions = append(suggestions, suggest.Suggestion{
				ID:         suggest.StableID("init", moduleGradle, "buildconfig-flag-new-block"),
				Kind:       "init",
				File:       moduleGradle,
				InsertAt:   at,
				Snippet:    blockSnippet(data, at, "\n    buildFeatures {\n        buildConfig = true\n    }"),
				Reason:     "Enable buildConfig generation (android.buildFeatures)",
				Confidence: 0.8,
			})
		}
	}

	if !strings.Contains(content, "APPSTORYS_API_TOKEN") {
		if at, ok := kotlin.LocateBlock(tree, data, "android", "defaultConfig"); ok {
			// Kotlin string templates (${...}) are a separate expression
			// context: the nested string literals inside it (the
			// getenv key and the "" default) must NOT be
			// backslash-escaped — only the outer string's own quotes
			// need that, to produce a value that's itself a quoted
			// Kotlin string once buildConfigField expands it.
			body := "\n        buildConfigField(\"String\", \"APPSTORYS_API_TOKEN\", " +
				`"\"${System.getenv("APPSTORYS_API_TOKEN") ?: ""}\""` + ")"
			suggestions = append(suggestions, suggest.Suggestion{
				ID:         suggest.StableID("init", moduleGradle, "buildconfig-field"),
				Kind:       "init",
				File:       moduleGradle,
				InsertAt:   at,
				Snippet:    blockSnippet(data, at, body),
				Reason:     "Add the APPSTORYS_API_TOKEN buildConfigField, read from the environment at build time",
				Confidence: 0.8,
			})
		}
	}

	return suggestions, nil
}

func fileExistsAt(root, rel string) bool {
	info, err := os.Stat(filepath.Join(root, rel))
	return err == nil && !info.IsDir()
}

func parseKotlinFile(data []byte) (tree *sitter.Tree, closeFn func(), err error) {
	pool, err := parse.NewPool(kotlin.Language(), 1)
	if err != nil {
		return nil, nil, err
	}
	t, err := pool.Parse(context.Background(), data)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("parsing: %w", err)
	}
	return t, func() { t.Close(); pool.Close() }, nil
}
