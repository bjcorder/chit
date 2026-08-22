// Package linear implements chit's Linear provider: IssueTracker only (no
// CodeHost — Linear has no notion of a pull request), talking directly to
// Linear's GraphQL API with a personal API key.
package linear

import (
	"context"
	"fmt"

	"github.com/bjcorder/chit/internal/provider"
	"github.com/bjcorder/chit/internal/secret"
)

// apiKeySecretName is the key chit's secret store looks the Linear personal
// API key up under (see internal/secret).
const apiKeySecretName = "linear-api-key"

func init() {
	provider.Register(provider.Descriptor{
		Name: "linear",
		NewIssueTracker: func(ctx context.Context, cfg provider.Config) (provider.IssueTracker, error) {
			return newFromSecretStore()
		},
	})
}

func newFromSecretStore() (*Provider, error) {
	store, err := secret.Open()
	if err != nil {
		return nil, fmt.Errorf("linear: opening secret store: %w", err)
	}
	apiKey, err := store.Get(apiKeySecretName)
	if err != nil {
		return nil, fmt.Errorf("linear: no personal API key stored — set one via `chit providers enable linear`: %w", err)
	}
	return New(apiKey), nil
}

// Provider implements provider.IssueTracker for Linear.
type Provider struct {
	client *graphQLClient
}

// New returns a Provider authenticating with apiKey.
func New(apiKey string) *Provider {
	return &Provider{client: newGraphQLClient(apiKey)}
}

func (p *Provider) Name() string { return "linear" }

// SupportsThreadedReplies is always true for Linear: comments have a real
// parent/child relationship at the API level (see docs/research/linear-api.md).
func (p *Provider) SupportsThreadedReplies() bool { return true }
