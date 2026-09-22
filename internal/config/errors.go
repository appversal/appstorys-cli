package config

import "fmt"

// ConfigError marks a failure the CLI layer should map to exit code 2
// (usage or config error) rather than a generic failure.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("config: %v", e.Err)
	}
	return fmt.Sprintf("config: %s: %v", e.Path, e.Err)
}

func (e *ConfigError) Unwrap() error { return e.Err }
