package tui

import (
	"github.com/bjcorder/chit/internal/cache"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

type rootContainersMsg struct {
	providerName string
	containers   []provider.Container
	err          error
}

type childContainersMsg struct {
	providerName string
	parentID     string
	containers   []provider.Container
	err          error
}

type issuesMsg struct {
	providerName string
	containerID  string
	issues       []domain.Issue
	err          error
}

type pullRequestsMsg struct {
	providerName string
	containerID  string
	prs          []domain.PullRequest
	err          error
}

type issueDetailMsg struct {
	providerName string
	issueID      string
	issue        domain.Issue
	err          error
}

type prDetailMsg struct {
	providerName string
	prID         string
	pr           domain.PullRequest
	err          error
}

type favoritesMsg struct {
	favorites []cache.Favorite
	err       error
}

type favoriteToggledMsg struct {
	providerName string
	containerID  string
	nowFavorite  bool
	err          error
}

type commentPostedMsg struct {
	providerName string
	issueID      string
	comment      domain.Comment
	err          error
}

type prActionMsg struct {
	providerName string
	prID         string
	action       string // "approve", "merge", "ready"
	err          error
}
