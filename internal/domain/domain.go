// Package domain holds the provider-agnostic types every chit UI screen renders.
// Providers translate their own data shapes into these; nothing outside a
// provider package should need to know whether an Issue came from GitHub or
// Linear.
package domain

import "time"

// Badge is a single label+color pair. Providers flatten state, priority,
// labels, CI status, draft-ness, etc. into badges rather than exposing
// typed, provider-specific fields to the UI layer.
type Badge struct {
	Label string
	Color string
}

type User struct {
	Login       string
	DisplayName string
	AvatarURL   string
}

// Comment is a single comment on an Issue or PullRequest. ParentID is empty
// for top-level comments and for every comment from a provider that doesn't
// support threading; see IssueTracker.SupportsThreadedReplies.
type Comment struct {
	ID        string
	Author    User
	Body      string
	CreatedAt time.Time
	ParentID  string
}

// Issue is a single issue-tracker item. Number is the provider's own
// human-facing identifier (e.g. "123" for GitHub, "ABC-456" for Linear).
type Issue struct {
	ID     string
	Number string
	Title  string
	Body   string
	URL    string
	State  string

	// StateBadge is the issue's state (open/closed/merged, or a Linear
	// workflow state), broken out from Badges so callers don't have to
	// rely on it being element 0 of a flat list. Badges holds everything
	// else — labels, priority, draft-ness, and so on.
	StateBadge Badge
	Badges     []Badge

	// Closed is true once an issue/PR is no longer active — a closed
	// GitHub issue, a closed or merged GitHub PR, or a Linear issue whose
	// workflow state type is "completed" or "canceled". Provider-specific
	// state semantics are normalized here rather than left for callers to
	// infer from State (a free-text, team-customizable string on Linear).
	Closed bool

	Author    User
	Assignees []User
	Comments  []Comment
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CheckStatus string

const (
	CheckPass      CheckStatus = "pass"
	CheckFail      CheckStatus = "fail"
	CheckPending   CheckStatus = "pending"
	CheckSkipped   CheckStatus = "skipped"
	CheckCancelled CheckStatus = "cancelled"
)

type CheckRun struct {
	Name     string
	Workflow string
	Status   CheckStatus
	URL      string
}

type Commit struct {
	SHA     string
	Message string
	Author  User
	URL     string
}

type MergeMethod string

const (
	MergeCommit MergeMethod = "merge"
	MergeSquash MergeMethod = "squash"
	MergeRebase MergeMethod = "rebase"
)

// PullRequest embeds Issue because GitHub PRs are, at the API level, issues
// with extra fields — modeling it as embedding rather than duplication keeps
// that relationship explicit.
type PullRequest struct {
	Issue

	IsDraft       bool
	Mergeable     bool
	BaseBranch    string
	HeadBranch    string
	Commits       []Commit
	Checks        []CheckRun
	AllowedMerges []MergeMethod
}
