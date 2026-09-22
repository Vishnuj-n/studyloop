package study

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

// GenerateManualFlashcards generates flashcards for a synthetic topic based on a page range (manual sandbox)
func (s *StudyService) GenerateManualFlashcards(notebookID string, startPage, endPage int) map[string]interface{} {
	cards, tier, err := s.generateFlashcardsCore(notebookID, startPage, endPage, nil)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	syntheticTopicID := fmt.Sprintf("marathon-%s-p%d-%d", notebookID, startPage, endPage)
	for i := range cards {
		cards[i].TopicID = syntheticTopicID
	}

	err = s.repo.SaveManualFlashcardsBatch(notebookID, cards)
	if err != nil {
		utils.Warnf("[FLASHCARD_PIPELINE] manual_flashcard_persistence result=error notebookID=%s err=%v", notebookID, err)
		return map[string]interface{}{"error": "failed to persist manual flashcards: " + err.Error()}
	}
	utils.Warnf("[FLASHCARD_PIPELINE] manual_flashcard_persistence result=ok notebookID=%s cardCount=%d", notebookID, len(cards))

	now := time.Now().Unix()
	return map[string]interface{}{
		"notebook_id":       notebookID,
		"existing":          false,
		"start_page":        startPage,
		"end_page":          endPage,
		"topic_id":          syntheticTopicID,
		"cards":             cards,
		"states":            map[string]models.FlashcardState{},
		"card_count":        len(cards),
		"llm_tier":          tier,
		"generated_at_unix": now,
	}
}

// GenerateFSRSCardsForTopic generates and persists FSRS flashcards in the core fsrs_cards table for a topic.
func (s *StudyService) GenerateFSRSCardsForTopic(topicID, notebookID string, startPage, endPage int) ([]models.Flashcard, map[string]models.FlashcardState, bool, string, error) {
	topicID = strings.TrimSpace(topicID)
	notebookID = strings.TrimSpace(notebookID)
	if topicID == "" || notebookID == "" {
		return nil, nil, false, "", fmt.Errorf("topic ID and notebook ID are required")
	}

	// ponytail: lookup latest quiz attempt and extract failed questions
	var failedQuestions []models.FailedQuestionDetail
	if payloadJSON, answersJSON, err := s.repo.GetLatestQuizAttemptDetailsByTopic(topicID); err == nil && payloadJSON != "" && answersJSON != "" {
		var payload models.QuizTaskPayload
		var answers []models.QuizAnswer
		if json.Unmarshal([]byte(payloadJSON), &payload) == nil && json.Unmarshal([]byte(answersJSON), &answers) == nil {
			selectedByQuestionID := make(map[string]string)
			for _, ans := range answers {
				selectedByQuestionID[ans.QuestionID] = strings.TrimSpace(ans.Selected)
			}
			for _, q := range payload.Questions {
				userAns := selectedByQuestionID[q.ID]
				if !strings.EqualFold(strings.TrimSpace(q.CorrectAnswer), userAns) {
					failedQuestions = append(failedQuestions, models.FailedQuestionDetail{
						Prompt:        q.Prompt,
						Options:       q.Options,
						CorrectAnswer: q.CorrectAnswer,
						UserAnswer:    userAns,
					})
				}
			}
		}
	}

	cards, tier, err := s.generateFlashcardsCore(notebookID, startPage, endPage, failedQuestions)
	if err != nil {
		return nil, nil, false, "", err
	}

	for i := range cards {
		cards[i].TopicID = topicID
	}

	topicTitle := topicID // Fallback title
	err = s.repo.EnsureTopicsBatch([]db.TopicBatchItem{{TopicID: topicID, Title: topicTitle}})
	if err != nil {
		return nil, nil, false, "", fmt.Errorf("failed to ensure topic: %w", err)
	}

	err = s.repo.EnsureNotebookTopic(notebookID, topicID)
	if err != nil {
		return nil, nil, false, "", fmt.Errorf("failed to link topic to notebook: %w", err)
	}

	// Start cards in Review state (bypass learning phase) with initial 1-day (24h) offset
	initialState := models.FlashcardState{
		StateCode:  2,   // 2 = Review state in models.go
		Stability:  1.0, // Default initial stability for Review state
		Difficulty: 5.0, // Default initial difficulty for Review state
	}
	now := time.Now().Unix()
	dueAt := now + 24*60*60 // 1 day initial offset across all generated cards
	utils.Warnf("[FSRS_CALIBRATION] Initializing flashcard due_at to 1-day offset (24h) for topicID=%s", topicID)

	states := make(map[string]models.FlashcardState, len(cards))
	for i := range cards {
		cards[i].DueAt = dueAt
		states[cards[i].ID] = initialState
	}

	cards, existing, err := s.repo.GetOrCreateFlashcardsForTopic(topicID, cards, states)
	if err != nil {
		return nil, nil, false, "", fmt.Errorf("failed to persist FSRS flashcards: %w", err)
	}

	cardIDs := make([]string, len(cards))
	for i, card := range cards {
		cardIDs[i] = card.ID
	}
	persistedStates, err := s.repo.GetFlashcardStatesByIDs(cardIDs)
	if err != nil {
		return nil, nil, false, "", fmt.Errorf("failed to fetch flashcard states: %w", err)
	}

	return cards, persistedStates, existing, tier, nil
}

