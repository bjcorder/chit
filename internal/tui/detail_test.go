package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

func TestIssueDetailScreenLoadsAndRenders(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{
		issueDetail: map[string]domain.Issue{
			"cli/cli#1": {
				ID: "cli/cli#1", Number: "1", Title: "Example issue",
				Body:     "Some **markdown** body.",
				Comments: []domain.Comment{{ID: "c1", Author: domain.User{Login: "alice"}, Body: "a comment"}},
			},
		},
	}

	s := newIssueDetailScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"}, "cli/cli#1")
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*issueDetailScreen)

	if !s.loaded || s.err != nil {
		t.Fatalf("loaded=%v err=%v", s.loaded, s.err)
	}
	if s.Title() != "#1" {
		t.Errorf("Title() = %q, want %q", s.Title(), "#1")
	}

	view := s.View(80, 24)
	if !strings.Contains(view, "Example issue") || !strings.Contains(view, "markdown") || !strings.Contains(view, "alice") {
		t.Errorf("view missing expected content: %q", view)
	}
}

func TestIssueDetailScreenSurfacesError(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{issueDetailErr: errFake}

	s := newIssueDetailScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"}, "cli/cli#1")
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*issueDetailScreen)

	if s.err == nil {
		t.Fatal("expected err to be set")
	}
	if s.Title() != "issue" {
		t.Errorf("Title() on error = %q, want fallback %q", s.Title(), "issue")
	}
	if !strings.Contains(s.View(80, 24), "error") {
		t.Errorf("View() should surface the error: %q", s.View(80, 24))
	}
}

func TestPRDetailScreenLoadsAndRenders(t *testing.T) {
	a := newTestApp(t)
	a.CodeHosts["gh"] = &fakeHost{
		prDetail: map[string]domain.PullRequest{
			"cli/cli#2": {
				Issue:      domain.Issue{ID: "cli/cli#2", Number: "2", Title: "Example PR", Author: domain.User{Login: "bob"}},
				BaseBranch: "main", HeadBranch: "feature",
				Commits: []domain.Commit{{SHA: "abcdef1234567", Message: "fix things"}},
				Checks:  []domain.CheckRun{{Name: "test", Status: domain.CheckPass}},
			},
		},
	}

	s := newPRDetailScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"}, "cli/cli#2")
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*prDetailScreen)

	if !s.loaded || s.err != nil {
		t.Fatalf("loaded=%v err=%v", s.loaded, s.err)
	}
	if s.Title() != "#2" {
		t.Errorf("Title() = %q, want %q", s.Title(), "#2")
	}

	view := s.View(80, 24)
	for _, want := range []string{"Example PR", "feature", "main", "abcdef1", "fix things", "test"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q: %q", want, view)
		}
	}
}

func TestPRDetailScreenSurfacesError(t *testing.T) {
	a := newTestApp(t)
	a.CodeHosts["gh"] = &fakeHost{prDetailErr: errFake}

	s := newPRDetailScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"}, "cli/cli#2")
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*prDetailScreen)

	if s.err == nil {
		t.Fatal("expected err to be set")
	}
	if !strings.Contains(s.View(80, 24), "error") {
		t.Errorf("View() should surface the error: %q", s.View(80, 24))
	}
}
