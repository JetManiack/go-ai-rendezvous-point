package storage

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type SearchResult struct {
	Threads []Thread `json:"threads"`
	Replies []Reply  `json:"replies"`
}

// Search performs a full-text search over thread titles/bodies and reply
// bodies, ranked by relevance, using whichever backend-specific full-text
// mechanism the connected database supports (SQLite FTS5 or Postgres
// tsvector — see search_sqlite.go / search_postgres.go).
func Search(db *gorm.DB, query string, limit int) (*SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if len(strings.Fields(query)) == 0 {
		return &SearchResult{}, nil
	}

	var result *SearchResult
	var err error
	if db.Dialector.Name() == "postgres" {
		result, err = searchPostgres(db, query, limit)
	} else {
		result, err = searchSQLite(db, query, limit)
	}
	if err != nil {
		return nil, err
	}

	// Tag names aren't indexed by either backend's full-text mechanism
	// (both only cover title/body), so a query word that names a tag
	// exactly (case-insensitively) would otherwise never surface threads
	// that only match via their tag — e.g. searching "ops" finds nothing
	// if no thread's title/body happens to contain the word "ops". Any
	// query word matching a tag name pulls in that tag's threads as
	// additional results (OR'd in, not required), deduplicated against
	// the full-text matches above and appended after them (full-text
	// relevance ranking doesn't apply to tag matches, so they sort by
	// recency instead).
	if len(result.Threads) < limit {
		tagThreads, err := matchThreadsByTagName(db, query, limit-len(result.Threads), result.Threads)
		if err != nil {
			return nil, fmt.Errorf("search threads by tag: %w", err)
		}
		result.Threads = append(result.Threads, tagThreads...)
	}

	return result, nil
}

// matchThreadsByTagName finds threads tagged with any word from query
// (case-insensitive exact match on tag name), excluding threads already
// present in exclude, newest first, capped at limit. Backend-agnostic —
// plain GORM Joins/Where, works identically on SQLite and Postgres.
func matchThreadsByTagName(db *gorm.DB, query string, limit int, exclude []Thread) ([]Thread, error) {
	words := strings.Fields(query)
	if len(words) == 0 || limit <= 0 {
		return nil, nil
	}
	lowerWords := make([]string, len(words))
	for i, w := range words {
		lowerWords[i] = strings.ToLower(w)
	}

	excludeIDs := make([]string, len(exclude))
	for i, t := range exclude {
		excludeIDs[i] = t.ID
	}

	var threads []Thread
	q := db.Table("threads").
		Joins("JOIN thread_tags ON thread_tags.thread_id = threads.id").
		Joins("JOIN tags ON tags.id = thread_tags.tag_id").
		Where("LOWER(tags.name) IN ?", lowerWords).
		Group("threads.id").
		Order("threads.created_at desc").
		Limit(limit)
	if len(excludeIDs) > 0 {
		q = q.Where("threads.id NOT IN ?", excludeIDs)
	}
	if err := q.Find(&threads).Error; err != nil {
		return nil, err
	}
	return threads, nil
}