func (s *StudyService) generateFlashcardsCore(notebookID string, startPage, endPage int, failedQuestions []models.FailedQuestionDetail) ([]models.Flashcard, string, error) {
	generationSource := "flashcard_pipeline_core"
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, "", fmt.Errorf("notebook ID is required")
	}
	if startPage <= 0 || endPage <= 0 || endPage < startPage {
		return nil, "", fmt.Errorf("invalid page range: start=%d end=%d", startPage, endPage)
	}

	notebookTitle := notebookID
	nb, err := s.repo.GetNotebookByID(notebookID)
	if err == nil && nb != nil && nb.Title != "" {
		notebookTitle = nb.Title
	}

	contextChunks, tokenCount, err := s.buildPageBoundedContext(notebookID, startPage, endPage)
	if err != nil {
		return nil, "", err
	}
	if len(contextChunks) == 0 {
		return nil, "", fmt.Errorf("no content found in page range %d-%d", startPage, endPage)
	}
	utils.Warnf("[FLASHCARD_PIPELINE] flashcard_auto_generation_batch generation_source=%s chunk_count=%d token_estimate=%d page_range=%d-%d", generationSource, len(contextChunks), tokenCount, startPage, endPage)
	contextText := buildContextTextFromChunks(contextChunks)

	llm, tier := s.selectLLM(contextText)
	if llm == nil {
		return nil, "", fmt.Errorf("no LLM provider available (tier: %s)", tier)
	}

	// Get model-specific token limits
	modelName := providerModelName(llm)
	limits := llm.GetLimits()
	maxInputTokens := limits.MaxInputTokens
	utils.Warnf("[FLASHCARD_PIPELINE] model_limits model=%s max_input=%d", modelName, maxInputTokens)
	if maxInputTokens <= 0 {
		return nil, "", fmt.Errorf("invalid or unconfigured MaxInputTokens (%d) for model %s", maxInputTokens, modelName)
	}

	// Default to 5 base flashcards. Additional cards will be added for failed questions in buildMarathonFlashcardPromptWithBudget.
	targetCount := 5

	// Build prompt with token budgeting
	prompt, promptTokenCount, includedChunkIDs := buildMarathonFlashcardPromptWithBudget(notebookTitle, startPage, endPage, contextChunks, targetCount, maxInputTokens, failedQuestions)
	if len(includedChunkIDs) == 0 {
		return nil, "", fmt.Errorf("no chunks fit within prompt token budget for page range %d-%d", startPage, endPage)
	}

	// Log token estimates before generation
	utils.Warnf("[FLASHCARD_PIPELINE] token_budget_estimate prompt_tokens=%d max_input=%d budget_used_pct=%.2f", promptTokenCount, maxInputTokens, float64(promptTokenCount)/float64(maxInputTokens)*100)

	if promptTokenCount > maxInputTokens {
		return nil, "", fmt.Errorf("prompt exceeds model context limit: %d > %d", promptTokenCount, maxInputTokens)
	}

	raw, err := llm.GenerateAnswer(prompt)
	if err != nil {
		return nil, "", s.FormatLLMError(err, tier)
	}

	// Validate output size before parsing
	outputTokenEstimate := len(strings.Fields(raw))
	utils.Warnf("[FLASHCARD_PIPELINE] output_validation output_tokens_est=%d", outputTokenEstimate)

	parsed, err := parseFlashcardLLMResponse(raw)
	if err != nil {
		return nil, "", fmt.Errorf("flashcard parsing failed: %w", err)
	}

	// Apply "Hard Slice" (The Array Truncation Trick) to prevent flashcard avalanche.
	maxCardsAllowed := 5 + len(failedQuestions)
	if len(parsed.Cards) > maxCardsAllowed {
		originalCount := len(parsed.Cards)
		parsed.Cards = parsed.Cards[:maxCardsAllowed]
		utils.Warnf("[FLASHCARD_PIPELINE] hard_slice applied original_count=%d capped_to=%d", originalCount, maxCardsAllowed)
	}

	now := time.Now().Unix()
	dueAt := now + 24*60*60 // Schedule new cards for delayed reinforcement, not same-day review.

	cards := make([]models.Flashcard, 0, len(parsed.Cards))
	allowedChunkIDs := make(map[string]struct{}, len(includedChunkIDs))
	for _, chunkID := range includedChunkIDs {
		allowedChunkIDs[chunkID] = struct{}{}
	}
	defaultChunkID := ""
	if len(includedChunkIDs) > 0 {
		defaultChunkID = includedChunkIDs[0]
	}

	for _, candidate := range parsed.Cards {
		sourceChunkID := strings.TrimSpace(candidate.SourceChunkID)
		cardPrompt := strings.TrimSpace(candidate.Prompt)
		answer := strings.TrimSpace(candidate.Answer)
		if cardPrompt == "" || answer == "" {
			utils.Warnf("Skipping flashcard: missing required prompt or answer")
			continue
		}
		if _, ok := allowedChunkIDs[sourceChunkID]; !ok {
			sourceChunkID = defaultChunkID
		}
		id := uuid.NewString()
		cards = append(cards, models.Flashcard{
			ID:            id,
			SourceChunkID: sourceChunkID,
			Prompt:        cardPrompt,
			Answer:        answer,
			DueAt:         dueAt,
			Suspended:     false,
		})
	}
	if len(cards) == 0 {
		return nil, "", fmt.Errorf("no valid flashcards generated from page range")
	}

	return cards, tier, nil
}

