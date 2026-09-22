package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeRepoConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, repoConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("writing repo config: %v", err)
	}
}

func TestLoadDefaultsOnly(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(LoadOptions{Root: dir})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Exclude) == 0 {
		t.Errorf("Exclude is empty, want default excludes")
	}
	if cfg.Platforms != nil {
		t.Errorf("Platforms = %v, want nil (no repo config present)", cfg.Platforms)
	}
}

func TestLoadRepoConfigOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	writeRepoConfig(t, dir, "version: 1\nplatforms: [android]\n")

	cfg, err := Load(LoadOptions{Root: dir})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Platforms) != 1 || cfg.Platforms[0] != "android" {
		t.Errorf("Platforms = %v, want [android]", cfg.Platforms)
	}
}

func TestLoadEnvOverridesRepoConfig(t *testing.T) {
	dir := t.TempDir()
	writeRepoConfig(t, dir, "version: 1\nplatforms: [android]\n")

	cfg, err := Load(LoadOptions{
		Root: dir,
		Env:  map[string]string{"APPSTORYS_CLI_PLATFORMS": "flutter,ios"},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Platforms) != 2 || cfg.Platforms[0] != "flutter" || cfg.Platforms[1] != "ios" {
		t.Errorf("Platforms = %v, want [flutter ios]", cfg.Platforms)
	}
}

func TestLoadFlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	writeRepoConfig(t, dir, "version: 1\nplatforms: [android]\n")

	cfg, err := Load(LoadOptions{
		Root: dir,
		Env:  map[string]string{"APPSTORYS_CLI_PLATFORMS": "flutter"},
		FlagOverrides: func(c *Config) {
			c.Platforms = []string{"ios"}
		},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Platforms) != 1 || cfg.Platforms[0] != "ios" {
		t.Errorf("Platforms = %v, want [ios]", cfg.Platforms)
	}
}

func TestLoadMalformedRepoConfig(t *testing.T) {
	dir := t.TempDir()
	writeRepoConfig(t, dir, "version: [this is not valid: yaml")

	_, err := Load(LoadOptions{Root: dir})
	if err == nil {
		t.Fatal("Load() expected error for malformed YAML, got nil")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("Load() error = %v (%T), want *ConfigError", err, err)
	}
}

func TestLoadExplicitPathMissing(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(LoadOptions{Root: dir, ExplicitPath: filepath.Join(dir, "does-not-exist.yaml")})
	if err == nil {
		t.Fatal("Load() expected error for missing explicit --config path, got nil")
	}
}
