package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bjcorder/chit/internal/provider"
)

// gqlRequest captures the query+variables a test server received, so tests
// can assert chit sent the right GraphQL operation.
type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// newTestServer starts an httptest.Server that returns respBody (raw JSON)
// for every request, and reports the last request's Authorization header
// and decoded body via the returned pointers.
func newTestServer(t *testing.T, respBody string, gotAuth *string, gotReq *gqlRequest) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotAuth = r.Header.Get("Authorization")
		if gotReq != nil {
			_ = json.NewDecoder(r.Body).Decode(gotReq)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestProvider(t *testing.T, respBody string, gotAuth *string, gotReq *gqlRequest) *Provider {
	srv := newTestServer(t, respBody, gotAuth, gotReq)
	p := New("lin_api_test123")
	p.client.endpoint = srv.URL
	return p
}

func TestAuthorizationHeaderHasNoBearerPrefix(t *testing.T) {
	var gotAuth string
	p := newTestProvider(t, `{"data":{"organization":{"id":"org1","name":"Acme"}}}`, &gotAuth, nil)

	if _, err := p.ListRootContainers(context.Background()); err != nil {
		t.Fatalf("ListRootContainers: %v", err)
	}
	if gotAuth != "lin_api_test123" {
		t.Errorf("Authorization header = %q, want bare API key with no Bearer prefix", gotAuth)
	}
}

func TestListRootContainersReturnsOneWorkspace(t *testing.T) {
	var gotAuth string
	p := newTestProvider(t, `{"data":{"organization":{"id":"org1","name":"Acme"}}}`, &gotAuth, nil)

	got, err := p.ListRootContainers(context.Background())
	if err != nil {
		t.Fatalf("ListRootContainers: %v", err)
	}
	if len(got) != 1 || got[0].ID != "org1" || got[0].Name != "Acme" || got[0].Kind != provider.KindRoot {
		t.Fatalf("got %+v", got)
	}
}

func TestListChildContainers(t *testing.T) {
	var gotAuth string
	resp := `{"data":{"teams":{"nodes":[{"id":"team1","name":"Engineering","key":"ENG"}]}}}`
	p := newTestProvider(t, resp, &gotAuth, nil)

	got, err := p.ListChildContainers(context.Background(), "org1")
	if err != nil {
		t.Fatalf("ListChildContainers: %v", err)
	}
	if len(got) != 1 || got[0].ID != "team1" || got[0].ParentID != "org1" || got[0].Kind != provider.KindChild {
		t.Fatalf("got %+v", got)
	}
}

const sampleIssueListResp = `{"data":{"team":{"issues":{"nodes":[
  {"id":"iss1","identifier":"ENG-123","title":"Fix the thing","url":"https://linear.app/acme/issue/ENG-123",
   "state":{"name":"In Progress","type":"started"},"priority":1,"priorityLabel":"Urgent",
   "assignee":{"displayName":"Jamie"},"labels":{"nodes":[{"name":"bug"}]},
   "createdAt":"2026-08-01T00:00:00Z","updatedAt":"2026-08-02T00:00:00Z"}
]}}}}`

func TestListIssues(t *testing.T) {
	var gotAuth string
	var gotReq gqlRequest
	p := newTestProvider(t, sampleIssueListResp, &gotAuth, &gotReq)

	got, err := p.ListIssues(context.Background(), "team1")
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if gotReq.Variables["teamId"] != "team1" {
		t.Errorf("sent teamId = %v, want %q", gotReq.Variables["teamId"], "team1")
	}
	if len(got) != 1 {
		t.Fatalf("got %d issues, want 1", len(got))
	}
	issue := got[0]
	if issue.ID != "iss1" || issue.Number != "ENG-123" {
		t.Errorf("ID/Number = %q/%q", issue.ID, issue.Number)
	}
	if len(issue.Assignees) != 1 || issue.Assignees[0].Login != "Jamie" {
		t.Errorf("Assignees = %+v", issue.Assignees)
	}
	var haveState, havePriority, haveLabel bool
	for _, b := range issue.Badges {
		switch b.Label {
		case "In Progress":
			haveState = true
		case "Urgent":
			havePriority = true
		case "bug":
			haveLabel = true
		}
	}
	if !haveState || !havePriority || !haveLabel {
		t.Errorf("Badges = %+v, want state+priority+label badges", issue.Badges)
	}
}

const sampleIssueDetailResp = `{"data":{"issue":{
  "id":"iss1","identifier":"ENG-123","title":"Fix the thing","url":"https://linear.app/acme/issue/ENG-123",
  "description":"issue body in markdown",
  "state":{"name":"In Progress","type":"started"},"priority":0,"priorityLabel":"No priority",
  "assignee":null,"labels":{"nodes":[]},
  "createdAt":"2026-08-01T00:00:00Z","updatedAt":"2026-08-02T00:00:00Z",
  "comments":{"nodes":[
    {"id":"c1","body":"top-level comment","createdAt":"2026-08-01T01:00:00Z","user":{"displayName":"Jamie"},"parent":null},
    {"id":"c2","body":"a reply","createdAt":"2026-08-01T02:00:00Z","user":{"displayName":"Alex"},"parent":{"id":"c1"}}
  ]}
}}}`

func TestGetIssueParsesThreadedComments(t *testing.T) {
	var gotAuth string
	var gotReq gqlRequest
	p := newTestProvider(t, sampleIssueDetailResp, &gotAuth, &gotReq)

	got, err := p.GetIssue(context.Background(), "team1", "iss1")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if got.Body != "issue body in markdown" {
		t.Errorf("Body = %q", got.Body)
	}
	if len(got.Comments) != 2 {
		t.Fatalf("Comments = %+v", got.Comments)
	}
	if got.Comments[0].ParentID != "" {
		t.Errorf("top-level comment ParentID = %q, want empty", got.Comments[0].ParentID)
	}
	if got.Comments[1].ParentID != "c1" {
		t.Errorf("reply ParentID = %q, want %q", got.Comments[1].ParentID, "c1")
	}
	// No priority badge should be added when priority is 0 ("No priority").
	for _, b := range got.Badges {
		if b.Label == "No priority" {
			t.Error("got a badge for priority 0, want it skipped")
		}
	}
}

func TestSupportsThreadedRepliesIsTrue(t *testing.T) {
	p := New("key")
	if !p.SupportsThreadedReplies() {
		t.Error("SupportsThreadedReplies() = false, want true for Linear")
	}
}

func TestAddCommentOmitsParentID(t *testing.T) {
	var gotAuth string
	var gotReq gqlRequest
	resp := `{"data":{"commentCreate":{"comment":{"id":"c3","body":"hello","createdAt":"2026-08-01T00:00:00Z","user":{"displayName":"Jamie"},"parent":null}}}}`
	p := newTestProvider(t, resp, &gotAuth, &gotReq)

	got, err := p.AddComment(context.Background(), "iss1", "hello")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if got.ID != "c3" || got.Author.Login != "Jamie" {
		t.Errorf("got %+v", got)
	}
	if _, ok := gotReq.Variables["parentId"]; ok {
		t.Errorf("variables included parentId for a top-level comment: %v", gotReq.Variables)
	}
}

func TestReplyToCommentSetsParentID(t *testing.T) {
	var gotAuth string
	var gotReq gqlRequest
	resp := `{"data":{"commentCreate":{"comment":{"id":"c4","body":"a reply","createdAt":"2026-08-01T00:00:00Z","user":{"displayName":"Jamie"},"parent":{"id":"c1"}}}}}`
	p := newTestProvider(t, resp, &gotAuth, &gotReq)

	got, err := p.ReplyToComment(context.Background(), "iss1", "c1", "a reply")
	if err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if got.ParentID != "c1" {
		t.Errorf("ParentID = %q, want %q", got.ParentID, "c1")
	}
	if gotReq.Variables["parentId"] != "c1" {
		t.Errorf("sent parentId = %v, want %q", gotReq.Variables["parentId"], "c1")
	}
}

func TestGraphQLErrorsSurfaceAsGoErrors(t *testing.T) {
	var gotAuth string
	resp := `{"errors":[{"message":"entity not found"}]}`
	p := newTestProvider(t, resp, &gotAuth, nil)

	_, err := p.ListRootContainers(context.Background())
	if err == nil || !strings.Contains(err.Error(), "entity not found") {
		t.Errorf("err = %v, want it to contain the GraphQL error message", err)
	}
}

func TestHTTPErrorStatusSurfacesAsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Authentication required"}`))
	}))
	defer srv.Close()

	p := New("bad-key")
	p.client.endpoint = srv.URL

	_, err := p.ListRootContainers(context.Background())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want it to mention HTTP 401", err)
	}
}

func TestRegisteredWithProviderRegistry(t *testing.T) {
	d, ok := provider.Get("linear")
	if !ok {
		t.Fatal(`provider.Get("linear") ok=false — init() didn't register`)
	}
	if d.NewIssueTracker == nil {
		t.Error("NewIssueTracker is nil")
	}
	if d.NewCodeHost != nil {
		t.Error("NewCodeHost is non-nil — Linear has no CodeHost capability")
	}
}
