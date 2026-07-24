package storage_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"go-ai-rendezvous-point/internal/storage"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openTestPostgresDB creates a uniquely-named schema on the Postgres
// instance at pgDSN, opens storage against it with that schema as the
// connection's search_path, and registers a cleanup to drop the schema —
// giving each test the same full isolation SQLite's per-test temp file
// gives it, without needing a separate database per test.
//
// pgDSN must not already contain a "?" query string (the docker-compose.yml
// DSN in this repo, e.g. "postgres://rendezvous:rendezvous@localhost:5432/rendezvous_test",
// satisfies this).
func openTestPostgresDB(t *testing.T, pgDSN string) *gorm.DB {
	t.Helper()

	schema := fmt.Sprintf("test_%d", rand.Int63())

	setup, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to set up test schema: %v", err)
	}
	if err := setup.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema)).Error; err != nil {
		t.Fatalf("CREATE SCHEMA %s: %v", schema, err)
	}
	t.Cleanup(func() {
		setup.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	})

	db, err := storage.Open(pgDSN + "?search_path=" + schema)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return db
}

func TestOpen_CreatesActorTableOnPostgres(t *testing.T) {
	pgDSN := os.Getenv("TEST_POSTGRES_DSN")
	if pgDSN == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping Postgres-backed test — run `docker compose up -d` and set TEST_POSTGRES_DSN=postgres://rendezvous:rendezvous@localhost:5432/rendezvous_test to enable")
	}

	db := openTestPostgresDB(t, pgDSN)

	actor := &storage.Actor{
		ID:          "actor-1",
		DisplayName: "agent-a",
		Kind:        storage.ActorKindAgent,
	}
	if err := db.Create(actor).Error; err != nil {
		t.Fatalf("Create(actor) error = %v", err)
	}

	var fetched storage.Actor
	if err := db.First(&fetched, "id = ?", "actor-1").Error; err != nil {
		t.Fatalf("First(actor) error = %v", err)
	}
	if fetched.DisplayName != "agent-a" {
		t.Errorf("DisplayName = %q, want %q", fetched.DisplayName, "agent-a")
	}
	if fetched.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
}
