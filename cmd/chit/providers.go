package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/bjcorder/chit/internal/config"
	"github.com/bjcorder/chit/internal/provider"
	"github.com/bjcorder/chit/internal/provider/linear"
)

func runProviders(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: chit providers <list|enable|disable|set-key> [name]")
	}

	switch args[0] {
	case "list":
		return providersList()
	case "enable":
		return providersSetEnabled(args[1:], true)
	case "disable":
		return providersSetEnabled(args[1:], false)
	case "set-key":
		return providersSetKey(args[1:])
	default:
		return fmt.Errorf("chit providers: unknown subcommand %q", args[0])
	}
}

func providersList() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	descriptors := provider.All()
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].Name < descriptors[j].Name })

	for _, d := range descriptors {
		status := "disabled"
		if cfg.Providers[d.Name].Enabled {
			status = "enabled"
		}
		fmt.Printf("%-10s %s\n", d.Name, status)
	}
	return nil
}

func providersSetEnabled(args []string, enabled bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: chit providers enable|disable <name>")
	}
	name := args[0]
	if _, ok := provider.Get(name); !ok {
		return fmt.Errorf("unknown provider %q (run `chit providers list` to see available providers)", name)
	}

	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	pc := cfg.Providers[name]
	pc.Enabled = enabled
	cfg.Providers[name] = pc
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	verb := "disabled"
	if enabled {
		verb = "enabled"
	}
	fmt.Printf("chit: %s %s\n", name, verb)
	return nil
}

func providersSetKey(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: chit providers set-key <name>")
	}
	name := args[0]

	switch name {
	case "linear":
		key, err := readSecret(fmt.Sprintf("Enter %s personal API key: ", name))
		if err != nil {
			return err
		}
		if key == "" {
			return fmt.Errorf("empty key, not saving")
		}
		if err := linear.SetAPIKey(key); err != nil {
			return err
		}
		fmt.Println("chit: API key saved.")
		return nil
	default:
		return fmt.Errorf("%q does not use a stored API key (e.g. github delegates auth entirely to the gh CLI)", name)
	}
}

// readSecret prompts on stderr and reads a line without echoing it when
// stdin is a terminal, falling back to a plain line read (e.g. piped input
// in scripts or tests) otherwise.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	defer fmt.Fprintln(os.Stderr)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return strings.TrimSpace(line), nil
}