func buildMarathonFlashcardPromptWithBudget(notebookTitle string, startPage, endPage int, contextChunks []models.ChunkWithContext, targetCount, maxInputTokens int, failedQuestions []models.FailedQuestionDetail) (string, int, []string) {
	// Template with empty chunks to calculate static template prompt overhead
	emptyTemplate := buildFlashcardStaticTemplate(notebookTitle, startPage, endPage, targetCount, failedQuestions)
	availableBudget, err := CalculateAvailableContextBudget(maxInputTokens, emptyTemplate)
	if err != nil || availableBudget < 1000 {
		availableBudget = 1000 // Minimum budget for meaningful content
	}

	baseTargetCount := targetCount
	// ponytail: increment target count by failed questions count to generate extra targeted corrective cards
	if len(failedQuestions) > 0 {
		targetCount += len(failedQuestions)
	}

	var b strings.Builder
	b.WriteString("You are an expert academic tutor and flashcard generator creating study materials for spaced repetition (FSRS).\n")
	b.WriteString("CRITICAL: Return ONLY valid JSON. No markdown. No code blocks. No explanations.\n")
	b.WriteString("Output must start with { and end with }. No prefix or suffix text.\n")
	fmt.Fprintf(&b, "Notebook: \"%s\"\n", notebookTitle)

	if len(failedQuestions) > 0 {
		b.WriteString("\n=== TARGETED REVIEW: TOPICS NEEDING REINFORCEMENT ===\n")
		b.WriteString("The user recently took a quiz and missed questions on the following concepts. Generate targeted corrective flashcards (1 per concept) addressing the core principles tested below:\n")
		for _, q := range failedQuestions {
			fmt.Fprintf(&b, "- Tested Concept: %s | Correct Ground Truth: %s\n", q.Prompt, q.CorrectAnswer)
		}
		b.WriteString("CRITICAL: Do NOT allow these targeted topics to crowd out or replace baseline coverage for the remaining pages.\n\n")
	}

	b.WriteString("\n=== JSON FORMAT (FOLLOW EXACTLY) ===\n")
	b.WriteString(`{"cards":[{"prompt":"Why can an early-layer weight be updated during backpropagation even though loss is only calculated at the output layer?","answer":"Because the output depends on early weights through successive intermediate layers, allowing the chain rule to propagate error gradients backward layer-by-layer."}]}` + "\n")
	b.WriteString("\n=== GOAL & KNOWLEDGE DENSITY ===\n")
	b.WriteString("Create a small set of high-value cards that help the learner reconstruct important concepts months or years later. Prioritize understanding over coverage and quantity.\n")
	b.WriteString("Generate 0 to targetCount cards based strictly on information density. There is NO minimum quota. Never create filler cards.\n")
	b.WriteString("- Use ONLY the information contained in the provided source material.\n")
	b.WriteString("- Ensure balanced concept coverage evenly distributed across the entire requested page range.\n")
	b.WriteString("- Concepts may be synthesized across multiple source chunks.\n")
	b.WriteString("\n=== PRIORITIZE ===\n")
	b.WriteString("- Cause-and-effect and 'why/how' mechanisms\n")
	b.WriteString("- Relationships between important concepts\n")
	b.WriteString("- Predictions: 'If X changes, what happens to Y and why?'\n")
	b.WriteString("- Failure modes, misconceptions, and important invariants\n")
	b.WriteString("- State transitions, mathematical/dimensional reasoning, and practical consequences\n")
	b.WriteString("\n=== AVOID ===\n")
	b.WriteString("- Shallow 'What is X?' definitions when deeper questions are possible\n")
	b.WriteString("- Author/book structure trivia or rhetorical questions\n")
	b.WriteString("- Incidental facts and boilerplate\n")
	b.WriteString("- Near-duplicate cards testing the same knowledge\n")
	b.WriteString("- Multiple cards about the same concept from slightly different angles\n")
	b.WriteString("\n=== CARD QUALITY & SELF-EVALUATION ===\n")
	b.WriteString("- Each card must test ONE important cognitive target.\n")
	b.WriteString("- Answers must be concise (1–3 sentences), technically accurate, self-contained, and explain the underlying mechanism.\n")
	b.WriteString("- Before returning a card, ask: 'Would remembering this card meaningfully improve ability to understand or apply this material months later?' If not, discard it.\n")
	b.WriteString("\n")
	if len(failedQuestions) > 0 {
		fmt.Fprintf(&b, "Generate up to %d flashcards total: %d balanced coverage cards across pages %d-%d, plus %d targeted reinforcement card(s) for the missed concepts above.\n", targetCount, baseTargetCount, startPage, endPage, len(failedQuestions))
	} else {
		fmt.Fprintf(&b, "Generate up to %d flashcards from the provided source material (pages %d-%d).\n", targetCount, startPage, endPage)
	}
	b.WriteString("Generate fewer if there are not enough distinct important concepts.\n")
	b.WriteString("\n=== SOURCE CHUNKS ===\n")

	// Trim chunks based on token budget
	currentTokens := 0
	var includedChunks []models.ChunkWithContext
	var includedChunkIDs []string
	truncatedCount := 0

	for _, chunk := range contextChunks {
		text := strings.TrimSpace(chunk.Text)
		if text == "" {
			continue
		}

		// Estimate tokens for this chunk with formatting
		chunkLine := fmt.Sprintf("- page_num: %d | text: %s\n", chunk.PageNum, text)
		chunkTokens, err := embeddings.CountTokens(chunkLine)
		if err != nil {
			// Fallback to word count if tokenization fails
			chunkTokens = len(strings.Fields(chunkLine))
		}

		// Check if adding this chunk would exceed budget
		if currentTokens+chunkTokens > availableBudget {
			truncatedCount++
			continue
		}

		includedChunks = append(includedChunks, chunk)
		includedChunkIDs = append(includedChunkIDs, chunk.ChunkID)
		currentTokens += chunkTokens
	}

	// Add chunks to prompt
	for _, chunk := range includedChunks {
		text := strings.TrimSpace(chunk.Text)
		fmt.Fprintf(&b, "- page_num: %d | text: %s\n", chunk.PageNum, text)
	}

	if truncatedCount > 0 {
		fmt.Fprintf(&b, "[...%d additional chunks truncated to stay within token budget...]\n", truncatedCount)
	}

	utils.Warnf("[FLASHCARD_PIPELINE] chunk_trimming total_chunks=%d included=%d truncated=%d budget_used=%d available=%d",
		len(contextChunks), len(includedChunks), truncatedCount, currentTokens, availableBudget)

	return b.String(), currentTokens, includedChunkIDs
}

