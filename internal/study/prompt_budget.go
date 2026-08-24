package study

import (
	"fmt"
	"strings"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

// scaledQuizQuestionCount dynamically scales quiz question count to content volume
// (1 question per ~400 words, clamped between 3 and 10 questions).
func scaledQuizQuestionCount(wordCount int) int {
	if wordCount <= 0 {
		return 3
	}
	count := wordCount / 400
	if count < 3 {
		return 3
	}
	if count > 10 {
		return 10
	}
	return count
}

// buildPageBoundedContext fetches structured chunk context for a notebook page range
// and returns (chunks, tokenCount, error).
// This is the canonical bounded context pipeline used by both manual and automatic flashcard generation.
func (s *StudyService) buildPageBoundedContext(notebookID string, startPage, endPage int) ([]models.ChunkWithContext, int, error) {
	utils.Warnf("[FLASHCARD_PIPELINE] buildPageBoundedContext entry notebookID=%s page_range=%d-%d", notebookID, startPage, endPage)
	chunks, err := s.repo.GetChunksWithContextByNotebookPageRange(notebookID, startPage, endPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load page-bounded context: %w", err)
	}
	utils.Warnf("[FLASHCARD_PIPELINE] buildPageBoundedContext raw_chunks=%d", len(chunks))
	if len(chunks) == 0 {
		return []models.ChunkWithContext{}, 0, nil
	}

	// Filter out front matter chunks if substantive chunks are present
	substantive := make([]models.ChunkWithContext, 0, len(chunks))
	for _, c := range chunks {
		if !isFrontMatterChunk(c.Text) {
			substantive = append(substantive, c)
		}
	}
	if len(substantive) > 0 {
		chunks = substantive
	}

	const maxContextChunks = 120
	if len(chunks) > maxContextChunks {
		chunks = chunks[:maxContextChunks]
	}

	finalTokenCount := calculatePromptTokenCount(chunks)
	utils.Warnf("[FLASHCARD_PIPELINE] buildPageBoundedContext exit chunks=%d token_count=%d", len(chunks), finalTokenCount)

	return chunks, finalTokenCount, nil
}

// calculatePromptTokenCount estimates the actual token count that will be sent to the LLM
// including prompt overhead and chunk formatting (chunk_id: | page_num: | text: format)
func calculatePromptTokenCount(chunks []models.ChunkWithContext) int {
	const maxContextChunks = 120
	limit := len(chunks)
	if limit > maxContextChunks {
		limit = maxContextChunks
	}

	baseOverhead := 200

	var contentBuilder strings.Builder
	for i := 0; i < limit; i++ {
		chunk := chunks[i]
		text := strings.TrimSpace(chunk.Text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&contentBuilder, "- chunk_id: %s | page_num: %d | text: %s\n", chunk.ChunkID, chunk.PageNum, text)
	}

	if len(chunks) > maxContextChunks {
		contentBuilder.WriteString("[...additional chunks truncated...]")
	}

	formattedContent := contentBuilder.String()

	if contentTokens, err := embeddings.CountTokens(formattedContent); err == nil {
		return baseOverhead + contentTokens
	} else {
		return baseOverhead + len(strings.Fields(formattedContent))
	}
}

func buildContextTextFromChunks(chunks []models.ChunkWithContext) string {
	var b strings.Builder
	for _, chunk := range chunks {
		text := strings.TrimSpace(chunk.Text)
		if text == "" {
			continue
		}
		b.WriteString(text)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

// CalculateAvailableContextBudget computes the exact remaining token budget for dynamic
// context chunks or text. It measures the token count of the static prompt template (instructions,
// rules, JSON schema) using the tokenizer and subtracts it from the model's MaxInputTokens.
func CalculateAvailableContextBudget(maxInputTokens int, templateText string) (int, error) {
	if maxInputTokens <= 0 {
		return 0, fmt.Errorf("invalid max input tokens: %d", maxInputTokens)
	}

	templateTokens, err := embeddings.CountTokens(templateText)
	if err != nil {
		return 0, fmt.Errorf("failed to count prompt template tokens: %w", err)
	}

	available := maxInputTokens - templateTokens
	if available <= 0 {
		return 0, fmt.Errorf("configured Max Input Tokens (%d) is too small to fit the prompt instructions (%d tokens)", maxInputTokens, templateTokens)
	}

	return available, nil
}

// BudgetChunksToLimit selects chunks that fit strictly within the given token budget using the tokenizer.
func BudgetChunksToLimit(chunks []models.ChunkWithContext, tokenBudget int) ([]models.ChunkWithContext, error) {
	if len(chunks) == 0 || tokenBudget <= 0 {
		return nil, nil
	}

	currentTokens := 0
	result := make([]models.ChunkWithContext, 0, len(chunks))

	for _, chunk := range chunks {
		text := strings.TrimSpace(chunk.Text)
		if text == "" {
			continue
		}

		chunkTokens, err := embeddings.CountTokens(text)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens for chunk %s: %w", chunk.ChunkID, err)
		}

		if len(result) > 0 && currentTokens+chunkTokens > tokenBudget {
			break
		}

		result = append(result, chunk)
		currentTokens += chunkTokens
	}

	return result, nil
}
