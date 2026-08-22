// Package app is chit's composition root: given the user's config, it
// opens the cache database and instantiates every enabled provider's
// IssueTracker/CodeHost, so both the CLI (`chit providers ...`) and the TUI
// share one place that turns config into running providers.
package app

import (
	"context"
	"fmt"

	"github.com/bjcorder/chit/internal/cache"
	"github.com/bjcorder/chit/internal/config"
	"github.com/bjcorder/chit/internal/provider"
)

// App holds everything a running chit session needs: the cache and every
// enabled provider's capabilities, keyed by provider name.
type App struct {
	Config        *config.Config
	Cache         *cache.Store
	IssueTrackers map[string]provider.IssueTracker
	CodeHosts     map[string]provider.CodeHost
}

// Load reads config.toml, opens the cache database, and instantiates every
// provider the config enables. A provider named in config but not
// registered (e.g. a typo, or a config written for a newer chit) is a hard
// error rather than a silent skip — chit should never run with less than
// what the user asked for without saying so.
func Load(ctx context.Context) (*App, error) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cachePath, err := cache.DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(ctx, cfgPath, cachePath)
}

// LoadFrom is Load with explicit paths, so tests can point it at a temp
// directory instead of the real XDG config/data locations.
func LoadFrom(ctx context.Context, cfgPath, cachePath string) (*App, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	store, err := cache.Open(cachePath)
	if err != nil {
		return nil, err
	}

	issueTrackers, codeHosts, err := instantiateProviders(ctx, cfg)
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	return &App{Config: cfg, Cache: store, IssueTrackers: issueTrackers, CodeHosts: codeHosts}, nil
}

func instantiateProviders(ctx context.Context, cfg *config.Config) (map[string]provider.IssueTracker, map[string]provider.CodeHost, error) {
	issueTrackers := map[string]provider.IssueTracker{}
	codeHosts := map[string]provider.CodeHost{}

	for name, pc := range cfg.Providers {
		if !pc.Enabled {
			continue
		}

		d, ok := provider.Get(name)
		if !ok {
			return nil, nil, fmt.Errorf("app: config enables unknown provider %q", name)
		}

		pcfg := provider.Config{Enabled: true, Extra: pc.Extra}
		if d.NewIssueTracker != nil {
			t, err := d.NewIssueTracker(ctx, pcfg)
			if err != nil {
				return nil, nil, fmt.Errorf("app: initializing %s issue tracker: %w", name, err)
			}
			issueTrackers[name] = t
		}
		if d.NewCodeHost != nil {
			h, err := d.NewCodeHost(ctx, pcfg)
			if err != nil {
				return nil, nil, fmt.Errorf("app: initializing %s code host: %w", name, err)
			}
			codeHosts[name] = h
		}
	}

	return issueTrackers, codeHosts, nil
}

func (a *App) Close() error {
	return a.Cache.Close()
}
