package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestJSONCommandsEmitEnvelopeEvenWhenEmpty pins the contract the
// exported skill documents in reference/output.md: every --format json
// command writes {"schemaVersion":1,"data":...}, and an empty top-level
// result is [] (never null, never a human sentence).
func TestJSONCommandsEmitEnvelopeEvenWhenEmpty(t *testing.T) {
	tests := []struct {
		name string
		root string
		args []string
	}{
		{"scan on a project with no SDK calls", "../../testdata/fixtures/flutter/minimal", []string{"scan", "--format", "json"}},
		{"integrate with nothing to suggest", "../../testdata/fixtures/android/sample-app", []string{"integrate", "--format", "json"}},
		{"events list --only events with no events", "../../testdata/fixtures/flutter/minimal", []string{"events", "list", "--only", "events", "--format", "json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := NewRootCmd()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs(append([]string{"--root", tt.root}, tt.args...))
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			var env struct {
				SchemaVersion int             `json:"schemaVersion"`
				Data          json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(out.Bytes(), &env); err != nil {
				t.Fatalf("output is not a JSON envelope: %v\n%s", err, out.String())
			}
			if env.SchemaVersion != 1 {
				t.Errorf("schemaVersion = %d, want 1", env.SchemaVersion)
			}
			if string(env.Data) != "[]" {
				t.Errorf("data = %s, want [] for an empty result", env.Data)
			}
		})
	}
}
