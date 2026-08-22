package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bjcorder/chit/internal/domain"
)

type ghIssueSummary struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	Author    ghUser    `json:"author"`
	Labels    []ghLabel `json:"labels"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s ghIssueSummary) toDomain(repo string) domain.Issue {
	badges := append([]domain.Badge{stateBadge(s.State)}, labelBadges(s.Labels)...)
	return domain.Issue{
		ID:        issueRef(repo, s.Number),
		Number:    strconv.Itoa(s.Number),
		Title:     s.Title,
		URL:       s.URL,
		State:     s.State,
		Badges:    badges,
		Author:    domain.User{Login: s.Author.Login},
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func stateBadge(state string) domain.Badge {
	switch strings.ToUpper(state) {
	case "OPEN":
		return domain.Badge{Label: "open", Color: "green"}
	case "CLOSED":
		return domain.Badge{Label: "closed", Color: "purple"}
	case "MERGED":
		return domain.Badge{Label: "merged", Color: "purple"}
	default:
		return domain.Badge{Label: strings.ToLower(state), Color: "gray"}
	}
}

const issueListFields = "number,title,state,labels,author,createdAt,updatedAt,url"

func (p *Provider) ListIssues(ctx context.Context, containerID string) ([]domain.Issue, error) {
	summaries, err := runJSON[[]ghIssueSummary](ctx, p, "issue", "list", "-R", containerID, "--state", "all", "--json", issueListFields, "--limit", "200")
	if err != nil {
		return nil, err
	}

	issues := make([]domain.Issue, 0, len(summaries))
	for _, s := range summaries {
		issues = append(issues, s.toDomain(containerID))
	}
	return issues, nil
}

type ghIssueDetail struct {
	ghIssueSummary
	Body     string      `json:"body"`
	Comments []ghComment `json:"comments"`
}

type ghComment struct {
	ID        string    `json:"id"`
	Author    ghUser    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

func (c ghComment) toDomain() domain.Comment {
	return domain.Comment{
		ID:        c.ID,
		Author:    domain.User{Login: c.Author.Login},
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
	}
}

const issueViewFields = issueListFields + ",body,comments"

func (p *Provider) GetIssue(ctx context.Context, containerID, issueID string) (domain.Issue, error) {
	_, number, err := parseIssueRef(issueID)
	if err != nil {
		return domain.Issue{}, err
	}

	detail, err := runJSON[ghIssueDetail](ctx, p, "issue", "view", strconv.Itoa(number), "-R", containerID, "--json", issueViewFields)
	if err != nil {
		return domain.Issue{}, err
	}

	issue := detail.toDomain(containerID)
	issue.Body = detail.Body
	issue.Comments = make([]domain.Comment, 0, len(detail.Comments))
	for _, c := range detail.Comments {
		issue.Comments = append(issue.Comments, c.toDomain())
	}
	return issue, nil
}

// SupportsThreadedReplies is always false for GitHub: issue comments have
// no reply-to relationship at the API level, only a flat ordered list.
func (p *Provider) SupportsThreadedReplies() bool { return false }

func (p *Provider) AddComment(ctx context.Context, issueID, body string) (domain.Comment, error) {
	repo, number, err := parseIssueRef(issueID)
	if err != nil {
		return domain.Comment{}, err
	}

	out, err := p.run(ctx, "issue", "comment", strconv.Itoa(number), "-R", repo, "--body", body)
	if err != nil {
		return domain.Comment{}, err
	}

	restID, err := parseCommentRESTID(strings.TrimSpace(string(out)))
	if err != nil {
		return domain.Comment{}, err
	}
	return p.fetchCommentByRESTID(ctx, repo, restID)
}

// ReplyToComment has nothing to actually thread under on GitHub, so it
// posts a new top-level comment that quotes the parent for context. If the
// parent comment can no longer be fetched (e.g. it was deleted), it falls
// back to posting body unquoted rather than failing the reply outright.
func (p *Provider) ReplyToComment(ctx context.Context, issueID, parentCommentID, body string) (domain.Comment, error) {
	author, quoted, err := p.fetchCommentForQuote(ctx, parentCommentID)
	text := body
	if err == nil {
		text = fmt.Sprintf("@%s wrote:\n%s\n\n%s", author, quoteBlock(quoted), body)
	}
	return p.AddComment(ctx, issueID, text)
}

func quoteBlock(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "> " + l
	}
	return strings.Join(lines, "\n")
}

// parseCommentRESTID extracts the numeric comment ID from the URL that
// `gh issue comment` prints on success, e.g.
// "https://github.com/owner/repo/issues/123#issuecomment-456".
func parseCommentRESTID(url string) (string, error) {
	const marker = "#issuecomment-"
	idx := strings.LastIndex(url, marker)
	if idx < 0 {
		return "", fmt.Errorf("github: unexpected `gh issue comment` output: %q", url)
	}
	return url[idx+len(marker):], nil
}

type ghRESTComment struct {
	NodeID    string    `json:"node_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      ghUser    `json:"user"`
}

// fetchCommentByRESTID re-fetches a just-created comment via the REST API
// to recover its GraphQL node ID (node_id) — `gh issue comment`'s own
// output only gives us the legacy numeric REST ID, but every other comment
// ID chit sees (from GetIssue) is the GraphQL node ID, and ReplyToComment
// needs that same ID shape to be able to look the comment back up later.
func (p *Provider) fetchCommentByRESTID(ctx context.Context, repo, restID string) (domain.Comment, error) {
	c, err := runJSON[ghRESTComment](ctx, p, "api", fmt.Sprintf("repos/%s/issues/comments/%s", repo, restID))
	if err != nil {
		return domain.Comment{}, err
	}
	return domain.Comment{
		ID:        c.NodeID,
		Author:    domain.User{Login: c.User.Login},
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
	}, nil
}

type ghNodeCommentResponse struct {
	Data struct {
		Node struct {
			Body   string `json:"body"`
			Author ghUser `json:"author"`
		} `json:"node"`
	} `json:"data"`
}

func (p *Provider) fetchCommentForQuote(ctx context.Context, commentNodeID string) (author, body string, err error) {
	resp, err := runJSON[ghNodeCommentResponse](ctx, p, "api", "graphql",
		"-f", "query=query($id: ID!) { node(id: $id) { ... on IssueComment { body author { login } } } }",
		"-f", "id="+commentNodeID,
	)
	if err != nil {
		return "", "", err
	}
	return resp.Data.Node.Author.Login, resp.Data.Node.Body, nil
}
