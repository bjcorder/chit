package tui

import (
	"strings"
	"testing"
)

func TestHintLabelOutOfRangeReturnsPlaceholder(t *testing.T) {
	for _, i := range []int{-1, maxHintIndex + 1} {
		if got := hintLabel(i); got != "?" {
			t.Errorf("hintLabel(%d) = %q, want %q", i, got, "?")
		}
	}
}

func TestHintLabelSequence(t *testing.T) {
	cases := map[int]string{0: "a", 1: "b", 25: "z", 26: "aa", 27: "ab", 51: "az", 52: "ba"}
	for i, want := range cases {
		if got := hintLabel(i); got != want {
			t.Errorf("hintLabel(%d) = %q, want %q", i, got, want)
		}
	}
}

func TestInjectLinkHintsMarkdownLink(t *testing.T) {
	out, hints := injectLinkHints("see [the docs](https://example.com/docs) for more", 0)
	if len(hints) != 1 || hints[0].label != "a" || hints[0].target != "https://example.com/docs" {
		t.Fatalf("hints = %+v", hints)
	}
	if !strings.Contains(out, "[a]") {
		t.Errorf("output missing inline hint marker: %q", out)
	}
}

func TestInjectLinkHintsCrossReference(t *testing.T) {
	out, hints := injectLinkHints("fixes #123 and relates to ENG-456", 0)
	if len(hints) != 2 {
		t.Fatalf("hints = %+v, want 2", hints)
	}
	if hints[0].target != "chit-ref:#123" || hints[1].target != "chit-ref:ENG-456" {
		t.Errorf("hints = %+v", hints)
	}
	if !strings.Contains(out, "[a]") || !strings.Contains(out, "[b]") {
		t.Errorf("output missing hint markers: %q", out)
	}
}

func TestInjectLinkHintsRespectsStartIndex(t *testing.T) {
	_, hints := injectLinkHints("[x](https://example.com)", 3)
	if len(hints) != 1 || hints[0].label != "d" {
		t.Fatalf("hints = %+v, want label starting from index 3 (\"d\")", hints)
	}
}

func TestFindHintExactMatch(t *testing.T) {
	hints := []linkHint{{label: "a", target: "url-a"}, {label: "b", target: "url-b"}}
	hint, exact, isPrefix := findHint(hints, "b")
	if !exact || !isPrefix || hint.target != "url-b" {
		t.Errorf("findHint(b) = %+v, %v, %v", hint, exact, isPrefix)
	}
}

func TestFindHintPrefixOnly(t *testing.T) {
	hints := []linkHint{{label: "aa", target: "url-aa"}}
	_, exact, isPrefix := findHint(hints, "a")
	if exact || !isPrefix {
		t.Errorf("findHint(a) with only \"aa\" present = exact=%v isPrefix=%v, want false,true", exact, isPrefix)
	}
}

func TestFindHintNoMatch(t *testing.T) {
	hints := []linkHint{{label: "a", target: "url-a"}}
	_, exact, isPrefix := findHint(hints, "z")
	if exact || isPrefix {
		t.Errorf("findHint(z) = exact=%v isPrefix=%v, want false,false", exact, isPrefix)
	}
}

func TestCrossRefIssueIDForGitHubStyle(t *testing.T) {
	if got := crossRefIssueID("cli/cli", "#123"); got != "cli/cli#123" {
		t.Errorf("crossRefIssueID = %q, want %q", got, "cli/cli#123")
	}
}

func TestCrossRefIssueIDForLinearStyle(t *testing.T) {
	if got := crossRefIssueID("team1", "ENG-456"); got != "ENG-456" {
		t.Errorf("crossRefIssueID = %q, want %q (identifier is already a valid Linear issue id)", got, "ENG-456")
	}
}
