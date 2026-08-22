package tui

import (
	"strings"

	"github.com/bjcorder/chit/internal/domain"
)

// buildCommentRows renders each comment as a one-line "author: snippet"
// row for the reply-target picker ('v' on a detail screen).
func buildCommentRows(comments []domain.Comment) []row {
	rows := make([]row, len(comments))
	for i, c := range comments {
		snippet := firstLine(c.Body)
		if len(snippet) > 60 {
			snippet = snippet[:60] + "…"
		}
		rows[i] = row{label: c.Author.Login + ": " + snippet, sourceIndex: i}
	}
	return rows
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
