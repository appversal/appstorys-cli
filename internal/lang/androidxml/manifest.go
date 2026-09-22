package androidxml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ManifestInfo is the subset of AndroidManifest.xml doctor's checks
// need: the Application class (for the init-location check) and the
// declared Activities (for Views-style screen discovery).
type ManifestInfo struct {
	Package         string
	ApplicationName string // as declared, e.g. ".App" or "com.example.app.App"; "" if absent
	Activities      []ManifestActivity
}

// ManifestActivity is one <activity> entry.
type ManifestActivity struct {
	Name string // as declared, e.g. ".HomeActivity"
	Line int
}

// ParseManifest reads AndroidManifest.xml's <application>/<activity>
// android:name attributes.
func ParseManifest(data []byte) (ManifestInfo, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var info ManifestInfo
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ManifestInfo{}, fmt.Errorf("androidxml: parsing manifest: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "manifest":
			info.Package = attrValue(start, "package")
		case "application":
			info.ApplicationName = attrValue(start, "name")
		case "activity":
			info.Activities = append(info.Activities, ManifestActivity{
				Name: attrValue(start, "name"),
				Line: lineAt(data, dec.InputOffset()),
			})
		}
	}
	return info, nil
}

// SimpleName reduces a manifest class reference to a bare class name:
// ".App" -> "App", ".ui.Home" -> "Home", "com.example.App" -> "App".
// Matching source-declared classes (which this CLI only tracks by
// simple name, not fully package-qualified) against manifest entries
// needs this simplification either way.
func SimpleName(qualified string) string {
	qualified = strings.TrimPrefix(qualified, ".")
	if idx := strings.LastIndex(qualified, "."); idx >= 0 {
		return qualified[idx+1:]
	}
	return qualified
}
