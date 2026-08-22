package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

// fakeRunner dispatches on the first two args (e.g. "issue list") so tests
// can stub out exactly the gh subcommands they exercise without caring
// about full argv order.
type fakeRunner struct {
	t  *testing.T
	on map[string]func(args []string) ([]byte, error)
}

func newFakeRunner(t *testing.T) *fakeRunner {
	return &fakeRunner{t: t, on: map[string]func(args []string) ([]byte, error){}}
}

func (f *fakeRunner) stub(key string, fn func(args []string) ([]byte, error)) {
	f.on[key] = fn
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	n := 2
	if len(args) < n {
		n = len(args)
	}
	key := strings.Join(args[:n], " ")
	fn, ok := f.on[key]
	if !ok {
		f.t.Fatalf("unexpected gh invocation: %v (no stub for %q)", args, key)
	}
	return fn(args)
}

func newTestProvider(r Runner) *Provider {
	return &Provider{runner: r}
}

func TestListRootContainers(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("api user", func(args []string) ([]byte, error) {
		return []byte(`{"login":"bjcorder"}`), nil
	})
	r.stub("api user/orgs", func(args []string) ([]byte, error) {
		return []byte(`[{"login":"acme"},{"login":"widgets-inc"}]`), nil
	})

	p := newTestProvider(r)
	got, err := p.ListRootContainers(context.Background())
	if err != nil {
		t.Fatalf("ListRootContainers: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d containers, want 3: %+v", len(got), got)
	}
	if got[0].Name != "bjcorder" {
		t.Errorf("got[0].Name = %q, want %q (own account first)", got[0].Name, "bjcorder")
	}
}

func TestListChildContainers(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("repo list", func(args []string) ([]byte, error) {
		return []byte(`[{"nameWithOwner":"cli/cli","name":"cli"},{"nameWithOwner":"cli/go-gh","name":"go-gh"}]`), nil
	})

	p := newTestProvider(r)
	got, err := p.ListChildContainers(context.Background(), "cli")
	if err != nil {
		t.Fatalf("ListChildContainers: %v", err)
	}
	if len(got) != 2 || got[0].ID != "cli/cli" || got[0].ParentID != "cli" {
		t.Fatalf("got %+v", got)
	}
}

const sampleIssueListJSON = `[{"assignees":[],"author":{"login":"annatchijova"},"createdAt":"2026-08-22T15:43:18Z","labels":[{"name":"needs-triage"}],"number":14238,"state":"OPEN","title":"Example issue","updatedAt":"2026-08-22T15:46:44Z","url":"https://github.com/cli/cli/issues/14238"}]`

func TestListIssues(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("issue list", func(args []string) ([]byte, error) {
		return []byte(sampleIssueListJSON), nil
	})

	p := newTestProvider(r)
	got, err := p.ListIssues(context.Background(), "cli/cli")
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d issues, want 1", len(got))
	}
	issue := got[0]
	if issue.ID != "cli/cli#14238" {
		t.Errorf("ID = %q, want %q", issue.ID, "cli/cli#14238")
	}
	if issue.Number != "14238" {
		t.Errorf("Number = %q, want %q", issue.Number, "14238")
	}
	if issue.StateBadge.Label != "open" {
		t.Errorf("StateBadge = %+v, want label %q", issue.StateBadge, "open")
	}
	if issue.Closed {
		t.Error("Closed = true, want false for an OPEN issue")
	}
	foundLabel := false
	for _, b := range issue.Badges {
		if b.Label == "needs-triage" {
			foundLabel = true
		}
	}
	if !foundLabel {
		t.Errorf("Badges = %+v, want a 'needs-triage' label badge", issue.Badges)
	}
}

const sampleIssueViewJSON = `{"assignees":[],"author":{"login":"annatchijova"},"body":"issue body text","comments":[{"author":{"login":"cli-triage"},"body":"a comment","createdAt":"2026-08-22T15:46:42Z","id":"IC_kwDODKw3uc8AAAABQL9ALQ"}],"createdAt":"2026-08-22T15:43:18Z","labels":[],"number":14238,"state":"OPEN","title":"Example issue","updatedAt":"2026-08-22T15:46:44Z","url":"https://github.com/cli/cli/issues/14238"}`

func TestGetIssue(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("issue view", func(args []string) ([]byte, error) {
		return []byte(sampleIssueViewJSON), nil
	})

	p := newTestProvider(r)
	got, err := p.GetIssue(context.Background(), "cli/cli", "cli/cli#14238")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if got.Body != "issue body text" {
		t.Errorf("Body = %q", got.Body)
	}
	if len(got.Comments) != 1 || got.Comments[0].ID != "IC_kwDODKw3uc8AAAABQL9ALQ" {
		t.Fatalf("Comments = %+v", got.Comments)
	}
}

func TestSupportsThreadedRepliesIsFalse(t *testing.T) {
	p := newTestProvider(newFakeRunner(t))
	if p.SupportsThreadedReplies() {
		t.Error("SupportsThreadedReplies() = true, want false for GitHub")
	}
}

