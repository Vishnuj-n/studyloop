package study

import (
	"fmt"
	"strings"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

// GenerateComprehensiveExam generates a short-answer written assessment question
// from the raw text of a notebook's page range (no RAG / ONNX).
func (s *StudyService) GenerateComprehensiveExam(notebookID string, startPage, endPage int) map[string]interface{} {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return map[string]interface{}{"error": "notebook ID is required"}
	}
	if startPage <= 0 || endPage <= 0 || endPage < startPage {
		return map[string]interface{}{"error": fmt.Sprintf("invalid page range: start=%d end=%d", startPage, endPage)}
	}

	contextChunks, tokenCount, err := s.buildPageBoundedContext(notebookID, startPage, endPage)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if len(contextChunks) == 0 {
		nb, nbErr := s.repo.GetNotebookByID(notebookID)
		if nbErr == nil && nb != nil && nb.StartPage > 0 && nb.EndPage > 0 {
			return map[string]interface{}{
				"error": fmt.Sprintf("no content found in page range %d-%d (notebook content is on pages %d-%d)", startPage, endPage, nb.StartPage, nb.EndPage),
			}
		}
		return map[string]interface{}{"error": fmt.Sprintf("no content found in page range %d-%d", startPage, endPage)}
	}

	rawContextText := buildContextTextFromChunks(contextChunks)

	llm, tier := s.selectLLM(rawContextText)
	if llm == nil {
		return map[string]interface{}{"error": "no LLM provider available (tier: " + tier + ")"}
	}

	limits := llm.GetLimits()
	templatePrompt := buildComprehensiveExamPrompt(notebookID, startPage, endPage, "")
	availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templatePrompt)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// Budget context chunks strictly to fit within available budget
	budgetedChunks, err := BudgetChunksToLimit(contextChunks, availableBudget)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if len(budgetedChunks) == 0 {
		return map[string]interface{}{"error": fmt.Sprintf("no content chunks fit within configured Max Input Tokens (%d)", limits.MaxInputTokens)}
	}
	contextText := buildContextTextFromChunks(budgetedChunks)

	utils.Warnf("[EXAMINER] generate_exam notebookID=%s page_range=%d-%d total_chunks=%d included_chunks=%d est_tokens=%d max_input=%d tier=%s model=%s",
		notebookID, startPage, endPage, len(contextChunks), len(budgetedChunks), tokenCount, limits.MaxInputTokens, tier, providerModelName(llm))

	prompt := buildComprehensiveExamPrompt(notebookID, startPage, endPage, contextText)
	raw, err := llm.GenerateAnswer(prompt)
	if err != nil {
		return map[string]interface{}{"error": "exam generation failed: " + err.Error()}
	}
	parsed, err := parseShortAnswerPromptLLMResponse(raw)
	if err != nil {
		return map[string]interface{}{"error": "exam prompt parsing failed: " + err.Error()}
	}
	questionPrompt := strings.TrimSpace(parsed.Prompt)
	if questionPrompt == "" {
		return map[string]interface{}{"error": "exam prompt generation returned empty question"}
	}

	tx, err := s.repo.Begin()
	if err != nil {
		return map[string]interface{}{"error": "failed to start database transaction: " + err.Error()}
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	syntheticTopicID := fmt.Sprintf("comprehensive-%s-p%d-%d", notebookID, startPage, endPage)

	if err := s.repo.EnsureTopicsBatchTx(tx, []db.TopicBatchItem{{
		TopicID: syntheticTopicID,
		Title:   fmt.Sprintf("Comprehensive %s p%d-%d", notebookID, startPage, endPage),
	}}); err != nil {
		utils.Warnf("failed to create synthetic topic %s for comprehensive exam in notebook %s: %v", syntheticTopicID, notebookID, err)
		return map[string]interface{}{"error": "failed to create synthetic topic for comprehensive exam: " + err.Error()}
	}

	question := models.WrittenQuestion{
		ID:              uuid.NewString(),
		TopicID:         syntheticTopicID,
		Prompt:          questionPrompt,
		SourcePageStart: startPage,
		SourcePageEnd:   endPage,
		LLMModel:        providerModelName(llm),
		PromptVersion:   "comprehensive-exam-v1",
	}
	if err := s.repo.CreateWrittenQuestionTx(tx, question); err != nil {
		return map[string]interface{}{"error": "failed to persist comprehensive exam question: " + err.Error()}
	}

	if err := tx.Commit(); err != nil {
		return map[string]interface{}{"error": "failed to commit database transaction: " + err.Error()}
	}
	committed = true

	return map[string]interface{}{
		"questionID":        question.ID,
		"prompt":            question.Prompt,
		"topicID":           syntheticTopicID,
		"notebook_id":       notebookID,
		"start_page":        startPage,
		"end_page":          endPage,
		"llm_tier":          tier,
		"source_page_start": startPage,
		"source_page_end":   endPage,
	}
}

