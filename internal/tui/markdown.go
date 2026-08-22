package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/bjcorder/chit/internal/domain"
)

// renderMarkdown renders md at the given width using glamourStyle (a
// standard glamour style name resolved once at startup — see
// resolveGlamourStyle; never glamour.WithAutoStyle here, which queries the
// terminal at render time and hangs when called while Bubble Tea is
// running). Falls back to the raw text if glamour fails to construct a
// renderer rather than showing nothing.
func renderMarkdown(md string, width int, glamourStyle string) string {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(glamourStyle), glamour.WithWordWrap(width))
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
}

// renderBody renders body as markdown, optionally injecting link hints
// first (see injectLinkHints). hints accumulates so labels stay unique
// across a whole screen's body + comments.
func renderBody(body string, width int, glamourStyle string, hintMode bool, hints *[]linkHint) string {
	if hintMode {
		var bodyHints []linkHint
		body, bodyHints = injectLinkHints(body, len(*hints))
		*hints = append(*hints, bodyHints...)
	}
	return renderMarkdown(body, width, glamourStyle)
}

func renderComments(comments []domain.Comment, width int, st styles, hintMode bool, hints *[]linkHint) string {
	var b strings.Builder
	b.WriteString(st.sectionHeader.Render(fmt.Sprintf("Comments (%d)", len(comments))))
	b.WriteString("\n")
	for _, c := range comments {
		indent := ""
		if c.ParentID != "" {
			indent = "  ↳ "
		}
		b.WriteString(st.dim.Render(indent + c.Author.Login + " · " + c.CreatedAt.Format("2006-01-02 15:04")))
		b.WriteString("\n")
		b.WriteString(renderBody(c.Body, width, st.glamourStyle, hintMode, hints))
		b.WriteString("\n")
	}
	return b.String()
}

// issueDetailContent renders an issue for display. When hintMode is true,
// every link/cross-reference in the body and comments gets an inline
// "[label]" marker, and the returned hints let the screen resolve a
// keypress back to a target.
func issueDetailContent(issue domain.Issue, width int, st styles, hintMode bool) (string, []linkHint) {
	var hints []linkHint
	var b strings.Builder

	b.WriteString(st.title.Render(fmt.Sprintf("#%s %s", issue.Number, issue.Title)))
	b.WriteString("\n")
	if len(issue.Badges) > 0 {
		b.WriteString(renderBadges(issue.Badges, st))
		b.WriteString("\n")
	}
	b.WriteString(st.dim.Render("opened by " + issue.Author.Login))
	b.WriteString("\n\n")
	b.WriteString(renderBody(issue.Body, width, st.glamourStyle, hintMode, &hints))

	if len(issue.Comments) > 0 {
		b.WriteString("\n")
		b.WriteString(renderComments(issue.Comments, width, st, hintMode, &hints))
	}
	return b.String(), hints
}

func checkBadge(status domain.CheckStatus, st styles) string {
	switch status {
	case domain.CheckPass:
		return st.badgeStyle("green").Render("✓")
	case domain.CheckFail:
		return st.badgeStyle("red").Render("✗")
	case domain.CheckPending:
		return st.badgeStyle("yellow").Render("●")
	case domain.CheckSkipped:
		return st.badgeStyle("gray").Render("−")
	case domain.CheckCancelled:
		return st.badgeStyle("gray").Render("⊘")
	default:
		return st.badgeStyle("gray").Render("?")
	}
}

func prDetailContent(pr domain.PullRequest, width int, st styles, hintMode bool) (string, []linkHint) {
	var hints []linkHint
	var b strings.Builder

	b.WriteString(st.title.Render(fmt.Sprintf("#%s %s", pr.Number, pr.Title)))
	b.WriteString("\n")
	if len(pr.Badges) > 0 {
		b.WriteString(renderBadges(pr.Badges, st))
		b.WriteString("\n")
	}
	b.WriteString(st.dim.Render(fmt.Sprintf("%s → %s · opened by %s", pr.HeadBranch, pr.BaseBranch, pr.Author.Login)))
	b.WriteString("\n\n")
	b.WriteString(renderBody(pr.Body, width, st.glamourStyle, hintMode, &hints))

	if len(pr.Commits) > 0 {
		b.WriteString("\n")
		b.WriteString(st.sectionHeader.Render(fmt.Sprintf("Commits (%d)", len(pr.Commits))))
		b.WriteString("\n")
		for _, c := range pr.Commits {
			sha := c.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			b.WriteString("  • " + st.dim.Render(sha) + " " + c.Message + "\n")
		}
	}

	if len(pr.Checks) > 0 {
		b.WriteString("\n")
		b.WriteString(st.sectionHeader.Render(fmt.Sprintf("Checks (%d)", len(pr.Checks))))
		b.WriteString("\n")
		for _, c := range pr.Checks {
			b.WriteString("  " + checkBadge(c.Status, st) + " " + c.Name + "\n")
		}
	}

	if len(pr.Comments) > 0 {
		b.WriteString("\n")
		b.WriteString(renderComments(pr.Comments, width, st, hintMode, &hints))
	}
	return b.String(), hints
}
