package linear

import (
	"context"

	"github.com/bjcorder/chit/internal/provider"
)

const organizationQuery = `{ organization { id name } }`

// ListRootContainers always returns exactly one container: a personal API
// key is scoped to a single Linear workspace (organization), so there's
// nothing to enumerate the way GitHub enumerates multiple orgs.
func (p *Provider) ListRootContainers(ctx context.Context) ([]provider.Container, error) {
	var resp struct {
		Organization struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"organization"`
	}
	if err := p.client.do(ctx, organizationQuery, nil, &resp); err != nil {
		return nil, err
	}
	return []provider.Container{{ID: resp.Organization.ID, Kind: provider.KindRoot, Name: resp.Organization.Name}}, nil
}

const teamsQuery = `{ teams(first: 100) { nodes { id name key } } }`

// ListChildContainers lists every team the API key can access. parentID is
// accepted for interface compliance but unused: Linear's `teams` query
// already scopes to what this key can see, and a personal key belongs to
// exactly one workspace, so there's no separate org to filter by.
func (p *Provider) ListChildContainers(ctx context.Context, parentID string) ([]provider.Container, error) {
	var resp struct {
		Teams struct {
			Nodes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Key  string `json:"key"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	if err := p.client.do(ctx, teamsQuery, nil, &resp); err != nil {
		return nil, err
	}

	containers := make([]provider.Container, 0, len(resp.Teams.Nodes))
	for _, t := range resp.Teams.Nodes {
		containers = append(containers, provider.Container{
			ID:       t.ID,
			Kind:     provider.KindChild,
			Name:     t.Key + " — " + t.Name,
			ParentID: parentID,
		})
	}
	return containers, nil
}
