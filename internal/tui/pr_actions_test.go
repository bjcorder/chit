package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

func loadedPRDetail(t *testing.T, pr domain.PullRequest) *prDetailScreen {
	t.Helper()
	a := newTestApp(t)
	a.CodeHosts["gh"] = &fakeHost{prDetail: map[string]domain.PullRequest{pr.ID: pr}}

	s := newPRDetailScreen(context.Background(), a, newStyles(), "gh", provider.Container{ID: "cli/cli"}, pr.ID)
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	return updated.(*prDetailScreen)
}

func key(k string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)} }

func TestApproveRequiresConfirmation(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}})

	updated, cmd := s.Update(key("a"))
	s = updated.(*prDetailScreen)
	if s.pendingAction != "approve" || cmd != nil {
		t.Fatalf("pendingAction=%q cmd=%v, want approve pending with no cmd yet", s.pendingAction, cmd)
	}

	updated, cmd = s.Update(key("y"))
	s = updated.(*prDetailScreen)
	if s.pendingAction != "" {
		t.Error("pendingAction should clear after confirming")
	}
	msg := runCmd(t, cmd).(prActionMsg)
	if msg.action != "approve" || msg.err != nil {
		t.Fatalf("prActionMsg = %+v", msg)
	}
}

func TestApproveCancelWithN(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}})
	updated, _ := s.Update(key("a"))
	s = updated.(*prDetailScreen)

	updated, cmd := s.Update(key("n"))
	s = updated.(*prDetailScreen)
	if s.pendingAction != "" || cmd != nil {
		t.Errorf("pendingAction=%q cmd=%v, want cleared with no action taken", s.pendingAction, cmd)
	}
}

func TestMergeWithNoAllowedMethodsRefuses(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}, AllowedMerges: nil})
	updated, cmd := s.Update(key("m"))
	s = updated.(*prDetailScreen)
	if s.pendingAction == "merge" || cmd != nil {
		t.Error("expected merge to be refused when no methods are allowed")
	}
	if s.status == "" {
		t.Error("expected a status message explaining why")
	}
}

func TestMergeCyclesThroughAllowedMethods(t *testing.T) {
	pr := domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}, AllowedMerges: []domain.MergeMethod{domain.MergeSquash, domain.MergeRebase}}
	s := loadedPRDetail(t, pr)

	updated, _ := s.Update(key("m"))
	s = updated.(*prDetailScreen)
	if s.pendingMergeMethod != domain.MergeSquash {
		t.Fatalf("first method = %q, want squash", s.pendingMergeMethod)
	}

	updated, _ = s.Update(key("m"))
	s = updated.(*prDetailScreen)
	if s.pendingMergeMethod != domain.MergeRebase {
		t.Fatalf("second method = %q, want rebase (cycled)", s.pendingMergeMethod)
	}

	updated, _ = s.Update(key("m"))
	s = updated.(*prDetailScreen)
	if s.pendingMergeMethod != domain.MergeSquash {
		t.Fatalf("third method = %q, want wrapped back to squash", s.pendingMergeMethod)
	}
}

func TestMergeConfirmSendsChosenMethod(t *testing.T) {
	pr := domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}, AllowedMerges: []domain.MergeMethod{domain.MergeRebase}}
	s := loadedPRDetail(t, pr)

	updated, _ := s.Update(key("m"))
	s = updated.(*prDetailScreen)
	_, cmd := s.Update(key("y"))

	msg := runCmd(t, cmd).(prActionMsg)
	if msg.action != "merge" {
		t.Fatalf("prActionMsg = %+v", msg)
	}
}

func TestMarkReadyOnlyAppliesToDrafts(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}, IsDraft: false})
	updated, cmd := s.Update(key("d"))
	s = updated.(*prDetailScreen)
	if s.pendingAction == "ready" || cmd != nil {
		t.Error("expected 'd' to be a no-op on a non-draft PR")
	}
}

func TestMarkReadyOnDraft(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}, IsDraft: true})
	updated, _ := s.Update(key("d"))
	s = updated.(*prDetailScreen)
	if s.pendingAction != "ready" {
		t.Fatalf("pendingAction = %q, want ready", s.pendingAction)
	}

	_, cmd := s.Update(key("y"))
	msg := runCmd(t, cmd).(prActionMsg)
	if msg.action != "ready" {
		t.Fatalf("prActionMsg = %+v", msg)
	}
}

func TestPRActionSuccessTriggersRefresh(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}})
	updated, cmd := s.Update(prActionMsg{providerName: "gh", prID: "cli/cli#1", action: "approve"})
	s = updated.(*prDetailScreen)

	if s.status != "approve succeeded" {
		t.Errorf("status = %q", s.status)
	}
	msg := runCmd(t, cmd)
	if _, ok := msg.(prDetailMsg); !ok {
		t.Fatalf("expected a fresh prDetailMsg load, got %T", msg)
	}
}

func TestPRActionErrorSurfacesStatusNoRefresh(t *testing.T) {
	s := loadedPRDetail(t, domain.PullRequest{Issue: domain.Issue{ID: "cli/cli#1"}})
	_, cmd := s.Update(prActionMsg{providerName: "gh", prID: "cli/cli#1", action: "merge", err: errFake})
	if cmd != nil {
		t.Error("expected no refresh cmd when the action failed")
	}
}
