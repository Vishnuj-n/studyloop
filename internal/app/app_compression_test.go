package app

import (
	"path/filepath"
	"testing"

	"ai-tutor/internal/db"
)

func TestGetTopicCompressionStats_Calculation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_compression.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if err := repo.SeedDemoDataForTests(); err != nil {
		t.Fatalf("failed to seed demo data: %v", err)
	}

	app := &App{
		repo: repo,
	}

	topicID := "os-scheduling"
	chunks, err := repo.GetChunksForTopic(topicID)
	if err != nil {
		t.Fatalf("failed to get chunks: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatalf("expected chunks for topic %s", topicID)
	}

	// 1. When chunks are uncompressed
	stats := app.GetTopicCompressionStats(topicID, 0, 0)
	if stats["error"] != nil {
		t.Fatalf("unexpected error: %v", stats["error"])
	}
	if stats["is_compressed"] != false {
		t.Errorf("expected is_compressed to be false, got: %v", stats["is_compressed"])
	}
	if stats["tokens_saved"].(int) != 0 {
		t.Errorf("expected 0 tokens saved, got: %v", stats["tokens_saved"])
	}

	// 2. Simulate compression on the first chunk
	chunkID := chunks[0].ID
	compText := "RR is preemptive algorithm: process assigned fixed time quantum."

	err = repo.UpdateChunkCompressedText(chunkID, compText, 10)
	if err != nil {
		t.Fatalf("failed to update test chunk: %v", err)
	}

	stats = app.GetTopicCompressionStats(topicID, 0, 0)
	if stats["is_compressed"] != true {
		t.Errorf("expected is_compressed to be true, got: %v", stats["is_compressed"])
	}
	rawTokens := stats["raw_tokens"].(int)
	compTokens := stats["compressed_tokens"].(int)
	savedTokens := stats["tokens_saved"].(int)
	savedPct := stats["saved_percentage"].(float64)

	if rawTokens <= compTokens {
		t.Fatalf("expected rawTokens (%d) > compTokens (%d)", rawTokens, compTokens)
	}
	if savedTokens <= 0 {
		t.Fatalf("expected positive savedTokens, got: %d", savedTokens)
	}
	if savedPct <= 0.0 {
		t.Fatalf("expected positive saved_percentage, got: %f", savedPct)
	}
}
