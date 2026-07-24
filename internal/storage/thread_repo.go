package storage

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrEmptyTitle = errors.New("title must not be empty")
	ErrEmptyBody  = errors.New("body must not be empty")
)

func CreateThread(db *gorm.DB, authorID, title, body string, tags []string) (*Thread, error) {
	if strings.TrimSpace(title) == "" {
		return nil, ErrEmptyTitle
	}
	if strings.TrimSpace(body) == "" {
		return nil, ErrEmptyBody
	}

	thread := &Thread{
		ID:       uuid.NewString(),
		Title:    title,
		Body:     body,
		Status:   "open",
		AuthorID: authorID,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(thread).Error; err != nil {
			return err
		}
		if err := tx.Create(&Watcher{ThreadID: thread.ID, ActorID: authorID}).Error; err != nil {
			return err
		}
		if err := tx.Create(&ThreadWatch{
			ActorID:    authorID,
			ThreadID:   thread.ID,
			LastReadAt: thread.CreatedAt,
		}).Error; err != nil {
			return err
		}
		return attachTags(tx, thread.ID, tags)
	})
	if err != nil {
		return nil, err
	}
	return thread, nil
}

// attachTags finds-or-creates each named tag and links it to threadID.
// Reusing an existing tag by name (rather than always inserting) keeps the
// Tag table free of duplicates when multiple threads share a tag.
func attachTags(tx *gorm.DB, threadID string, tagNames []string) error {
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&Tag{ID: uuid.NewString(), Name: name}).Error; err != nil {
			return err
		}

		var tag Tag
		if err := tx.Where("name = ?", name).First(&tag).Error; err != nil {
			return err
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&ThreadTag{ThreadID: threadID, TagID: tag.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

var mentionPattern = regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)

func AddReply(db *gorm.DB, threadID, authorID, body string, extraWatcherIDs []string) (*Reply, error) {
	if strings.TrimSpace(body) == "" {
		return nil, ErrEmptyBody
	}

	reply := &Reply{
		ID:       uuid.NewString(),
		ThreadID: threadID,
		Body:     body,
		AuthorID: authorID,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var thread Thread
		if err := tx.First(&thread, "id = ?", threadID).Error; err != nil {
			return fmt.Errorf("thread %q not found: %w", threadID, err)
		}

		if err := tx.Create(reply).Error; err != nil {
			return err
		}

		watcherIDs := append([]string{authorID}, extraWatcherIDs...)
		for _, watcherID := range watcherIDs {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&Watcher{ThreadID: threadID, ActorID: watcherID}).Error; err != nil {
				return err
			}
		}

		if err := upsertThreadWatch(tx, authorID, threadID, reply.CreatedAt); err != nil {
			return err
		}

		// Extra watchers (excluding the author, handled above with the
		// reply's own timestamp) get a ThreadWatch row only if they don't
		// already have one, so the whole thread history shows up as unread
		// for someone newly added as a watcher, without resetting the read
		// cursor of someone already watching.
		for _, watcherID := range extraWatcherIDs {
			if watcherID == authorID {
				continue
			}
			if err := ensureThreadWatch(tx, watcherID, threadID); err != nil {
				return err
			}
		}

		return createMentions(tx, &reply.ID, nil, authorID, body)
	})
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func upsertThreadWatch(tx *gorm.DB, actorID, threadID string, readAt time.Time) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "actor_id"}, {Name: "thread_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_read_at", "updated_at"}),
	}).Create(&ThreadWatch{
		ActorID:    actorID,
		ThreadID:   threadID,
		LastReadAt: readAt,
	}).Error
}

// ensureThreadWatch creates a ThreadWatch row for actorID/threadID if one
// doesn't already exist, leaving LastReadAt as the zero time.Time so the
// entire thread history is treated as unread. It does nothing if a row
// already exists, so it never resets an existing read cursor.
func ensureThreadWatch(tx *gorm.DB, actorID, threadID string) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ThreadWatch{
		ActorID:  actorID,
		ThreadID: threadID,
		// LastReadAt left as zero time.Time: a newly-added watcher hasn't
		// read anything in this thread yet, so everything is unread.
	}).Error
}

func createMentions(tx *gorm.DB, replyID, threadID *string, authorID, body string) error {
	matches := mentionPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := map[string]bool{}
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true

		var actor Actor
		if err := tx.Where("display_name = ?", name).First(&actor).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue // unresolvable @name in free-form body text; not an error
			}
			return err
		}
		if actor.ID == authorID {
			continue // don't notify yourself
		}

		mention := &Mention{
			ID:               uuid.NewString(),
			ReplyID:          replyID,
			ThreadID:         threadID,
			MentionedActorID: actor.ID,
		}
		if err := tx.Create(mention).Error; err != nil {
			return err
		}
	}
	return nil
}
