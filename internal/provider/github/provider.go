// Package github implements chit's GitHub provider by shelling out to the
// `gh` CLI for every operation. gh already owns GitHub authentication
// end-to-end (it resolves and stores its own token via the OS keyring), so
// this package never handles a GitHub token directly.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

func init() {
	provider.Register(provider.Descriptor{
		Name: "github",
		NewIssueTracker: func(ctx context.Context, cfg provider.Config) (provider.IssueTracker, error) {
			return New(), nil
		},
		NewCodeHost: func(ctx context.Context, cfg provider.Config) (provider.CodeHost, error) {
			return New(), nil
		},
	})
}

// Provider implements both provider.IssueTracker and provider.CodeHost for
// GitHub.
//
// domain.Issue.ID and domain.PullRequest.ID are encoded as "owner/repo#N":
// both interfaces' comment/review/merge methods take only an issueID/prID,
// with no separate containerID, so each ID has to carry enough information
// on its own for Provider to reconstruct the `gh ... -R owner/repo <N>`
// invocation it needs.
type Provider struct {
	runner Runner

	userOnce  sync.Once
	userLogin string
	userErr   error
}

// New returns a Provider backed by the real gh binary.
func New() *Provider {
	return &Provider{runner: execRunner{}}
}

func (p *Provider) Name() string { return "github" }

func (p *Provider) run(ctx context.Context, args ...string) ([]byte, error) {
	return p.runner.Run(ctx, args...)
}

func runJSON[T any](ctx context.Context, p *Provider, args ...string) (T, error) {
	var zero T
	out, err := p.run(ctx, args...)
	if err != nil {
		return zero, err
	}
	var v T
	if err := json.Unmarshal(out, &v); err != nil {
		return zero, fmt.Errorf("parsing gh output: %w", err)
	}
	return v, nil
}

// currentUser returns the authenticated user's login, fetched once and
// cached for the lifetime of this Provider.
func (p *Provider) currentUser(ctx context.Context) (string, error) {
	p.userOnce.Do(func() {
		u, err := runJSON[ghUser](ctx, p, "api", "user")
		p.userLogin, p.userErr = u.Login, err
	})
	return p.userLogin, p.userErr
}

type ghUser struct {
	Login string `json:"login"`
}

type ghLabel struct {
	Name string `json:"name"`
}

// issueRef formats a domain-facing "owner/repo#N" ID.
func issueRef(repo string, number int) string {
	return fmt.Sprintf("%s#%d", repo, number)
}

// parseIssueRef splits an "owner/repo#N" ID back into its parts.
func parseIssueRef(id string) (repo string, number int, err error) {
	idx := strings.LastIndex(id, "#")
	if idx < 0 {
		return "", 0, fmt.Errorf("github: invalid issue/PR id %q", id)
	}
	repo = id[:idx]
	number, err = strconv.Atoi(id[idx+1:])
	if err != nil {
		return "", 0, fmt.Errorf("github: invalid issue/PR id %q: %w", id, err)
	}
	return repo, number, nil
}

func labelBadges(labels []ghLabel) []domain.Badge {
	badges := make([]domain.Badge, 0, len(labels))
	for _, l := range labels {
		badges = append(badges, domain.Badge{Label: l.Name, Color: "label"})
	}
	return badges
}