func buildComprehensiveExamPrompt(notebookID string, startPage, endPage int, contextText string) string {
	var b strings.Builder
	b.WriteString("You are an AI tutor generating a short-answer assessment question.\n")
	fmt.Fprintf(&b, "Generate exactly one short-answer question grounded in pages %d-%d of notebook '%s'.\n",
		startPage, endPage, notebookID)
	b.WriteString(`Return STRICT JSON only in this shape: {"prompt":"..."}.` + "\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Ask exactly one question.\n")
	b.WriteString("- Keep it concise (max 30 words).\n")
	b.WriteString("- Require understanding, not pure definition recall.\n")
	b.WriteString("- Do not include answer choices, rubric, preamble, or markdown.\n")
	b.WriteString("\n=== SOURCE MATERIAL ===\n")
	b.WriteString(contextText)
	return b.String()
}

// ScoreShortAnswer scores one persisted short-answer prompt and updates FSRS.
func (s *StudyService) ScoreShortAnswer(questionID, userAnswer string) map[string]interface{} {
	questionID = strings.TrimSpace(questionID)
	userAnswer = strings.TrimSpace(userAnswer)
	if questionID == "" || userAnswer == "" {
		return map[string]interface{}{"error": "question ID and user answer are required"}
	}
	if s.fastLLMProvider == nil {
		return map[string]interface{}{"error": "FAST_LLM provider not initialized"}
	}

	question, err := s.repo.GetWrittenQuestionByID(questionID)
	if err != nil {
		return map[string]interface{}{"error": "failed to fetch written question: " + err.Error()}
	}
	if question == nil {
		return map[string]interface{}{"error": "written question not found"}
	}

	scorePrompt := fmt.Sprintf(`You are grading a student's short answer.
Return STRICT JSON only in this shape: {"score":number,"feedback":"..."}.

Scoring rubric:
- Score must be an integer from 1 to 10.
- 1-3 = major misunderstandings or mostly incorrect.
- 4-5 = partially correct with clear gaps.
- 6-8 = mostly correct with some omissions.
- 9-10 = strong, precise, and concise.
- Feedback must be concise (max 2 sentences), specific, and actionable.

Question: %s
Student answer: %s`, question.Prompt, userAnswer)

	raw, err := s.fastLLMProvider.GenerateAnswer(scorePrompt)
	if err != nil {
		return map[string]interface{}{"error": "short-answer scoring failed: " + err.Error()}
	}
	parsed, err := parseShortAnswerScoreLLMResponse(raw)
	if err != nil {
		return map[string]interface{}{"error": "short-answer scoring parse failed: " + err.Error()}
	}
	score := parsed.Score
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}

	tx, err := s.repo.Begin()
	if err != nil {
		return map[string]interface{}{"error": "failed to begin transaction: " + err.Error()}
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	writtenAnswer := models.WrittenAnswer{
		QuestionID:    question.ID,
		Score:         score,
		Feedback:      strings.TrimSpace(parsed.Feedback),
		UserAnswer:    userAnswer,
		SourceHeading: question.SourceHeading,
	}
	if err := s.repo.SaveWrittenAnswerTx(tx, writtenAnswer); err != nil {
		return map[string]interface{}{"error": "failed to save written answer: " + err.Error()}
	}
	if err := tx.Commit(); err != nil {
		return map[string]interface{}{"error": "failed to commit transaction: " + err.Error()}
	}
	committed = true

	return map[string]interface{}{
		"question_id":       question.ID,
		"prompt":            question.Prompt,
		"score":             score,
		"feedback":          strings.TrimSpace(parsed.Feedback),
		"source_page_start": question.SourcePageStart,
		"source_page_end":   question.SourcePageEnd,
		"source_heading":    question.SourceHeading,
	}
}
