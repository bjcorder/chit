# Linear API Research (for chit's Linear provider)

Research notes on Linear's public developer API (developers.linear.app / linear.app/developers), gathered to inform the design of a "Linear provider" plugin for chit. Note: as of this research, `developers.linear.app` 301-redirects to `linear.app/developers` — the docs live at the latter now, and this file cites the live URLs.

## 1. Auth model

- Linear supports two authentication methods for the GraphQL API: **personal API keys** and **OAuth2**. Personal API keys are the recommended path "for personal scripts"; OAuth2 is recommended when "building an application for others to use" (source: https://linear.app/developers/oauth-2-0-authentication).
- Personal API keys can be created **directly from account settings, with no OAuth app registration required**: "Admins and permitted Members can create personal API keys from Settings > Account > Security & Access" (source: https://linear.app/docs/api-and-webhooks). The dedicated settings doc also documents this location, described as letting you "Create API keys for your account with specific permissions, optionally scoped to particular teams" (source: https://linear.app/docs/security-and-access).
- The UI flow: profile icon → Settings → Security & access → Personal API keys section → "New API key" → name it → Create. The key is shown only once; if lost, a new one must be generated. Existing keys can be viewed/revoked from the same menu (source: https://linear.app/docs/api-and-webhooks).
- Personal API key permission model: for each key you choose either full access to the data your user can access, or restrict it to specific permissions — **Read, Write, Admin, Create issues, Create comments** — and you can additionally scope a key to specific teams in the workspace (source: https://linear.app/docs/api-and-webhooks).
- Personal API keys are user-scoped (inherit the creating user's permissions) and long-lived (do not expire unless revoked) (source: https://linear.app/docs/api-and-webhooks).
- Separate OAuth2 **scopes** exist for OAuth apps (not the same list as personal-key permissions above): `read` (default, "Read access for the user's account. This scope will always be present."), `write` ("Write access for the user's account..."), `issues:create` ("Allows creating new issues and their attachments"), `comments:create` ("Allows creating new issue comments"), `timeSchedule:write` ("Allows creating and modifying time schedules"), and `admin` ("Full access to admin level endpoints. You should never ask for this permission unless it's absolutely needed"). Additional agent-specific scopes (e.g. `app:assignable`, `app:mentionable`) exist and are documented separately for app/agent authentication (source: https://linear.app/developers/oauth-2-0-authentication).
- Auth header formats differ by method: personal API key requests use `Authorization: <API_KEY>` (no "Bearer" prefix); OAuth2 requests use `Authorization: Bearer <ACCESS_TOKEN>` (source: https://linear.app/developers/graphql).
- **Implication for chit**: a single-user TUI tool can authenticate with just a personal API key generated from Settings > Account > Security & Access — no OAuth app registration, redirect URI, or client secret needed, closely mirroring how a GitHub PAT is used today.

## 2. API shape

- Linear's public API is **GraphQL-only**; the docs make no mention of a general-purpose REST surface for issue/project data (source: https://linear.app/developers/graphql).
- The GraphQL endpoint is a single URL: **`https://api.linear.app/graphql`**, accessed via HTTP POST (source: https://linear.app/developers/graphql; corroborated by https://linear.app/developers/oauth-2-0-authentication).
- For exploration without installing anything, Linear points developers at Apollo Studio's schema explorer/sandbox for the Linear API rather than a separate REST reference (source: https://linear.app/developers/graphql; schema explorer at https://studio.apollographql.com/public/Linear-API/schema/reference?variant=current).
- A narrow non-GraphQL exception exists for **file uploads/attachments**, which use a separate authenticated upload flow documented on its own page, but this is not a general REST API for reading/writing issues (source: https://linear.app/developers/file-storage-authentication).

## 3. Hierarchy

- A **Workspace** is the top-level container: "A Linear workspace is the container for all issues, teams and other concepts relating to an individual company" (source: https://linear.app/docs/conceptual-model).
- **Teams** are "the primary organizational unit in Linear. Each team owns its own workflow, triage process, and planning cadence" (source: https://linear.app/docs/conceptual-model). Teams can be structured into parent teams and sub-teams, where sub-teams can inherit settings such as workflow configuration, cycles, or labels from a parent team (source: https://linear.app/docs/teams).
- **Cycles** are team-specific repeating planning periods for issues — "Cycles are team-specific. You can set up cycles so that all teams follow the same schedule, but you can't view more than one team's cycles at once" (source: https://linear.app/docs/conceptual-model).
- **Projects** group issues around a shared outcome (e.g. a feature launch) and are comprised of issues plus optional documents; unlike teams, a project can be **shared across multiple teams** rather than being owned by exactly one (source: https://linear.app/docs/conceptual-model; https://linear.app/docs/projects).
- **Issues** are the fundamental unit of work; per the schema, every `Issue` belongs to exactly one `Team` (`team: Team!`, "Every issue must belong to exactly one team, which determines the available workflow states, labels, and other team-specific configuration"), and optionally to one `Cycle` and one `Project` (both nullable) (source: https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql).
- **Team is the closer analogue to a GitHub "repo"**, not Project: every issue has exactly one owning team (mandatory, 1:1, like a repo owning its issues), the team defines the workflow-state set, labels, and other issue configuration, and cycles are strictly team-scoped. Projects are a cross-cutting, optional grouping that issues from multiple teams can share, more like a milestone/theme spanning repos than a container equivalent to a repo (source: https://linear.app/docs/conceptual-model; schema confirmation at https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql).
- **Implication for chit**: if chit models a GitHub "repo" as its top-level browsable container, the natural Linear equivalent to list/select is **Team**, with Project available as a secondary optional filter/grouping dimension (similar to a milestone), and Cycle as a further team-scoped time-boxed filter.

## 4. Issue fields

Pulled directly from the `type Issue implements Node { ... }` definition in Linear's canonical GraphQL SDL (source: https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql, `type Issue` block). Field name — type — doc comment:

- `state: WorkflowState!` — "The workflow state (issue status) that the issue is currently in. Workflow states represent the issue's progress through the team's workflow, such as Triage, Todo, In Progress, Done, or Canceled."
- `labels(...): IssueLabelConnection!` — paginated connection; "Labels associated with this issue."
- `priority: Float!` — "The priority of the issue. 0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low."
- `priorityLabel: String!` — "Label for the priority." (human-readable string form of the numeric priority)
- `assignee: User` — "The user to whom the issue is assigned. Null if the issue is unassigned."
- `estimate: Float` — "The estimate of the complexity of the issue. The specific scale used depends on the team's estimation configuration (e.g., points, T-shirt sizes). Null if no estimate has been set."
- `cycle: Cycle` — "The cycle that the issue is associated with. Null if the issue is not part of any cycle."
- `project: Project` — "The project that the issue is associated with. Null if the issue is not part of any project."
- `projectMilestone: ProjectMilestone` — "The project milestone that the issue is associated with. Null if the issue is not assigned to a specific milestone within its project."
- `subscribers(...): UserConnection!` — paginated connection; "Users who are subscribed to the issue."
- `team: Team!` — "The team that the issue belongs to. Every issue must belong to exactly one team..."
- `title: String!` — "The issue's title. This is the primary human-readable summary of the work item."
- `identifier: String!` — "Issue's human readable identifier (e.g. ENG-123)."
- `number: Float!` — "The issue's unique number, scoped to the issue's team. Together with the team key, this forms the issue's human-readable identifier (e.g., ENG-123)."
- `url: String!` — "Issue URL."
- `dueDate: TimelessDate` — "The date at which the issue is due."
- `parent: Issue` / `children(...): IssueConnection!` — sub-issue hierarchy ("The parent of the issue." / "Children of the issue.")
- `sortOrder: Float!` — "The order of the item in relation to other items in the organization. Used for manual sorting in list views." (there is also a deprecated `boardOrder: Float!` superseded by `sortOrder`)
- `comments(...): CommentConnection!` — "Comments associated with the issue, including inline comments on the issue's description."
- `creator: User`, `assignee: User`, `delegate: User` (AI agent delegate), `botActor: ActorBot` — actor-related fields.
- `createdAt: DateTime!`, `completedAt: DateTime`, `canceledAt: DateTime`, `autoArchivedAt`, `autoClosedAt`, `archivedAt`, `addedToCycleAt`, `addedToProjectAt`, `addedToTeamAt` — lifecycle timestamps.
- `description: String` — issue body in markdown; `documentContent: DocumentContent` (alpha) — richer description representation.
- `attachments(...): AttachmentConnection!` — linked external attachments (e.g. PRs, links).

These map closely onto GitHub Issue "badges": `state`↔state/status, `labels`↔labels, `priority`/`priorityLabel`↔(no direct GitHub equivalent — GitHub has no native priority field), `assignee`↔assignee, `estimate`↔(no GitHub equivalent — closest is a custom field/points), `cycle`↔(no GitHub equivalent — closest is a Milestone/Sprint), `project`↔(loosely, GitHub Projects, but Linear's Project is issue-native rather than a separate board object), `subscribers`↔GitHub's notification subscribers.

## 5. Comments

- Linear's `Comment` type has a genuine **parent/thread field**: `parent: Comment` — "The parent comment under which the current comment is nested. Null for top-level comments that are not replies." There is also `parentId: String` (the raw ID form) and a `children(...): CommentConnection!` field — "The children of the comment." (source: https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql, `type Comment` block).
- This confirms Linear comments are **threaded, not a flat list** — a comment can be a reply nested under another comment, and each comment can be queried for its own replies via `children`.
- The end-user docs corroborate this at the product level: "Threads allow you to continue on a topic mentioned in a comment," with a reply-arrow icon on individual comments to create threaded responses, and threads that can be marked resolved (source: https://linear.app/docs/comment-on-issues).
- On the `Issue` type, `comments(...): CommentConnection!` returns "Comments associated with the issue, including inline comments on the issue's description" — this is the top-level entry point; callers reconstruct the tree using each comment's `parent`/`children` (source: https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql).
- **Implication for chit**: rendering a Linear issue's comments as a flat scrollback (as GitHub Issues effectively are) would lose real structure — chit's Linear provider should either render reply nesting/indentation or explicitly flatten with a "replying to" indicator, since Linear's own UI treats threading as a first-class feature.

## 6. Rate limits

Documented on Linear's dedicated rate-limiting page (source: https://linear.app/developers/rate-limiting). Two independent limit dimensions apply simultaneously, both using "a leaky bucket algorithm with constant token refill rates":

**Request-count limits (per hour):**
| Authentication | Limit | Scope | Period |
|---|---|---|---|
| API key | 2,500 requests | per User | 1 hour |
| OAuth app | 5,000 requests | per User (or App User) | 1 hour |
| Unauthenticated | 600 requests | per IP address | 1 hour |

**Complexity-point limits (per hour):**
| Authentication | Limit | Scope | Period |
|---|---|---|---|
| API key | 3,000,000 points | per User | 1 hour |
| OAuth app | 2,000,000 points | per User (or App User) | 1 hour |
| Unauthenticated | 100,000 points | per IP address | 1 hour |

- A **single query** is additionally capped at a maximum complexity of **10,000 points**, regardless of remaining hourly budget (source: https://linear.app/developers/rate-limiting).
- Responses include rate-limit headers tracking remaining quota and reset time; exceeding a limit returns HTTP 400 with error code `RATELIMITED` (source: https://linear.app/developers/rate-limiting).
- Higher limits are available on request through Linear support (source: https://linear.app/developers/rate-limiting).
- **Implication for chit**: personal API-key auth (2,500 req/hr, 3M points/hr) is actually *more* generous on the points budget than OAuth-app auth, reinforcing that a single-user PAT-based flow is both simpler and comparably or more capable for chit's use case than building OAuth.

## 7. SDKs

- Linear officially maintains and publishes a **TypeScript SDK**, `@linear/sdk`, installable via `npm install @linear/sdk`, "written in Typescript but can also be used in any Javascript environment" (source: https://linear.app/developers/sdk).
- Its source lives in Linear's own monorepo at `github.com/linear/linear`, specifically under `packages/sdk` (source: https://github.com/linear/linear/tree/master/packages/sdk). The canonical GraphQL schema SDL file used to generate it is checked into that same package at `packages/sdk/src/schema.graphql` (source: https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql).
- **No official Go SDK exists.** The `linear/linear` monorepo's open-source packages are all TypeScript/JavaScript-focused (`sdk`, `import`, `codegen-doc`, `codegen-sdk`, `codegen-test`) with no Go package present (source: https://github.com/linear/linear). The docs' SDK page likewise makes no mention of a Go, Python, or other official language client (source: https://linear.app/developers/sdk).
- The **community Go ecosystem for Linear is small and immature** — no widely-adopted, high-star Go SDK was found. Candidates surfaced:
  - **`chainguard-sandbox/go-linear`** (https://github.com/chainguard-sandbox/go-linear) — the most feature-complete option found: a type-safe Go SDK generated from Linear's schema, plus a CLI and an MCP server. Its own README states explicitly "Not official: This is a third-party client. Official Linear SDKs at https://github.com/linear." At time of research it had 7 stars / 5 forks but a substantial commit history (506 commits), suggesting real development activity despite low star count (source: https://github.com/chainguard-sandbox/go-linear).
  - **`securiteru/linear-cli`** (https://github.com/securiteru/linear-cli) — a Go CLI (not a standalone SDK library) covering "35+ commands for issues, projects, cycles, initiatives, webhooks," 2 stars, described as "Built for agentic workflows" (source: https://github.com/securiteru/linear-cli).
  - **`carlosflorencio/linear-cli`** (https://github.com/carlosflorencio/linear-cli) — a small, explicitly work-in-progress Go CLI wrapping the API, 0 stars (source: https://github.com/carlosflorencio/linear-cli).
  - Given the low adoption across all community Go options, **chit's Linear provider will likely need to hand-roll a thin Go GraphQL client against `https://api.linear.app/graphql`** (e.g. using a generic GraphQL client library plus generated types from Linear's public `schema.graphql`) rather than depending on an existing Go SDK.