func TestAddComment(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("issue comment", func(args []string) ([]byte, error) {
		return []byte("https://github.com/cli/cli/issues/14238#issuecomment-5381242925\n"), nil
	})
	r.stub("api repos/cli/cli/issues/comments/5381242925", func(args []string) ([]byte, error) {
		return []byte(`{"node_id":"IC_new","body":"hello","created_at":"2026-08-22T16:00:00Z","user":{"login":"bjcorder"}}`), nil
	})

	p := newTestProvider(r)
	got, err := p.AddComment(context.Background(), "cli/cli#14238", "hello")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if got.ID != "IC_new" || got.Body != "hello" || got.Author.Login != "bjcorder" {
		t.Errorf("got %+v", got)
	}
}

func TestReplyToCommentQuotesParent(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("api graphql", func(args []string) ([]byte, error) {
		return []byte(`{"data":{"node":{"body":"original text","author":{"login":"annatchijova"}}}}`), nil
	})

	var postedBody string
	r.stub("issue comment", func(args []string) ([]byte, error) {
		for i, a := range args {
			if a == "--body" && i+1 < len(args) {
				postedBody = args[i+1]
			}
		}
		return []byte("https://github.com/cli/cli/issues/14238#issuecomment-999\n"), nil
	})
	r.stub("api repos/cli/cli/issues/comments/999", func(args []string) ([]byte, error) {
		out, err := json.Marshal(map[string]any{
			"node_id": "IC_reply", "body": postedBody,
			"created_at": "2026-08-22T16:00:00Z", "user": map[string]string{"login": "bjcorder"},
		})
		return out, err
	})

	p := newTestProvider(r)
	_, err := p.ReplyToComment(context.Background(), "cli/cli#14238", "IC_kwDODKw3uc8AAAABQL9ALQ", "my reply")
	if err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if !strings.Contains(postedBody, "@annatchijova wrote") || !strings.Contains(postedBody, "> original text") || !strings.Contains(postedBody, "my reply") {
		t.Errorf("posted body = %q, want it to quote the parent and include the reply", postedBody)
	}
}

func TestReplyToCommentFallsBackWhenParentUnreachable(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("api graphql", func(args []string) ([]byte, error) {
		return nil, fmt.Errorf("not found")
	})
	var postedBody string
	r.stub("issue comment", func(args []string) ([]byte, error) {
		for i, a := range args {
			if a == "--body" && i+1 < len(args) {
				postedBody = args[i+1]
			}
		}
		return []byte("https://github.com/cli/cli/issues/14238#issuecomment-999\n"), nil
	})
	r.stub("api repos/cli/cli/issues/comments/999", func(args []string) ([]byte, error) {
		return []byte(`{"node_id":"IC_reply","body":"","created_at":"2026-08-22T16:00:00Z","user":{"login":"bjcorder"}}`), nil
	})

	p := newTestProvider(r)
	if _, err := p.ReplyToComment(context.Background(), "cli/cli#14238", "IC_deleted", "my reply"); err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if postedBody != "my reply" {
		t.Errorf("postedBody = %q, want unquoted fallback %q", postedBody, "my reply")
	}
}

const samplePRListJSON = `[{"author":{"login":"MaramHarsha"},"createdAt":"2026-08-21T03:05:55Z","isDraft":false,"labels":[],"number":14217,"state":"OPEN","title":"Example PR","updatedAt":"2026-08-21T03:07:24Z","url":"https://github.com/cli/cli/pull/14217"}]`

func TestListPullRequests(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("pr list", func(args []string) ([]byte, error) {
		return []byte(samplePRListJSON), nil
	})

	p := newTestProvider(r)
	got, err := p.ListPullRequests(context.Background(), "cli/cli")
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(got) != 1 || got[0].ID != "cli/cli#14217" || got[0].IsDraft {
		t.Fatalf("got %+v", got)
	}
}

func TestClosedFieldAcrossIssueAndPRStates(t *testing.T) {
	issueOpen := ghIssueSummary{State: "OPEN"}.toDomain("cli/cli")
	if issueOpen.Closed {
		t.Error("issue Closed = true for state OPEN")
	}
	issueClosed := ghIssueSummary{State: "CLOSED"}.toDomain("cli/cli")
	if !issueClosed.Closed {
		t.Error("issue Closed = false for state CLOSED")
	}

	prCases := []struct {
		state string
		want  bool
	}{
		{"OPEN", false},
		{"CLOSED", true},
		{"MERGED", true},
	}
	for _, c := range prCases {
		pr := ghPRSummary{State: c.state}.toDomain("cli/cli")
		if pr.Closed != c.want {
			t.Errorf("PR state %q: Closed = %v, want %v", c.state, pr.Closed, c.want)
		}
	}
}

const samplePRViewJSON = `{"author":{"login":"MaramHarsha"},"baseRefName":"trunk","commits":[{"authoredDate":"2026-08-21T02:51:31Z","authors":[{"login":"MaramHarsha","name":"MaramHarsha"}],"committedDate":"2026-08-21T03:04:59Z","messageHeadline":"Treat gist contents as binary only when they are not valid UTF-8","oid":"95129b3e317a4c990df6b55268a17bd172addab1"}],"comments":[],"createdAt":"2026-08-21T03:05:55Z","headRefName":"fix/gist-utf8-contents","isDraft":false,"labels":[],"mergeable":"MERGEABLE","number":14217,"state":"OPEN","title":"Example PR","updatedAt":"2026-08-21T03:07:24Z","url":"https://github.com/cli/cli/pull/14217"}`

