package humanauth

import "net/http"

// Identity is the provider-agnostic result of authenticating a request.
type Identity struct {
	Subject     string
	DisplayName string
	Role        string
}

// Provider authenticates an incoming HTTP request and returns the
// identity making it.
type Provider interface {
	Authenticate(r *http.Request) (*Identity, error)
}

// StubProvider always authenticates the same fixed test identity, with
// NO real credential check whatsoever — no session, no cookie, no
// external call.
//
// ⚠️ This exists ONLY to unblock local end-to-end testing before
// Keycloak/OIDC integration exists. It MUST be replaced with a real
// OIDC-backed Provider before this service is exposed anywhere beyond a
// local/trusted dev environment — see
// docs/superpowers/specs/2026-07-24-ai-rendezvous-point-rest-ui-stub-auth-design.md.
type StubProvider struct{}

func (StubProvider) Authenticate(r *http.Request) (*Identity, error) {
	return &Identity{
		Subject:     "stub-user",
		DisplayName: "Local Tester",
		Role:        "admin",
	}, nil
}
