package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

// detailMode tracks which of a detail screen's input modes is active. In
// modeNormal, keys navigate/scroll and trigger actions; the other modes
// borrow the whole keyboard for a single purpose until they finish or are
// canceled with 'q'.
type detailMode string

const (
	modeNormal   detailMode = ""
	modeComments detailMode = "comments"
	modeHints    detailMode = "hints"
)

type issueDetailScreen struct {
	ctx          context.Context
	a            *app.App
	st           styles
	providerName string
	container    provider.Container
	issueID      string

	issue  domain.Issue
	loaded bool
	err    error

	vp viewport.Model

	mode          detailMode
	hints         []linkHint
	hintBuffer    string
	commentRows   []row
	commentCursor int

	status string
}

func newIssueDetailScreen(ctx context.Context, a *app.App, st styles, providerName string, container provider.Container, issueID string) *issueDetailScreen {
	return &issueDetailScreen{ctx: ctx, a: a, st: st, providerName: providerName, container: container, issueID: issueID, vp: viewport.New(0, 0)}
}

func (s *issueDetailScreen) Title() string {
	if s.loaded && s.err == nil {
		return "#" + s.issue.Number
	}
	return "issue"
}

func (s *issueDetailScreen) Init() tea.Cmd {
	return loadIssueDetail(s.ctx, s.a, s.providerName, s.container.ID, s.issueID, false)
}

func (s *issueDetailScreen) renderContent() {
	content, hints := issueDetailContent(s.issue, s.vp.Width, s.st, s.mode == modeHints)
	s.hints = hints
	s.vp.SetContent(content)
}

func (s *issueDetailScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case issueDetailMsg:
		if msg.providerName != s.providerName || msg.issueID != s.issueID {
			return s, nil
		}
		s.loaded = true
		s.err = msg.err
		s.status = ""
		if msg.err == nil {
			s.issue = msg.issue
			s.renderContent()
		}
		return s, nil

	case editorClosedMsg:
		return s, handleEditorClosed(msg, &s.status, s.ctx, s.a, s.providerName)

	case commentPostedMsg:
		if msg.providerName != s.providerName || msg.issueID != s.issueID {
			return s, nil
		}
		if msg.err != nil {
			s.status = "comment: " + msg.err.Error()
			return s, nil
		}
		s.status = "comment posted"
		return s, loadIssueDetail(s.ctx, s.a, s.providerName, s.container.ID, s.issueID, true)

	case tea.KeyMsg:
		switch s.mode {
		case modeComments:
			return s.updateCommentsMode(msg)
		case modeHints:
			return s.updateHintsMode(msg)
		default:
			if cmd, handled := s.updateNormalMode(msg); handled {
				return s, cmd
			}
		}
	}

	var cmd tea.Cmd
	s.vp, cmd = s.vp.Update(msg)
	return s, cmd
}

func (s *issueDetailScreen) updateNormalMode(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "r":
		return loadIssueDetail(s.ctx, s.a, s.providerName, s.container.ID, s.issueID, true), true
	case "c":
		return openEditorCmd(s.issue.ID, ""), true
	case "v":
		s.mode = modeComments
		s.commentRows = buildCommentRows(s.issue.Comments)
		s.commentCursor = initialCursor(s.commentRows)
		return nil, true
	case "f":
		s.mode = modeHints
		s.hintBuffer = ""
		s.renderContent()
		return nil, true
	}
	return nil, false
}

func (s *issueDetailScreen) updateCommentsMode(msg tea.KeyMsg) (screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		s.commentCursor = moveCursor(s.commentRows, s.commentCursor, 1)
	case "k", "up":
		s.commentCursor = moveCursor(s.commentRows, s.commentCursor, -1)
	case "enter":
		s.mode = modeNormal
		if len(s.commentRows) == 0 {
			return s, nil
		}
		parentID := s.issue.Comments[s.commentRows[s.commentCursor].sourceIndex].ID
		return s, openEditorCmd(s.issue.ID, parentID)
	case "q":
		s.mode = modeNormal
	}
	return s, nil
}

func (s *issueDetailScreen) updateHintsMode(msg tea.KeyMsg) (screen, tea.Cmd) {
	cmd := resolveHintKey(msg, s.ctx, s.a, s.st, s.providerName, s.container, &s.mode, &s.hintBuffer, s.hints, &s.status)
	s.renderContent()
	return s, cmd
}

func (s *issueDetailScreen) View(width, height int) string {
	if !s.loaded {
		return s.st.dim.Render("loading…")
	}
	if s.err != nil {
		return s.st.errorText.Render("error: " + s.err.Error())
	}

	if s.mode == modeComments {
		return renderRowsScrolled(s.commentRows, s.commentCursor, height-1, s.st) + "\n" + s.st.help.Render("j/k move  enter reply  q cancel")
	}

	if s.vp.Width != width || s.vp.Height != height {
		s.vp.Width, s.vp.Height = width, height
		s.renderContent()
	}
	body := s.vp.View()
	if s.mode == modeHints {
		body += "\n" + s.st.help.Render("type a hint label to open it, any other key cancels ("+s.hintBuffer+")")
	} else if s.status != "" {
		body += "\n" + s.st.dim.Render(s.status)
	}
	return body
}

