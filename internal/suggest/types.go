// Package suggest will hold the init/screen/event detectors that produce
// Suggestion values for the `integrate` command (Phase 3). Only the
// Suggestion type itself — fully specified by the spec's core types —
// lives here so far, since internal/rules' Finding.Fix needs a real type
// to point at rather than a placeholder that would need migrating later.
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
