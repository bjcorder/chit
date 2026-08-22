package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

// containerLister is satisfied by both provider.IssueTracker and
// provider.CodeHost — both declare identical ListRootContainers/
// ListChildContainers methods, since a container (an org/repo, a
// workspace/team) is a navigation concept shared by both capabilities, not
// specific to issues or PRs.
type containerLister interface {
	ListRootContainers(ctx context.Context) ([]provider.Container, error)
	ListChildContainers(ctx context.Context, parentID string) ([]provider.Container, error)
}

// containerListerFor picks whichever capability a provider has for listing
// containers. For GitHub, IssueTracker and CodeHost are backed by the same
// object and return identical results either way; Linear only has
// IssueTracker.
func containerListerFor(a *app.App, providerName string) (containerLister, bool) {
	if t, ok := a.IssueTrackers[providerName]; ok {
		return t, true
	}
	if h, ok := a.CodeHosts[providerName]; ok {
		return h, true
	}
	return nil, false
}

func loadRootContainers(ctx context.Context, a *app.App, providerName string, forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			if containers, ok, err := a.Cache.RootContainers(ctx, providerName); err == nil && ok {
				return rootContainersMsg{providerName: providerName, containers: containers}
			}
		}

		lister, ok := containerListerFor(a, providerName)
		if !ok {
			return rootContainersMsg{providerName: providerName, err: errUnknownProvider(providerName)}
		}
		containers, err := lister.ListRootContainers(ctx)
		if err != nil {
			return rootContainersMsg{providerName: providerName, err: err}
		}
		_ = a.Cache.SetRootContainers(ctx, providerName, containers)
		return rootContainersMsg{providerName: providerName, containers: containers}
	}
}

func loadChildContainers(ctx context.Context, a *app.App, providerName, parentID string, forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			if containers, ok, err := a.Cache.ChildContainers(ctx, providerName, parentID); err == nil && ok {
				return childContainersMsg{providerName: providerName, parentID: parentID, containers: containers}
			}
		}

		lister, ok := containerListerFor(a, providerName)
		if !ok {
			return childContainersMsg{providerName: providerName, parentID: parentID, err: errUnknownProvider(providerName)}
		}
		containers, err := lister.ListChildContainers(ctx, parentID)
		if err != nil {
			return childContainersMsg{providerName: providerName, parentID: parentID, err: err}
		}
		_ = a.Cache.SetChildContainers(ctx, providerName, parentID, containers)
		return childContainersMsg{providerName: providerName, parentID: parentID, containers: containers}
	}
}

func loadIssues(ctx context.Context, a *app.App, providerName, containerID string, forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			if issues, ok, err := a.Cache.Issues(ctx, providerName, containerID); err == nil && ok {
				return issuesMsg{providerName: providerName, containerID: containerID, issues: issues}
			}
		}

		tracker, ok := a.IssueTrackers[providerName]
		if !ok {
			return issuesMsg{providerName: providerName, containerID: containerID, err: errUnknownProvider(providerName)}
		}
		issues, err := tracker.ListIssues(ctx, containerID)
		if err != nil {
			return issuesMsg{providerName: providerName, containerID: containerID, err: err}
		}
		_ = a.Cache.SetIssues(ctx, providerName, containerID, issues)
		return issuesMsg{providerName: providerName, containerID: containerID, issues: issues}
	}
}

func loadPullRequests(ctx context.Context, a *app.App, providerName, containerID string, forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			if prs, ok, err := a.Cache.PullRequests(ctx, providerName, containerID); err == nil && ok {
				return pullRequestsMsg{providerName: providerName, containerID: containerID, prs: prs}
			}
		}

		host, ok := a.CodeHosts[providerName]
		if !ok {
			return pullRequestsMsg{providerName: providerName, containerID: containerID, err: errUnknownProvider(providerName)}
		}
		prs, err := host.ListPullRequests(ctx, containerID)
		if err != nil {
			return pullRequestsMsg{providerName: providerName, containerID: containerID, err: err}
		}
		_ = a.Cache.SetPullRequests(ctx, providerName, containerID, prs)
		return pullRequestsMsg{providerName: providerName, containerID: containerID, prs: prs}
	}
}

