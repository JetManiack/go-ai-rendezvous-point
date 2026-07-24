package restapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"go-ai-rendezvous-point/internal/humanauth"
)

// NewHandler builds the full REST API handler, mounted with no path
// prefix (the caller mounts it under /api — see cmd/rendezvous/main.go).
// Every route requires human authentication via provider.
func NewHandler(db *gorm.DB, provider humanauth.Provider) http.Handler {
	r := chi.NewRouter()
	r.Use(humanauth.RequireHumanAuth(db, provider))

	r.Route("/threads", func(r chi.Router) {
		r.Get("/", listThreadsHandler(db))
		r.With(humanauth.RequireAdmin).Post("/", createThreadHandler(db))
		r.Get("/{id}", getThreadHandler(db))
		r.With(humanauth.RequireAdmin).Post("/{id}/replies", addReplyHandler(db))
		r.With(humanauth.RequireAdmin).Patch("/{id}", updateThreadStatusHandler(db))
	})

	r.Get("/search", searchHandler(db))

	r.Get("/actors", listActorsByIDHandler(db))

	r.Get("/profiles/{actorID}", getProfileHandler(db))
	r.With(humanauth.RequireAdmin).Put("/profiles/{actorID}", updateProfileByIDHandler(db))
	r.Put("/me/profile", updateOwnProfileHandler(db))

	r.Route("/agents", func(r chi.Router) {
		r.Use(humanauth.RequireAdmin)
		r.Get("/", listAgentsHandler(db))
		r.Post("/", createAgentHandler(db))
		r.Delete("/{id}", deleteAgentHandler(db))
		r.Post("/{id}/tokens", issueTokenHandler(db))
		r.Delete("/{id}/tokens/{tokenID}", revokeTokenHandler(db))
	})

	r.Get("/me", meHandler())

	return r
}
