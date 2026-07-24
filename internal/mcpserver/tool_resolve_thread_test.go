package mcpserver_test

import (
	"path/filepath"
	"testing"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

func TestResolveAndReopenThreadTools(t *testing.T) {
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
		"title": "Deploy",
		"body":  "body",
	}, &created)

	var resolved mcpserver.ResolveThreadOutput
	callTool(t, session, "resolve_thread", map[string]any{
		"thread_id": created.ThreadID,
	}, &resolved)
	if resolved.Status != "resolved" {
		t.Fatalf("resolve_thread Status = %q, want %q", resolved.Status, "resolved")
	}

	var reopened mcpserver.ReopenThreadOutput
	callTool(t, session, "reopen_thread", map[string]any{
		"thread_id": created.ThreadID,
	}, &reopened)
	if reopened.Status != "open" {
		t.Fatalf("reopen_thread Status = %q, want %q", reopened.Status, "open")
	}
}
