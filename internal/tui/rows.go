package tui

import (
	"strings"

	"github.com/bjcorder/chit/internal/domain"
)

// row is one line in a list-type screen (home, children, items). A header
// row (header != "") is a non-selectable section divider — cursor movement
// skips over it.
type row struct {
	header      string
	label       string
	badges      []domain.Badge
	favorite    bool
	section     string // which underlying data source sourceIndex refers to; screens with only one source can leave this empty
	sourceIndex int    // index into the screen's own typed data slice for that section
}

func renderBadges(badges []domain.Badge, st styles) string {
	parts := make([]string, len(badges))
	for i, b := range badges {
		parts[i] = st.badgeStyle(b.Color).Render(b.Label)
	}
	return strings.Join(parts, " ")
}

func renderRows(rows []row, cursor int, st styles) string {
	var b strings.Builder
	for i, r := range rows {
		if r.header != "" {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(st.sectionHeader.Render(r.header))
			b.WriteString("\n")
			continue
		}

		label := r.label
		if r.favorite {
			label = "★ " + label
		}
		if len(r.badges) > 0 {
			label += "  " + renderBadges(r.badges, st)
		}

		if i == cursor {
			b.WriteString(st.selected.Render("> " + label))
		} else {
			b.WriteString("  " + label)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// moveCursor returns the new cursor position after moving delta rows
// (skipping non-selectable header rows), clamped to the first/last
// selectable row rather than wrapping.
func moveCursor(rows []row, cursor, delta int) int {
	if len(rows) == 0 {
		return 0
	}

	step := 1
	if delta < 0 {
		step = -1
	}

	next := cursor
	for i := 0; i < len(rows); i++ {
		candidate := next + step
		if candidate < 0 || candidate >= len(rows) {
			break
		}
		next = candidate
		if rows[next].header == "" {
			if delta > 0 {
				delta--
			} else {
				delta++
			}
			if delta == 0 {
				return next
			}
		}
	}
	return firstSelectable(rows, next)
}

// firstSelectable returns from (if selectable) or the nearest selectable
// row, preferring later rows first, then earlier ones.
func firstSelectable(rows []row, from int) int {
	if from >= 0 && from < len(rows) && rows[from].header == "" {
		return from
	}
	for i := from; i < len(rows); i++ {
		if rows[i].header == "" {
			return i
		}
	}
	for i := from; i >= 0; i-- {
		if rows[i].header == "" {
			return i
		}
	}
	return 0
}

// initialCursor returns the index of the first selectable row.
func initialCursor(rows []row) int {
	for i, r := range rows {
		if r.header == "" {
			return i
		}
	}
	return 0
}