const sampleChecksJSON = `[{"bucket":"pass","link":"https://github.com/cli/cli/actions/runs/1/job/1","name":"test","state":"SUCCESS","workflow":"CI"},{"bucket":"skipping","link":"https://github.com/cli/cli/actions/runs/1/job/2","name":"deploy","state":"SKIPPED","workflow":"CI"}]`

const sampleMergeSettingsJSON = `{"deleteBranchOnMerge":true,"mergeCommitAllowed":true,"rebaseMergeAllowed":true,"squashMergeAllowed":true}`

func TestGetPullRequest(t *testing.T) {
	r := newFakeRunner(t)
	r.stub("pr view", func(args []string) ([]byte, error) {
		return []byte(samplePRViewJSON), nil
	})
	r.stub("pr checks", func(args []string) ([]byte, error) {
		return []byte(sampleChecksJSON), fmt.Errorf("exit status 8") // gh exits non-zero when any check isn't passing
	})
	r.stub("repo view", func(args []string) ([]byte, error) {
		return []byte(sampleMergeSettingsJSON), nil
	})

	p := newTestProvider(r)
	got, err := p.GetPullRequest(context.Background(), "cli/cli", "cli/cli#14217")
	if err != nil {
		t.Fatalf("GetPullRequest: %v", err)
	}
	if got.BaseBranch != "trunk" || got.HeadBranch != "fix/gist-utf8-contents" || !got.Mergeable {
		t.Errorf("got %+v", got)
	}
	if len(got.Commits) != 1 || got.Commits[0].SHA != "95129b3e317a4c990df6b55268a17bd172addab1" {
		t.Fatalf("Commits = %+v", got.Commits)
	}
	if got.Commits[0].URL != "https://github.com/cli/cli/commit/95129b3e317a4c990df6b55268a17bd172addab1" {
		t.Errorf("Commit URL = %q", got.Commits[0].URL)
	}
	if len(got.Checks) != 2 || got.Checks[0].Status != domain.CheckPass || got.Checks[1].Status != domain.CheckSkipped {
		t.Fatalf("Checks = %+v", got.Checks)
	}
	wantMethods := []domain.MergeMethod{domain.MergeCommit, domain.MergeSquash, domain.MergeRebase}
	if len(got.AllowedMerges) != len(wantMethods) {
		t.Fatalf("AllowedMerges = %+v", got.AllowedMerges)
	}
}

func TestApprovePullRequest(t *testing.T) {
	r := newFakeRunner(t)
	var gotArgs []string
	r.stub("pr review", func(args []string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	})

	p := newTestProvider(r)
	if err := p.ApprovePullRequest(context.Background(), "cli/cli#14217", "looks good"); err != nil {
		t.Fatalf("ApprovePullRequest: %v", err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "--approve") || !strings.Contains(joined, "-R cli/cli") || !strings.Contains(joined, "14217") {
		t.Errorf("args = %v", gotArgs)
	}
}

func TestMergePullRequestUsesRequestedMethod(t *testing.T) {
	r := newFakeRunner(t)
	var gotArgs []string
	r.stub("pr merge", func(args []string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	})

	p := newTestProvider(r)
	if err := p.MergePullRequest(context.Background(), "cli/cli#14217", domain.MergeSquash, true); err != nil {
		t.Fatalf("MergePullRequest: %v", err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "--squash") || !strings.Contains(joined, "--delete-branch") {
		t.Errorf("args = %v", gotArgs)
	}
}

func TestMarkReadyForReview(t *testing.T) {
	r := newFakeRunner(t)
	var gotArgs []string
	r.stub("pr ready", func(args []string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	})

	p := newTestProvider(r)
	if err := p.MarkReadyForReview(context.Background(), "cli/cli#14217"); err != nil {
		t.Fatalf("MarkReadyForReview: %v", err)
	}
	if !strings.Contains(strings.Join(gotArgs, " "), "14217") {
		t.Errorf("args = %v", gotArgs)
	}
}

func TestParseIssueRefRejectsMalformedID(t *testing.T) {
	if _, _, err := parseIssueRef("not-a-valid-id"); err == nil {
		t.Error("expected error for ID with no '#'")
	}
	if _, _, err := parseIssueRef("owner/repo#not-a-number"); err == nil {
		t.Error("expected error for non-numeric issue number")
	}
}

func TestRegisteredWithProviderRegistry(t *testing.T) {
	d, ok := provider.Get("github")
	if !ok {
		t.Fatal(`provider.Get("github") ok=false — init() didn't register`)
	}
	if d.NewIssueTracker == nil {
		t.Error("NewIssueTracker is nil")
	}
	if d.NewCodeHost == nil {
		t.Error("NewCodeHost is nil")
	}
}
