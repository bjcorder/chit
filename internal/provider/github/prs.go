package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bjcorder/chit/internal/domain"
)

type ghPRSummary struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	IsDraft   bool      `json:"isDraft"`
	URL       string    `json:"url"`
	Author    ghUser    `json:"author"`
	Labels    []ghLabel `json:"labels"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s ghPRSummary) toDomain(repo string) domain.PullRequest {
	badges := labelBadges(s.Labels)
	if s.IsDraft {
		badges = append(badges, domain.Badge{Label: "draft", Color: "gray"})
	}
	closed := strings.EqualFold(s.State, "CLOSED") || strings.EqualFold(s.State, "MERGED")
	return domain.PullRequest{
		Issue: domain.Issue{
			ID:         issueRef(repo, s.Number),
			Number:     strconv.Itoa(s.Number),
			Title:      s.Title,
			URL:        s.URL,
			State:      s.State,
			StateBadge: stateBadge(s.State),
			Badges:     badges,
			Closed:     closed,
			Author:     domain.User{Login: s.Author.Login},
			CreatedAt:  s.CreatedAt,
			UpdatedAt:  s.UpdatedAt,
		},
		IsDraft: s.IsDraft,
	}
}

const prListFields = "number,title,state,isDraft,labels,author,createdAt,updatedAt,url"

func (p *Provider) ListPullRequests(ctx context.Context, containerID string) ([]domain.PullRequest, error) {
	summaries, err := runJSON[[]ghPRSummary](ctx, p, "pr", "list", "-R", containerID, "--state", "all", "--json", prListFields, "--limit", "200")
	if err != nil {
		return nil, err
	}

	prs := make([]domain.PullRequest, 0, len(summaries))
	for _, s := range summaries {
		prs = append(prs, s.toDomain(containerID))
	}
	return prs, nil
}

type ghPRDetail struct {
	ghPRSummary
	Body        string      `json:"body"`
	BaseRefName string      `json:"baseRefName"`
	HeadRefName string      `json:"headRefName"`
	Mergeable   string      `json:"mergeable"`
	Comments    []ghComment `json:"comments"`
	Commits     []ghCommit  `json:"commits"`
}

type ghCommit struct {
	OID             string           `json:"oid"`
	MessageHeadline string           `json:"messageHeadline"`
	Authors         []ghCommitAuthor `json:"authors"`
}

type ghCommitAuthor struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

const prViewFields = prListFields + ",body,baseRefName,headRefName,mergeable,comments,commits"

func (p *Provider) GetPullRequest(ctx context.Context, containerID, prID string) (domain.PullRequest, error) {
	_, number, err := parseIssueRef(prID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	detail, err := runJSON[ghPRDetail](ctx, p, "pr", "view", strconv.Itoa(number), "-R", containerID, "--json", prViewFields)
	if err != nil {
		return domain.PullRequest{}, err
	}

	pr := detail.toDomain(containerID)
	pr.Body = detail.Body
	pr.BaseBranch = detail.BaseRefName
	pr.HeadBranch = detail.HeadRefName
	pr.Mergeable = strings.EqualFold(detail.Mergeable, "MERGEABLE")

	pr.Comments = make([]domain.Comment, 0, len(detail.Comments))
	for _, c := range detail.Comments {
		pr.Comments = append(pr.Comments, c.toDomain())
	}

	pr.Commits = make([]domain.Commit, 0, len(detail.Commits))
	for _, c := range detail.Commits {
		pr.Commits = append(pr.Commits, domain.Commit{
			SHA:     c.OID,
			Message: c.MessageHeadline,
			Author:  domain.User{Login: commitAuthorLogin(c.Authors)},
			URL:     fmt.Sprintf("https://github.com/%s/commit/%s", containerID, c.OID),
		})
	}

	checks, err := p.pullRequestChecks(ctx, containerID, number)
	if err != nil {
		return domain.PullRequest{}, err
	}
	pr.Checks = checks

	allowed, err := p.allowedMergeMethods(ctx, containerID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	pr.AllowedMerges = allowed

	return pr, nil
}

func commitAuthorLogin(authors []ghCommitAuthor) string {
	if len(authors) == 0 {
		return ""
	}
	if authors[0].Login != "" {
		return authors[0].Login
	}
	return authors[0].Name
}

type ghCheck struct {
	Bucket   string `json:"bucket"`
	Name     string `json:"name"`
	Workflow string `json:"workflow"`
	Link     string `json:"link"`
}

func (p *Provider) pullRequestChecks(ctx context.Context, containerID string, number int) ([]domain.CheckRun, error) {
	// gh's own exit code reflects whether checks are passing, not whether
	// the command itself worked — stdout is still a valid JSON array
	// (possibly empty) whenever checks exist, so the error only matters if
	// there's no output to parse at all.
	out, _ := p.run(ctx, "pr", "checks", strconv.Itoa(number), "-R", containerID, "--json", "bucket,name,workflow,link")
	if len(out) == 0 {
		return nil, nil
	}

	var checks []ghCheck
	if uerr := json.Unmarshal(out, &checks); uerr != nil {
		return nil, uerr
	}

	runs := make([]domain.CheckRun, 0, len(checks))
	for _, c := range checks {
		runs = append(runs, domain.CheckRun{
			Name:     c.Name,
			Workflow: c.Workflow,
			Status:   checkBucketStatus(c.Bucket),
			URL:      c.Link,
		})
	}
	return runs, nil
}

func checkBucketStatus(bucket string) domain.CheckStatus {
	switch strings.ToLower(bucket) {
	case "pass":
		return domain.CheckPass
	case "fail":
		return domain.CheckFail
	case "pending":
		return domain.CheckPending
	case "skipping":
		return domain.CheckSkipped
	case "cancel":
		return domain.CheckCancelled
	default:
		return domain.CheckPending
	}
}

type ghRepoMergeSettings struct {
	SquashMergeAllowed bool `json:"squashMergeAllowed"`
	MergeCommitAllowed bool `json:"mergeCommitAllowed"`
	RebaseMergeAllowed bool `json:"rebaseMergeAllowed"`
}

func (p *Provider) allowedMergeMethods(ctx context.Context, containerID string) ([]domain.MergeMethod, error) {
	settings, err := runJSON[ghRepoMergeSettings](ctx, p, "repo", "view", containerID, "--json", "squashMergeAllowed,mergeCommitAllowed,rebaseMergeAllowed")
	if err != nil {
		return nil, err
	}

	var methods []domain.MergeMethod
	if settings.MergeCommitAllowed {
		methods = append(methods, domain.MergeCommit)
	}
	if settings.SquashMergeAllowed {
		methods = append(methods, domain.MergeSquash)
	}
	if settings.RebaseMergeAllowed {
		methods = append(methods, domain.MergeRebase)
	}
	return methods, nil
}

func (p *Provider) ApprovePullRequest(ctx context.Context, prID, body string) error {
	repo, number, err := parseIssueRef(prID)
	if err != nil {
		return err
	}
	args := []string{"pr", "review", strconv.Itoa(number), "-R", repo, "--approve"}
	if body != "" {
		args = append(args, "--body", body)
	}
	_, err = p.run(ctx, args...)
	return err
}

func (p *Provider) MergePullRequest(ctx context.Context, prID string, method domain.MergeMethod, deleteBranch bool) error {
	repo, number, err := parseIssueRef(prID)
	if err != nil {
		return err
	}

	args := []string{"pr", "merge", strconv.Itoa(number), "-R", repo}
	switch method {
	case domain.MergeSquash:
		args = append(args, "--squash")
	case domain.MergeRebase:
		args = append(args, "--rebase")
	case domain.MergeCommit:
		args = append(args, "--merge")
	default:
		return fmt.Errorf("github: unknown merge method %q", method)
	}
	if deleteBranch {
		args = append(args, "--delete-branch")
	}

	_, err = p.run(ctx, args...)
	return err
}

func (p *Provider) MarkReadyForReview(ctx context.Context, prID string) error {
	repo, number, err := parseIssueRef(prID)
	if err != nil {
		return err
	}
	_, err = p.run(ctx, "pr", "ready", strconv.Itoa(number), "-R", repo)
	return err
}
