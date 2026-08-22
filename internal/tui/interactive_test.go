package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

func loadedIssueDetail(t *testing.T, comments ...domain.Comment) *issueDetailScreen {
	t.Helper()
	a := newTestApp(t)
	issue := domain.Issue{ID: "cli/cli#1", Number: "1", Title: "Bug", Body: "body [x](https://example.com)", Comments: comments}
	a.IssueTrackers["gh"] = &fakeTracker{issueDetail: map[string]domain.Issue{"cli/cli#1": issue}}

	s := newIssueDetailScreen(context.Background(), a, newStyles(), "gh", provider.Container{ID: "cli/cli"}, "cli/cli#1")
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	return updated.(*issueDetailScreen)
}

func TestComposeCommentKeyReturnsNonNilCmd(t *testing.T) {
	s := loadedIssueDetail(t)
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd == nil {
		t.Fatal("expected 'c' to return a non-nil cmd (opening $EDITOR)")
	}
}

func TestEditorClosedWithEmptyBodyDoesNotPost(t *testing.T) {
	s := loadedIssueDetail(t)
	tmp := writeTempFile(t, "   \n\n  ")

	_, cmd := s.Update(editorClosedMsg{tmpPath: tmp, issueID: "cli/cli#1"})
	if cmd != nil {
		t.Error("expected no post cmd for an empty/whitespace-only comment")
	}
	if s.status == "" {
		t.Error("expected a status message explaining nothing was posted")
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("temp file should be cleaned up after reading")
	}
}

func TestEditorClosedWithErrorSurfacesStatus(t *testing.T) {
	s := loadedIssueDetail(t)
	_, cmd := s.Update(editorClosedMsg{issueID: "cli/cli#1", err: errFake})
	if cmd != nil {
		t.Error("expected no post cmd when the editor itself errored")
	}
	if s.status == "" {
		t.Error("expected an error status message")
	}
}

func TestEditorClosedPostsTopLevelComment(t *testing.T) {
	s := loadedIssueDetail(t)
	tmp := writeTempFile(t, "hello world")

	_, cmd := s.Update(editorClosedMsg{tmpPath: tmp, issueID: "cli/cli#1"})
	msg := runCmd(t, cmd).(commentPostedMsg)
	if msg.err != nil || msg.comment.Body != "hello world" {
		t.Fatalf("commentPostedMsg = %+v", msg)
	}
}

func TestEditorClosedPostsReplyWithParentID(t *testing.T) {
	s := loadedIssueDetail(t)
	s.a.IssueTrackers["gh"] = &fakeTracker{threaded: true}
	tmp := writeTempFile(t, "a reply")

	_, cmd := s.Update(editorClosedMsg{tmpPath: tmp, issueID: "cli/cli#1", parentCommentID: "c1"})
	msg := runCmd(t, cmd).(commentPostedMsg)
	if msg.comment.ParentID != "c1" {
		t.Errorf("comment.ParentID = %q, want %q", msg.comment.ParentID, "c1")
	}
}

func TestCommentPostedMsgTriggersForceRefresh(t *testing.T) {
	s := loadedIssueDetail(t)
	updated, cmd := s.Update(commentPostedMsg{providerName: "gh", issueID: "cli/cli#1", comment: domain.Comment{ID: "new"}})
	s = updated.(*issueDetailScreen)

	if s.status != "comment posted" {
		t.Errorf("status = %q", s.status)
	}
	msg := runCmd(t, cmd)
	if _, ok := msg.(issueDetailMsg); !ok {
		t.Fatalf("expected the post to trigger a fresh issueDetailMsg load, got %T", msg)
	}
}

func TestCommentPostedMsgErrorDoesNotRefresh(t *testing.T) {
	s := loadedIssueDetail(t)
	_, cmd := s.Update(commentPostedMsg{providerName: "gh", issueID: "cli/cli#1", err: errFake})
	if cmd != nil {
		t.Error("expected no refresh cmd when posting the comment failed")
	}
}

func TestCommentsModeListsAndSelectsReplyTarget(t *testing.T) {
	s := loadedIssueDetail(t, domain.Comment{ID: "c1", Author: domain.User{Login: "alice"}, Body: "first"}, domain.Comment{ID: "c2", Author: domain.User{Login: "bob"}, Body: "second"})

	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	s = updated.(*issueDetailScreen)
	if s.mode != modeComments || len(s.commentRows) != 2 {
		t.Fatalf("mode=%q rows=%d, want modeComments with 2 rows", s.mode, len(s.commentRows))
	}

	updated, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	s = updated.(*issueDetailScreen)
	if s.commentCursor != 1 {
		t.Fatalf("commentCursor = %d, want 1", s.commentCursor)
	}

	updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s = updated.(*issueDetailScreen)
	if s.mode != modeNormal {
		t.Error("expected mode to return to normal after selecting a reply target")
	}
	if cmd == nil {
		t.Error("expected enter to return a non-nil cmd (opening $EDITOR for the reply)")
	}
}

func TestCommentsModeQCancels(t *testing.T) {
	s := loadedIssueDetail(t, domain.Comment{ID: "c1"})
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	s = updated.(*issueDetailScreen)

	updated, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	s = updated.(*issueDetailScreen)
	if s.mode != modeNormal {
		t.Errorf("mode = %q, want modeNormal after 'q'", s.mode)
	}
}

func TestHintModeExactMatchOnCrossRefPushesScreen(t *testing.T) {
	a := newTestApp(t)
	issue := domain.Issue{ID: "cli/cli#1", Number: "1", Body: "fixes #42"}
	a.IssueTrackers["gh"] = &fakeTracker{issueDetail: map[string]domain.Issue{"cli/cli#1": issue}}
	s := newIssueDetailScreen(context.Background(), a, newStyles(), "gh", provider.Container{ID: "cli/cli"}, "cli/cli#1")
	msg := runCmd(t, s.Init())
	updated, _ := s.Update(msg)
	s = updated.(*issueDetailScreen)

	updated, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	s = updated.(*issueDetailScreen)
	if s.mode != modeHints || len(s.hints) == 0 {
		t.Fatalf("mode=%q hints=%+v, want modeHints with at least one hint", s.mode, s.hints)
	}

	label := s.hints[0].label
	updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(label)})
	s = updated.(*issueDetailScreen)
	if s.mode != modeNormal {
		t.Error("expected mode to reset after resolving a hint")
	}
	pushMsg := runCmd(t, cmd).(pushScreenMsg)
	if _, ok := pushMsg.screen.(*issueDetailScreen); !ok {
		t.Fatalf("pushed screen = %T, want *issueDetailScreen for a #NN cross-reference", pushMsg.screen)
	}
}

func TestHintModeNoMatchCancels(t *testing.T) {
	s := loadedIssueDetail(t)
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	s = updated.(*issueDetailScreen)

	updated, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	s = updated.(*issueDetailScreen)
	if s.mode != modeNormal {
		t.Errorf("mode = %q, want modeNormal after an unmatched key", s.mode)
	}
}

func TestHintModeQCancels(t *testing.T) {
	s := loadedIssueDetail(t)
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	s = updated.(*issueDetailScreen)

	updated, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	s = updated.(*issueDetailScreen)
	if s.mode != modeNormal || s.hintBuffer != "" {
		t.Errorf("mode=%q buffer=%q, want reset after 'q'", s.mode, s.hintBuffer)
	}
}

func writeTempFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "comment.md")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}
