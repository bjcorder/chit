// Package config reads chit's config.toml — currently just which providers
// are enabled and their provider-specific settings.
package config

import (
	"bytes"
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

// DataDir returns the XDG-compliant data directory chit uses for its cache
// database and secret-store fallback vault, ~/.local/share/chit on Linux.
// os.UserConfigDir handles XDG_CONFIG_HOME natively; the stdlib has no
// equivalent for XDG_DATA_HOME, so it's resolved by hand here.
func DataDir() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "chit"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "chit"), nil
}

// Load reads and parses the config file at path. A missing file is not an
// error — it returns an empty Config, since a fresh chit install has no
// providers enabled yet.
func Load(path string) (*Config, error) {
	cfg := &Config{Providers: map[string]ProviderConfig{}}

	data, err := os.ReadFile(path) // #nosec G304 -- path is either DefaultPath()'s result or an explicit path the caller supplied, never attacker-controlled input
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

// Save writes cfg to path as TOML, creating the parent directory if needed.
// Used by the `chit providers enable/disable` CLI helpers, which edit the
// config file rather than requiring an in-TUI settings screen.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
