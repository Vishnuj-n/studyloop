package study

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

// GenerateComprehensiveExam generates a short-answer written assessment question
// synthesized from quiz questions or from the raw text of a notebook's page range (no RAG / ONNX).
func (s *StudyService) GenerateComprehensiveExam(notebookID string, startPage, endPage int) map[string]interface{} {
	return s.GenerateVivaExam(notebookID, startPage, endPage, "")
}

// GenerateVivaExam generates a viva question from either a quiz task or page range.
func (s *StudyService) GenerateVivaExam(notebookID string, startPage, endPage int, quizTaskID string) map[string]interface{} {
	notebookID = strings.TrimSpace(notebookID)
	quizTaskID = strings.TrimSpace(quizTaskID)
	if notebookID == "" && quizTaskID == "" {
		return map[string]interface{}{"error": "notebook ID or quiz task ID is required"}
	}

	notebookTitle := notebookID
	var quizQuestions []models.QuizTaskQuestion

	// If quizTaskID is provided, attempt to load questions from the quiz task
	if quizTaskID != "" {
		task, err := s.repo.GetTaskByID(quizTaskID)
		if err == nil {
			if notebookID == "" {
				notebookID = task.NotebookID
			}
			if startPage <= 0 {
				startPage = task.StartPage
			}
			if endPage <= 0 {
				endPage = task.EndPage
			}

			if task.TaskType == models.StudyTaskTypeMilestoneExam {
				if compiledPayload, cErr := CompileMilestonePayload(s.repo, &task); cErr == nil && len(compiledPayload.Questions) > 0 {
					quizQuestions = compiledPayload.Questions
				}
			} else if strings.TrimSpace(task.PayloadJSON) != "" {
				var payload models.QuizTaskPayload
				if uErr := json.Unmarshal([]byte(task.PayloadJSON), &payload); uErr == nil && len(payload.Questions) > 0 {
					quizQuestions = payload.Questions
				}
			}
		}
	}

	if nb, nbErr := s.repo.GetNotebookByID(notebookID); nbErr == nil && nb != nil && strings.TrimSpace(nb.Title) != "" {
		notebookTitle = strings.TrimSpace(nb.Title)
	}

	var prompt string
	var llm LLMProvider
	var tier string

	if len(quizQuestions) > 0 {
		// MCQ-derived Viva Synthesis
		rawContextText := buildContextTextFromQuizQuestions(quizQuestions)
		llm, tier = s.selectLLM(rawContextText)
		if llm == nil {
			return map[string]interface{}{"error": "no LLM provider available (tier: " + tier + ")"}
		}

		limits := llm.GetLimits()
		templatePrompt := buildVivaFromQuizPrompt(notebookTitle, startPage, endPage, "")
		availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templatePrompt)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		contextText, err := BudgetTextSample(rawContextText, availableBudget)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		if strings.TrimSpace(contextText) == "" {
			return map[string]interface{}{"error": fmt.Sprintf("no quiz context fits within configured Max Input Tokens (%d)", limits.MaxInputTokens)}
		}

		prompt = buildVivaFromQuizPrompt(notebookTitle, startPage, endPage, contextText)
	} else {
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
		llm, tier = s.selectLLM(rawContextText)
		if llm == nil {
			return map[string]interface{}{"error": "no LLM provider available (tier: " + tier + ")"}
		}

		limits := llm.GetLimits()
		templatePrompt := buildComprehensiveExamPrompt(notebookTitle, startPage, endPage, "")
		availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templatePrompt)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

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

		prompt = buildComprehensiveExamPrompt(notebookTitle, startPage, endPage, contextText)
	}

	raw, err := llm.GenerateAnswer(prompt)
	if err != nil {
		formattedErr := s.FormatLLMError(err, tier)
		return map[string]interface{}{"error": formattedErr.Error()}
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
		PromptVersion:   "comprehensive-exam-v2",
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

func buildContextTextFromQuizQuestions(questions []models.QuizTaskQuestion) string {
	var b strings.Builder
	for i, q := range questions {
		fmt.Fprintf(&b, "Question %d: %s\n", i+1, strings.TrimSpace(q.Prompt))
		if strings.TrimSpace(q.CorrectAnswer) != "" {
			fmt.Fprintf(&b, "Correct Answer: %s\n", strings.TrimSpace(q.CorrectAnswer))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func buildVivaFromQuizPrompt(notebookTitle string, startPage, endPage int, quizContext string) string {
	var b strings.Builder
	b.WriteString("You are an expert AI tutor conducting an oral viva examination.\n")
	fmt.Fprintf(&b, "The student just answered the following multiple-choice questions from pages %d-%d of notebook '%s'.\n\n",
		startPage, endPage, notebookTitle)

	b.WriteString(`Return STRICT JSON only in this shape: {"prompt":"..."}.` + "\n")
	b.WriteString("Your task is to synthesize ONE open-ended spoken viva question based on the concepts tested in these questions.\n")
	b.WriteString("The viva question must test deep conceptual understanding and intuition—requiring the student to explain the underlying mechanism, why/how a principle works, or how concepts connect in plain words, without asking for mathematical formula memorization.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Ask exactly one clear synthesis question.\n")
	b.WriteString("- Target core concepts, intuition, mechanisms, and cause-and-effect.\n")
	b.WriteString("- Avoid asking for mathematical derivations, formulas, or pure rote recall.\n")
	b.WriteString("- Avoid simple repetition of a multiple-choice question.\n")
	b.WriteString("- Encourage a 30–90 second spoken explanation.\n")
	b.WriteString("- Maximum 35 words.\n")
	b.WriteString("- Do not include answer choices, rubric, hints, preamble, or markdown.\n")
	b.WriteString("\n=== QUIZ QUESTIONS AND CONCEPTS ===\n")
	b.WriteString(quizContext)

	return b.String()
}

func buildComprehensiveExamPrompt(notebookTitle string, startPage, endPage int, contextText string) string {
	var b strings.Builder
	b.WriteString("You are an AI tutor generating one oral viva question.\n")
	fmt.Fprintf(&b, "Generate exactly ONE high-quality question based ONLY on the supplied material from pages %d-%d of notebook '%s'.\n",
		startPage, endPage, notebookTitle)

	b.WriteString(`Return STRICT JSON only in this shape: {"prompt":"..."}.` + "\n")
	b.WriteString("Test genuine conceptual understanding and intuition rather than definition or formula recall. Target the most important concept or mechanism in the material. Require the student to explain WHY/HOW, apply the concept, predict an outcome, or explain cause and effect in plain terms.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Ask exactly one clear question.\n")
	b.WriteString("- Focus on conceptual understanding, mechanisms, and intuition, not formulas or mathematical derivations.\n")
	b.WriteString("- Use ONLY information supported by the supplied material; do not rely on outside knowledge.\n")
	b.WriteString("- Avoid trivial factual recall, yes/no questions, and questions answerable by copying a sentence.\n")
	b.WriteString("- Encourage a 30–90 second spoken explanation.\n")
	b.WriteString("- Maximum 35 words.\n")
	b.WriteString("- Do not include answer choices, rubric, hints, preamble, or markdown.\n")
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

	scorePrompt := fmt.Sprintf(`You are grading a student's answer in a conceptual assessment.
Return STRICT JSON only in this shape: {"score":number,"feedback":"..."}.

Scoring Philosophy & Rubric:
- Score must be an integer from 1 to 10.
- Focus strictly on CONCEPTUAL UNDERSTANDING, intuitive reasoning, and core principles—NOT formulas, equations, math derivations, or rote jargon.
- Do NOT penalize the student for omitting mathematical formulas or textbook jargon if their conceptual explanation is sound.
- 1-3 = major conceptual misunderstanding.
- 4-5 = partially correct intuition with notable gaps.
- 6-8 = solid conceptual grasp with minor omissions.
- 9-10 = clear, accurate, and insightful understanding.

Feedback Guidelines:
- Keep the feedback CONCISE, direct, and easy to read (avoid lengthy walls of text or unnecessary fluff).
- Format in clean Markdown with clear headers and bullet points.
- Use plain language and intuitive mental models. Format key terms with inline backticks (e.g., ` + "`term`" + `).
- If any math is strictly needed, use $inline$ or $$display$$—never bare parentheses for math equations.
- Feedback structure (keep each section concise, 2-4 sentences or short bullet points):
  ### Evaluation
  - **What was done well**: Key concepts or intuitions the student understood correctly.
  - **Gaps and misunderstandings**: Specific conceptual gaps or misconceptions (do not critique lack of formulas).
  ### Expected Answer
  A concise, intuitive explanation of the core concept and mechanism in plain English (2-4 sentences max).

Question: %s
Student answer: %s`, question.Prompt, userAnswer)

	raw, err := s.fastLLMProvider.GenerateAnswer(scorePrompt)
	if err != nil {
		formattedErr := s.FormatLLMError(err, "fast")
		return map[string]interface{}{"error": formattedErr.Error()}
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
