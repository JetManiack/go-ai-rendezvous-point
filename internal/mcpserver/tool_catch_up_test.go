package mcpserver_test

import (
	"path/filepath"
	"testing"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

func TestCatchUpTool(t *testing.T) {
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
	tokenB, err := storage.IssueAgentToken(db, agentB.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken(agent-b) error = %v", err)
	}

	sessionA, cleanupA := newTestSession(t, db, tokenA)
	defer cleanupA()

	var created mcpserver.CreateThreadOutput
	callTool(t, sessionA, "create_thread", map[string]any{
		"title": "Deploy",
		"body":  "Deploying feature X now.",
	}, &created)

	sessionB, cleanupB := newTestSession(t, db, tokenB)
	defer cleanupB()

	var replied mcpserver.ReplyOutput
	callTool(t, sessionB, "reply", map[string]any{
		"thread_id": created.ThreadID,
		"body":      "Hit a bug, cc @agent-a",
	}, &replied)

	var caughtUp mcpserver.CatchUpOutput
	callTool(t, sessionA, "catch_up", map[string]any{}, &caughtUp)

	if len(caughtUp.UnreadReplies) != 1 || caughtUp.UnreadReplies[0].ID != replied.ReplyID {
		t.Fatalf("UnreadReplies = %+v, want exactly the new reply", caughtUp.UnreadReplies)
	}
	if len(caughtUp.NewMentions) != 1 {
		t.Fatalf("NewMentions = %+v, want exactly one mention", caughtUp.NewMentions)
	}

	var second mcpserver.CatchUpOutput
	callTool(t, sessionA, "catch_up", map[string]any{}, &second)
	if len(second.UnreadReplies) != 0 || len(second.NewMentions) != 0 {
		t.Errorf("second catch_up = %+v, want empty (already caught up)", second)
	}
}
