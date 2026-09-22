// Package symbols loads per-platform AppStorys SDK facts from embedded
// YAML files. Facts are kept as data, not Go code, so a new SDK release
// is a data change plus fixture updates.
package symbols

// Map is one platform's SDK facts: the receivers and call symbols the
// extractors match against, plus lint-relevant constants.
type Map struct {
	Platform           string   `yaml:"platform"`
	SDKVersion         string   `yaml:"sdk_version"`
	Import             string   `yaml:"import"`
	Receivers          []string `yaml:"receivers"`
	Symbols            []Symbol `yaml:"symbols"`
	ReservedEvents     []string `yaml:"reserved_events"`
	MergedMetadataKeys []string `yaml:"merged_metadata_keys"`
}

// Symbol describes one SDK call or constructor the extractors recognize,
// and where in its arguments the concept-relevant values live.
type Symbol struct {
	Concept      string  `yaml:"concept"`
	Call         string  `yaml:"call,omitempty"`
	Constructor  string  `yaml:"constructor,omitempty"`
	NameArg      *ArgRef `yaml:"name_arg,omitempty"`
	PositionsArg *ArgRef `yaml:"positions_arg,omitempty"`
	PropsArg     *ArgRef `yaml:"props_arg,omitempty"`
	PositionArg  *ArgRef `yaml:"position_arg,omitempty"`
	MustBeInside string  `yaml:"must_be_inside,omitempty"`
}

// ArgRef points at a call argument either by position or by name.
type ArgRef struct {
	Positional *int   `yaml:"positional,omitempty"`
	Named      string `yaml:"named,omitempty"`
}
