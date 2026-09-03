package retrieval

import (
	"math"
	"testing"

	"ai-tutor/internal/models"
)

func TestHybridRRFScoring(t *testing.T) {
	chunks := []models.Chunk{
		{ID: "c1", Text: "Go goroutines and channels concurrency"},
		{ID: "c2", Text: "Memory management and garbage collector"},
		{ID: "c3", Text: "Python dynamic typing interpreter"},
	}

	engine := NewEngine(nil, nil)
	for _, c := range chunks {
		engine.AddChunk(c)
	}

	// Case 1: Pure lexical when embedder is nil
	res, err := engine.searchWithScope(
		"test scope",
		"goroutines concurrency",
		2,
		func() ([]models.Chunk, error) { return chunks, nil },
		func([]float32, int) ([]string, error) { return nil, nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected results from lexical match")
	}
	if res[0].ChunkID != "c1" {
		t.Fatalf("expected chunk c1 to rank first for lexical match, got %s", res[0].ChunkID)
	}

	// Verify reciprocal rank formula: 1 / (60 + 1) ~= 0.016393
	expectedScore := 1.0 / (60.0 + 1.0)
	if math.Abs(res[0].Score-expectedScore) > 1e-4 {
		t.Fatalf("expected RRF score close to %f, got %f", expectedScore, res[0].Score)
	}
}

func TestHybridRRFCompounding(t *testing.T) {
	chunks := []models.Chunk{
		{ID: "c1", Text: "Go goroutines and channels concurrency"},
		{ID: "c2", Text: "Go concurrency patterns in production"},
	}

	engine := NewEngine(nil, nil)
	for _, c := range chunks {
		engine.AddChunk(c)
	}

	// Test with query matching both chunks lexically, but c1 has higher lexical relevance (score)
	res, err := engine.searchWithScope(
		"test scope",
		"concurrency",
		2,
		func() ([]models.Chunk, error) { return chunks, nil },
		func([]float32, int) ([]string, error) { return nil, nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
}
