package study

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"ai-tutor/internal/db"
)

func TestDynamicCompressionThreshold(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if err := repo.SeedDemoDataForTests(); err != nil {
		t.Fatalf("failed to seed demo data: %v", err)
	}

	service := NewStudyService(Config{Repo: repo})

	topicID := "os-scheduling"

	// Case 1: Mode = DISABLED -> skips compression
	settings, err := repo.GetUserSettings()
	if err != nil {
		t.Fatalf("failed to get settings: %v", err)
	}
	settings.PromptCompressionMode = "DISABLED"
	settings.TargetSessionWords = 10 // small threshold, but disabled
	if err := repo.UpdateUserSettings(*settings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	err = service.CompressTopicChunks(context.Background(), topicID)
	if err != nil {
		t.Fatalf("unexpected error when disabled: %v", err)
	}

	chunks, err := repo.GetChunksForTopic(topicID)
	if err != nil {
		t.Fatalf("failed to get chunks: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatalf("expected seeded chunks, got none")
	}
	if chunks[0].CompressedText != "" {
		t.Errorf("expected empty compressed text when disabled, got %q", chunks[0].CompressedText)
	}

	// Case 2: Mode = OVER_LIMIT, but tokens (small seeded text) <= Dynamic Budget (3000 * 1.33 = ~3990) -> skips
	settings.PromptCompressionMode = "OVER_LIMIT"
	settings.TargetSessionWords = 3000
	if err := repo.UpdateUserSettings(*settings); err != nil {
		t.Fatalf("failed to save settings for Case 2: %v", err)
	}

	err = service.CompressTopicChunks(context.Background(), topicID)
	if err != nil {
		t.Fatalf("unexpected error under threshold: %v", err)
	}

	chunks, _ = repo.GetChunksForTopic(topicID)
	if chunks[0].CompressedText != "" {
		t.Errorf("expected empty compressed text when under limit, got %q", chunks[0].CompressedText)
	}
}

func TestCompressTopicChunksPrioritizingRange(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if err := repo.SeedDemoDataForTests(); err != nil {
		t.Fatalf("failed to seed demo data: %v", err)
	}

	service := NewStudyService(Config{Repo: repo})
	topicID := "os-scheduling"

	// When disabled, onRangeComplete and onAllComplete should be invoked immediately
	settings, _ := repo.GetUserSettings()
	settings.PromptCompressionMode = "DISABLED"
	_ = repo.UpdateUserSettings(*settings)

	rangeDone := false
	if err := service.CompressTopicChunksWithPriorityRange(context.Background(), topicID, 1, 5, func() {
		rangeDone = true
	}); err != nil {
		t.Fatalf("unexpected error when compression is disabled: %v", err)
	}

	if !rangeDone {
		t.Errorf("expected onRangeComplete to be invoked when compression is disabled")
	}

	// Test async method
	rangeAsyncDone := make(chan bool, 1)
	allAsyncDone := make(chan bool, 1)
	service.CompressTopicChunksPrioritizingRangeAsync(context.Background(), topicID, 1, 5, func() {
		rangeAsyncDone <- true
	}, func() {
		allAsyncDone <- true
	})

	select {
	case <-rangeAsyncDone:
		// success
	case <-time.After(3 * time.Second):
		t.Errorf("timed out waiting for range callback")
	}

	select {
	case <-allAsyncDone:
		// success
	case <-time.After(3 * time.Second):
		t.Errorf("timed out waiting for all complete callback")
	}
}

