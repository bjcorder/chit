// Package config reads chit's config.toml — currently just which providers
// are enabled and their provider-specific settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ProviderConfig struct {
	Enabled bool              `toml:"enabled"`
	Extra   map[string]string `toml:"extra"`
}

type Config struct {
	Providers map[string]ProviderConfig `toml:"providers"`
}

// DefaultPath returns the XDG-compliant config file location,
// ~/.config/chit/config.toml on Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving user config dir: %w", err)
	}
	return filepath.Join(dir, "chit", "config.toml"), nil
}

// Load reads and parses the config file at path. A missing file is not an
// error — it returns an empty Config, since a fresh chit install has no
// providers enabled yet.
func Load(path string) (*Config, error) {
	cfg := &Config{Providers: map[string]ProviderConfig{}}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}
	return cfg, nil
}
