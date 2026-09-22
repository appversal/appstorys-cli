package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/androidxml"
	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

// DoctorResult is one row of `doctor`'s CHECK/SCREEN/FILE:LINE/TRACKED
// AS/OVERLAY/STATUS table, per the spec's F6 output.
type DoctorResult struct {
	Check     string // "init" | "screen"
	Screen    string // "-" for the init check
	File      string
	Line      int
	TrackedAs string // literal screen name, "<dynamic>", or "-"
	Overlay   string // "root" | "missing" | "-"
	Status    string // "pass" | "fail: <reason>" | "warn: heuristic: <reason>"
}

// doctorAndroid runs the spec's F6 checks for Android: init at the
// entry point, and screen tracking + overlay host per screen.
//
// Screen discovery here only covers the spec's "Views" row (Activities
// declared in AndroidManifest.xml) — Compose NavHost destination
// discovery needs lambda-body containment analysis (which composable
// block a call site falls inside) that this phase's extraction doesn't
// retain, so it isn't implemented. Tracking/overlay-host correlation is
// done at file granularity (does this Activity's own .kt file contain a
// screen/overlay-host call anywhere), a heuristic the spec explicitly
// allows for cases doctor can't fully resolve statically.
func doctorAndroid(app *App, sites []extract.CallSite) ([]DoctorResult, error) {
	manifestPath, err := findManifest(app.Root)
	if err != nil {
		return nil, err
	}
	if manifestPath == "" {
		return []DoctorResult{{
			Check: "init", Screen: "-", File: "-",
			Status: "fail: AndroidManifest.xml not found",
		}}, nil
	}
	data, err := os.ReadFile(filepath.Join(app.Root, manifestPath))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
	}
	manifest, err := androidxml.ParseManifest(data)
	if err != nil {
		return nil, err
	}

	classes, err := allKotlinClasses(app)
	if err != nil {
		return nil, err
	}

	results := []DoctorResult{checkInit(manifest, classes, sites)}
	results = append(results, checkScreens(manifest, classes, sites)...)
	return results, nil
}

func checkInit(manifest androidxml.ManifestInfo, classes []kotlin.ClassInfo, sites []extract.CallSite) DoctorResult {
	var inits []extract.CallSite
	for _, s := range sites {
		if s.Kind == "init" {
			inits = append(inits, s)
		}
	}
	if len(inits) == 0 {
		return DoctorResult{Check: "init", Screen: "-", File: "-", Status: "fail: no init call found"}
	}
	if len(inits) > 1 {
		return DoctorResult{Check: "init", Screen: "-", File: inits[0].File, Line: inits[0].Line,
			Status: fmt.Sprintf("fail: %d init calls found, expected exactly one", len(inits))}
	}
	site := inits[0]

	if manifest.ApplicationName == "" {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: "fail: no Application class registered in AndroidManifest.xml (android:name)"}
	}
	appClass := androidxml.SimpleName(manifest.ApplicationName)

	enclosingClass, enclosingFunc, _ := strings.Cut(site.Enclosing, ".")
	if enclosingFunc == "" {
		enclosingFunc, enclosingClass = enclosingClass, ""
	}

	if enclosingClass != appClass {
		display := enclosingClass
		if display == "" {
			display = "(no enclosing class)"
		}
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: fmt.Sprintf("fail: init is in %s, not the manifest's Application class %q", display, appClass)}
	}
	if enclosingFunc != "onCreate" {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: fmt.Sprintf("fail: init is in %s.%s, not onCreate", enclosingClass, enclosingFunc)}
	}

	var isApplication bool
	for _, c := range classes {
		if c.Name == appClass {
			for _, sup := range c.Supertypes {
				if sup == "Application" {
					isApplication = true
				}
			}
		}
	}
	if !isApplication {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: fmt.Sprintf("warn: heuristic: could not confirm %q extends Application", appClass)}
	}

	return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line, Status: "pass"}
}

func checkScreens(manifest androidxml.ManifestInfo, classes []kotlin.ClassInfo, sites []extract.CallSite) []DoctorResult {
	var results []DoctorResult
	for _, act := range manifest.Activities {
		simple := androidxml.SimpleName(act.Name)
		if simple == "" {
			continue
		}

		var classInfo *kotlin.ClassInfo
		for i := range classes {
			if classes[i].Name == simple {
				classInfo = &classes[i]
				break
			}
		}
		if classInfo == nil {
			results = append(results, DoctorResult{
				Check: "screen", Screen: simple, File: "-", TrackedAs: "-", Overlay: "-",
				Status: "warn: heuristic: source file for this Activity was not found",
			})
			continue
		}

		var tracked *extract.CallSite
		var overlayFound bool
		for i := range sites {
			s := &sites[i]
			if s.File != classInfo.File {
				continue
			}
			if s.Kind == "screen" && tracked == nil {
				tracked = s
			}
			if s.Kind == "overlay-host" {
				overlayFound = true
			}
		}

		trackedAs := "-"
		switch {
		case tracked != nil && tracked.Dynamic:
			trackedAs = "<dynamic>"
		case tracked != nil && tracked.Name != "":
			trackedAs = tracked.Name
		}
		overlay := "missing"
		if overlayFound {
			overlay = "root"
		}

		status := "pass"
		switch {
		case tracked == nil:
			status = "fail: not tracked"
		case !overlayFound:
			status = "fail: no overlay host"
		}

		results = append(results, DoctorResult{
			Check: "screen", Screen: simple, File: classInfo.File, Line: classInfo.Line,
			TrackedAs: trackedAs, Overlay: overlay, Status: status,
		})
	}
	return results
}

func findManifest(root string) (string, error) {
	matches, err := parse.Walk(root, parse.WalkOptions{Include: []string{"AndroidManifest.xml"}})
	if err != nil {
		return "", err
	}
	for _, m := range matches {
		if strings.Contains(m, "src/main/") {
			return m, nil
		}
	}
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", nil
}

func allKotlinClasses(app *App) ([]kotlin.ClassInfo, error) {
	adapter, _ := project.ByPlatform(project.Android)
	files, err := adapter.Files(app.Root)
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}

	pool, err := parse.NewPool(kotlin.Language(), 0)
	if err != nil {
		return nil, fmt.Errorf("starting parser pool: %w", err)
	}
	defer pool.Close()

	var classes []kotlin.ClassInfo
	for _, rel := range files {
		if filepath.Ext(rel) != ".kt" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(app.Root, rel))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", rel, err)
		}
		tree, err := pool.Parse(context.Background(), data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", rel, err)
		}
		classes = append(classes, kotlin.Classes(tree, data, rel)...)
		tree.Close()
	}
	return classes, nil
}
