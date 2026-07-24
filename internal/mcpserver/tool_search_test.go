package mcpserver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

func TestSearchTool_FindsMatchesAndRejectsUnsupportedMode(t *testing.T) {
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

	var created mcpserver.CreateThreadOutput
	callTool(t, session, "create_thread", map[string]any{
		"title": "Deploy pipeline broken",
		"body":  "The nightly deploy pipeline is failing on staging.",
	}, &created)

	var found mcpserver.SearchOutput
	callTool(t, session, "search", map[string]any{
		"query": "pipeline",
	}, &found)
	if len(found.Threads) != 1 || found.Threads[0].ID != created.ThreadID {
		t.Fatalf("Threads = %+v, want exactly the created thread", found.Threads)
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query": "pipeline",
			"mode":  "semantic",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(search, mode=semantic) error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool(search, mode=semantic) succeeded, want a tool error since semantic mode isn't supported yet")
	}
}
