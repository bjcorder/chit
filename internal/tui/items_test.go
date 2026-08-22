package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil cmd")
	}
	return cmd()
}

func TestItemsScreenLoadsIssuesOnInit(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{
		issues: map[string][]domain.Issue{"cli/cli": {{ID: "cli/cli#1", Number: "1", Title: "Bug"}}},
	}

	s := newItemsScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli", Name: "cli"})
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*itemsScreen)

	if !s.issuesLoaded || len(s.issues) != 1 || s.issues[0].Title != "Bug" {
		t.Fatalf("issues not loaded correctly: %+v", s.issues)
	}
	if !strings.Contains(s.View(80, 24), "Bug") {
		t.Errorf("View() missing issue title: %q", s.View(80, 24))
	}
}

func TestItemsScreenSurfacesProviderError(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{issuesErr: errFake}

	s := newItemsScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"})
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*itemsScreen)

	if s.issuesErr == nil {
		t.Fatal("expected issuesErr to be set")
	}
	if !strings.Contains(s.View(80, 24), "error") {
		t.Errorf("View() should surface the error: %q", s.View(80, 24))
	}
}

func TestItemsScreenTogglesToPRsOnlyWhenCodeHostAvailable(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["linear"] = &fakeTracker{}
	// linear has no CodeHost registered

	s := newItemsScreen(context.Background(), a, newStyles("notty"), "linear", provider.Container{ID: "team1"})
	updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	s = updated.(*itemsScreen)

	if s.showPRs {
		t.Error("showPRs became true for a provider with no CodeHost")
	}
	if cmd != nil {
		t.Error("expected no cmd when 'p' is a no-op")
	}
}

func TestItemsScreenTogglesToPRsAndLoadsThem(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{}
	a.CodeHosts["gh"] = &fakeHost{
		prs: map[string][]domain.PullRequest{
			"cli/cli": {{Issue: domain.Issue{ID: "cli/cli#2", Number: "2", Title: "A PR"}}},
		},
	}

	s := newItemsScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"})
	updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	s = updated.(*itemsScreen)
	if !s.showPRs {
		t.Fatal("expected showPRs=true after 'p'")
	}

	msg := runCmd(t, cmd)
	updated, _ = s.Update(msg)
	s = updated.(*itemsScreen)

	if !s.prsLoaded || len(s.prs) != 1 || s.prs[0].Title != "A PR" {
		t.Fatalf("PRs not loaded correctly: %+v", s.prs)
	}
}

func TestItemsScreenEnterPushesIssueDetail(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{
		issues: map[string][]domain.Issue{"cli/cli": {{ID: "cli/cli#1", Number: "1", Title: "Bug"}}},
	}

	s := newItemsScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"})
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*itemsScreen)

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pushMsg := runCmd(t, cmd).(pushScreenMsg)

	detail, ok := pushMsg.screen.(*issueDetailScreen)
	if !ok {
		t.Fatalf("pushed screen is %T, want *issueDetailScreen", pushMsg.screen)
	}
	if detail.issueID != "cli/cli#1" {
		t.Errorf("pushed detail screen issueID = %q, want %q", detail.issueID, "cli/cli#1")
	}
}

func TestItemsScreenRefreshBypassesCache(t *testing.T) {
	a := newTestApp(t)
	tracker := &fakeTracker{issues: map[string][]domain.Issue{"cli/cli": {{ID: "cli/cli#1", Title: "v1"}}}}
	a.IssueTrackers["gh"] = tracker

	s := newItemsScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "cli/cli"})
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*itemsScreen)

	// Change what the provider would return, then force a refresh — a
	// cache-only read would still see "v1".
	tracker.issues["cli/cli"] = []domain.Issue{{ID: "cli/cli#1", Title: "v2"}}
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	msg = runCmd(t, cmd)
	updated, _ = s.Update(msg)
	s = updated.(*itemsScreen)

	if s.issues[0].Title != "v2" {
		t.Errorf("issues[0].Title = %q after refresh, want %q (cache should be bypassed)", s.issues[0].Title, "v2")
	}
}
