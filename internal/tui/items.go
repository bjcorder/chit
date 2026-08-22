package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

// itemsScreen lists issues (or, for providers with CodeHost support, PRs —
// toggled with 'p') in a single leaf container.
type itemsScreen struct {
	ctx          context.Context
	a            *app.App
	st           styles
	providerName string
	container    provider.Container
	hasPRs       bool

	showPRs bool

	issues       []domain.Issue
	issuesLoaded bool
	issuesErr    error

	prs       []domain.PullRequest
	prsLoaded bool
	prsErr    error

	rows   []row
	cursor int
	status string
}

func newItemsScreen(ctx context.Context, a *app.App, st styles, providerName string, container provider.Container) *itemsScreen {
	_, hasPRs := a.CodeHosts[providerName]
	return &itemsScreen{ctx: ctx, a: a, st: st, providerName: providerName, container: container, hasPRs: hasPRs}
}

func (s *itemsScreen) Title() string { return s.container.Name }

func (s *itemsScreen) Init() tea.Cmd {
	return loadIssues(s.ctx, s.a, s.providerName, s.container.ID, false)
}

func (s *itemsScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case issuesMsg:
		if msg.providerName != s.providerName || msg.containerID != s.container.ID {
			return s, nil
		}
		s.issuesLoaded = true
		s.issuesErr = msg.err
		if msg.err == nil {
			s.issues = msg.issues
		}
		s.rebuildRows()
		return s, nil

	case pullRequestsMsg:
		if msg.providerName != s.providerName || msg.containerID != s.container.ID {
			return s, nil
		}
		s.prsLoaded = true
		s.prsErr = msg.err
		if msg.err == nil {
			s.prs = msg.prs
		}
		s.rebuildRows()
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			s.cursor = moveCursor(s.rows, s.cursor, 1)
		case "k", "up":
			s.cursor = moveCursor(s.rows, s.cursor, -1)
		case "p":
			if s.hasPRs {
				s.showPRs = !s.showPRs
				s.cursor = 0
				s.rebuildRows()
				if s.showPRs && !s.prsLoaded {
					s.status = "loading PRs…"
					return s, loadPullRequests(s.ctx, s.a, s.providerName, s.container.ID, false)
				}
			}
		case "r":
			s.status = "refreshing…"
			if s.showPRs {
				return s, loadPullRequests(s.ctx, s.a, s.providerName, s.container.ID, true)
			}
			return s, loadIssues(s.ctx, s.a, s.providerName, s.container.ID, true)
		case "enter", "l":
			return s, s.selectAtCursor()
		}
	}
	return s, nil
}

func (s *itemsScreen) rebuildRows() {
	var rows []row
	if s.showPRs {
		for i, pr := range s.prs {
			rows = append(rows, row{label: "#" + pr.Number + " " + pr.Title, badges: pr.Badges, sourceIndex: i})
		}
	} else {
		for i, issue := range s.issues {
			rows = append(rows, row{label: "#" + issue.Number + " " + issue.Title, badges: issue.Badges, sourceIndex: i})
		}
	}
	s.rows = rows
	if s.cursor >= len(rows) {
		s.cursor = initialCursor(rows)
	}
}

func (s *itemsScreen) selectAtCursor() tea.Cmd {
	if s.cursor < 0 || s.cursor >= len(s.rows) {
		return nil
	}
	idx := s.rows[s.cursor].sourceIndex
	if s.showPRs {
		return pushScreen(newPRDetailScreen(s.ctx, s.a, s.st, s.providerName, s.container, s.prs[idx].ID))
	}
	return pushScreen(newIssueDetailScreen(s.ctx, s.a, s.st, s.providerName, s.container, s.issues[idx].ID))
}

func (s *itemsScreen) View(width, height int) string {
	loaded, err, empty := s.issuesLoaded, s.issuesErr, len(s.issues) == 0
	kind := "issues"
	if s.showPRs {
		loaded, err, empty = s.prsLoaded, s.prsErr, len(s.prs) == 0
		kind = "PRs"
	}

	if !loaded {
		return s.st.dim.Render("loading…")
	}
	if err != nil {
		return s.st.errorText.Render("error: " + err.Error())
	}

	mode := s.st.dim.Render("[" + kind + "]")
	if empty {
		return mode + "\n" + s.st.dim.Render("(none)")
	}

	rowsHeight := height - 2 // reserve one line each for the mode indicator and the status message
	body := mode + "\n" + renderRowsScrolled(s.rows, s.cursor, rowsHeight, s.st)
	if s.status != "" {
		body += "\n" + s.st.dim.Render(s.status)
	}
	return body
}
