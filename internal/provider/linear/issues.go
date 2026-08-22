package linear

import (
	"context"
	"fmt"
	"time"

	"github.com/bjcorder/chit/internal/domain"
)

type linIssueNode struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	State      struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Priority      float64 `json:"priority"`
	PriorityLabel string  `json:"priorityLabel"`
	Assignee      *struct {
		DisplayName string `json:"displayName"`
	} `json:"assignee"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (n linIssueNode) toDomain() domain.Issue {
	badges := []domain.Badge{stateBadge(n.State.Name, n.State.Type)}
	if n.Priority > 0 {
		badges = append(badges, domain.Badge{Label: n.PriorityLabel, Color: priorityColor(n.Priority)})
	}
	for _, l := range n.Labels.Nodes {
		badges = append(badges, domain.Badge{Label: l.Name, Color: "label"})
	}

	issue := domain.Issue{
		ID:        n.ID,
		Number:    n.Identifier,
		Title:     n.Title,
		URL:       n.URL,
		State:     n.State.Name,
		Badges:    badges,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
	if n.Assignee != nil {
		issue.Assignees = []domain.User{{Login: n.Assignee.DisplayName}}
	}
	return issue
}

// stateBadge colors a workflow-state badge by Linear's own state *type*
// (triage/backlog/unstarted/started/completed/canceled) rather than by
// name, since state names are team-customizable but the type enum isn't.
func stateBadge(name, stateType string) domain.Badge {
	switch stateType {
	case "completed":
		return domain.Badge{Label: name, Color: "green"}
	case "canceled":
		return domain.Badge{Label: name, Color: "red"}
	case "started":
		return domain.Badge{Label: name, Color: "blue"}
	case "triage":
		return domain.Badge{Label: name, Color: "yellow"}
	default: // backlog, unstarted
		return domain.Badge{Label: name, Color: "gray"}
	}
}

// priorityColor maps Linear's 1-4 priority scale (1=Urgent..4=Low; 0=No
// priority, which callers skip badging entirely).
func priorityColor(priority float64) string {
	switch priority {
	case 1:
		return "red"
	case 2:
		return "orange"
	case 3:
		return "yellow"
	default:
		return "gray"
	}
}

const issueListQuery = `query($teamId: String!) {
  team(id: $teamId) {
    issues(first: 250) {
      nodes {
        id identifier title url
        state { name type }
        priority priorityLabel
        assignee { displayName }
        labels(first: 20) { nodes { name } }
        createdAt updatedAt
      }
    }
  }
}`

func (p *Provider) ListIssues(ctx context.Context, containerID string) ([]domain.Issue, error) {
	var resp struct {
		Team struct {
			Issues struct {
				Nodes []linIssueNode `json:"nodes"`
			} `json:"issues"`
		} `json:"team"`
	}
	if err := p.client.do(ctx, issueListQuery, map[string]any{"teamId": containerID}, &resp); err != nil {
		return nil, err
	}

	issues := make([]domain.Issue, 0, len(resp.Team.Issues.Nodes))
	for _, n := range resp.Team.Issues.Nodes {
		issues = append(issues, n.toDomain())
	}
	return issues, nil
}

const issueDetailQuery = `query($id: String!) {
  issue(id: $id) {
    id identifier title url description
    state { name type }
    priority priorityLabel
    assignee { displayName }
    labels(first: 20) { nodes { name } }
    createdAt updatedAt
    comments(first: 250) {
      nodes {
        id body createdAt
        user { displayName }
        parent { id }
      }
    }
  }
}`

type linCommentNode struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	User      *struct {
		DisplayName string `json:"displayName"`
	} `json:"user"`
	Parent *struct {
		ID string `json:"id"`
	} `json:"parent"`
}

func (n linCommentNode) toDomain() domain.Comment {
	c := domain.Comment{ID: n.ID, Body: n.Body, CreatedAt: n.CreatedAt}
	if n.User != nil {
		c.Author = domain.User{Login: n.User.DisplayName}
	}
	if n.Parent != nil {
		c.ParentID = n.Parent.ID
	}
	return c
}

func (p *Provider) GetIssue(ctx context.Context, containerID, issueID string) (domain.Issue, error) {
	var resp struct {
		Issue struct {
			linIssueNode
			Description string `json:"description"`
			Comments    struct {
				Nodes []linCommentNode `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	}
	if err := p.client.do(ctx, issueDetailQuery, map[string]any{"id": issueID}, &resp); err != nil {
		return domain.Issue{}, err
	}
	if resp.Issue.ID == "" {
		return domain.Issue{}, fmt.Errorf("linear: issue %q not found", issueID)
	}

	issue := resp.Issue.toDomain()
	issue.Body = resp.Issue.Description
	issue.Comments = make([]domain.Comment, 0, len(resp.Issue.Comments.Nodes))
	for _, c := range resp.Issue.Comments.Nodes {
		issue.Comments = append(issue.Comments, c.toDomain())
	}
	return issue, nil
}

const commentCreateMutation = `mutation($issueId: String!, $body: String!, $parentId: String) {
  commentCreate(input: { issueId: $issueId, body: $body, parentId: $parentId }) {
    comment { id body createdAt user { displayName } parent { id } }
  }
}`

func (p *Provider) createComment(ctx context.Context, issueID, body, parentID string) (domain.Comment, error) {
	vars := map[string]any{"issueId": issueID, "body": body}
	if parentID != "" {
		vars["parentId"] = parentID
	}

	var resp struct {
		CommentCreate struct {
			Comment linCommentNode `json:"comment"`
		} `json:"commentCreate"`
	}
	if err := p.client.do(ctx, commentCreateMutation, vars, &resp); err != nil {
		return domain.Comment{}, err
	}
	return resp.CommentCreate.Comment.toDomain(), nil
}

func (p *Provider) AddComment(ctx context.Context, issueID, body string) (domain.Comment, error) {
	return p.createComment(ctx, issueID, body, "")
}

// ReplyToComment nests the new comment under parentCommentID via
// CommentCreateInput.parentId — Linear supports this natively, unlike
// GitHub.
func (p *Provider) ReplyToComment(ctx context.Context, issueID, parentCommentID, body string) (domain.Comment, error) {
	return p.createComment(ctx, issueID, body, parentCommentID)
}
