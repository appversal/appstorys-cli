package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const repoConfigFileName = ".appstorys-cli.yaml"

// LoadOptions controls how Load resolves a Config.
type LoadOptions struct {
	// Root is the project root to look for .appstorys-cli.yaml in.
	Root string
	// ExplicitPath, if set (the --config flag), is used instead of
	// Root/.appstorys-cli.yaml. It is an error for an explicit path not
	// to exist; an implicit one is simply skipped.
	ExplicitPath string
	// Env is consulted for APPSTORYS_CLI_* overrides. Callers normally
	// pass a map built from os.Environ(); tests can pass a fixed map.
	Env map[string]string
	// FlagOverrides, if set, is applied last, after env, so CLI flags
	// always win.
	FlagOverrides func(*Config)
}

// Load resolves a Config following flags -> env -> repo config ->
// defaults precedence (each stage overrides the previous where it sets a
// value).
func Load(opts LoadOptions) (*Config, error) {
	cfg := Default()

	path := opts.ExplicitPath
	required := path != ""
	if path == "" {
		path = filepath.Join(opts.Root, repoConfigFileName)
	}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		var repoCfg Config
		if err := yaml.Unmarshal(data, &repoCfg); err != nil {
			return nil, &ConfigError{Path: path, Err: err}
		}
		mergeConfig(&cfg, repoCfg)
	case os.IsNotExist(err) && !required:
		// No repo config, and none was explicitly requested: defaults
		// stand.
	default:
		return nil, &ConfigError{Path: path, Err: err}
	}

	applyEnv(&cfg, opts.Env)

	if opts.FlagOverrides != nil {
		opts.FlagOverrides(&cfg)
	}

	return &cfg, nil
}

// mergeConfig overrides base with every field repoCfg explicitly set.
func mergeConfig(base *Config, repoCfg Config) {
	if repoCfg.Version != 0 {
		base.Version = repoCfg.Version
	}
	if repoCfg.Platforms != nil {
		base.Platforms = repoCfg.Platforms
	}
	if repoCfg.Include != nil {
		base.Include = repoCfg.Include
	}
	if repoCfg.Exclude != nil {
		base.Exclude = repoCfg.Exclude
	}
	if repoCfg.Receivers != nil {
		base.Receivers = repoCfg.Receivers
	}
	if repoCfg.Wrappers != nil {
		base.Wrappers = repoCfg.Wrappers
	}
	if repoCfg.Naming.Events != "" {
		base.Naming.Events = repoCfg.Naming.Events
	}
	if repoCfg.Rules != nil {
		base.Rules = repoCfg.Rules
	}
	if repoCfg.Screens.Ignore != nil {
		base.Screens.Ignore = repoCfg.Screens.Ignore
	}
	if repoCfg.Screens.Extra != nil {
		base.Screens.Extra = repoCfg.Screens.Extra
	}
	if repoCfg.Init.TokenExpr != "" {
		base.Init.TokenExpr = repoCfg.Init.TokenExpr
	}
	if repoCfg.Integrate.MinConfidence != 0 {
		base.Integrate.MinConfidence = repoCfg.Integrate.MinConfidence
	}
	if repoCfg.Integrate.Flows != nil {
		base.Integrate.Flows = repoCfg.Integrate.Flows
	}
}

// applyEnv overrides the small set of fields that matter this phase from
// APPSTORYS_CLI_* environment variables. Expand this list as later
// phases add config fields that need env overrides.
func applyEnv(cfg *Config, env map[string]string) {
	if v, ok := env["APPSTORYS_CLI_PLATFORMS"]; ok && v != "" {
		cfg.Platforms = strings.Split(v, ",")
	}
	if v, ok := env["APPSTORYS_CLI_INCLUDE"]; ok && v != "" {
		cfg.Include = strings.Split(v, ",")
	}
	if v, ok := env["APPSTORYS_CLI_EXCLUDE"]; ok && v != "" {
		cfg.Exclude = strings.Split(v, ",")
	}
}
