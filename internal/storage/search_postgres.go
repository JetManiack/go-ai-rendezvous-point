package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// setupSearchPostgres adds a tsvector column and a GIN index to threads
// and replies, and BEFORE INSERT triggers that populate it — the Postgres
// equivalent of search_sqlite.go's setupFTS. Every statement uses IF NOT
// EXISTS/OR REPLACE, so it's safe to call on every Open.
//
// BEFORE INSERT (not AFTER, unlike SQLite's FTS5 triggers) because
// Postgres can only populate a column on the row currently being inserted
// from a BEFORE trigger — an AFTER INSERT trigger can no longer modify
// that row. The underlying intent matches setupFTS exactly: insert-time
// population only, no update handling, because nothing in this codebase
// updates a Thread's Title/Body or a Reply's Body today. If that ever
// changes, this trigger (and setupFTS's SQLite triggers) MUST be updated
// together, or the full-text index will silently drift from the
// underlying tables.
func setupSearchPostgres(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE threads ADD COLUMN IF NOT EXISTS search_vector tsvector`,
		`ALTER TABLE replies ADD COLUMN IF NOT EXISTS search_vector tsvector`,
		`CREATE INDEX IF NOT EXISTS threads_search_vector_idx ON threads USING GIN (search_vector)`,
		`CREATE INDEX IF NOT EXISTS replies_search_vector_idx ON replies USING GIN (search_vector)`,
		`CREATE OR REPLACE FUNCTION threads_search_vector_update() RETURNS trigger AS $$
			BEGIN
				NEW.search_vector := to_tsvector('english', coalesce(NEW.title, '') || ' ' || coalesce(NEW.body, ''));
				RETURN NEW;
			END
		$$ LANGUAGE plpgsql`,
		`CREATE OR REPLACE FUNCTION replies_search_vector_update() RETURNS trigger AS $$
			BEGIN
				NEW.search_vector := to_tsvector('english', coalesce(NEW.body, ''));
				RETURN NEW;
			END
		$$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS threads_search_vector_trigger ON threads`,
		`CREATE TRIGGER threads_search_vector_trigger BEFORE INSERT ON threads
			FOR EACH ROW EXECUTE FUNCTION threads_search_vector_update()`,
		`DROP TRIGGER IF EXISTS replies_search_vector_trigger ON replies`,
		`CREATE TRIGGER replies_search_vector_trigger BEFORE INSERT ON replies
			FOR EACH ROW EXECUTE FUNCTION replies_search_vector_update()`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("setup postgres search: exec %q: %w", stmt, err)
		}
	}
	return nil
}

// searchPostgres performs a full-text search over thread titles/bodies and
// reply bodies using Postgres tsvector/GIN, ranked by relevance (ts_rank —
// higher is better, the opposite convention from SQLite's bm25 in
// search_sqlite.go, so results are ordered DESC here).
// plainto_tsquery gives "every word must appear, any order" semantics,
// matching ftsMatchExpr's tested behavior in search_sqlite.go — not
// websearch_to_tsquery's richer operator syntax, which would give
// different behavior depending on which backend is deployed.
// query is guaranteed non-empty by Search, which checks this before
// dispatching to either backend.
func searchPostgres(db *gorm.DB, query string, limit int) (*SearchResult, error) {
	var threads []Thread
	if err := db.Raw(`
		SELECT * FROM threads
		WHERE search_vector @@ plainto_tsquery('english', ?)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', ?)) DESC
		LIMIT ?
	`, query, query, limit).Scan(&threads).Error; err != nil {
		return nil, fmt.Errorf("search threads: %w", err)
	}

	var replies []Reply
	if err := db.Raw(`
		SELECT * FROM replies
		WHERE search_vector @@ plainto_tsquery('english', ?)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', ?)) DESC
		LIMIT ?
	`, query, query, limit).Scan(&replies).Error; err != nil {
		return nil, fmt.Errorf("search replies: %w", err)
	}

	return &SearchResult{Threads: threads, Replies: replies}, nil
}
