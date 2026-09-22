// Package androidxml detects AppStorys symbols in Android layout XML:
// OverlayLayoutView as a layout root (overlay-host) and WidgetView's
// app:position attribute (placement) — the two layout-XML symbols named
// in the spec's concept map, distinct from Compose's overlayElements()/
// Widget() calls that internal/lang/kotlin already handles.
//
// This isn't tree-sitter-based like the other language adapters: layout
// XML is simple attribute-value data, not call expressions to match
// against the symbol map's call/constructor schema, so a plain
// encoding/xml scan is the more direct fit. It produces extract.CallSite
// values directly rather than going through extract.Match.
package androidxml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/project"
)

// Scan parses an Android layout XML file's elements and returns an
// overlay-host CallSite for every OverlayLayoutView element and a
// placement CallSite for every WidgetView element (using its position
// attribute, matched by local name so an "app:" or other namespace
// prefix doesn't matter).
func Scan(file string, data []byte) ([]extract.CallSite, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var sites []extract.CallSite
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("androidxml: parsing %s: %w", file, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		line := lineAt(data, dec.InputOffset())
		switch {
		case isViewClass(start.Name.Local, "OverlayLayoutView"):
			sites = append(sites, extract.CallSite{
				Kind:     "overlay-host",
				File:     file,
				Line:     line,
				Via:      "OverlayLayoutView",
				Platform: project.Android,
			})
		case isViewClass(start.Name.Local, "WidgetView"):
			sites = append(sites, extract.CallSite{
				Kind:     "placement",
				File:     file,
				Line:     line,
				Via:      "WidgetView",
				Position: attrValue(start, "position"),
				Platform: project.Android,
			})
		}
	}
	return sites, nil
}

// isViewClass reports whether an XML element name is the given custom
// view class — Android layout XML writes custom views as their
// fully-qualified class name (e.g. "com.appversal.appstorys.WidgetView"),
// not a plain local name, so this matches either the bare name or a
// "....ClassName" suffix.
func isViewClass(elementName, className string) bool {
	return elementName == className || strings.HasSuffix(elementName, "."+className)
}

func attrValue(el xml.StartElement, local string) string {
	for _, a := range el.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func lineAt(data []byte, offset int64) int {
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	return 1 + bytes.Count(data[:offset], []byte("\n"))
}
