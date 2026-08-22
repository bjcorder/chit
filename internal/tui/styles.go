package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	sectionHeader lipgloss.Style
	selected      lipgloss.Style
	dim           lipgloss.Style
	errorText     lipgloss.Style
	badge         map[string]lipgloss.Style
	help          lipgloss.Style
	title         lipgloss.Style
}

func newStyles() styles {
	badgeBase := lipgloss.NewStyle().Padding(0, 1).Bold(true)
	colorOf := func(fg string) lipgloss.Style { return badgeBase.Foreground(lipgloss.Color(fg)) }

	return styles{
		sectionHeader: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginTop(1),
		selected:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		errorText:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		help:          lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
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