func loadIssueDetail(ctx context.Context, a *app.App, providerName, containerID, issueID string, forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			if issue, ok, err := a.Cache.IssueDetail(ctx, providerName, issueID); err == nil && ok {
				return issueDetailMsg{providerName: providerName, issueID: issueID, issue: issue}
			}
		}

		tracker, ok := a.IssueTrackers[providerName]
		if !ok {
			return issueDetailMsg{providerName: providerName, issueID: issueID, err: errUnknownProvider(providerName)}
		}
		issue, err := tracker.GetIssue(ctx, containerID, issueID)
		if err != nil {
			return issueDetailMsg{providerName: providerName, issueID: issueID, err: err}
		}
		_ = a.Cache.SetIssueDetail(ctx, providerName, issueID, issue)
		return issueDetailMsg{providerName: providerName, issueID: issueID, issue: issue}
	}
}

func loadPRDetail(ctx context.Context, a *app.App, providerName, containerID, prID string, forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			if pr, ok, err := a.Cache.PRDetail(ctx, providerName, prID); err == nil && ok {
				return prDetailMsg{providerName: providerName, prID: prID, pr: pr}
			}
		}

		host, ok := a.CodeHosts[providerName]
		if !ok {
			return prDetailMsg{providerName: providerName, prID: prID, err: errUnknownProvider(providerName)}
		}
		pr, err := host.GetPullRequest(ctx, containerID, prID)
		if err != nil {
			return prDetailMsg{providerName: providerName, prID: prID, err: err}
		}
		_ = a.Cache.SetPRDetail(ctx, providerName, prID, pr)
		return prDetailMsg{providerName: providerName, prID: prID, pr: pr}
	}
}

func loadFavorites(ctx context.Context, a *app.App) tea.Cmd {
	return func() tea.Msg {
		favorites, err := a.Cache.Favorites(ctx)
		return favoritesMsg{favorites: favorites, err: err}
	}
}

func toggleFavorite(ctx context.Context, a *app.App, providerName string, c provider.Container, currentlyFavorite bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if currentlyFavorite {
			err = a.Cache.RemoveFavorite(ctx, providerName, c.ID)
		} else {
			err = a.Cache.AddFavorite(ctx, providerName, c)
		}
		return favoriteToggledMsg{providerName: providerName, containerID: c.ID, nowFavorite: !currentlyFavorite, err: err}
	}
}

func postComment(ctx context.Context, a *app.App, providerName, issueID, body string) tea.Cmd {
	return func() tea.Msg {
		tracker, ok := a.IssueTrackers[providerName]
		if !ok {
			return commentPostedMsg{providerName: providerName, issueID: issueID, err: errUnknownProvider(providerName)}
		}
		c, err := tracker.AddComment(ctx, issueID, body)
		return commentPostedMsg{providerName: providerName, issueID: issueID, comment: c, err: err}
	}
}

func postReply(ctx context.Context, a *app.App, providerName, issueID, parentCommentID, body string) tea.Cmd {
	return func() tea.Msg {
		tracker, ok := a.IssueTrackers[providerName]
		if !ok {
			return commentPostedMsg{providerName: providerName, issueID: issueID, err: errUnknownProvider(providerName)}
		}
		c, err := tracker.ReplyToComment(ctx, issueID, parentCommentID, body)
		return commentPostedMsg{providerName: providerName, issueID: issueID, comment: c, err: err}
	}
}

func approvePR(ctx context.Context, a *app.App, providerName, prID string) tea.Cmd {
	return func() tea.Msg {
		host, ok := a.CodeHosts[providerName]
		if !ok {
			return prActionMsg{providerName: providerName, prID: prID, action: "approve", err: errUnknownProvider(providerName)}
		}
		err := host.ApprovePullRequest(ctx, prID, "")
		return prActionMsg{providerName: providerName, prID: prID, action: "approve", err: err}
	}
}

func mergePR(ctx context.Context, a *app.App, providerName, prID string, method domain.MergeMethod) tea.Cmd {
	return func() tea.Msg {
		host, ok := a.CodeHosts[providerName]
		if !ok {
			return prActionMsg{providerName: providerName, prID: prID, action: "merge", err: errUnknownProvider(providerName)}
		}
		err := host.MergePullRequest(ctx, prID, method, false)
		return prActionMsg{providerName: providerName, prID: prID, action: "merge", err: err}
	}
}

func markReady(ctx context.Context, a *app.App, providerName, prID string) tea.Cmd {
	return func() tea.Msg {
		host, ok := a.CodeHosts[providerName]
		if !ok {
			return prActionMsg{providerName: providerName, prID: prID, action: "ready", err: errUnknownProvider(providerName)}
		}
		err := host.MarkReadyForReview(ctx, prID)
		return prActionMsg{providerName: providerName, prID: prID, action: "ready", err: err}
	}
}

func errUnknownProvider(name string) error {
	return fmt.Errorf("tui: no provider registered as %q", name)
}