type prDetailScreen struct {
	ctx          context.Context
	a            *app.App
	st           styles
	providerName string
	container    provider.Container
	prID         string

	pr     domain.PullRequest
	loaded bool
	err    error

	vp viewport.Model

	mode          detailMode
	hints         []linkHint
	hintBuffer    string
	commentRows   []row
	commentCursor int

	pendingAction      string // "", "approve", "merge", "ready"
	pendingMergeMethod domain.MergeMethod

	status string
}

func newPRDetailScreen(ctx context.Context, a *app.App, st styles, providerName string, container provider.Container, prID string) *prDetailScreen {
	return &prDetailScreen{ctx: ctx, a: a, st: st, providerName: providerName, container: container, prID: prID, vp: viewport.New(0, 0)}
}

func (s *prDetailScreen) Title() string {
	if s.loaded && s.err == nil {
		return "#" + s.pr.Number
	}
	return "PR"
}

func (s *prDetailScreen) Init() tea.Cmd {
	return loadPRDetail(s.ctx, s.a, s.providerName, s.container.ID, s.prID, false)
}

func (s *prDetailScreen) renderContent() {
	content, hints := prDetailContent(s.pr, s.vp.Width, s.st, s.mode == modeHints)
	s.hints = hints
	s.vp.SetContent(content)
}

func (s *prDetailScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case prDetailMsg:
		if msg.providerName != s.providerName || msg.prID != s.prID {
			return s, nil
		}
		s.loaded = true
		s.err = msg.err
		s.status = ""
		if msg.err == nil {
			s.pr = msg.pr
			s.renderContent()
		}
		return s, nil

	case editorClosedMsg:
		return s, handleEditorClosed(msg, &s.status, s.ctx, s.a, s.providerName)

	case commentPostedMsg:
		if msg.providerName != s.providerName || msg.issueID != s.prID {
			return s, nil
		}
		if msg.err != nil {
			s.status = "comment: " + msg.err.Error()
			return s, nil
		}
		s.status = "comment posted"
		return s, loadPRDetail(s.ctx, s.a, s.providerName, s.container.ID, s.prID, true)

	case prActionMsg:
		if msg.providerName != s.providerName || msg.prID != s.prID {
			return s, nil
		}
		if msg.err != nil {
			s.status = msg.action + ": " + msg.err.Error()
			return s, nil
		}
		s.status = msg.action + " succeeded"
		return s, loadPRDetail(s.ctx, s.a, s.providerName, s.container.ID, s.prID, true)

	case tea.KeyMsg:
		switch s.mode {
		case modeComments:
			return s.updateCommentsMode(msg)
		case modeHints:
			return s.updateHintsMode(msg)
		default:
			if cmd, handled := s.updateNormalMode(msg); handled {
				return s, cmd
			}
		}
	}

	var cmd tea.Cmd
	s.vp, cmd = s.vp.Update(msg)
	return s, cmd
}

func (s *prDetailScreen) updateNormalMode(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "r":
		return loadPRDetail(s.ctx, s.a, s.providerName, s.container.ID, s.prID, true), true
	case "c":
		return openEditorCmd(s.pr.ID, ""), true
	case "v":
		s.mode = modeComments
		s.commentRows = buildCommentRows(s.pr.Comments)
		s.commentCursor = initialCursor(s.commentRows)
		return nil, true
	case "f":
		s.mode = modeHints
		s.hintBuffer = ""
		s.renderContent()
		return nil, true
	case "a":
		s.pendingAction = "approve"
		s.status = "approve this PR? (y/n)"
		return nil, true
	case "m":
		return nil, s.beginOrCycleMerge()
	case "d":
		if !s.pr.IsDraft {
			s.status = "not a draft"
			return nil, true
		}
		s.pendingAction = "ready"
		s.status = "mark ready for review? (y/n)"
		return nil, true
	case "y":
		return s.confirmPendingAction(), true
	case "n":
		if s.pendingAction != "" {
			s.pendingAction = ""
			s.status = "canceled"
		}
		return nil, true
	}
	return nil, false
}

// beginOrCycleMerge handles 'm': the first press proposes the first
// allowed merge method, and repeated presses (while the merge confirmation
// is still pending) cycle through the rest — repos can allow more than one
// method, and chit shouldn't guess which one the user wants.
func (s *prDetailScreen) beginOrCycleMerge() bool {
	if len(s.pr.AllowedMerges) == 0 {
		s.status = "no merge method allowed by this repo's settings"
		return true
	}
	idx := 0
	if s.pendingAction == "merge" {
		for i, m := range s.pr.AllowedMerges {
			if m == s.pendingMergeMethod {
				idx = (i + 1) % len(s.pr.AllowedMerges)
				break
			}
		}
	}
	s.pendingAction = "merge"
	s.pendingMergeMethod = s.pr.AllowedMerges[idx]
	s.status = "merge via " + string(s.pendingMergeMethod) + "? (y confirm, m cycle method, n cancel)"
	return true
}

