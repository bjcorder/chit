package tui

import (
	"testing"
	"time"

	"github.com/bjcorder/chit/internal/domain"
)

// TestRenderMarkdownDoesNotHang guards against a real regression: glamour's
// WithAutoStyle() queries the terminal's background color at render time,
// which blocks for termenv.OSCTimeout (5s) once Bubble Tea's own stdin
// reader is running (see resolveGlamourStyle's doc comment for the full
// story). renderMarkdown must only ever use a pre-resolved standard style
// (WithStandardStyle) — this test fails loudly, instead of just slowly, if
// that ever regresses.
func TestRenderMarkdownDoesNotHang(t *testing.T) {
	done := make(chan string, 1)
	go func() {
		done <- renderMarkdown("some **markdown** with a [link](https://example.com)", 80, "dark")
	}()

	select {
	case out := <-done:
		if out == "" {
			t.Error("renderMarkdown returned empty output")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("renderMarkdown took longer than 2s — likely querying the terminal (WithAutoStyle) instead of using a pre-resolved style")
	}
}

func TestIssueDetailContentDoesNotHang(t *testing.T) {
	issue := domain.Issue{Number: "1", Title: "x", Body: "body", Comments: []domain.Comment{{Body: "c"}}}
	done := make(chan struct{}, 1)
	go func() {
		issueDetailContent(issue, 80, newStyles("dark"), false)
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("issueDetailContent took longer than 2s")
	}
}

func TestResolveGlamourStyleReturnsKnownStyle(t *testing.T) {
	done := make(chan string, 1)
	go func() { done <- resolveGlamourStyle() }()

	select {
	case got := <-done:
		switch got {
		case "dark", "light", "notty":
		default:
			t.Errorf("resolveGlamourStyle() = %q, want one of dark/light/notty", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resolveGlamourStyle took longer than 2s")
	}
}
