package tui

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// linkHint is one link/cross-reference chit found in a body or comment,
// labeled for the hint overlay (see injectLinkHints).
type linkHint struct {
	label  string
	target string // an http(s) URL, or "chit-ref:<text>" for an internal cross-reference
}

// mdLinkRe matches standard markdown links: [text](url).
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

// crossRefRe matches GitHub-style issue/PR references (#123) and
// Linear-style identifiers (ENG-456).
var crossRefRe = regexp.MustCompile(`#(\d+)|\b([A-Z][A-Z0-9]{1,9}-\d+)\b`)

// maxHintIndex bounds hintLabel's input well above any realistic number of
// links in a single issue/PR body — an explicit ceiling so the int->rune
// conversions below can never see a value anywhere near overflowing.
const maxHintIndex = 26*26*26 - 1

// hintLabel produces a-z, then aa-az, ba-bz, ... for i in [0, maxHintIndex]
// — the same base-26 letter-labeling scheme Vimium-style hint overlays use.
// Out-of-range input (which shouldn't happen — callers only ever pass a
// running count of hints found so far) returns "?" rather than a hint that
// could collide or misrender.
func hintLabel(i int) string {
	if i < 0 || i > maxHintIndex {
		return "?"
	}
	if i < 26 {
		return string(rune('a' + i))
	}
	return hintLabel(i/26-1) + string(rune('a'+i%26))
}

// injectLinkHints rewrites md so every link and cross-reference carries an
// inline "[label]" marker next to its visible text, riding along with
// glamour's own rendering and word-wrap rather than requiring a separate
// overlay pass against already-rendered, wrapped ANSI output. startIndex
// lets callers assign globally unique labels across multiple pieces of
// content (an issue body plus each of its comments).
func injectLinkHints(md string, startIndex int) (string, []linkHint) {
	var hints []linkHint
	next := func() string { return hintLabel(startIndex + len(hints)) }

	md = mdLinkRe.ReplaceAllStringFunc(md, func(m string) string {
		sub := mdLinkRe.FindStringSubmatch(m)
		text, url := sub[1], sub[2]
		label := next()
		hints = append(hints, linkHint{label: label, target: url})
		return "[" + text + " [" + label + "]](" + url + ")"
	})

	md = crossRefRe.ReplaceAllStringFunc(md, func(m string) string {
		label := next()
		hints = append(hints, linkHint{label: label, target: "chit-ref:" + m})
		return "[" + m + " [" + label + "]](chit-ref:" + m + ")"
	})

	return md, hints
}

// findHint returns the hint whose label exactly matches input, and whether
// input is at least a prefix of some hint's label (so callers can tell
// "no such hint, cancel" apart from "keep buffering keystrokes").
func findHint(hints []linkHint, input string) (hint linkHint, exact bool, isPrefix bool) {
	for _, h := range hints {
		if h.label == input {
			return h, true, true
		}
		if strings.HasPrefix(h.label, input) {
			isPrefix = true
		}
	}
	return linkHint{}, false, isPrefix
}

// openURL launches url in the user's browser ($BROWSER, falling back to
// xdg-open) without blocking the TUI. url comes from provider-supplied
// content (an issue/PR body), so it's restricted to http(s) links before
// ever reaching exec.Command — both to avoid handing a browser launcher a
// scheme it shouldn't act on (javascript:, file:, ...) and, more directly,
// to guarantee it can never start with '-' and be misread as a flag by
// whatever's on the other end of $BROWSER.
func openURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("refusing to open non-http(s) link: %q", url)
	}

	browser := os.Getenv("BROWSER")
	if browser == "" {
		browser = "xdg-open"
	}
	// #nosec G204,G702 -- browser is the user's own $BROWSER or the fixed "xdg-open"; url is validated http(s) above and passed as a single argv element, never shell-interpreted
	return exec.Command(browser, url).Start()
}

// crossRefIssueID converts a matched cross-reference like "#123" into the
// issue/PR ID chit uses to look it up within the same container.
// "ENG-456"-style Linear identifiers are already valid IDs as-is.
func crossRefIssueID(containerID, ref string) string {
	if strings.HasPrefix(ref, "#") {
		return containerID + "#" + ref[1:]
	}
	return ref
}
