package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// defaults are the spec's F2 table severities.
var defaults = map[string]Severity{
	"reserved-event-name":       SeverityError,
	"metadata-key-overwritten":  SeverityWarning,
	"empty-event-name":          SeverityError,
	"dynamic-name":              SeverityInfo,
	"near-duplicate-name":       SeverityWarning,
	"property-type-conflict":    SeverityWarning,
	"pii-property":              SeverityWarning,
	"init-missing":              SeverityError,
	"init-multiple":             SeverityWarning,
	"duplicate-widget-position": SeverityWarning,
}

var piiKeys = []string{"email", "phone", "password", "token", "address"}

// Run evaluates every rule this phase implements (see the package doc
// for what's deferred) against sites, using m for platform facts
// (reserved event names, merged metadata keys). overrides is
// Config.Rules from .appstorys-cli.yaml; a rule set to "off" there is
// skipped entirely.
func Run(sites []extract.CallSite, m *symbols.Map, overrides map[string]string) []Finding {
	sev := func(ruleID string) (Severity, bool) {
		if v, ok := overrides[ruleID]; ok {
			s := Severity(v)
			return s, s != SeverityOff
		}
		return defaults[ruleID], true
	}

	var findings []Finding
	add := func(ruleID string, site extract.CallSite, format string, args ...any) {
		s, on := sev(ruleID)
		if !on {
			return
		}
		findings = append(findings, Finding{
			RuleID:   ruleID,
			Severity: s,
			Message:  fmt.Sprintf(format, args...),
			Site:     site,
		})
	}

	reservedEventName(sites, m, add)
	metadataKeyOverwritten(sites, m, add)
	emptyAndDynamicName(sites, add)
	nearDuplicateName(sites, add)
	propertyTypeConflict(sites, add)
	piiProperty(sites, add)
	initCount(sites, add)
	duplicateWidgetPosition(sites, add)

	return findings
}

type adder func(ruleID string, site extract.CallSite, format string, args ...any)

func reservedEventName(sites []extract.CallSite, m *symbols.Map, add adder) {
	reserved := make(map[string]bool, len(m.ReservedEvents))
	for _, r := range m.ReservedEvents {
		reserved[r] = true
	}
	for _, s := range sites {
		if s.Kind != "event" || s.Dynamic || s.Name == "" {
			continue
		}
		if reserved[s.Name] {
			add("reserved-event-name", s, "%q is a reserved SDK event name and cannot be used as a custom event", s.Name)
		}
	}
}

func metadataKeyOverwritten(sites []extract.CallSite, m *symbols.Map, add adder) {
	overwritten := make(map[string]bool, len(m.MergedMetadataKeys))
	for _, k := range m.MergedMetadataKeys {
		overwritten[k] = true
	}
	for _, s := range sites {
		if s.Kind != "event" {
			continue
		}
		for key := range s.Properties {
			if overwritten[key] {
				add("metadata-key-overwritten", s, "metadata key %q is overwritten by the SDK with device info", key)
			}
		}
	}
}

func emptyAndDynamicName(sites []extract.CallSite, add adder) {
	for _, s := range sites {
		if s.Kind != "event" && s.Kind != "screen" {
			continue
		}
		if s.Dynamic {
			add("dynamic-name", s, "%s name is not a literal and cannot be registered", s.Kind)
			continue
		}
		if s.Name == "" {
			add("empty-event-name", s, "%s name is empty", s.Kind)
		}
	}
}

func nearDuplicateName(sites []extract.CallSite, add adder) {
	for _, kind := range []string{"event", "screen"} {
		firstSeen := map[string]extract.CallSite{}
		var order []string
		for _, s := range sites {
			if s.Kind != kind || s.Dynamic || s.Name == "" {
				continue
			}
			if _, ok := firstSeen[s.Name]; !ok {
				firstSeen[s.Name] = s
				order = append(order, s.Name)
			}
		}
		sort.Strings(order)
		for i := 1; i < len(order); i++ {
			for j := 0; j < i; j++ {
				a, b := order[i], order[j]
				if a == b {
					continue
				}
				if normalize(a) == normalize(b) || levenshtein(a, b) <= 2 {
					add("near-duplicate-name", firstSeen[a], "%q is a near-duplicate of %q", a, b)
					break
				}
			}
		}
	}
}

func propertyTypeConflict(sites []extract.CallSite, add adder) {
	type key struct{ name, prop string }
	seenType := map[key]extract.PropType{}
	reported := map[key]bool{}
	for _, s := range sites {
		if s.Kind != "event" || s.Name == "" {
			continue
		}
		for prop, t := range s.Properties {
			k := key{s.Name, prop}
			if prev, ok := seenType[k]; ok {
				if prev != t && !reported[k] {
					add("property-type-conflict", s, "property %q on event %q is sent as both %s and %s", prop, s.Name, prev, t)
					reported[k] = true
				}
				continue
			}
			seenType[k] = t
		}
	}
}

func piiProperty(sites []extract.CallSite, add adder) {
	pii := make(map[string]bool, len(piiKeys))
	for _, k := range piiKeys {
		pii[k] = true
	}
	for _, s := range sites {
		if s.Kind != "event" {
			continue
		}
		for key := range s.Properties {
			if pii[strings.ToLower(key)] {
				add("pii-property", s, "property %q looks like it may carry personal data", key)
			}
		}
	}
}

func initCount(sites []extract.CallSite, add adder) {
	var inits []extract.CallSite
	for _, s := range sites {
		if s.Kind == "init" {
			inits = append(inits, s)
		}
	}
	if len(inits) == 0 {
		add("init-missing", extract.CallSite{}, "no AppStorys init call was found")
		return
	}
	for _, extra := range inits[1:] {
		add("init-multiple", extra, "more than one AppStorys init call was found")
	}
}

func duplicateWidgetPosition(sites []extract.CallSite, add adder) {
	type key struct{ file, position string }
	seen := map[key][]extract.CallSite{}
	var order []key
	for _, s := range sites {
		if s.Kind != "placement" || s.Position == "" {
			continue
		}
		k := key{s.File, s.Position}
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}
		seen[k] = append(seen[k], s)
	}
	for _, k := range order {
		group := seen[k]
		if len(group) < 2 {
			continue
		}
		for _, s := range group[1:] {
			add("duplicate-widget-position", s, "widget position %q is used more than once in %s", k.position, k.file)
		}
	}
}
