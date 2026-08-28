package study

import (
	"strings"
	"testing"

	"ai-tutor/internal/models"
)

func TestBuildMarathonFlashcardPromptWithBudget_IncludesChunks(t *testing.T) {
	chunks := []models.ChunkWithContext{
		{
			ChunkID: "chunk_001",
			PageNum: 15,
			Text:    "Neural networks consist of input, hidden, and output layers.",
		},
		{
			ChunkID: "chunk_002",
			PageNum: 16,
			Text:    "Gradient descent minimizes the loss function by updating weights.",
		},
	}

	prompt, currentTokens, includedIDs := buildMarathonFlashcardPromptWithBudget("Deep Learning", 15, 20, chunks, 5, 8000, nil)

	if len(includedIDs) != 2 {
		t.Errorf("expected 2 included chunk IDs, got %d", len(includedIDs))
	}
	if currentTokens <= 0 {
		t.Errorf("expected positive token count, got %d", currentTokens)
	}
	if !strings.Contains(prompt, "=== SOURCE CHUNKS ===") {
		t.Errorf("prompt missing === SOURCE CHUNKS === header")
	}
	if !strings.Contains(prompt, "page_num: 15") || !strings.Contains(prompt, "page_num: 16") {
		t.Errorf("prompt expected to contain page_num: 15 and page_num: 16, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "chunk_id:") {
		t.Errorf("prompt should not contain chunk_id, got:\n%s", prompt)
	}
}

func TestBuildMarathonFlashcardPromptWithBudget_EmptyChunks(t *testing.T) {
	prompt, _, includedIDs := buildMarathonFlashcardPromptWithBudget("Empty Book", 1, 5, nil, 5, 8000, nil)

	if len(includedIDs) != 0 {
		t.Errorf("expected 0 included chunk IDs for nil chunks, got %d", len(includedIDs))
	}
	if !strings.Contains(prompt, "=== SOURCE CHUNKS ===") {
		t.Errorf("prompt missing === SOURCE CHUNKS === header")
	}
	// Verify that no chunk lines are rendered when context is empty
	if strings.Contains(prompt, "- page_num:") || strings.Contains(prompt, "chunk_id:") {
		t.Errorf("expected no chunk lines for empty context, got:\n%s", prompt)
	}
}
