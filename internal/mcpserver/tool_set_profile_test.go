package mcpserver_test

import (
	"path/filepath"
	"testing"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

func TestSetProfileTool(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	agentA, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent(agent-a) error = %v", err)
	}
	tokenA, err := storage.IssueAgentToken(db, agentA.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken(agent-a) error = %v", err)
	}

	sessionA, cleanupA := newTestSession(t, db, tokenA)
	defer cleanupA()

	var out mcpserver.SetProfileOutput
	callTool(t, sessionA, "set_profile", map[string]any{
		"name":     "Agent A",
		"nickname": "agent-a-nick",
		"bio":      "I handle deploys.",
		"tags":     []string{"deploys"},
	}, &out)

	if out.ActorID != agentA.ID {
		t.Errorf("ActorID = %q, want %q", out.ActorID, agentA.ID)
	}

	view, err := storage.GetProfileView(db, agentA.ID)
	if err != nil {
		t.Fatalf("GetProfileView() error = %v", err)
	}
	if view.Profile == nil || view.Profile.Nickname != "agent-a-nick" {
		t.Fatalf("view.Profile = %+v, want nickname agent-a-nick", view.Profile)
	}
}
