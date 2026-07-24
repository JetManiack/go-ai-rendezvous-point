package storage

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// setupFTS creates the FTS5 virtual tables and the triggers that keep them
// in sync with Thread/Reply inserts. Every statement uses IF NOT EXISTS, so
// it's safe to call on every Open.
//
// Only AFTER INSERT triggers are defined. Nothing in this codebase updates
// a Thread's Title/Body or a Reply's Body (resolve_thread/reopen_thread
// only ever update Status, which isn't indexed), and nothing deletes a
// Thread or Reply. If either capability is added later, matching AFTER
// UPDATE/AFTER DELETE triggers on thread_fts/reply_fts (and the Postgres
// equivalent in search_postgres.go) MUST be added in that same change, or
// the full-text index will silently drift from the underlying tables.
func setupFTS(db *gorm.DB) error {
	statements := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS thread_fts USING fts5(title, body, content='threads', content_rowid='rowid')`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS reply_fts USING fts5(body, content='replies', content_rowid='rowid')`,
		`CREATE TRIGGER IF NOT EXISTS threads_ai AFTER INSERT ON threads BEGIN
			INSERT INTO thread_fts(rowid, title, body) VALUES (new.rowid, new.title, new.body);
		END`,
		`CREATE TRIGGER IF NOT EXISTS replies_ai AFTER INSERT ON replies BEGIN
			INSERT INTO reply_fts(rowid, body) VALUES (new.rowid, new.body);
		END`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("setup FTS: exec %q: %w", stmt, err)
		}
	}
	return nil
}

// ftsQuote wraps query as a single FTS5 string literal, so its contents are
// always matched as literal text rather than parsed as FTS5 query syntax.
//
// Without this, FTS5's query-string grammar treats characters like `-`, `"`,
// `:`, `(`, `)`, and `*` as operators (NOT, column filters, grouping,
// prefix search, ...) rather than literal text. For example, an unquoted
// query of "nonexistent-keyword-xyz" is parsed as a column filter
// expression and fails with "no such column: keyword" instead of simply
// matching nothing — verified directly against this driver's SQLite build.
// Quoting also prevents user-supplied search input from being able to
// inject arbitrary FTS5 query operators.
func ftsQuote(query string) string {
	return `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
}

// ftsMatchExpr builds an FTS5 MATCH expression that requires every
// whitespace-separated word in query to be present, in any order and not
// necessarily adjacent — the "contains all these words" semantics agents
// expect from a keyword search. Each word is quoted individually via
// ftsQuote (so the same escaping/injection-safety applies per-word), and
// the quoted words are joined with a space, which FTS5 interprets as an
// implicit AND between adjacent terms. This intentionally differs from
// quoting the whole query as one phrase, which would require the words to
// appear adjacent and in that exact order.
func ftsMatchExpr(query string) string {
	fields := strings.Fields(query)
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = ftsQuote(f)
	}
	return strings.Join(quoted, " ")
}

// searchSQLite performs a full-text search over thread titles/bodies and
// reply bodies using SQLite FTS5, ranked by relevance (bm25 — lower is
// better, matching FTS5's own convention, so the ORDER BY needs no DESC).
// query is guaranteed non-empty by Search, which checks this before
// dispatching to either backend.
func searchSQLite(db *gorm.DB, query string, limit int) (*SearchResult, error) {
	matchExpr := ftsMatchExpr(query)

	var threads []Thread
	if err := db.Raw(`
		SELECT threads.* FROM threads
		JOIN thread_fts ON thread_fts.rowid = threads.rowid
		WHERE thread_fts MATCH ?
		ORDER BY bm25(thread_fts)
		LIMIT ?
	`, matchExpr, limit).Scan(&threads).Error; err != nil {
		return nil, fmt.Errorf("search threads: %w", err)
	}

	var replies []Reply
	if err := db.Raw(`
		SELECT replies.* FROM replies
		JOIN reply_fts ON reply_fts.rowid = replies.rowid
		WHERE reply_fts MATCH ?
		ORDER BY bm25(reply_fts)
		LIMIT ?
	`, matchExpr, limit).Scan(&replies).Error; err != nil {
		return nil, fmt.Errorf("search replies: %w", err)
	}

	return &SearchResult{Threads: threads, Replies: replies}, nil
}
