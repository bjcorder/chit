package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

type issueDetailScreen struct {
	ctx          context.Context
	a            *app.App
	st           styles
	providerName string
	container    provider.Container
	issueID      string

	issue  domain.Issue
	loaded bool
	err    error

	vp viewport.Model
}

func newIssueDetailScreen(ctx context.Context, a *app.App, st styles, providerName string, container provider.Container, issueID string) *issueDetailScreen {
	return &issueDetailScreen{ctx: ctx, a: a, st: st, providerName: providerName, container: container, issueID: issueID, vp: viewport.New(0, 0)}
}

func (s *issueDetailScreen) Title() string {
	if s.loaded && s.err == nil {
		return "#" + s.issue.Number
	}
	return "issue"
}

func (s *issueDetailScreen) Init() tea.Cmd {
	return loadIssueDetail(s.ctx, s.a, s.providerName, s.container.ID, s.issueID, false)
}

func (s *issueDetailScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case issueDetailMsg:
		if msg.providerName != s.providerName || msg.issueID != s.issueID {
			return s, nil
		}
		s.loaded = true
		s.err = msg.err
		if msg.err == nil {
			s.issue = msg.issue
			s.vp.SetContent(issueDetailContent(s.issue, s.vp.Width, s.st))
		}
		return s, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			return s, loadIssueDetail(s.ctx, s.a, s.providerName, s.container.ID, s.issueID, true)
		}
	}

	var cmd tea.Cmd
	s.vp, cmd = s.vp.Update(msg)
	return s, cmd
}

// View lazily (re)sizes and re-renders the viewport when the available
// width/height changes, since the screen interface only learns the current
// terminal size here, not via a message the screen's Update sees directly.
func (s *issueDetailScreen) View(width, height int) string {
	if !s.loaded {
		return s.st.dim.Render("loading…")
	}
	if s.err != nil {
		return s.st.errorText.Render("error: " + s.err.Error())
	}

	if s.vp.Width != width || s.vp.Height != height {
		s.vp.Width, s.vp.Height = width, height
		s.vp.SetContent(issueDetailContent(s.issue, s.vp.Width, s.st))
	}
	return s.vp.View()
}

type prDetailScreen struct {
	ctx          context.Context
	a            *app.App
	st           styles
	providerName string
	container    provider.Container
	prID         string

	pr     domain.PullRequest
	loaded bool
	err    error

	vp viewport.Model
}

func newPRDetailScreen(ctx context.Context, a *app.App, st styles, providerName string, container provider.Container, prID string) *prDetailScreen {
	return &prDetailScreen{ctx: ctx, a: a, st: st, providerName: providerName, container: container, prID: prID, vp: viewport.New(0, 0)}
}

func (s *prDetailScreen) Title() string {
	if s.loaded && s.err == nil {
		return "#" + s.pr.Number
	}
	return "PR"
}

func (s *prDetailScreen) Init() tea.Cmd {
	return loadPRDetail(s.ctx, s.a, s.providerName, s.container.ID, s.prID, false)
}

func (s *prDetailScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case prDetailMsg:
		if msg.providerName != s.providerName || msg.prID != s.prID {
			return s, nil
		}
		s.loaded = true
		s.err = msg.err
		if msg.err == nil {
			s.pr = msg.pr
			s.vp.SetContent(prDetailContent(s.pr, s.vp.Width, s.st))
		}
		return s, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			return s, loadPRDetail(s.ctx, s.a, s.providerName, s.container.ID, s.prID, true)
		}
	}

	var cmd tea.Cmd
	s.vp, cmd = s.vp.Update(msg)
	return s, cmd
}

func (s *prDetailScreen) View(width, height int) string {
	if !s.loaded {
		return s.st.dim.Render("loading…")
	}
	if s.err != nil {
		return s.st.errorText.Render("error: " + s.err.Error())
	}

	if s.vp.Width != width || s.vp.Height != height {
		s.vp.Width, s.vp.Height = width, height
		s.vp.SetContent(prDetailContent(s.pr, s.vp.Width, s.st))
	}
	return s.vp.View()
}
