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

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	cfg := &Config{Providers: map[string]ProviderConfig{
		"github": {Enabled: true},
		"linear": {Enabled: false, Extra: map[string]string{"api_key_ref": "chit/linear-api-key"}},
	}}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if !got.Providers["github"].Enabled {
		t.Errorf("providers[github].Enabled = false, want true")
	}
	if got.Providers["linear"].Enabled {
		t.Errorf("providers[linear].Enabled = true, want false")
	}
	if got.Providers["linear"].Extra["api_key_ref"] != "chit/linear-api-key" {
		t.Errorf("linear.Extra[api_key_ref] = %q, want %q", got.Providers["linear"].Extra["api_key_ref"], "chit/linear-api-key")
	}
}

func TestDataDirRespectsXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if want := filepath.Join("/custom/data", "chit"); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestDataDirFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	home, _ := os.UserHomeDir()
	if want := filepath.Join(home, ".local", "share", "chit"); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}