func (s *prDetailScreen) confirmPendingAction() tea.Cmd {
	action := s.pendingAction
	s.pendingAction = ""
	switch action {
	case "approve":
		s.status = "approving…"
		return approvePR(s.ctx, s.a, s.providerName, s.pr.ID)
	case "merge":
		s.status = "merging…"
		return mergePR(s.ctx, s.a, s.providerName, s.pr.ID, s.pendingMergeMethod)
	case "ready":
		s.status = "marking ready…"
		return markReady(s.ctx, s.a, s.providerName, s.pr.ID)
	}
	return nil
}

func (s *prDetailScreen) updateCommentsMode(msg tea.KeyMsg) (screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		s.commentCursor = moveCursor(s.commentRows, s.commentCursor, 1)
	case "k", "up":
		s.commentCursor = moveCursor(s.commentRows, s.commentCursor, -1)
	case "enter":
		s.mode = modeNormal
		if len(s.commentRows) == 0 {
			return s, nil
		}
		parentID := s.pr.Comments[s.commentRows[s.commentCursor].sourceIndex].ID
		return s, openEditorCmd(s.pr.ID, parentID)
	case "q":
		s.mode = modeNormal
	}
	return s, nil
}

func (s *prDetailScreen) updateHintsMode(msg tea.KeyMsg) (screen, tea.Cmd) {
	cmd := resolveHintKey(msg, s.ctx, s.a, s.st, s.providerName, s.container, &s.mode, &s.hintBuffer, s.hints, &s.status)
	s.renderContent()
	return s, cmd
}

func (s *prDetailScreen) View(width, height int) string {
	if !s.loaded {
		return s.st.dim.Render("loading…")
	}
	if s.err != nil {
		return s.st.errorText.Render("error: " + s.err.Error())
	}

	if s.mode == modeComments {
		return renderRowsScrolled(s.commentRows, s.commentCursor, height-1, s.st) + "\n" + s.st.help.Render("j/k move  enter reply  q cancel")
	}

	if s.vp.Width != width || s.vp.Height != height {
		s.vp.Width, s.vp.Height = width, height
		s.renderContent()
	}
	body := s.vp.View()
	if s.mode == modeHints {
		body += "\n" + s.st.help.Render("type a hint label to open it, any other key cancels ("+s.hintBuffer+")")
	} else if s.status != "" {
		body += "\n" + s.st.dim.Render(s.status)
	}
	return body
}

// resolveHintKey is shared by both detail screens' hint-mode key handling:
// buffer the keystroke, and on an exact label match either open the URL or
// push a new detail screen for an internal cross-reference.
func resolveHintKey(msg tea.KeyMsg, ctx context.Context, a *app.App, st styles, providerName string, container provider.Container, mode *detailMode, hintBuffer *string, hints []linkHint, status *string) tea.Cmd {
	if msg.String() == "q" {
		*mode = modeNormal
		*hintBuffer = ""
		return nil
	}
	if msg.Type != tea.KeyRunes {
		*mode = modeNormal
		return nil
	}

	*hintBuffer += string(msg.Runes)
	hint, exact, isPrefix := findHint(hints, *hintBuffer)
	if exact {
		*mode = modeNormal
		*hintBuffer = ""
		if strings.HasPrefix(hint.target, "chit-ref:") {
			ref := strings.TrimPrefix(hint.target, "chit-ref:")
			return pushScreen(newIssueDetailScreen(ctx, a, st, providerName, container, crossRefIssueID(container.ID, ref)))
		}
		if err := openURL(hint.target); err != nil {
			*status = "opening link: " + err.Error()
		}
		return nil
	}
	if !isPrefix {
		*mode = modeNormal
		*hintBuffer = ""
	}
	return nil
}

// handleEditorClosed is shared by both detail screens: read back what the
// user wrote, and if non-empty, post it as a comment or reply.
func handleEditorClosed(msg editorClosedMsg, status *string, ctx context.Context, a *app.App, providerName string) tea.Cmd {
	if msg.err != nil {
		*status = "editor: " + msg.err.Error()
		return nil
	}
	body, err := readAndCleanupTempFile(msg.tmpPath)
	if err != nil {
		*status = "editor: " + err.Error()
		return nil
	}
	if strings.TrimSpace(body) == "" {
		*status = "empty comment, not posted"
		return nil
	}
	if msg.parentCommentID != "" {
		return postReply(ctx, a, providerName, msg.issueID, msg.parentCommentID, body)
	}
	return postComment(ctx, a, providerName, msg.issueID, body)
}
