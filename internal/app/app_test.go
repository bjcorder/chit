package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

// fakeIssueTracker/fakeCodeHost are minimal stand-ins registered directly
// with the provider registry for this test, independent of the real
// github/linear packages.
type fakeIssueTracker struct{ name string }

func (f *fakeIssueTracker) Name() string { return f.name }
func (f *fakeIssueTracker) ListRootContainers(ctx context.Context) ([]provider.Container, error) {
	return nil, nil
}
func (f *fakeIssueTracker) ListChildContainers(ctx context.Context, parentID string) ([]provider.Container, error) {
	return nil, nil
}
func (f *fakeIssueTracker) ListIssues(ctx context.Context, containerID string) ([]domain.Issue, error) {
	return nil, nil
}
func (f *fakeIssueTracker) GetIssue(ctx context.Context, containerID, issueID string) (domain.Issue, error) {
	return domain.Issue{}, nil
}
func (f *fakeIssueTracker) AddComment(ctx context.Context, issueID, body string) (domain.Comment, error) {
	return domain.Comment{}, nil
}
func (f *fakeIssueTracker) SupportsThreadedReplies() bool { return false }
func (f *fakeIssueTracker) ReplyToComment(ctx context.Context, issueID, parentCommentID, body string) (domain.Comment, error) {
	return domain.Comment{}, nil
}

func registerFakeProvider(t *testing.T, name string, withCodeHost bool) {
	t.Helper()
	d := provider.Descriptor{
		Name: name,
		NewIssueTracker: func(ctx context.Context, cfg provider.Config) (provider.IssueTracker, error) {
			return &fakeIssueTracker{name: name}, nil
		},
	}
	provider.Register(d)
	_ = withCodeHost // GitHub-style CodeHost registration isn't needed by these tests yet
}

func writeConfig(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}

func TestLoadFromInstantiatesOnlyEnabledProviders(t *testing.T) {
	registerFakeProvider(t, "fake-app-test-1", false)
	registerFakeProvider(t, "fake-app-test-2", false)

	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, `
[providers.fake-app-test-1]
enabled = true

[providers.fake-app-test-2]
enabled = false
`)
	cachePath := filepath.Join(dir, "chit.db")

	a, err := LoadFrom(context.Background(), cfgPath, cachePath)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	defer a.Close() //nolint:errcheck

	if _, ok := a.IssueTrackers["fake-app-test-1"]; !ok {
		t.Error("enabled provider was not instantiated")
	}
	if _, ok := a.IssueTrackers["fake-app-test-2"]; ok {
		t.Error("disabled provider was instantiated")
	}
}

func TestLoadFromErrorsOnUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, `
[providers.this-provider-does-not-exist]
enabled = true
`)
	cachePath := filepath.Join(dir, "chit.db")

	if _, err := LoadFrom(context.Background(), cfgPath, cachePath); err == nil {
		t.Error("expected an error for an enabled but unregistered provider")
	}
}

func TestLoadFromWithNoProvidersEnabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "")
	cachePath := filepath.Join(dir, "chit.db")

	a, err := LoadFrom(context.Background(), cfgPath, cachePath)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	defer a.Close() //nolint:errcheck

	if len(a.IssueTrackers) != 0 || len(a.CodeHosts) != 0 {
		t.Errorf("got IssueTrackers=%v CodeHosts=%v, want both empty", a.IssueTrackers, a.CodeHosts)
	}
}

func TestLoadFromOpensCacheAtGivenPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "")
	cachePath := filepath.Join(dir, "chit.db")

	a, err := LoadFrom(context.Background(), cfgPath, cachePath)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	defer a.Close() //nolint:errcheck

	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache DB not created at %s: %v", cachePath, err)
	}
}
