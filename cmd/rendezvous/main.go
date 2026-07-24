package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"go-ai-rendezvous-point/internal/frontend"
	"go-ai-rendezvous-point/internal/humanauth"
	"go-ai-rendezvous-point/internal/mcpserver"
	"go-ai-rendezvous-point/internal/restapi"
	"go-ai-rendezvous-point/internal/storage"
)

func newRootCommand() *cli.Command {
	return &cli.Command{
		Name:  "rendezvous",
		Usage: "AI Rendezvous Point MCP server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen-addr",
				Value:   ":8080",
				Sources: cli.EnvVars("LISTEN_ADDR"),
			},
			&cli.StringFlag{
				Name:    "db-dsn",
				Value:   "data/rendezvous.db",
				Sources: cli.EnvVars("DB_DSN"),
			},
			&cli.BoolFlag{
				Name:    "auth-stub",
				Usage:   "use a fixed, always-admin test identity instead of real Keycloak/OIDC auth — local development only, never set this in a real deployment",
				Sources: cli.EnvVars("AUTH_STUB"),
			},
			&cli.StringFlag{
				Name:    "oidc-issuer",
				Usage:   "Keycloak realm issuer URL, e.g. https://keycloak.internal/realms/rendezvous",
				Sources: cli.EnvVars("OIDC_ISSUER"),
			},
			&cli.StringFlag{
				Name:    "oidc-client-id",
				Usage:   "client ID of the dedicated OIDC client configured in Keycloak for this app",
				Sources: cli.EnvVars("OIDC_CLIENT_ID"),
			},
			&cli.StringFlag{
				Name:    "oidc-client-secret",
				Usage:   "client secret of the dedicated OIDC client configured in Keycloak for this app",
				Sources: cli.EnvVars("OIDC_CLIENT_SECRET"),
			},
			&cli.StringFlag{
				Name:    "public-url",
				Usage:   "this server's externally-reachable base URL (used to build the OIDC redirect URI <public-url>/auth/callback — must match what's configured on the Keycloak client)",
				Sources: cli.EnvVars("PUBLIC_URL"),
			},
			&cli.StringFlag{
				Name:    "admin-group",
				Value:   "admins",
				Usage:   "Keycloak group (from the ID token's groups claim) whose members get role admin; everyone else gets viewer",
				Sources: cli.EnvVars("ADMIN_GROUP"),
			},
			&cli.StringFlag{
				Name:    "session-encryption-key",
				Usage:   "base64-encoded 32-byte key used to encrypt session refresh tokens at rest (generate one with: openssl rand -base64 32)",
				Sources: cli.EnvVars("SESSION_ENCRYPTION_KEY"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			db, err := storage.Open(cmd.String("db-dsn"))
			if err != nil {
				return err
			}

			var authProvider humanauth.Provider
			var oidcHandlers humanauth.OIDCHandlers
			useOIDC := !cmd.Bool("auth-stub")

			if useOIDC {
				if cmd.String("oidc-issuer") == "" || cmd.String("oidc-client-id") == "" || cmd.String("oidc-client-secret") == "" || cmd.String("public-url") == "" || cmd.String("session-encryption-key") == "" {
					return fmt.Errorf("OIDC auth requires --oidc-issuer, --oidc-client-id, --oidc-client-secret, --public-url, and --session-encryption-key (or pass --auth-stub for local development)")
				}
				encryptionKey, err := base64.StdEncoding.DecodeString(cmd.String("session-encryption-key"))
				if err != nil {
					return fmt.Errorf("--session-encryption-key must be valid base64: %w", err)
				}
				if len(encryptionKey) != 32 {
					return fmt.Errorf("--session-encryption-key must decode to exactly 32 bytes, got %d (generate one with: openssl rand -base64 32)", len(encryptionKey))
				}
				cfg := humanauth.OIDCConfig{
					Issuer:        cmd.String("oidc-issuer"),
					ClientID:      cmd.String("oidc-client-id"),
					ClientSecret:  cmd.String("oidc-client-secret"),
					PublicURL:     cmd.String("public-url"),
					AdminGroup:    cmd.String("admin-group"),
					EncryptionKey: encryptionKey,
				}
				provider, handlers, err := humanauth.NewOIDCHandlers(ctx, db, cfg)
				if err != nil {
					return fmt.Errorf("configure OIDC: %w", err)
				}
				authProvider = provider
				oidcHandlers = handlers
			} else {
				// ⚠️ StubProvider is a mandatory-to-replace interim measure: it
				// authenticates every request as a fixed always-admin identity
				// with no real credential check. Only reachable via --auth-stub,
				// which must never be set outside local development. See
				// docs/superpowers/specs/2026-07-24-ai-rendezvous-point-oidc-design.md.
				authProvider = humanauth.StubProvider{}
				slog.Warn("human auth running in STUB mode — do not deploy to production")
			}

			frontendFS, err := frontend.FS(false)
			if err != nil {
				return err
			}

			mux := http.NewServeMux()
			mux.Handle("/mcp", mcpserver.NewHTTPHandler(db))
			mux.Handle("/api/", http.StripPrefix("/api", restapi.NewHandler(db, authProvider)))
			if useOIDC {
				mux.HandleFunc("/auth/login", oidcHandlers.Login)
				mux.HandleFunc("/auth/callback", oidcHandlers.Callback)
				mux.HandleFunc("/auth/logout", oidcHandlers.Logout)
			}
			mux.Handle("/", http.FileServer(frontendFS))

			addr := cmd.String("listen-addr")
			// WriteTimeout is safe for now since catch_up is pull-based and
			// no server-initiated streaming push channel is built (per the
			// design doc); if long-lived SSE streaming is added later, this
			// may need to become per-handler or be removed.
			server := &http.Server{
				Addr:              addr,
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       120 * time.Second,
			}

			serveErr := make(chan error, 1)
			go func() {
				slog.Info("starting server", "addr", addr)
				serveErr <- server.ListenAndServe()
			}()

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			case err := <-serveErr:
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}
		},
	}
}

func main() {
	if err := newRootCommand().Run(context.Background(), os.Args); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
