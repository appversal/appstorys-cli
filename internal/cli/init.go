package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/androidxml"
	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// planInitAndroid figures out what's missing for Android's init step —
// an Application subclass, its manifest registration, and the
// AppStorys.initialize(...) call itself — and returns the Suggestions
// needed to fix it.
//
// Dependency wiring (the JitPack repo in settings.gradle.kts, the SDK
// dependency and buildConfigField in the app module's build.gradle.kts)
// isn't generated yet: reliably inserting into those Gradle Kotlin DSL
// blocks needs trailing-lambda call detection this phase's Kotlin
// extractor doesn't have (it only handles parenthesized call
// arguments), and getting that wrong risks a broken build file — a much
// worse outcome than just not proposing the edit yet.
func planInitAndroid(app *App, sites []extract.CallSite, classes []kotlin.ClassInfo) ([]suggest.Suggestion, error) {
	manifestPath, err := findManifest(app.Root)
	if err != nil {
		return nil, err
	}
	if manifestPath == "" {
		return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf("AndroidManifest.xml not found under %s", app.Root)}
	}
	manifestData, err := os.ReadFile(filepath.Join(app.Root, manifestPath))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
	}
	manifest, err := androidxml.ParseManifest(manifestData)
	if err != nil {
		return nil, err
	}

	tokenExpr := app.Config.Init.TokenExpr
	if tokenExpr == "" {
		tokenExpr = "BuildConfig.APPSTORYS_API_TOKEN"
	}

	registeredClass := androidxml.SimpleName(manifest.ApplicationName)
	var appClass *kotlin.ClassInfo
	for i := range classes {
		if !hasSupertype(classes[i], "Application") {
			continue
		}
		if classes[i].Name == registeredClass {
			appClass = &classes[i]
			break
		}
		if appClass == nil {
			appClass = &classes[i]
		}
	}

	var suggestions []suggest.Suggestion

	if appClass == nil {
		file, content, err := newApplicationClassFile(app.Root, manifestPath, manifest, tokenExpr)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggest.Suggestion{
			ID:         suggest.StableID("init", file, "new-application-class"),
			Kind:       "init",
			File:       file,
			Snippet:    content,
			Reason:     "No Application subclass found; creating one with the AppStorys init call",
			Confidence: 0.9,
			Requires:   []string{"register in manifest"},
		})
		suggestions = append(suggestions, manifestRegisterSuggestion(manifestPath, manifestData, "App"))
		gradle, err := planGradleAndroid(app, manifestPath)
		if err != nil {
			return nil, err
		}
		return append(suggestions, gradle...), nil
	}

	if registeredClass != appClass.Name {
		suggestions = append(suggestions, manifestRegisterSuggestion(manifestPath, manifestData, appClass.Name))
	}

	var hasInitSomewhere bool
	for _, s := range sites {
		if s.Kind == "init" {
			hasInitSomewhere = true
			break
		}
	}
	if !hasInitSomewhere {
		s, err := initCallSuggestion(app.Root, *appClass, tokenExpr)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}

	gradle, err := planGradleAndroid(app, manifestPath)
	if err != nil {
		return nil, err
	}
	return append(suggestions, gradle...), nil
}

func hasSupertype(c kotlin.ClassInfo, name string) bool {
	for _, s := range c.Supertypes {
		if s == name {
			return true
		}
	}
	return false
}

func manifestRegisterSuggestion(manifestPath string, manifestData []byte, className string) suggest.Suggestion {
	insertAt := 0
	if idx := bytes.Index(manifestData, []byte("<application")); idx >= 0 {
		insertAt = idx + len("<application")
	}
	return suggest.Suggestion{
		ID:         suggest.StableID("init", manifestPath, "register-application"),
		Kind:       "init",
		File:       manifestPath,
		InsertAt:   insertAt,
		Snippet:    fmt.Sprintf(` android:name=".%s"`, className),
		Reason:     fmt.Sprintf("Register %s as the app's Application class in AndroidManifest.xml", className),
		Confidence: 0.9,
	}
}

