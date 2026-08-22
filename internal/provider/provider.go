// Package provider defines the two capability interfaces chit's providers
// implement — IssueTracker and CodeHost — and the compile-time registry
// that lets a provider package register itself via init() while still
// leaving it up to the user's config which providers actually run.
//
// A provider package (internal/provider/github, internal/provider/linear)
// calls Register from its own init(). cmd/chit blank-imports every known
// provider package so they're all linked into the binary; config.Providers
// then decides which ones get instantiated at startup.
package provider

import (
	"context"
	"fmt"

	"github.com/bjcorder/chit/internal/domain"
)

// ContainerKind distinguishes the two levels chit's navigation model
// supports: a root container (GitHub org, Linear workspace) and a child
// container one level down (GitHub repo, Linear team).
type ContainerKind string

const (
	KindRoot  ContainerKind = "root"
	KindChild ContainerKind = "child"
)

type Container struct {
	ID       string
	Kind     ContainerKind
	Name     string
	ParentID string
}

// IssueTracker is implemented by any provider that can list and manage
// issues. GitHub and Linear both implement this.
type IssueTracker interface {
	Name() string
	ListRootContainers(ctx context.Context) ([]Container, error)
	ListChildContainers(ctx context.Context, parentID string) ([]Container, error)
	ListIssues(ctx context.Context, containerID string) ([]domain.Issue, error)
	GetIssue(ctx context.Context, containerID, issueID string) (domain.Issue, error)
	AddComment(ctx context.Context, issueID, body string) (domain.Comment, error)

	// SupportsThreadedReplies reports whether ReplyToComment nests under an
	// existing comment (Linear) or is only able to post a new top-level
	// comment (GitHub, which has no reply-to at the API level).
	SupportsThreadedReplies() bool
	ReplyToComment(ctx context.Context, issueID, parentCommentID, body string) (domain.Comment, error)
}

// CodeHost is implemented by any provider that can list and manage pull
// requests. Only GitHub implements this for now.
type CodeHost interface {
	Name() string
	ListRootContainers(ctx context.Context) ([]Container, error)
	ListChildContainers(ctx context.Context, parentID string) ([]Container, error)
	ListPullRequests(ctx context.Context, containerID string) ([]domain.PullRequest, error)
	GetPullRequest(ctx context.Context, containerID, prID string) (domain.PullRequest, error)
	ApprovePullRequest(ctx context.Context, prID, body string) error
	MergePullRequest(ctx context.Context, prID string, method domain.MergeMethod, deleteBranch bool) error
	MarkReadyForReview(ctx context.Context, prID string) error
}

// Config is the per-provider configuration block a factory receives —
// currently just whether it's enabled, plus a bag of provider-specific
// settings (e.g. Linear's API-key keyring reference) read from config.toml.
type Config struct {
	Enabled bool
	Extra   map[string]string
}

type IssueTrackerFactory func(ctx context.Context, cfg Config) (IssueTracker, error)
type CodeHostFactory func(ctx context.Context, cfg Config) (CodeHost, error)

// Descriptor is what a provider package registers at init() time. Either
// factory may be nil — Linear, for example, only sets NewIssueTracker.
type Descriptor struct {
	Name            string
	NewIssueTracker IssueTrackerFactory
	NewCodeHost     CodeHostFactory
}

var registry = map[string]Descriptor{}

// Register adds a provider descriptor to the compile-time registry. It
// panics on a duplicate name, since that can only happen from a programming
// error (two provider packages claiming the same Name), never from user
// input or runtime state.
func Register(d Descriptor) {
	if d.Name == "" {
		panic("provider: Register called with empty Name")
	}
	if _, exists := registry[d.Name]; exists {
		panic(fmt.Sprintf("provider: %q registered twice", d.Name))
	}
	registry[d.Name] = d
}

// All returns every provider linked into the binary, regardless of whether
// the user's config enables it.
func All() []Descriptor {
	out := make([]Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	return out
}

// Get looks up a single registered provider by name.
func Get(name string) (Descriptor, bool) {
	d, ok := registry[name]
	return d, ok
}
