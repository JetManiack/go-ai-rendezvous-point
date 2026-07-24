package mcpserver_test

import (
	"path/filepath"
	"testing"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

func TestWatchAndUnwatchThreadTools(t *testing.T) {
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
	agentC, err := storage.CreateAgent(db, "agent-c")
	if err != nil {
		t.Fatalf("CreateAgent(agent-c) error = %v", err)
	}
	tokenA, err := storage.IssueAgentToken(db, agentA.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken(agent-a) error = %v", err)
	}
	tokenC, err := storage.IssueAgentToken(db, agentC.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken(agent-c) error = %v", err)
	}

	sessionA, cleanupA := newTestSession(t, db, tokenA)
	defer cleanupA()

	var created mcpserver.CreateThreadOutput
	callTool(t, sessionA, "create_thread", map[string]any{
		"title": "Deploy",
		"body":  "body",
	}, &created)

	sessionC, cleanupC := newTestSession(t, db, tokenC)
	defer cleanupC()

	var watched mcpserver.WatchThreadOutput
	callTool(t, sessionC, "watch_thread", map[string]any{
		"thread_id": created.ThreadID,
	}, &watched)
	if !watched.Watching {
		t.Fatal("WatchThreadOutput.Watching = false, want true")
	}

	if _, err := storage.AddReply(db, created.ThreadID, agentB.ID, "update", nil); err != nil {
		t.Fatalf("AddReply() error = %v", err)
	}

	var caughtUp mcpserver.CatchUpOutput
	callTool(t, sessionC, "catch_up", map[string]any{}, &caughtUp)
	if len(caughtUp.UnreadReplies) != 1 {
		t.Fatalf("UnreadReplies while watching = %+v, want exactly one reply", caughtUp.UnreadReplies)
	}

	var unwatched mcpserver.UnwatchThreadOutput
	callTool(t, sessionC, "unwatch_thread", map[string]any{
		"thread_id": created.ThreadID,
	}, &unwatched)
	if unwatched.Watching {
		t.Fatal("UnwatchThreadOutput.Watching = true, want false")
	}

	if _, err := storage.AddReply(db, created.ThreadID, agentB.ID, "another update", nil); err != nil {
		t.Fatalf("second AddReply() error = %v", err)
	}

	var secondCatchUp mcpserver.CatchUpOutput
	callTool(t, sessionC, "catch_up", map[string]any{}, &secondCatchUp)
	if len(secondCatchUp.UnreadReplies) != 0 {
		t.Fatalf("UnreadReplies after unwatch = %+v, want empty", secondCatchUp.UnreadReplies)
	}
}
