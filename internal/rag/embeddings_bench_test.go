package rag

import (
	"fmt"
	"path/filepath"
	"testing"

	"ai-tutor/internal/db"
)

func initBenchDB(b *testing.B) {
	tempDB := filepath.Join(b.TempDir(), "rag-bench.db")
	if err := db.Init(tempDB, ""); err != nil {
		b.Fatalf("failed to init rag bench db: %v", err)
	}

	// Seed more data for benchmark
	topicID := "bench-topic"
	_, err := db.GetConnection().Exec(`INSERT INTO topics (id, title, status) VALUES (?, ?, ?)`, topicID, "Bench Topic", "reading")
	if err != nil {
		b.Fatalf("failed to insert topic: %v", err)
	}

	for i := 0; i < 100; i++ {
		parentID := fmt.Sprintf("parent-%d", i)
		_, err = db.GetConnection().Exec(`
			INSERT INTO parents (id, topic_id, heading, order_index, content_text)
			VALUES (?, ?, ?, ?, ?)
		`, parentID, topicID, fmt.Sprintf("Heading %d", i), i, fmt.Sprintf("Content for parent %d", i))
		if err != nil {
			b.Fatalf("failed to insert parent %d: %v", i)
		}
	}
}

func BenchmarkBuildContext_N1(b *testing.B) {
	initBenchDB(b)
	defer db.Close()

	results := make([]RetrievalResult, 50)
	for i := 0; i < 50; i++ {
		results[i] = RetrievalResult{
			ParentID: fmt.Sprintf("parent-%d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := BuildContext(results, "bench-topic")
		if err != nil {
			b.Fatal(err)
		}
	}
}
