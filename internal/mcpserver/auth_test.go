package mcpserver_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/storage"
)

func TestRequireAgentToken_RejectsMissingToken(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	handler := mcpserver.RequireAgentToken(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAgentToken_RejectsUnknownToken(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	handler := mcpserver.RequireAgentToken(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer arp_not-a-real-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAgentToken_AcceptsValidTokenAndSetsActor(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	actor, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	token, err := storage.IssueAgentToken(db, actor.ID)
	if err != nil {
		t.Fatalf("IssueAgentToken() error = %v", err)
	}

	var gotActorID string
	handler := mcpserver.RequireAgentToken(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a, ok := mcpserver.ActorFromContext(r.Context())
		if !ok {
			t.Error("ActorFromContext() found no actor")
		} else {
			gotActorID = a.ID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotActorID != actor.ID {
		t.Errorf("actor in context = %q, want %q", gotActorID, actor.ID)
	}
}
