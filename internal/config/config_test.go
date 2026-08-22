package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("Providers = %v, want empty", cfg.Providers)
	}
}

func TestLoadParsesProviders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	const contents = `
[providers.github]
enabled = true

[providers.linear]
enabled = false
extra = { api_key_ref = "chit/linear-api-key" }
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	gh, ok := cfg.Providers["github"]
	if !ok || !gh.Enabled {
		t.Errorf("providers[github] = %+v, ok=%v, want enabled=true", gh, ok)
	}

	lin, ok := cfg.Providers["linear"]
	if !ok || lin.Enabled {
		t.Errorf("providers[linear] = %+v, ok=%v, want enabled=false", lin, ok)
	}
	if lin.Extra["api_key_ref"] != "chit/linear-api-key" {
		t.Errorf("linear.Extra[api_key_ref] = %q, want %q", lin.Extra["api_key_ref"], "chit/linear-api-key")
	}
}

func TestLoadRejectsMalformedToml(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("not valid toml [[["), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Error("Load with malformed TOML returned nil error")
	}
}
