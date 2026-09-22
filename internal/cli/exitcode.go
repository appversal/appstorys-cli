// Package cli wires Cobra commands, global flags and output streams into
// one App, and maps errors to the spec's exit codes.
package cli

import "fmt"

// Exit codes, per the spec's "Exit codes" table. Only ExitOK,
// ExitUsageOrConfig and ExitProjectUnsupported are reachable this phase;
// the rest are declared now since they're part of one shared contract
// later phases (findings, registration, refusal) plug into.
const (
	ExitOK                 = 0
	ExitFindings           = 1
	ExitUsageOrConfig      = 2
	ExitProjectUnsupported = 3
	ExitRegistrationFailed = 4
	ExitRefused            = 5
)

// CLIError pairs an error with the process exit code it should produce.
type CLIError struct {
	Code int
	Err  error
}

func (e *CLIError) Error() string { return fmt.Sprintf("%v", e.Err) }
func (e *CLIError) Unwrap() error { return e.Err }
