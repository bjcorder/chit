// Command chit is a TUI for browsing and acting on issues and pull requests
// across pluggable providers (GitHub, Linear, ...).
package main

import (
	"fmt"
	"os"

	"github.com/bjcorder/chit/internal/config"
	"github.com/bjcorder/chit/internal/provider"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "chit:", err)
		os.Exit(1)
	}
}

func run() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}

	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	var enabled []string
	for name, pc := range cfg.Providers {
		if !pc.Enabled {
			continue
		}
		if _, ok := provider.Get(name); !ok {
			return fmt.Errorf("config enables unknown provider %q", name)
		}
		enabled = append(enabled, name)
	}

	if len(enabled) == 0 {
		fmt.Printf("chit: no providers enabled — edit %s or run `chit providers enable <name>`\n", path)
		return nil
	}

	fmt.Println("chit: TUI not built yet — enabled providers:", enabled)
	return nil
}
