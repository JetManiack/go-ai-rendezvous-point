package mcpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

// newTestSessionWithResourceUpdates behaves like newTestSession
// (server_test.go) but also captures every "notifications/resources/updated"
// URI the server sends, via a buffered channel — for tests that observe
// push notifications rather than just call tools. Deliberately not folded
// into newTestSession itself, to avoid touching every existing tool test.
func newTestSessionWithResourceUpdates(t *testing.T, db *gorm.DB, token string) (*mcp.ClientSession, chan string, func()) {
	t.Helper()

	srv := httptest.NewServer(mcpserver.NewHTTPHandler(db))

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.Header.Set("Authorization", "Bearer "+token)
			return http.DefaultTransport.RoundTrip(r)
		}),
	}

	updates := make(chan string, 16)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, &mcp.ClientOptions{
		ResourceUpdatedHandler: func(ctx context.Context, req *mcp.ResourceUpdatedNotificationRequest) {
			updates <- req.Params.URI
		},
	})
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   srv.URL + "/mcp",
		HTTPClient: httpClient,
	}, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("client.Connect() error = %v", err)
	}

	return session, updates, func() {
		session.Close()
		srv.Close()
	}
}

func TestCatchUpTool_IncludesActorID(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	agent, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	token, err := storage.IssueAgentToken(db, agent.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken() error = %v", err)
	}

	session, cleanup := newTestSession(t, db, token)
	defer cleanup()

	var out mcpserver.CatchUpOutput
	callTool(t, session, "catch_up", map[string]any{}, &out)

	if out.ActorID != agent.ID {
		t.Errorf("ActorID = %q, want %q", out.ActorID, agent.ID)
	}
}

func TestSubscribeToOwnCatchUpResource_Succeeds(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	agent, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	token, err := storage.IssueAgentToken(db, agent.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken() error = %v", err)
	}

	session, _, cleanup := newTestSessionWithResourceUpdates(t, db, token)
	defer cleanup()

	if err := session.Subscribe(context.Background(), &mcp.SubscribeParams{URI: "rendezvous://catchup/" + agent.ID}); err != nil {
		t.Fatalf("Subscribe() to own resource error = %v", err)
	}
}

func TestSubscribeToAnotherActorsCatchUpResource_Fails(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	agentA, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent(agent-a) error = %v", err)
	}
	agentB, err := storage.CreateAgent(db, "agent-b")
	if err != nil {
		t.Fatalf("CreateAgent(agent-b) error = %v", err)
	}
	tokenA, err := storage.IssueAgentToken(db, agentA.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken(agent-a) error = %v", err)
	}

	session, _, cleanup := newTestSessionWithResourceUpdates(t, db, tokenA)
	defer cleanup()

	err = session.Subscribe(context.Background(), &mcp.SubscribeParams{URI: "rendezvous://catchup/" + agentB.ID})
	if err == nil {
		t.Fatal("Subscribe() to another actor's resource succeeded, want an error")
	}
}

func TestReadOwnCatchUpResource_ReturnsSummaryWithoutMarkingSeen(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	agentA, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent(agent-a) error = %v", err)
	}
	agentB, err := storage.CreateAgent(db, "agent-b")
	if err != nil {
		t.Fatalf("CreateAgent(agent-b) error = %v", err)
	}
	tokenB, err := storage.IssueAgentToken(db, agentB.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken(agent-b) error = %v", err)
	}
	if _, err := storage.CreateThread(db, agentA.ID, "Deploy", "Deploying, cc @agent-b", nil); err != nil {
		t.Fatalf("CreateThread() error = %v", err)
	}

	session, _, cleanup := newTestSessionWithResourceUpdates(t, db, tokenB)
	defer cleanup()

	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "rendezvous://catchup/" + agentB.ID})
	if err != nil {
		t.Fatalf("ReadResource() error = %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("len(Contents) = %d, want 1", len(result.Contents))
	}
	var summary storage.CatchUpSummary
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &summary); err != nil {
		t.Fatalf("unmarshal resource contents: %v", err)
	}
	if summary.UnseenMentionCount != 1 {
		t.Errorf("UnseenMentionCount = %d, want 1", summary.UnseenMentionCount)
	}

	// Reading the resource must not have marked the mention seen.
	var out mcpserver.CatchUpOutput
	callTool(t, session, "catch_up", map[string]any{}, &out)
	if len(out.NewMentions) != 1 {
		t.Errorf("catch_up NewMentions after a resource read = %d, want 1 (resource read must be non-destructive)", len(out.NewMentions))
	}
}
