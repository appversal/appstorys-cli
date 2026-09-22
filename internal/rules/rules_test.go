package rules

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

func has(t *testing.T, findings []Finding, ruleID string) *Finding {
	t.Helper()
	for i := range findings {
		if findings[i].RuleID == ruleID {
			return &findings[i]
		}
	}
	return nil
}

func TestRun(t *testing.T) {
	m := &symbols.Map{
		ReservedEvents:     []string{"viewed", "clicked"},
		MergedMetadataKeys: []string{"platform", "model"},
	}

	sites := []extract.CallSite{
		{Kind: "init", File: "App.kt"},
		{Kind: "init", File: "Other.kt"}, // triggers init-multiple

		{Kind: "event", Name: "viewed", File: "A.kt"}, // reserved-event-name
		{Kind: "event", Name: "Login", File: "A.kt", Properties: map[string]extract.PropType{
			"method":   extract.PropString,
			"platform": extract.PropString, // metadata-key-overwritten
			"email":    extract.PropString, // pii-property
		}},
		{Kind: "event", Name: "Login", File: "B.kt", Properties: map[string]extract.PropType{
			"method": extract.PropNumber, // property-type-conflict vs A.kt's string
		}},
		{Kind: "event", Name: "login", File: "C.kt"}, // near-duplicate-name of "Login"
		{Kind: "event", Dynamic: true, File: "D.kt"}, // dynamic-name
		{Kind: "event", Name: "", File: "E.kt"},      // empty-event-name

		{Kind: "placement", Position: "widget_one", File: "Home.kt"},
		{Kind: "placement", Position: "widget_one", File: "Home.kt"}, // duplicate-widget-position
	}

	findings := Run(sites, m, nil)

	for _, ruleID := range []string{
		"init-multiple",
		"reserved-event-name",
		"metadata-key-overwritten",
		"pii-property",
		"property-type-conflict",
		"near-duplicate-name",
		"dynamic-name",
		"empty-event-name",
		"duplicate-widget-position",
	} {
		if has(t, findings, ruleID) == nil {
			t.Errorf("missing expected finding for rule %q", ruleID)
		}
	}

	if f := has(t, findings, "init-missing"); f != nil {
		t.Errorf("unexpected init-missing finding: %+v", f)
	}
}

func TestRunInitMissing(t *testing.T) {
	findings := Run(nil, &symbols.Map{}, nil)
	f := has(t, findings, "init-missing")
	if f == nil {
		t.Fatal("expected init-missing finding for an empty site list")
	}
	if f.Severity != SeverityError {
		t.Errorf("init-missing severity = %q, want %q", f.Severity, SeverityError)
	}
}

func TestRunSeverityOverride(t *testing.T) {
	sites := []extract.CallSite{{Kind: "event", Name: "viewed"}}
	m := &symbols.Map{ReservedEvents: []string{"viewed"}}

	findings := Run(sites, m, map[string]string{"reserved-event-name": "off"})
	if has(t, findings, "reserved-event-name") != nil {
		t.Error("expected reserved-event-name to be suppressed by override")
	}

	findings = Run(sites, m, map[string]string{"reserved-event-name": "warning"})
	f := has(t, findings, "reserved-event-name")
	if f == nil {
		t.Fatal("expected reserved-event-name finding")
	}
	if f.Severity != SeverityWarning {
		t.Errorf("severity = %q, want %q (override)", f.Severity, SeverityWarning)
	}
}
