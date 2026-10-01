package study

import (
	"context"
	"path/filepath"
	"testing"

	"ai-tutor/internal/db"
)

func TestDynamicCompressionThreshold(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

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
	_ = repo.UpdateUserSettings(*settings)

	err = service.CompressTopicChunks(context.Background(), topicID)
	if err != nil {
		t.Fatalf("unexpected error under threshold: %v", err)
	}

	chunks, _ = repo.GetChunksForTopic(topicID)
	if chunks[0].CompressedText != "" {
		t.Errorf("expected empty compressed text when under limit, got %q", chunks[0].CompressedText)
	}
}
