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

func TestBuildMarathonFlashcardPromptWithBudget_MitigatesMisconceptionBias(t *testing.T) {
	chunks := []models.ChunkWithContext{
		{ChunkID: "c1", PageNum: 1, Text: "Matrix operations definition."},
		{ChunkID: "c2", PageNum: 2, Text: "Eigenvectors and eigenvalues properties."},
		{ChunkID: "c3", PageNum: 3, Text: "Singular Value Decomposition applications."},
	}

	failed := []models.FailedQuestionDetail{
		{
			Prompt:        "What is an Eigenvector?",
			CorrectAnswer: "A vector whose direction does not change under linear transform.",
			UserAnswer:    "A zero vector with constant magnitude.",
		},
	}

	prompt, _, includedIDs := buildMarathonFlashcardPromptWithBudget("Linear Algebra", 1, 3, chunks, 5, 8000, failed)

	if len(includedIDs) != 3 {
		t.Errorf("expected 3 included chunk IDs, got %d", len(includedIDs))
	}

	// Verify misconception section header
	if !strings.Contains(prompt, "=== TARGETED REVIEW: TOPICS NEEDING REINFORCEMENT ===") {
		t.Errorf("prompt missing topics needing reinforcement section")
	}

	// Verify tested concept and correct answer are included
	if !strings.Contains(prompt, "What is an Eigenvector?") || !strings.Contains(prompt, "A vector whose direction does not change under linear transform.") {
		t.Errorf("prompt missing correct concept or ground truth")
	}

	// Verify false distractor (user answer) is omitted to avoid negative prompting
	if strings.Contains(prompt, "User's wrong selection:") || strings.Contains(prompt, "A zero vector with constant magnitude.") {
		t.Errorf("prompt contains distractor or user wrong selection, which introduces negative priming")
	}

	// Verify balanced coverage instructions
	if !strings.Contains(prompt, "Ensure balanced concept coverage evenly distributed") {
		t.Errorf("prompt missing balanced concept coverage rule")
	}
	if !strings.Contains(prompt, "Generate up to 6 flashcards total: 5 balanced coverage cards across pages 1-3, plus 1 targeted reinforcement card(s)") {
		t.Errorf("prompt missing explicit quota breakdown instruction, got:\n%s", prompt)
	}
}

