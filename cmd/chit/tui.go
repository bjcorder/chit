package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/config"
	"github.com/bjcorder/chit/internal/tui"
)

func runTUI() error {
	ctx := context.Background()

	a, err := app.Load(ctx)
	if err != nil {
		return err
	}
	defer a.Close() //nolint:errcheck

	if len(a.IssueTrackers) == 0 && len(a.CodeHosts) == 0 {
		path, _ := config.DefaultPath()
		fmt.Printf("chit: no providers enabled — edit %s or run `chit providers enable <name>`\n", path)
		return nil
	}

	m := tui.New(ctx, a)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}
	return nil
}
