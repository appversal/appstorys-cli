package project

import sitter "github.com/tree-sitter/go-tree-sitter"

// Query is a placeholder for a tree-sitter query descriptor. Its real
// shape (query source, capture names, concept mapping) lands with
// internal/lang in Phase 1.
type Query struct {
	Name    string
	Pattern string
}

// Detector and Formatter are placeholders for concepts that belong to
// packages which don't exist yet: Detector to internal/suggest ("...
// detectors -> Suggestion" in the spec's module layout), Formatter to
// internal/patch ("insertion planning, diff, apply, formatters").
//
// TODO(phase1): if internal/suggest or internal/patch need to import
// project.Platform or project.ProjectInfo (likely — a Detector has to
// know which platform it's running against), then Adapter importing
// those packages' real types back would create an import cycle
// (project -> suggest/patch -> project). The likely fix is splitting
// Adapter into narrower interfaces defined at the point of use (e.g. a
// suggest.PlatformSource interface satisfied by project.Adapter)
// instead of one fat interface living in project. Deferred rather than
// forced here, since neither package exists yet.
type Detector = any
type Formatter = any

// Adapter is the spec's per-platform adapter interface, trimmed to what
// Phase 0 implements. Platform, Detect and Files are fully implemented
// by every concrete adapter; the remaining methods are stubbed (returning
// zero values, never panicking) so the interface is satisfiable ahead of
// Phase 1's real extraction, suggestion and patch logic.
type Adapter interface {
	Platform() Platform
	Detect(root string) (bool, ProjectInfo, error)
	Files(root string) ([]string, error)
	Language(file string) *sitter.Language
	CallQueries() []Query
	InitDetector() Detector
	ScreenDetectors() []Detector
	EventDetectors() []Detector
	Formatter() Formatter
}
