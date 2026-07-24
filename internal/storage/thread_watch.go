package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WatchThread subscribes actorID to threadID: it ensures both a Watcher
// row and a ThreadWatch row exist, reusing ensureThreadWatch (defined in
// thread_repo.go) so a newly-added watcher sees the thread's full history
// as unread on their next catch_up, exactly like an extra watcher added
// via reply's Watchers parameter.
func WatchThread(db *gorm.DB, actorID, threadID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var thread Thread
		if err := tx.First(&thread, "id = ?", threadID).Error; err != nil {
			return fmt.Errorf("thread %q not found: %w", threadID, err)
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&Watcher{ThreadID: threadID, ActorID: actorID}).Error; err != nil {
			return err
		}

		return ensureThreadWatch(tx, actorID, threadID)
	})
}

// UnwatchThread unsubscribes actorID from threadID. It deletes both the
// Watcher row and the ThreadWatch row — deleting only Watcher would leave
// catch_up (which reads ThreadWatch, not Watcher) still surfacing this
// thread's future replies, defeating the point of unwatching. Unwatching
// a thread the actor doesn't currently watch is a no-op, not an error.
func UnwatchThread(db *gorm.DB, actorID, threadID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("thread_id = ? AND actor_id = ?", threadID, actorID).
			Delete(&Watcher{}).Error; err != nil {
			return err
		}
		return tx.Where("thread_id = ? AND actor_id = ?", threadID, actorID).
			Delete(&ThreadWatch{}).Error
	})
}
