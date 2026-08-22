package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

type styles struct {
	sectionHeader lipgloss.Style
	selected      lipgloss.Style
	dim           lipgloss.Style
	errorText     lipgloss.Style
	badge         map[string]lipgloss.Style
	help          lipgloss.Style
	title         lipgloss.Style

	// glamourStyle is a glamour standard style name ("dark", "light", or
	// "notty") resolved exactly once, before the Bubble Tea program starts
	// (see resolveGlamourStyle) — never glamour.WithAutoStyle(), which
	// queries the terminal's background color at render time and hangs for
	// termenv.OSCTimeout (5s) when that query races Bubble Tea's own stdin
	// reader, which it always does once the program is running.
	glamourStyle string
}

func newStyles(glamourStyle string) styles {
	badgeBase := lipgloss.NewStyle().Padding(0, 1).Bold(true)
	colorOf := func(fg string) lipgloss.Style { return badgeBase.Foreground(lipgloss.Color(fg)) }

	return styles{
		sectionHeader: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginTop(1),
		selected:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		errorText:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		help:          lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		glamourStyle:  glamourStyle,
		badge: map[string]lipgloss.Style{
			"green":  colorOf("2"),
			"red":    colorOf("1"),
			"yellow": colorOf("3"),
			"blue":   colorOf("4"),
			"purple": colorOf("5"),
			"orange": colorOf("208"),
			"gray":   colorOf("240"),
			"label":  colorOf("7"),
		},
	}
}

func (s styles) badgeStyle(color string) lipgloss.Style {
	if st, ok := s.badge[color]; ok {
		return st
	}
	return s.badge["gray"]
}

// resolveGlamourStyle picks glamour's dark/light/notty standard style by
// querying the terminal exactly once. This MUST run before
// tea.NewProgram(...).Run() takes over stdin: termenv.HasDarkBackground
// writes an OSC query and blocks reading the terminal's response, and once
// Bubble Tea's own input loop is running it's also reading stdin, so the
// two race — in practice the query never wins and this blocks for the
// full termenv.OSCTimeout (5s) instead of returning immediately. Calling
// it here, before the program starts, avoids the race entirely; nothing
// after this point may call it again.
func resolveGlamourStyle() string {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return "notty"
	}
	if termenv.HasDarkBackground() {
		return "dark"
	}
	return "light"
}
