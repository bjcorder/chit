package provider

import (
	"context"
	"testing"

	"github.com/bjcorder/chit/internal/domain"
)

type fakeIssueTracker struct{ name string }

func (f *fakeIssueTracker) Name() string { return f.name }
func (f *fakeIssueTracker) ListRootContainers(ctx context.Context) ([]Container, error) {
	return nil, nil
}
func (f *fakeIssueTracker) ListChildContainers(ctx context.Context, parentID string) ([]Container, error) {
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

func withCleanRegistry(t *testing.T) {
	t.Helper()
	saved := registry
	registry = map[string]Descriptor{}
	t.Cleanup(func() { registry = saved })
}

func TestRegisterAndGet(t *testing.T) {
	withCleanRegistry(t)

	Register(Descriptor{
		Name: "fake",
		NewIssueTracker: func(ctx context.Context, cfg Config) (IssueTracker, error) {
			return &fakeIssueTracker{name: "fake"}, nil
		},
	})

	d, ok := Get("fake")
	if !ok {
		t.Fatal("Get(\"fake\") returned ok=false after Register")
	}
	if d.NewCodeHost != nil {
		t.Error("expected NewCodeHost to be nil for an issue-tracker-only provider")
	}

	tracker, err := d.NewIssueTracker(context.Background(), Config{Enabled: true})
	if err != nil {
		t.Fatalf("NewIssueTracker: %v", err)
	}
	if tracker.Name() != "fake" {
		t.Errorf("Name() = %q, want %q", tracker.Name(), "fake")
	}

	if len(All()) != 1 {
		t.Errorf("All() returned %d descriptors, want 1", len(All()))
	}

	if _, ok := Get("does-not-exist"); ok {
		t.Error("Get for unregistered name returned ok=true")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	withCleanRegistry(t)

	Register(Descriptor{Name: "dup"})

	defer func() {
		if recover() == nil {
			t.Error("expected Register to panic on duplicate name")
		}
	}()
	Register(Descriptor{Name: "dup"})
}

func TestRegisterEmptyNamePanics(t *testing.T) {
	withCleanRegistry(t)

	defer func() {
		if recover() == nil {
			t.Error("expected Register to panic on empty name")
		}
	}()
	Register(Descriptor{Name: ""})
}
