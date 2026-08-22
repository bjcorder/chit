package tui

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/cache"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

// fakeTracker is a controllable provider.IssueTracker double for testing
// screens without hitting a real provider.
type fakeTracker struct {
	rootContainers  []provider.Container
	rootErr         error
	childContainers map[string][]provider.Container
	childErr        error
	issues          map[string][]domain.Issue
	issuesErr       error
	issueDetail     map[string]domain.Issue
	issueDetailErr  error
	threaded        bool
}

func (f *fakeTracker) Name() string { return "fake" }
func (f *fakeTracker) ListRootContainers(ctx context.Context) ([]provider.Container, error) {
	return f.rootContainers, f.rootErr
}
func (f *fakeTracker) ListChildContainers(ctx context.Context, parentID string) ([]provider.Container, error) {
	return f.childContainers[parentID], f.childErr
}
func (f *fakeTracker) ListIssues(ctx context.Context, containerID string) ([]domain.Issue, error) {
	return f.issues[containerID], f.issuesErr
}
func (f *fakeTracker) GetIssue(ctx context.Context, containerID, issueID string) (domain.Issue, error) {
	return f.issueDetail[issueID], f.issueDetailErr
}
func (f *fakeTracker) AddComment(ctx context.Context, issueID, body string) (domain.Comment, error) {
	return domain.Comment{ID: "new", Body: body}, nil
}
func (f *fakeTracker) SupportsThreadedReplies() bool { return f.threaded }
func (f *fakeTracker) ReplyToComment(ctx context.Context, issueID, parentCommentID, body string) (domain.Comment, error) {
	return domain.Comment{ID: "reply", Body: body, ParentID: parentCommentID}, nil
}

// fakeHost is a controllable provider.CodeHost double.
type fakeHost struct {
	rootContainers  []provider.Container
	childContainers map[string][]provider.Container
	prs             map[string][]domain.PullRequest
	prsErr          error
	prDetail        map[string]domain.PullRequest
	prDetailErr     error
}

func (f *fakeHost) Name() string { return "fake" }
func (f *fakeHost) ListRootContainers(ctx context.Context) ([]provider.Container, error) {
	return f.rootContainers, nil
}
func (f *fakeHost) ListChildContainers(ctx context.Context, parentID string) ([]provider.Container, error) {
	return f.childContainers[parentID], nil
}
func (f *fakeHost) ListPullRequests(ctx context.Context, containerID string) ([]domain.PullRequest, error) {
	return f.prs[containerID], f.prsErr
}
func (f *fakeHost) GetPullRequest(ctx context.Context, containerID, prID string) (domain.PullRequest, error) {
	return f.prDetail[prID], f.prDetailErr
}
func (f *fakeHost) ApprovePullRequest(ctx context.Context, prID, body string) error { return nil }
func (f *fakeHost) MergePullRequest(ctx context.Context, prID string, method domain.MergeMethod, deleteBranch bool) error {
	return nil
}
func (f *fakeHost) MarkReadyForReview(ctx context.Context, prID string) error { return nil }

var errFake = errors.New("fake provider error")

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	store, err := cache.Open(filepath.Join(t.TempDir(), "chit.db"))
	if err != nil {
		t.Fatalf("cache.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &app.App{
		Cache:         store,
		IssueTrackers: map[string]provider.IssueTracker{},
		CodeHosts:     map[string]provider.CodeHost{},
	}
}
