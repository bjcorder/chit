package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bjcorder/chit/internal/domain"
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

func TestScrollWindowFitsEverythingWhenShorterThanHeight(t *testing.T) {
	start, end := scrollWindow(5, 2, 10)
	if start != 0 || end != 5 {
		t.Errorf("scrollWindow = (%d,%d), want (0,5) when total <= height", start, end)
	}
}

func TestScrollWindowCentersCursor(t *testing.T) {
	// 100 rows, height 10, cursor in the middle — window should be
	// centered around the cursor, not pinned to either edge.
	start, end := scrollWindow(100, 50, 10)
	if start > 50 || end <= 50 {
		t.Fatalf("window (%d,%d) does not contain cursor 50", start, end)
	}
	if end-start != 10 {
		t.Errorf("window size = %d, want 10", end-start)
	}
}

func TestScrollWindowClampsAtStart(t *testing.T) {
	start, end := scrollWindow(100, 0, 10)
	if start != 0 || end != 10 {
		t.Errorf("scrollWindow at cursor=0 = (%d,%d), want (0,10)", start, end)
	}
}

func TestScrollWindowClampsAtEnd(t *testing.T) {
	start, end := scrollWindow(100, 99, 10)
	if start != 90 || end != 100 {
		t.Errorf("scrollWindow at cursor=99 = (%d,%d), want (90,100)", start, end)
	}
}

func TestScrollWindowCursorAlwaysInRange(t *testing.T) {
	for _, total := range []int{1, 5, 10, 50} {
		for cursor := 0; cursor < total; cursor++ {
			for _, height := range []int{1, 2, 3, 7} {
				start, end := scrollWindow(total, cursor, height)
				if cursor < start || cursor >= end {
					t.Fatalf("total=%d cursor=%d height=%d -> window (%d,%d) excludes cursor", total, cursor, height, start, end)
				}
			}
		}
	}
}

func TestRenderRowsScrolledFitsWithoutWindowing(t *testing.T) {
	rows := plainRows("a", "b", "c")
	out := renderRowsScrolled(rows, 1, 10, newStyles("notty"))
	if strings.Contains(out, "more above") || strings.Contains(out, "more below") {
		t.Errorf("unexpected scroll indicators when everything fits: %q", out)
	}
}

func TestRenderRowsScrolledShowsIndicators(t *testing.T) {
	labels := make([]string, 30)
	for i := range labels {
		labels[i] = fmt.Sprintf("item-%02d", i)
	}
	rows := plainRows(labels...)

	// Cursor in the middle: both "more above" and "more below" should show.
	out := renderRowsScrolled(rows, 15, 10, newStyles("notty"))
	if !strings.Contains(out, "more above") || !strings.Contains(out, "more below") {
		t.Errorf("expected both scroll indicators with cursor in the middle: %q", out)
	}

	// Cursor at the very top: only "more below".
	out = renderRowsScrolled(rows, 0, 10, newStyles("notty"))
	if strings.Contains(out, "more above") {
		t.Errorf("unexpected 'more above' with cursor at the top: %q", out)
	}
	if !strings.Contains(out, "more below") {
		t.Errorf("expected 'more below' with cursor at the top: %q", out)
	}
	if !strings.Contains(out, "item-00") {
		t.Errorf("expected the row at the cursor to be visible: %q", out)
	}
}

func TestRenderRowsScrolledKeepsSelectedRowVisible(t *testing.T) {
	labels := make([]string, 50)
	for i := range labels {
		labels[i] = fmt.Sprintf("item-%02d", i)
	}
	rows := plainRows(labels...)

	out := renderRowsScrolled(rows, 42, 10, newStyles("notty"))
	if !strings.Contains(out, "item-42") {
		t.Errorf("selected row item-42 not present in scrolled output: %q", out)
	}
}

func TestRenderRowsPutsLeadingBadgeBeforeTitleAndTrailingBadgesAfter(t *testing.T) {
	st := newStyles("notty")
	leading := domain.Badge{Label: "open", Color: "green"}
	r := row{
		prefix:       "#123",
		leadingBadge: &leading,
		label:        "Fix the thing",
		badges:       []domain.Badge{{Label: "bug", Color: "label"}},
	}
	out := renderRows([]row{r}, 0, st)

	numberIdx := strings.Index(out, "#123")
	openIdx := strings.Index(out, "open")
	titleIdx := strings.Index(out, "Fix the thing")
	bugIdx := strings.Index(out, "bug")

	if numberIdx < 0 || openIdx < 0 || titleIdx < 0 || bugIdx < 0 {
		t.Fatalf("output missing expected content: %q", out)
	}
	if numberIdx >= openIdx || openIdx >= titleIdx || titleIdx >= bugIdx {
		t.Errorf("wrong order — want #123, then open, then title, then bug; got: %q", out)
	}
}

func TestRenderRowsWithoutPrefixOrLeadingBadgeUnchanged(t *testing.T) {
	st := newStyles("notty")
	r := row{label: "some-repo", badges: []domain.Badge{{Label: "x", Color: "gray"}}}
	out := renderRows([]row{r}, 0, st)
	if !strings.Contains(out, "some-repo") || !strings.Contains(out, "x") {
		t.Errorf("output missing expected content: %q", out)
	}
}
