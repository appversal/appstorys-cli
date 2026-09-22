// Package rules evaluates lint rules over a set of extracted CallSites,
// producing Findings for `lint` and `validate`.
//
// This phase implements the rules that only need the flat CallSite list
// itself: reserved-event-name, metadata-key-overwritten, empty-event-name,
// dynamic-name, near-duplicate-name, property-type-conflict, pii-property,
// init-missing, init-multiple and duplicate-widget-position. The rest of
// the spec's F2 table needs information this phase doesn't have yet:
// static screen/nav discovery (init-location, missing-overlay-host,
// duplicate-overlay-host, overlay-outside-stack, provider-required,
// root-wrappers-missing, track-before-init, screen-without-untrack — all
// squarely `doctor`'s job, Phase 2), a babel config reader
// (babel-plugin-missing), a local registered.json (unregistered-event,
// Phase 2), or backend-supplied limits (max-lengths). hardcoded-token is
// also deferred: the spec lists the exact token-parameter position per
// platform as an open question, and the symbol maps don't encode one.
package rules

import (
	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// Severity is a finding's severity level.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
	SeverityOff     Severity = "off"
)

// Finding is one lint result.
type Finding struct {
	RuleID   string
	Severity Severity
	Message  string
	Site     extract.CallSite
	Fix      *suggest.Suggestion
}
