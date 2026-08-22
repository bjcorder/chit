package tui

import (
	"strings"
	"testing"
)

func plainRows(labels ...string) []row {
	rows := make([]row, len(labels))
	for i, l := range labels {
		rows[i] = row{label: l}
	}
	return rows
}

func TestMoveCursorDownSkipsHeaders(t *testing.T) {
	rows := []row{{header: "Section"}, {label: "a"}, {label: "b"}}
	got := moveCursor(rows, 1, 1)
	if got != 2 {
		t.Errorf("moveCursor down = %d, want 2", got)
	}
}

func TestMoveCursorUpSkipsHeaders(t *testing.T) {
	rows := []row{{header: "H1"}, {label: "a"}, {header: "H2"}, {label: "b"}}
	got := moveCursor(rows, 3, -1)
	if got != 1 {
		t.Errorf("moveCursor up = %d, want 1 (skipping H2)", got)
	}
}

func TestMoveCursorClampsAtEnd(t *testing.T) {
	rows := plainRows("a", "b")
	got := moveCursor(rows, 1, 1)
	if got != 1 {
		t.Errorf("moveCursor past end = %d, want clamped to 1", got)
	}
}

func TestMoveCursorClampsAtStart(t *testing.T) {
	rows := []row{{header: "H"}, {label: "a"}, {label: "b"}}
	got := moveCursor(rows, 1, -1)
	if got != 1 {
		t.Errorf("moveCursor before start = %d, want clamped to 1 (first selectable)", got)
	}
}

func TestMoveCursorOnEmptyRows(t *testing.T) {
	if got := moveCursor(nil, 0, 1); got != 0 {
		t.Errorf("moveCursor on empty = %d, want 0", got)
	}
}

func TestInitialCursorSkipsLeadingHeader(t *testing.T) {
	rows := []row{{header: "H"}, {label: "a"}}
	if got := initialCursor(rows); got != 1 {
		t.Errorf("initialCursor = %d, want 1", got)
	}
}

func TestInitialCursorOnAllHeaders(t *testing.T) {
	rows := []row{{header: "H1"}, {header: "H2"}}
	if got := initialCursor(rows); got != 0 {
		t.Errorf("initialCursor = %d, want 0 (degenerate fallback)", got)
	}
}

func TestRenderRowsIncludesAllLabels(t *testing.T) {
	rows := plainRows("alpha", "beta")
	out := renderRows(rows, 1, newStyles("notty"))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Errorf("output missing expected labels: %q", out)
	}
}
