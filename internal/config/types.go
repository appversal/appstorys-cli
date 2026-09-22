// Package config resolves .appstorys-cli.yaml, environment variables and
// CLI flags into one Config, following the precedence flags -> env ->
// repo config -> defaults.
package config

// Config is the parsed shape of .appstorys-cli.yaml.
type Config struct {
	Version   int               `yaml:"version"`
	Platforms []string          `yaml:"platforms,omitempty"`
	Include   []string          `yaml:"include,omitempty"`
	Exclude   []string          `yaml:"exclude,omitempty"`
	Receivers []string          `yaml:"receivers,omitempty"`
	Wrappers  []WrapperConfig   `yaml:"wrappers,omitempty"`
	Naming    NamingConfig      `yaml:"naming,omitempty"`
	Rules     map[string]string `yaml:"rules,omitempty"`
	Screens   ScreensConfig     `yaml:"screens,omitempty"`
	Init      InitConfig        `yaml:"init,omitempty"`
	Integrate IntegrateConfig   `yaml:"integrate,omitempty"`
}

// WrapperConfig describes one of the app's own helper functions around an
// SDK call, so extractors recognize it as if it were the SDK symbol
// itself.
type WrapperConfig struct {
	Name     string `yaml:"name"`
	Concept  string `yaml:"concept"`
	NameArg  int    `yaml:"name_arg"`
	PropsArg int    `yaml:"props_arg"`
}

// NamingConfig holds naming-convention preferences used by lint rules.
type NamingConfig struct {
	Events string `yaml:"events,omitempty"`
}

// ScreensConfig lets the developer correct static screen discovery.
type ScreensConfig struct {
	Ignore []string `yaml:"ignore,omitempty"`
	Extra  []string `yaml:"extra,omitempty"`
}

// InitConfig overrides how the generated init call reads its token.
type InitConfig struct {
	TokenExpr string `yaml:"token_expr,omitempty"`
}

// IntegrateConfig holds defaults for the `integrate` command.
type IntegrateConfig struct {
	MinConfidence float64  `yaml:"min_confidence,omitempty"`
	Flows         []string `yaml:"flows,omitempty"`
}