func newApplicationClassFile(root, manifestPath string, manifest androidxml.ManifestInfo, tokenExpr string) (file, content string, err error) {
	pkg, err := derivePackage(root, manifestPath, manifest)
	if err != nil {
		return "", "", err
	}

	manifestDir := filepath.Dir(manifestPath)
	srcDir := filepath.Join(manifestDir, "java")
	if _, statErr := os.Stat(filepath.Join(root, srcDir)); statErr != nil {
		if _, kErr := os.Stat(filepath.Join(root, manifestDir, "kotlin")); kErr == nil {
			srcDir = filepath.Join(manifestDir, "kotlin")
		}
	}
	packagePath := strings.ReplaceAll(pkg, ".", "/")
	file = filepath.Join(srcDir, packagePath, "App.kt")

	content = fmt.Sprintf(`package %s

import android.app.Application
import com.appversal.appstorys.AppStorys

class App : Application() {
    override fun onCreate() {
        super.onCreate()
        AppStorys.initialize(token = %s)
    }
}
`, pkg, tokenExpr)
	return file, content, nil
}

// derivePackage resolves the app's package name: the manifest's
// package attribute if present, otherwise (newer AGP projects) the
// `namespace = "..."` assignment inside the app module's
// android { ... } block in build.gradle.kts.
func derivePackage(root, manifestPath string, manifest androidxml.ManifestInfo) (string, error) {
	if manifest.Package != "" {
		return manifest.Package, nil
	}

	moduleDir := filepath.Dir(filepath.Dir(filepath.Dir(manifestPath)))
	moduleGradle := filepath.Join(moduleDir, "build.gradle.kts")
	if fileExistsAt(root, moduleGradle) {
		data, err := os.ReadFile(filepath.Join(root, moduleGradle))
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", moduleGradle, err)
		}
		tree, closeFn, err := parseKotlinFile(data)
		if err != nil {
			return "", err
		}
		defer closeFn()
		if block, ok := kotlin.FindBlockNode(tree, data, "android"); ok {
			if ns, ok := kotlin.FindStringAssignment(block, data, "namespace"); ok {
				return ns, nil
			}
		}
	}

	return "", &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf(
		"could not determine the app's package: AndroidManifest.xml has no package attribute, and %s has no android.namespace assignment — create the Application class manually",
		moduleGradle)}
}

func initCallSuggestion(root string, appClass kotlin.ClassInfo, tokenExpr string) (suggest.Suggestion, error) {
	data, err := os.ReadFile(filepath.Join(root, appClass.File))
	if err != nil {
		return suggest.Suggestion{}, fmt.Errorf("reading %s: %w", appClass.File, err)
	}
	pool, err := parse.NewPool(kotlin.Language(), 1)
	if err != nil {
		return suggest.Suggestion{}, err
	}
	defer pool.Close()
	tree, err := pool.Parse(context.Background(), data)
	if err != nil {
		return suggest.Suggestion{}, fmt.Errorf("parsing %s: %w", appClass.File, err)
	}
	defer tree.Close()

	bodyAt, methods, found := kotlin.LocateClass(tree, data, appClass.Name)
	if !found {
		return suggest.Suggestion{}, fmt.Errorf("could not re-locate class %s in %s", appClass.Name, appClass.File)
	}

	call := fmt.Sprintf("AppStorys.initialize(token = %s)", tokenExpr)

	if at, ok := methods["onCreate"]; ok {
		return suggest.Suggestion{
			ID:         suggest.StableID("init", appClass.File, "onCreate-init-call"),
			Kind:       "init",
			File:       appClass.File,
			InsertAt:   at,
			Snippet:    "\n        " + call,
			Reason:     fmt.Sprintf("Add the AppStorys init call to %s.onCreate", appClass.Name),
			Confidence: 0.9,
		}, nil
	}

	snippet := fmt.Sprintf("\n    override fun onCreate() {\n        super.onCreate()\n        %s\n    }\n", call)
	return suggest.Suggestion{
		ID:         suggest.StableID("init", appClass.File, "new-onCreate"),
		Kind:       "init",
		File:       appClass.File,
		InsertAt:   bodyAt,
		Snippet:    snippet,
		Reason:     fmt.Sprintf("%s has no onCreate; adding one with the AppStorys init call", appClass.Name),
		Confidence: 0.85,
	}, nil
}
