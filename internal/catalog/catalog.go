// Package catalog builds the deduplicated events/screens/positions
// catalog `events list` shows, from a set of extracted CallSites. The
// register/approval flow itself (F3's core backend-facing feature) is
// Phase 2 work — this phase only builds the catalog that flow will show.
package catalog

import (
	"slices"
	"sort"
	"strings"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/project"
)

// Item is one deduplicated catalog entry.
type Item struct {
	Name       string
	Kind       string // event | screen | position
	Platforms  []project.Platform
	CallSites  int
	Properties map[string]extract.PropType
}

// Catalog is the built events/screens/positions catalog.
type Catalog struct {
	Events    []Item
	Screens   []Item
	Positions []Item
	// Dynamic holds call sites whose name isn't a literal, listed
	// separately since they can't be registered (spec F3: "Dynamic
	// names are listed as unregistrable, with their call sites.").
	Dynamic []extract.CallSite
}

// Build deduplicates sites into a Catalog.
//
// Positions are collected from both screen calls' positions_arg and
// placement calls' position_arg, but aren't associated back to the
// screen they belong to (the request payload's {screen, position}
// pairing) — that needs static screen/nav discovery, which is `doctor`
// territory (Phase 2), not available from a flat CallSite list yet.
func Build(sites []extract.CallSite) Catalog {
	events := map[string]*Item{}
	screens := map[string]*Item{}
	positions := map[string]*Item{}
	var dynamic []extract.CallSite

	addPlatform := func(it *Item, p project.Platform) {
		if !slices.Contains(it.Platforms, p) {
			it.Platforms = append(it.Platforms, p)
		}
	}
	mergeProps := func(it *Item, props map[string]extract.PropType) {
		if len(props) == 0 {
			return
		}
		if it.Properties == nil {
			it.Properties = map[string]extract.PropType{}
		}
		for k, v := range props {
			if _, ok := it.Properties[k]; !ok {
				it.Properties[k] = v
			}
		}
	}
	upsertPosition := func(name string, p project.Platform) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		it, ok := positions[name]
		if !ok {
			it = &Item{Name: name, Kind: "position"}
			positions[name] = it
		}
		it.CallSites++
		addPlatform(it, p)
	}

	for _, s := range sites {
		switch s.Kind {
		case "event":
			if s.Dynamic || s.Name == "" {
				dynamic = append(dynamic, s)
				continue
			}
			it, ok := events[s.Name]
			if !ok {
				it = &Item{Name: s.Name, Kind: "event"}
				events[s.Name] = it
			}
			it.CallSites++
			addPlatform(it, s.Platform)
			mergeProps(it, s.Properties)

		case "screen":
			if s.Dynamic || s.Name == "" {
				dynamic = append(dynamic, s)
				continue
			}
			it, ok := screens[s.Name]
			if !ok {
				it = &Item{Name: s.Name, Kind: "screen"}
				screens[s.Name] = it
			}
			it.CallSites++
			addPlatform(it, s.Platform)
			for pos := range strings.SplitSeq(s.Position, ",") {
				upsertPosition(pos, s.Platform)
			}

		case "placement":
			upsertPosition(s.Position, s.Platform)
		}
	}

	return Catalog{
		Events:    sortedValues(events),
		Screens:   sortedValues(screens),
		Positions: sortedValues(positions),
		Dynamic:   dynamic,
	}
}

func sortedValues(m map[string]*Item) []Item {
	out := make([]Item, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
