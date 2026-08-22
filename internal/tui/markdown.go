package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/bjcorder/chit/internal/domain"
)

// renderMarkdown renders md at the given width, falling back to the raw
// text if glamour fails to construct a renderer (e.g. an unusual
// $COLORTERM) rather than showing nothing.
func renderMarkdown(md string, width int) string {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(width))
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
}

func renderComments(comments []domain.Comment, width int, st styles) string {
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
		b.WriteString(renderMarkdown(c.Body, width))
		b.WriteString("\n")
	}
	return b.String()
}

func issueDetailContent(issue domain.Issue, width int, st styles) string {
	var b strings.Builder
	b.WriteString(st.title.Render(fmt.Sprintf("#%s %s", issue.Number, issue.Title)))
	b.WriteString("\n")
	if len(issue.Badges) > 0 {
		b.WriteString(renderBadges(issue.Badges, st))
		b.WriteString("\n")
	}
	b.WriteString(st.dim.Render("opened by " + issue.Author.Login))
	b.WriteString("\n\n")
	b.WriteString(renderMarkdown(issue.Body, width))

	if len(issue.Comments) > 0 {
		b.WriteString("\n")
		b.WriteString(renderComments(issue.Comments, width, st))
	}
	return b.String()
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

func prDetailContent(pr domain.PullRequest, width int, st styles) string {
	var b strings.Builder
	b.WriteString(st.title.Render(fmt.Sprintf("#%s %s", pr.Number, pr.Title)))
	b.WriteString("\n")
	if len(pr.Badges) > 0 {
		b.WriteString(renderBadges(pr.Badges, st))
		b.WriteString("\n")
	}
	b.WriteString(st.dim.Render(fmt.Sprintf("%s → %s · opened by %s", pr.HeadBranch, pr.BaseBranch, pr.Author.Login)))
	b.WriteString("\n\n")
	b.WriteString(renderMarkdown(pr.Body, width))

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
		b.WriteString(renderComments(pr.Comments, width, st))
	}
	return b.String()
}
