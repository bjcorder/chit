// Command chit is a TUI for browsing and acting on issues and pull requests
// across pluggable providers (GitHub, Linear, ...).
package main

import (
	"fmt"
	"os"

	_ "github.com/bjcorder/chit/internal/provider/github"
	_ "github.com/bjcorder/chit/internal/provider/linear"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "chit:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "providers" {
		return runProviders(args[1:])
	}
	return runTUI()
}
