package github

import (
	"context"

	"github.com/bjcorder/chit/internal/provider"
)

// ListRootContainers returns the authenticated user's own account plus
// every organization they belong to — the two kinds of owner a repo can
// have on GitHub.
func (p *Provider) ListRootContainers(ctx context.Context) ([]provider.Container, error) {
	me, err := p.currentUser(ctx)
	if err != nil {
		return nil, err
	}

	orgs, err := runJSON[[]ghUser](ctx, p, "api", "user/orgs", "--method", "GET", "-f", "per_page=100")
	if err != nil {
		return nil, err
	}

	containers := make([]provider.Container, 0, len(orgs)+1)
	containers = append(containers, provider.Container{ID: me, Kind: provider.KindRoot, Name: me})
	for _, o := range orgs {
		containers = append(containers, provider.Container{ID: o.Login, Kind: provider.KindRoot, Name: o.Login})
	}
	return containers, nil
}

// ListChildContainers lists the repos owned by parentID (a user login or
// org login from ListRootContainers).
func (p *Provider) ListChildContainers(ctx context.Context, parentID string) ([]provider.Container, error) {
	type repo struct {
		NameWithOwner string `json:"nameWithOwner"`
		Name          string `json:"name"`
	}

	repos, err := runJSON[[]repo](ctx, p, "repo", "list", parentID, "--json", "nameWithOwner,name", "--limit", "200")
	if err != nil {
		return nil, err
	}

	containers := make([]provider.Container, 0, len(repos))
	for _, r := range repos {
		containers = append(containers, provider.Container{ID: r.NameWithOwner, Kind: provider.KindChild, Name: r.Name, ParentID: parentID})
	}
	return containers, nil
}
