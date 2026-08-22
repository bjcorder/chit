package cache

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "chit.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestRootContainersMissReturnsFalse(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_, ok, err := s.RootContainers(ctx, "github")
	if err != nil {
		t.Fatalf("RootContainers: %v", err)
	}
	if ok {
		t.Error("ok = true for an empty cache, want false")
	}
}

func TestRootContainersRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	want := []provider.Container{{ID: "bjcorder", Kind: provider.KindRoot, Name: "bjcorder"}}
	if err := s.SetRootContainers(ctx, "github", want); err != nil {
		t.Fatalf("SetRootContainers: %v", err)
	}

	got, ok, err := s.RootContainers(ctx, "github")
	if err != nil {
		t.Fatalf("RootContainers: %v", err)
	}
	if !ok || len(got) != 1 || got[0].ID != "bjcorder" {
		t.Fatalf("got %+v, ok=%v", got, ok)
	}
}

func TestSetRootContainersOverwrites(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.SetRootContainers(ctx, "github", []provider.Container{{ID: "old"}})
	_ = s.SetRootContainers(ctx, "github", []provider.Container{{ID: "new"}})

	got, _, err := s.RootContainers(ctx, "github")
	if err != nil {
		t.Fatalf("RootContainers: %v", err)
	}
	if len(got) != 1 || got[0].ID != "new" {
		t.Fatalf("got %+v, want overwritten to just 'new'", got)
	}
}

func TestChildContainersKeyedByParent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.SetChildContainers(ctx, "github", "org-a", []provider.Container{{ID: "org-a/repo1"}})
	_ = s.SetChildContainers(ctx, "github", "org-b", []provider.Container{{ID: "org-b/repo2"}})

	got, ok, err := s.ChildContainers(ctx, "github", "org-a")
	if err != nil || !ok {
		t.Fatalf("ChildContainers: ok=%v err=%v", ok, err)
	}
	if len(got) != 1 || got[0].ID != "org-a/repo1" {
		t.Fatalf("got %+v, want only org-a's repos", got)
	}
}

func TestProvidersDoNotCollide(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.SetRootContainers(ctx, "github", []provider.Container{{ID: "gh-root"}})
	_ = s.SetRootContainers(ctx, "linear", []provider.Container{{ID: "lin-root"}})

	gh, _, _ := s.RootContainers(ctx, "github")
	lin, _, _ := s.RootContainers(ctx, "linear")
	if len(gh) != 1 || gh[0].ID != "gh-root" {
		t.Errorf("github = %+v", gh)
	}
	if len(lin) != 1 || lin[0].ID != "lin-root" {
		t.Errorf("linear = %+v", lin)
	}
}

func TestIssuesRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	want := []domain.Issue{{ID: "cli/cli#1", Title: "Example"}}
	if err := s.SetIssues(ctx, "github", "cli/cli", want); err != nil {
		t.Fatalf("SetIssues: %v", err)
	}

	got, ok, err := s.Issues(ctx, "github", "cli/cli")
	if err != nil || !ok || len(got) != 1 || got[0].Title != "Example" {
		t.Fatalf("got %+v, ok=%v, err=%v", got, ok, err)
	}
}

func TestIssueDetailRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	want := domain.Issue{ID: "cli/cli#1", Title: "Example", Body: "the body", Comments: []domain.Comment{{ID: "c1", Body: "hi"}}}
	if err := s.SetIssueDetail(ctx, "github", "cli/cli#1", want); err != nil {
		t.Fatalf("SetIssueDetail: %v", err)
	}

	got, ok, err := s.IssueDetail(ctx, "github", "cli/cli#1")
	if err != nil || !ok || got.Body != "the body" || len(got.Comments) != 1 {
		t.Fatalf("got %+v, ok=%v, err=%v", got, ok, err)
	}
}

func TestPullRequestsRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	want := []domain.PullRequest{{Issue: domain.Issue{ID: "cli/cli#2", Title: "A PR"}, IsDraft: true}}
	if err := s.SetPullRequests(ctx, "github", "cli/cli", want); err != nil {
		t.Fatalf("SetPullRequests: %v", err)
	}

	got, ok, err := s.PullRequests(ctx, "github", "cli/cli")
	if err != nil || !ok || len(got) != 1 || !got[0].IsDraft {
		t.Fatalf("got %+v, ok=%v, err=%v", got, ok, err)
	}
}

func TestPRDetailRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	want := domain.PullRequest{
		Issue:   domain.Issue{ID: "cli/cli#2", Title: "A PR"},
		Commits: []domain.Commit{{SHA: "abc123", Message: "fix things"}},
		Checks:  []domain.CheckRun{{Name: "test", Status: domain.CheckPass}},
	}
	if err := s.SetPRDetail(ctx, "github", "cli/cli#2", want); err != nil {
		t.Fatalf("SetPRDetail: %v", err)
	}

	got, ok, err := s.PRDetail(ctx, "github", "cli/cli#2")
	if err != nil || !ok || len(got.Commits) != 1 || len(got.Checks) != 1 {
		t.Fatalf("got %+v, ok=%v, err=%v", got, ok, err)
	}
}

func TestReopeningExistingDatabasePreservesData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chit.db")
	ctx := context.Background()

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s1.SetRootContainers(ctx, "github", []provider.Container{{ID: "persisted"}}); err != nil {
		t.Fatalf("SetRootContainers: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer s2.Close() //nolint:errcheck

	got, ok, err := s2.RootContainers(ctx, "github")
	if err != nil || !ok || len(got) != 1 || got[0].ID != "persisted" {
		t.Fatalf("got %+v, ok=%v, err=%v — data did not survive reopen", got, ok, err)
	}
}
