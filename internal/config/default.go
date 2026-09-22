package config

// Default returns the built-in configuration used when no
// .appstorys-cli.yaml is present and no overrides apply.
func Default() Config {
	return Config{
		Version: 1,
		Exclude: []string{
			"**/node_modules/**",
			"**/.dart_tool/**",
			"**/Pods/**",
			"**/build/**",
			"**/generated/**",
			"**/.git/**",
		},
	}
}
