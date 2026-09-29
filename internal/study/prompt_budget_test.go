package study

import (
	"strings"
	"testing"
)

func TestBudgetContextBlocks(t *testing.T) {
	blocks := []string{
		"Block 1: Introduction to algorithms and data structures.",
		"Block 2: Sorting algorithms including quicksort, mergesort, and heapsort.",
		"Block 3: Graph search algorithms including BFS and DFS in detail.",
	}
	citations := []string{
		"Page 1",
		"Page 2",
		"Page 3",
	}

	t.Run("Empty input", func(t *testing.T) {
		resBlocks, resCites, err := BudgetContextBlocks(nil, nil, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resBlocks) != 0 || len(resCites) != 0 {
			t.Fatalf("expected empty results, got blocks=%d cites=%d", len(resBlocks), len(resCites))
		}
	})

	t.Run("Zero budget", func(t *testing.T) {
		resBlocks, resCites, err := BudgetContextBlocks(blocks, citations, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resBlocks) != 0 || len(resCites) != 0 {
			t.Fatalf("expected empty results, got blocks=%d cites=%d", len(resBlocks), len(resCites))
		}
	})

	t.Run("Full fit within generous budget", func(t *testing.T) {
		resBlocks, resCites, err := BudgetContextBlocks(blocks, citations, 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resBlocks) != 3 || len(resCites) != 3 {
			t.Fatalf("expected 3 blocks and 3 cites, got %d and %d", len(resBlocks), len(resCites))
		}
		for i := range blocks {
			if resBlocks[i] != blocks[i] {
				t.Errorf("block %d mismatch: got %q, want %q", i, resBlocks[i], blocks[i])
			}
			if resCites[i] != citations[i] {
				t.Errorf("citation %d mismatch: got %q, want %q", i, resCites[i], citations[i])
			}
		}
	})

	t.Run("Truncates boundary block with partial budget", func(t *testing.T) {
		// A tight budget that fits block 1 completely, but only part of block 2
		resBlocks, resCites, err := BudgetContextBlocks(blocks, citations, 22)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resBlocks) < 1 {
			t.Fatalf("expected at least 1 block, got %d", len(resBlocks))
		}
		if len(resBlocks) != len(resCites) {
			t.Fatalf("mismatched lengths: blocks=%d cites=%d", len(resBlocks), len(resCites))
		}
		if resBlocks[0] != blocks[0] {
			t.Errorf("first block should be intact, got %q", resBlocks[0])
		}
		if len(resBlocks) > 1 {
			// Second block should be truncated (e.g., tokenizer decodes subwords with spacing differences)
			truncatedWords := strings.Fields(strings.ToLower(strings.TrimSuffix(resBlocks[1], "...")))
			originalLower := strings.ToLower(blocks[1])
			for _, w := range truncatedWords {
				if w == ":" || w == "." || w == "," {
					continue
				}
				if !strings.Contains(originalLower, w) {
					t.Errorf("word %q from truncated block not found in original %q", w, blocks[1])
				}
			}
		}
	})

	t.Run("Minimal budget fallback keeps first block truncated", func(t *testing.T) {
		resBlocks, resCites, err := BudgetContextBlocks(blocks, citations, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resBlocks) != 1 || len(resCites) != 1 {
			t.Fatalf("expected 1 fallback block and cite, got blocks=%d cites=%d", len(resBlocks), len(resCites))
		}
		if resCites[0] != citations[0] {
			t.Errorf("expected citation %q, got %q", citations[0], resCites[0])
		}
	})
}
