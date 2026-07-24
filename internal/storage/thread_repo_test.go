package storage_test

import (
	"errors"
	"testing"

	"go-ai-rendezvous-point/internal/storage"
)

func TestCreateThread_AddsAuthorAsWatcher(t *testing.T) {
	db := openTestDB(t)
	author, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	thread, err := storage.CreateThread(db, author.ID, "New feature X", "Shipped feature X, see docs.", nil)
	if err != nil {
		t.Fatalf("CreateThread() error = %v", err)
	}
	if thread.Status != "open" {
		t.Errorf("Status = %q, want %q", thread.Status, "open")
	}
	if thread.Title != "New feature X" {
		t.Errorf("Title = %q, want %q", thread.Title, "New feature X")
	}

	var watcher storage.Watcher
	if err := db.First(&watcher, "thread_id = ? AND actor_id = ?", thread.ID, author.ID).Error; err != nil {
		t.Fatalf("expected author to be a watcher: %v", err)
	}

	var watch storage.ThreadWatch
	if err := db.First(&watch, "thread_id = ? AND actor_id = ?", thread.ID, author.ID).Error; err != nil {
		t.Fatalf("expected a ThreadWatch row for the author: %v", err)
	}
	if watch.LastReadAt.IsZero() {
		t.Error("LastReadAt was not set")
	}
}

func TestCreateThread_RejectsEmptyOrWhitespaceOnlyTitleAndBody(t *testing.T) {
	db := openTestDB(t)
	author, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	cases := []struct {
		name    string
		title   string
		body    string
		wantErr error
	}{
		{"empty title", "", "a body", storage.ErrEmptyTitle},
		{"whitespace-only title", "   ", "a body", storage.ErrEmptyTitle},
		{"empty body", "a title", "", storage.ErrEmptyBody},
		{"whitespace-only body", "a title", "  \t\n", storage.ErrEmptyBody},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := storage.CreateThread(db, author.ID, tc.title, tc.body, nil)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CreateThread() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateThread_AttachesTags(t *testing.T) {
	db := openTestDB(t)
	author, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	thread, err := storage.CreateThread(db, author.ID, "New feature X", "Shipped feature X.", []string{"feature", "deploy"})
	if err != nil {
		t.Fatalf("CreateThread() error = %v", err)
	}

	var tagCount int64
	db.Model(&storage.Tag{}).Where("name IN ?", []string{"feature", "deploy"}).Count(&tagCount)
	if tagCount != 2 {
		t.Fatalf("tag count = %d, want 2", tagCount)
	}

	var joinCount int64
	db.Model(&storage.ThreadTag{}).Where("thread_id = ?", thread.ID).Count(&joinCount)
	if joinCount != 2 {
		t.Fatalf("thread_tag join count = %d, want 2", joinCount)
	}
}

func TestCreateThread_ReusesExistingTagAcrossThreads(t *testing.T) {
	db := openTestDB(t)
	author, err := storage.CreateAgent(db, "agent-a")
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	thread1, err := storage.CreateThread(db, author.ID, "Thread 1", "Body 1", []string{"shared-tag"})
	if err != nil {
		t.Fatalf("CreateThread(1) error = %v", err)
	}
	thread2, err := storage.CreateThread(db, author.ID, "Thread 2", "Body 2", []string{"shared-tag"})
	if err != nil {
		t.Fatalf("CreateThread(2) error = %v", err)
	}

	var tagCount int64
	db.Model(&storage.Tag{}).Where("name = ?", "shared-tag").Count(&tagCount)
	if tagCount != 1 {
		t.Fatalf("tag count = %d, want exactly 1 shared tag row (not duplicated)", tagCount)
	}

	var joinCount int64
	db.Model(&storage.ThreadTag{}).Where("thread_id IN ?", []string{thread1.ID, thread2.ID}).Count(&joinCount)
	if joinCount != 2 {
		t.Fatalf("thread_tag join count = %d, want 2 (one per thread)", joinCount)
	}
}
