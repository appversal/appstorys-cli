package project

// ProjectInfo describes one detected platform project rooted at Root.
type ProjectInfo struct {
	Platform Platform
	Root     string
	// MarkerFiles are the paths (relative to Root) whose presence
	// caused Detect to match, kept for --verbose output and debugging.
	MarkerFiles []string
	// NavLibrary is the detected navigation library (e.g. "compose-navigation",
	// "go_router"). Best-effort; empty when unknown. Real detection
	// lands in Phase 1 alongside the screen detectors that need it.
	NavLibrary string
	// Extra holds adapter-specific facts that don't warrant their own
	// field yet.
	Extra map[string]string
}
