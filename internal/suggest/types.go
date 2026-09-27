// Package suggest holds the Suggestion type (per the spec's core types)
// and StableID. The `init` command's Android planning logic in
// internal/cli produces Suggestion values using this package's types,
// but the screen/event detectors `integrate` needs (confidence scoring,
// dedup against existing calls) aren't implemented yet — Phase 3.
package suggest

// Suggestion is a proposed edit: an init call, an overlay host, a screen
// tracking call, or an event call site to insert.
type Suggestion struct {
	ID         string // stable hash of kind+file+anchor
	Kind       string // init | overlay-host | screen | event | nav-hook
	File       string
	Line       int
	Anchor     string
	InsertAt   int // byte offset from tree-sitter node
	Snippet    string
	Confidence float64 // 0..1
	Reason     string
	Requires   []string // e.g. "add import", "register in manifest"
}