func buildFlashcardStaticTemplate(notebookTitle string, startPage, endPage, targetCount int, failedQuestions []models.FailedQuestionDetail) string {
	var b strings.Builder
	b.WriteString("You are an expert academic tutor and flashcard generator creating study materials for spaced repetition (FSRS).\n")
	b.WriteString("CRITICAL: Return ONLY valid JSON. No markdown. No code blocks. No explanations.\n")
	b.WriteString("Output must start with { and end with }. No prefix or suffix text.\n")
	fmt.Fprintf(&b, "Notebook: \"%s\"\n", notebookTitle)

	if len(failedQuestions) > 0 {
		b.WriteString("\n=== TARGETED REVIEW: TOPICS NEEDING REINFORCEMENT ===\n")
		for _, q := range failedQuestions {
			fmt.Fprintf(&b, "- Tested Concept: %s | Correct Ground Truth: %s\n", q.Prompt, q.CorrectAnswer)
		}
		b.WriteString("CRITICAL: Do NOT allow these targeted topics to crowd out or replace baseline coverage for the remaining pages.\n\n")
	}

	b.WriteString("\n=== JSON FORMAT (FOLLOW EXACTLY) ===\n")
	b.WriteString(`{"cards":[{"prompt":"Why can an early-layer weight be updated during backpropagation even though loss is only calculated at the output layer?","answer":"Because the output depends on early weights through successive intermediate layers, allowing the chain rule to propagate error gradients backward layer-by-layer."}]}` + "\n")
	b.WriteString("\n=== GOAL & KNOWLEDGE DENSITY ===\n")
	b.WriteString("Create a small set of high-value cards that help the learner reconstruct important concepts months or years later. Prioritize understanding over coverage and quantity.\n")
	b.WriteString("Generate 0 to targetCount cards based strictly on information density. There is NO minimum quota. Never create filler cards.\n")
	b.WriteString("- Use ONLY the information contained in the provided source material.\n")
	b.WriteString("- Ensure balanced concept coverage evenly distributed across the entire requested page range.\n")
	b.WriteString("- Concepts may be synthesized across multiple source chunks.\n")
	b.WriteString("\n=== PRIORITIZE ===\n")
	b.WriteString("- Cause-and-effect and 'why/how' mechanisms\n")
	b.WriteString("- Relationships between important concepts\n")
	b.WriteString("- Predictions: 'If X changes, what happens to Y and why?'\n")
	b.WriteString("- Failure modes, misconceptions, and important invariants\n")
	b.WriteString("- State transitions, mathematical/dimensional reasoning, and practical consequences\n")
	b.WriteString("\n=== AVOID ===\n")
	b.WriteString("- Shallow 'What is X?' definitions when deeper questions are possible\n")
	b.WriteString("- Author/book structure trivia or rhetorical questions\n")
	b.WriteString("- Incidental facts and boilerplate\n")
	b.WriteString("- Near-duplicate cards testing the same knowledge\n")
	b.WriteString("- Multiple cards about the same concept from slightly different angles\n")
	b.WriteString("\n=== CARD QUALITY & SELF-EVALUATION ===\n")
	b.WriteString("- Each card must test ONE important cognitive target.\n")
	b.WriteString("- Answers must be concise (1–3 sentences), technically accurate, self-contained, and explain the underlying mechanism.\n")
	b.WriteString("- Before returning a card, ask: 'Would remembering this card meaningfully improve ability to understand or apply this material months later?' If not, discard it.\n")
	b.WriteString("\n")
	fmt.Fprintf(&b, "Generate up to %d flashcards from the provided source material (pages %d-%d).\n", targetCount, startPage, endPage)
	b.WriteString("Generate fewer if there are not enough distinct important concepts.\n")
	b.WriteString("\n=== SOURCE CHUNKS ===\n")
	return b.String()
}

