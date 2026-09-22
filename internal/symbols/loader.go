package symbols

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed android.yaml flutter.yaml reactnative.yaml ios.yaml
var embedded embed.FS

var knownPlatforms = map[string]string{
	"android":      "android.yaml",
	"flutter":      "flutter.yaml",
	"react-native": "reactnative.yaml",
	"ios":          "ios.yaml",
}

// Load reads and validates every embedded per-platform symbol map,
// keyed by platform name.
func Load() (map[string]*Map, error) {
	out := make(map[string]*Map, len(knownPlatforms))
	for platform, file := range knownPlatforms {
		m, err := loadFile(file)
		if err != nil {
			return nil, fmt.Errorf("symbols: loading %s: %w", file, err)
		}
		if m.Platform != platform {
			return nil, fmt.Errorf("symbols: %s: platform field %q does not match expected %q", file, m.Platform, platform)
		}
		if err := validate(m); err != nil {
			return nil, fmt.Errorf("symbols: %s: %w", file, err)
		}
		out[platform] = m
	}
	return out, nil
}

// LoadPlatform reads and validates a single platform's symbol map.
func LoadPlatform(platform string) (*Map, error) {
	file, ok := knownPlatforms[platform]
	if !ok {
		return nil, fmt.Errorf("symbols: unknown platform %q", platform)
	}
	m, err := loadFile(file)
	if err != nil {
		return nil, fmt.Errorf("symbols: loading %s: %w", file, err)
	}
	if err := validate(m); err != nil {
		return nil, fmt.Errorf("symbols: %s: %w", file, err)
	}
	return m, nil
}

func loadFile(name string) (*Map, error) {
	data, err := embedded.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var m Map
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func validate(m *Map) error {
	if m.Platform == "" {
		return fmt.Errorf("missing platform")
	}
	if _, ok := knownPlatforms[m.Platform]; !ok {
		return fmt.Errorf("unknown platform %q", m.Platform)
	}
	if m.SDKVersion == "" {
		return fmt.Errorf("missing sdk_version")
	}
	// A concept may have several symbols (e.g. several placement kinds:
	// Stories, Widget, Reels, Milestone), so uniqueness is checked on
	// the symbol's actual identifier (call or constructor) instead.
	seen := make(map[string]bool, len(m.Symbols))
	for _, s := range m.Symbols {
		id := s.Call
		if id == "" {
			id = s.Constructor
		}
		if id == "" {
			return fmt.Errorf("symbol with concept %q has neither call nor constructor", s.Concept)
		}
		if seen[id] {
			return fmt.Errorf("duplicate symbol %q", id)
		}
		seen[id] = true
	}
	return nil
}
