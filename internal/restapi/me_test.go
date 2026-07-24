package restapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMe_ReturnsAuthenticatedIdentity(t *testing.T) {
	_, handler := openTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /me status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got struct {
		ActorID     string `json:"actor_id"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal /me response: %v", err)
	}
	if got.ActorID == "" {
		t.Error("ActorID is empty")
	}
	if got.DisplayName != "Local Tester" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Local Tester")
	}
	if got.Role != "admin" {
		t.Errorf("Role = %q, want %q", got.Role, "admin")
	}
}
