package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/config"
)

// runTUI is a placeholder until the internal/tui package exists: it wires
// up config, the cache, and every enabled provider (proving the whole
// composition root works end to end) and reports what it would have
// launched the TUI with.
func runTUI() error {
	a, err := app.Load(context.Background())
	if err != nil {
		return err
	}
	defer a.Close() //nolint:errcheck

	if len(a.IssueTrackers) == 0 && len(a.CodeHosts) == 0 {
		path, _ := config.DefaultPath()
		fmt.Printf("chit: no providers enabled — edit %s or run `chit providers enable <name>`\n", path)
		return nil
	}

	fmt.Println("chit: TUI not built yet — providers ready:", enabledProviderNames(a))
	return nil
}

func enabledProviderNames(a *app.App) []string {
	seen := map[string]bool{}
	var names []string
	for name := range a.IssueTrackers {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for name := range a.CodeHosts {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
