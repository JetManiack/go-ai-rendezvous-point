package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// ResolveThread marks threadID as resolved.
func ResolveThread(db *gorm.DB, threadID string) (*Thread, error) {
	return setThreadStatus(db, threadID, "resolved")
}

// ReopenThread marks threadID as open again.
func ReopenThread(db *gorm.DB, threadID string) (*Thread, error) {
	return setThreadStatus(db, threadID, "open")
}

func setThreadStatus(db *gorm.DB, threadID, status string) (*Thread, error) {
	var thread Thread
	if err := db.First(&thread, "id = ?", threadID).Error; err != nil {
		return nil, fmt.Errorf("thread %q not found: %w", threadID, err)
	}

	if err := db.Model(&thread).Update("status", status).Error; err != nil {
		return nil, err
	}
	thread.Status = status
	return &thread, nil
}
