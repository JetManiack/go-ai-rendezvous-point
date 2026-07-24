package storage

import (
	"time"

	"gorm.io/gorm"
)

type CatchUpResult struct {
	UnreadReplies []Reply
	NewMentions   []Mention
}

// CatchUp returns everything new for actorID since their last call: unread
// replies in every thread they watch, and mentions of them not yet seen.
// Calling it marks the returned items as read/seen, so a second call with
// no new activity returns an empty result.
func CatchUp(db *gorm.DB, actorID string) (*CatchUpResult, error) {
	result := &CatchUpResult{}

	err := db.Transaction(func(tx *gorm.DB) error {
		var watches []ThreadWatch
		if err := tx.Where("actor_id = ?", actorID).Find(&watches).Error; err != nil {
			return err
		}

		for _, w := range watches {
			var replies []Reply
			if err := tx.Where("thread_id = ? AND created_at > ?", w.ThreadID, w.LastReadAt).
				Order("created_at asc").Find(&replies).Error; err != nil {
				return err
			}
			if len(replies) == 0 {
				continue
			}
			result.UnreadReplies = append(result.UnreadReplies, replies...)

			latest := replies[len(replies)-1].CreatedAt
			if err := tx.Model(&ThreadWatch{}).
				Where("actor_id = ? AND thread_id = ?", actorID, w.ThreadID).
				Updates(map[string]interface{}{"last_read_at": latest, "updated_at": time.Now()}).Error; err != nil {
				return err
			}
		}

		var mentions []Mention
		if err := tx.Where("mentioned_actor_id = ? AND seen_at IS NULL", actorID).Find(&mentions).Error; err != nil {
			return err
		}
		result.NewMentions = mentions

		if len(mentions) > 0 {
			now := time.Now()
			if err := tx.Model(&Mention{}).
				Where("mentioned_actor_id = ? AND seen_at IS NULL", actorID).
				Update("seen_at", now).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
