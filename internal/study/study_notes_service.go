package study

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

const studyNoteSystemPrompt = `You are a high-yield academic study notes assistant.
Generate a structured, concise, highly intelligible study note for the provided topic and reference material.

Requirements:
- Target length: 100 to 150 words.
- Format: Plain markdown with bold terms and concise bullet points.
- Tone: Rigorous, clear, conceptual, direct.
- Style: DO NOT use emojis. DO NOT include greetings, preamble, or meta explanations.
- Structure:
  - **Core Definition / Concept**: 1-2 sentence high-level summary of what it is and why it matters.
  - **Key Mechanisms / Principles**: 2-3 essential high-yield bullet points.
  - **Key Formula / Rule of Thumb / Takeaway**: 1 concluding key takeaway or operational rule.
`

// GenerateTopicStudyNote creates or regenerates a structured study note for a whole topic.
func (s *StudyService) GenerateTopicStudyNote(topicID, notebookID string) (*models.TopicStudyNote, error) {
	return s.GenerateTopicStudyNoteForRange(topicID, notebookID, 0, 0)
}

// GenerateTopicStudyNoteForRange creates or updates a structured 100-150 word study note for a specific reading session range
// (or the entire topic if startPage/endPage are 0). It prioritizes compressed text from LLMLingua-2.
func (s *StudyService) GenerateTopicStudyNoteForRange(topicID, notebookID string, startPage, endPage int) (*models.TopicStudyNote, error) {
	topicID = strings.TrimSpace(topicID)
	notebookID = strings.TrimSpace(notebookID)
	if topicID == "" {
		return nil, fmt.Errorf("topic ID is required")
	}

	// Resolve notebook ID if not provided
	if notebookID == "" {
		if nbID, nbErr := s.repo.GetNotebookIDForTopic(topicID); nbErr == nil && nbID != "" {
			notebookID = nbID
		}
	}

	// Fetch candidate chunks (scoped to session page range if provided)
	var chunks []models.Chunk
	var err error
	if startPage > 0 && endPage >= startPage {
		chunks, err = s.repo.GetChunksForTopicPageRange(topicID, startPage, endPage)
	} else {
		chunks, err = s.repo.GetChunksForTopic(topicID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chunks for topic %s: %w", topicID, err)
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no content chunks available for topic %s", topicID)
	}

	// Fetch topic title for prompt grounding
	topicTitle := topicID
	if t, tErr := s.repo.GetTopic(topicID); tErr == nil && t != nil && t.Title != "" {
		topicTitle = t.Title
	}

	// Sort chunks: top importance score first, then page number
	sortedChunks := make([]models.Chunk, len(chunks))
	copy(sortedChunks, chunks)
	sort.SliceStable(sortedChunks, func(i, j int) bool {
		if sortedChunks[i].ImportanceScore != sortedChunks[j].ImportanceScore {
			return sortedChunks[i].ImportanceScore > sortedChunks[j].ImportanceScore
		}
		return sortedChunks[i].PageNum < sortedChunks[j].PageNum
	})

	// Select LLM provider
	provider, tier := s.selectLLM("")
	if provider == nil {
		return nil, fmt.Errorf("no LLM provider configured")
	}

	limits := provider.GetLimits()
	maxInput := limits.MaxInputTokens
	if maxInput <= 0 {
		maxInput = 4000
	}

	// Assemble excerpt within budget (~70% of max input tokens)
	var contentBuilder strings.Builder
	currentTokens := 0
	tokenBudget := int(float64(maxInput) * 0.70)
	if tokenBudget < 500 {
		tokenBudget = 500
	}

	for _, c := range sortedChunks {
		chunkContent := c.CompressedText
		if strings.TrimSpace(chunkContent) == "" {
			chunkContent = c.Text
		}
		chunkContent = strings.TrimSpace(chunkContent)
		if chunkContent == "" {
			continue
		}

		estTokens, _ := embeddings.CountTokens(chunkContent)
		if currentTokens+estTokens > tokenBudget && currentTokens > 0 {
			break
		}

		if contentBuilder.Len() > 0 {
			contentBuilder.WriteString("\n\n")
		}
		if c.PageNum > 0 {
			fmt.Fprintf(&contentBuilder, "[Page %d]\n%s", c.PageNum, chunkContent)
		} else {
			contentBuilder.WriteString(chunkContent)
		}
		currentTokens += estTokens
	}

	referenceText := contentBuilder.String()
	if strings.TrimSpace(referenceText) == "" {
		return nil, fmt.Errorf("no substantive text content found for topic %s", topicID)
	}

	sectionContext := ""
	if startPage > 0 && endPage >= startPage {
		sectionContext = fmt.Sprintf(" (Pages %d–%d)", startPage, endPage)
	}

	prompt := fmt.Sprintf("%s\n\nTopic Title: %s%s\n\nReference Material:\n%s\n\nStructured Study Note:", studyNoteSystemPrompt, topicTitle, sectionContext, referenceText)

	answer, genErr := provider.GenerateAnswer(prompt)
	if genErr != nil {
		return nil, s.FormatLLMError(genErr, tier)
	}

	cleanedContent := strings.TrimSpace(answer)
	// Strip markdown code fences if LLM wrapped it in ```markdown ... ```
	if strings.HasPrefix(cleanedContent, "```markdown") {
		cleanedContent = strings.TrimPrefix(cleanedContent, "```markdown")
		cleanedContent = strings.TrimSuffix(cleanedContent, "```")
		cleanedContent = strings.TrimSpace(cleanedContent)
	} else if strings.HasPrefix(cleanedContent, "```") && strings.HasSuffix(cleanedContent, "```") {
		cleanedContent = strings.TrimPrefix(cleanedContent, "```")
		cleanedContent = strings.TrimSuffix(cleanedContent, "```")
		cleanedContent = strings.TrimSpace(cleanedContent)
	}

	finalContent := cleanedContent
	if startPage > 0 && endPage >= startPage {
		sectionHeader := fmt.Sprintf("### Pages %d–%d", startPage, endPage)
		sectionBlock := fmt.Sprintf("%s\n%s", sectionHeader, cleanedContent)

		existing, _ := s.repo.GetTopicStudyNote(topicID)
		if existing != nil && strings.TrimSpace(existing.Content) != "" {
			finalContent = mergeSectionIntoNote(existing.Content, sectionHeader, sectionBlock)
		} else {
			finalContent = sectionBlock
		}
	}

	note := models.TopicStudyNote{
		TopicID:        topicID,
		NotebookID:     notebookID,
		Content:        finalContent,
		LastReviewedAt: 0,
	}

	if err := s.repo.UpsertTopicStudyNote(note); err != nil {
		return nil, fmt.Errorf("failed to save topic study note: %w", err)
	}

	utils.Infof("[STUDY_NOTES] successfully generated study note for topic %s%s (tier=%s)", topicID, sectionContext, tier)
	return s.repo.GetTopicStudyNote(topicID)
}

func mergeSectionIntoNote(existingContent, sectionHeader, newSectionBlock string) string {
	existingContent = strings.TrimSpace(existingContent)
	if existingContent == "" {
		return newSectionBlock
	}

	headerIdx := strings.Index(existingContent, sectionHeader)
	if headerIdx == -1 {
		return fmt.Sprintf("%s\n\n%s", existingContent, newSectionBlock)
	}

	before := existingContent[:headerIdx]
	afterSection := existingContent[headerIdx+len(sectionHeader):]
	nextHeaderIdx := strings.Index(afterSection, "\n### ")
	if nextHeaderIdx != -1 {
		rest := afterSection[nextHeaderIdx:]
		return strings.TrimSpace(fmt.Sprintf("%s%s\n%s", before, newSectionBlock, rest))
	}

	return strings.TrimSpace(fmt.Sprintf("%s%s", before, newSectionBlock))
}

// GenerateTopicStudyNoteAsync triggers asynchronous study note generation in the background.
func (s *StudyService) GenerateTopicStudyNoteAsync(ctx context.Context, topicID, notebookID string, startPage, endPage int) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Errorf("[STUDY_NOTES] panic in GenerateTopicStudyNoteAsync: %v", r)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		select {
		case <-bgCtx.Done():
			return
		default:
		}

		if _, err := s.GenerateTopicStudyNoteForRange(topicID, notebookID, startPage, endPage); err != nil {
			utils.Warnf("[STUDY_NOTES] async note generation failed for topic %s (pages %d-%d): %v", topicID, startPage, endPage, err)
		}
	}()
}

// GetTopicStudyNote retrieves the study note for a given topic ID.
func (s *StudyService) GetTopicStudyNote(topicID string) (*models.TopicStudyNote, error) {
	return s.repo.GetTopicStudyNote(topicID)
}

// UpdateTopicStudyNote updates the markdown text of a topic study note.
func (s *StudyService) UpdateTopicStudyNote(topicID, content string) error {
	return s.repo.UpdateTopicStudyNoteContent(topicID, content)
}

// GetNotesByNotebook returns all study notes for topics belonging to a notebook.
func (s *StudyService) GetNotesByNotebook(notebookID string) ([]models.TopicStudyNote, error) {
	return s.repo.GetNotesByNotebook(notebookID)
}

// MarkTopicReviewed updates the last reviewed timestamp for a topic's study note.
func (s *StudyService) MarkTopicReviewed(topicID string) error {
	return s.repo.MarkTopicStudyNoteReviewed(topicID)
}
