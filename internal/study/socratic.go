package study

// socratic.go — ONLY file that imports internal/retrieval.
// The GenerateShortAnswerPrompt method is the sole consumer of the
// SemanticSearch engine. All other study flows use page-bounded SQL injection.

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
	"ai-tutor/internal/retrieval"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

const socraticRescueInstructions = `You are an Adaptive Concept Tutor helping a student who struggled with a quiz or concept.
Act like an encouraging, highly clear human tutor.
Prefer concrete examples, step-by-step clarity, and direct explanations over abstract or indirect questioning.

Goal:
First explain what went wrong and why, teach the core concepts clearly with examples, and then verify the student's understanding.

Rules:
- Stay within the retrieved material.
- The student cannot see the retrieved material. Do NOT refer to "retrieved material", "provided text", "context", "document", or "source". Talk to the student naturally as if you both know the subject matter.
- When wrong answers are provided:
  1. Identify what the student misunderstood.
  2. Explain the relevant concept clearly and simply using concrete examples.
  3. Explain why the student's answer was incorrect and why the correct answer is right.
  4. Look for underlying concepts causing multiple mistakes rather than treating every wrong answer in isolation.
- Focus only on what the student needs to understand from their mistakes. Do not reteach material they already understand.
- End your response with exactly one short question to check whether the student understood the concept.
- If the student gets a follow-up question wrong, explain the concept again using a different perspective or example before moving on.
- Keep responses focused, friendly, and clear.`

const directRescueInstructions = `You are a Direct and Concise AI Concept Tutor helping a student who struggled with a quiz or concept.
Act like a clear, direct, and helpful expert tutor.

Goal:
Directly and concisely explain what went wrong, teach the correct concepts clearly, and give immediate clarity so the student can master the material and retake their quiz.

Rules:
- Stay within the retrieved material.
- The student cannot see the retrieved material. Do NOT refer to "retrieved material", "provided text", "context", "document", or "source". Talk to the student naturally as if you both know the subject matter.
- When wrong answers are provided:
  1. Directly explain what was misunderstood.
  2. Clearly state why the chosen answer was incorrect and why the correct answer is right.
  3. Explain the core concept directly, simply, and concisely without unnecessary fluff.
- Answer any student questions and doubts directly with straightforward explanations.
- Do NOT withhold answers, give vague hints, or force question-based guessing games.
- Do NOT end with mandatory quiz questions or tests unless explicitly requested by the student.`

const detailedRescueInstructions = `You are a Step-by-Step AI Concept Tutor helping a student who struggled with a quiz or concept.
Act like a thorough, supportive, and structured expert tutor.

Goal:
Provide comprehensive conceptual walkthroughs, real-world analogies, and illustrative examples to thoroughly resolve misunderstandings and explain the correct concepts.

Rules:
- Stay within the retrieved material.
- The student cannot see the retrieved material. Do NOT refer to "retrieved material", "provided text", "context", "document", or "source". Talk to the student naturally as if you both know the subject matter.
- When wrong answers are provided:
  1. Break down the misconception step by step.
  2. Explain the core principles using intuitive analogies and clear examples.
  3. Detail why the chosen answer was incorrect and why the correct answer is right.
- Provide structured, in-depth explanations that build solid conceptual foundations.
- Answer student doubts directly, clearly, and thoroughly.`

const socraticGeneralInstructions = `You are an adaptive Socratic tutor helping a student understand material from the retrieved content.
Act like a human tutor talking to a confused student.
Prefer concrete examples over abstract analysis.
Start from the student's likely confusion.

Goal:
Help the student discover the answer through guided thinking, not answer substitution.

Rules:
- Stay within the retrieved material.
- The student cannot see the retrieved material. Do NOT refer to "retrieved material", "provided text", "context", "document", or "source". Talk to the student naturally as if you both know the subject matter.
- First identify what the student is being asked to do (theme identification, concept understanding, comparison, argument analysis, application, etc.).
- Stay at the same level of abstraction as the question.
- Guide using questions and hints before explanations.
- Build on the student's current understanding.
- Help the student notice evidence, patterns, contrasts, causes, and assumptions.
- Do not create study plans, teaching plans, summaries, or new tasks unless requested.
- Do not provide the final answer unless asked or the student is clearly stuck.
- Keep responses concise and focused.
- Continue the conversation naturally. Reference what the student said before.

Hint Progression:
Observation → Pattern → Concept → Near Answer → Full Explanation

Response Format Guidelines:
- Respond in a natural, conversational manner.
- Directly respond to the student's input: validate if they are correct, partially correct, or incorrect, and explain why briefly using the retrieved material. If they ask a question, answer it directly and clearly.
- End your response with exactly one short probing question to guide them further. If helpful, you may add a hint below the question labeled 'Hint:'.`

const directGeneralInstructions = `You are a Direct and Concise AI Tutor helping a student learn and resolve doubts.
Act like a clear, direct, and knowledgeable tutor.

Goal:
Directly and concisely answer the student's questions, resolve their doubts, and explain concepts with zero fluff, zero evasion, and zero roundabout guessing games.

Rules:
- Stay within the retrieved material.
- The student cannot see the retrieved material. Do NOT refer to "retrieved material", "provided text", "context", "document", or "source". Talk to the student naturally as if you both know the subject matter.
- When the student asks a question, has a doubt, or needs an explanation:
  1. Answer directly, clearly, and immediately without withholding the answer or being evasive.
  2. Give a direct explanation with key takeaways and concrete points.
  3. Point out why things work the way they do in straightforward language.
- Do NOT answer with questions instead of answers. Do NOT force the student into a guessing game or Socratic riddle.
- Do NOT end your response with mandatory quiz questions or test interrogations.
- Keep responses sharp, focused, and directly helpful.`

const detailedGeneralInstructions = `You are a Comprehensive Step-by-Step AI Tutor helping a student deeply understand material.
Act like a structured, thorough, and highly articulate mentor.

Goal:
Provide deep conceptual walkthroughs, intuitive analogies, and clear illustrative examples to help the student thoroughly master the subject and resolve any doubts.

Rules:
- Stay within the retrieved material.
- The student cannot see the retrieved material. Do NOT refer to "retrieved material", "provided text", "context", "document", or "source". Talk to the student naturally as if you both know the subject matter.
- When the student asks a question or has a doubt:
  1. Answer directly and comprehensively.
  2. Break down the answer into structured, step-by-step points.
  3. Use intuitive analogies and real-world examples to make complex ideas crystal clear.
  4. Clearly explain the "why" and "how" behind each concept.
- Answer directly and thoroughly without hiding information.
- Keep tone supportive, clear, and educational.`

// GenerateShortAnswerPrompt creates, persists, and returns one grounded short-answer
// question for the Socratic mode.  It is the only method in the study package
// that calls the vector retrieval engine.
func (s *StudyService) GenerateShortAnswerPrompt(topicID string) map[string]interface{} {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return map[string]interface{}{"error": "topic ID is required"}
	}
	if s.fastLLMProvider == nil {
		return map[string]interface{}{"error": "FAST_LLM provider not initialized"}
	}
	if s.retrievalEngine == nil {
		return map[string]interface{}{"error": "retrieval engine not initialized"}
	}

	// Semantic search for the most relevant chunks in this topic.
	results, err := s.retrievalEngine.SemanticSearch(
		topicID,
		"Generate exactly one short-answer assessment question grounded in the retrieved material.",
		5, 0, 0,
	)
	if err != nil {
		return map[string]interface{}{"error": "retrieval failed: " + err.Error()}
	}
	if len(results) == 0 {
		return map[string]interface{}{"error": "no relevant content found for Socratic question"}
	}

	// Build context from top results.
	var contextBuilder strings.Builder
	chunkIDs := make([]string, 0, len(results))
	for _, r := range results {
		contextBuilder.WriteString(r.Text)
		contextBuilder.WriteByte('\n')
		chunkIDs = append(chunkIDs, r.ChunkID)
	}
	contextText := strings.TrimSpace(contextBuilder.String())

	prompt := fmt.Sprintf(`You are an AI tutor generating a short-answer assessment question.
Act like a human tutor talking to a confused student.
Prefer concrete examples over abstract analysis.
Start from the student's likely confusion.
Use ONLY the material below. Return STRICT JSON only in this shape: {"prompt":"..."}.
Rules:
- Ask exactly one question.
- Keep it concise (max 30 words).
- Require understanding, not pure definition recall.
- Do not include answer choices, rubric, preamble, or markdown.

Retrieved material:
%s`, contextText)

	raw, err := s.fastLLMProvider.GenerateAnswer(prompt)
	if err != nil {
		formattedErr := s.FormatLLMError(err, "fast")
		return map[string]interface{}{"error": formattedErr.Error()}
	}
	parsed, err := parseShortAnswerPromptLLMResponse(raw)
	if err != nil {
		return map[string]interface{}{"error": "short-answer prompt parsing failed: " + err.Error()}
	}
	questionPrompt := strings.TrimSpace(parsed.Prompt)
	if questionPrompt == "" {
		return map[string]interface{}{"error": "short-answer prompt generation returned empty prompt"}
	}

	// Resolve lineage from cited chunks.
	sourceHeading, sourcePageStart, sourcePageEnd := s.resolveSocraticLineage(topicID, chunkIDs)

	question := models.WrittenQuestion{
		ID:              uuid.NewString(),
		TopicID:         topicID,
		Prompt:          questionPrompt,
		SourceHeading:   sourceHeading,
		SourcePageStart: sourcePageStart,
		SourcePageEnd:   sourcePageEnd,
		LLMModel:        providerModelName(s.fastLLMProvider),
		PromptVersion:   "written-v1-persisted",
	}
	if err := s.repo.CreateWrittenQuestion(question); err != nil {
		return map[string]interface{}{"error": "failed to persist short-answer prompt: " + err.Error()}
	}
	return map[string]interface{}{
		"questionID":        question.ID,
		"prompt":            question.Prompt,
		"topicID":           topicID,
		"source_heading":    question.SourceHeading,
		"source_page_start": question.SourcePageStart,
		"source_page_end":   question.SourcePageEnd,
	}
}

// resolveSocraticLineage resolves the heading / page range from chunk IDs.
func (s *StudyService) resolveSocraticLineage(topicID string, chunkIDs []string) (string, int, int) {
	if len(chunkIDs) == 0 {
		return "", 0, 0
	}
	headingPageRanges, err := s.repo.GetTopicHeadingPageRanges(topicID)
	if err != nil {
		utils.Warnf("could not resolve socratic lineage for topic %s: %v", topicID, err)
		return "", 0, 0
	}
	sourcePageStart, sourcePageEnd := 0, 0
	for _, cid := range chunkIDs {
		pageRange, ok := headingPageRanges[cid]
		if !ok {
			continue
		}
		if sourcePageStart == 0 || pageRange[0] < sourcePageStart {
			sourcePageStart = pageRange[0]
		}
		if pageRange[1] > sourcePageEnd {
			sourcePageEnd = pageRange[1]
		}
	}
	sourceHeading := ""
	if sourcePageStart > 0 {
		sourceHeading = fmt.Sprintf("Page %d", sourcePageStart)
	}
	return sourceHeading, sourcePageStart, sourcePageEnd
}

func (s *StudyService) AskSocratic(notebookID string, topicID string, question string, conversationHistory []map[string]string) (map[string]interface{}, error) {
	notebookID = strings.TrimSpace(notebookID)
	topicID = strings.TrimSpace(topicID)
	question = strings.TrimSpace(question)
	if notebookID == "" {
		return nil, retrieval.ErrInvalidNotebookContext
	}
	if s.fastLLMProvider == nil {
		return nil, fmt.Errorf("FAST_LLM provider not initialized")
	}
	if s.retrievalEngine == nil {
		return nil, fmt.Errorf("retrieval engine not initialized")
	}
	// ponytail: check for active remedial task and retrieve failed questions payload
	var failedQuestionsSummary string
	if topicID != "" {
		if payloadJSON, err := s.repo.GetActiveRemedialTaskPayloadByTopic(topicID); err == nil && payloadJSON != "" {
			var payload struct {
				FailedQuestions []models.FailedQuestionDetail `json:"failed_questions"`
			}
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err == nil && len(payload.FailedQuestions) > 0 {
				var sb strings.Builder
				sb.WriteString("\n=== WRONG ANSWERS ===\n")
				sb.WriteString("The student failed these quiz questions recently. Focus guidance on these concepts:\n")
				for _, q := range payload.FailedQuestions {
					fmt.Fprintf(&sb, "- Q: %s | User answered: %s | Correct: %s\n", q.Prompt, q.UserAnswer, q.CorrectAnswer)
				}
				sb.WriteString("\n")
				failedQuestionsSummary = sb.String()
			}
		}
	}

	tutorStyle := "socratic"
	if s.repo != nil {
		if userSettings, err := s.repo.GetUserSettings(); err == nil && userSettings != nil && userSettings.TutorStyle != "" {
			tutorStyle = userSettings.TutorStyle
		}
	}

	if question == "" || question == "__START__" {
		if failedQuestionsSummary != "" {
			switch tutorStyle {
			case "direct":
				question = "I studied this material and took a quiz, but I struggled with some questions. Please analyze my wrong answers, explain what I misunderstood, and directly explain why the correct answers are right."
			case "detailed":
				question = "I studied this material and took a quiz, but I struggled with some questions. Please give me a detailed step-by-step walkthrough of what I misunderstood, explaining the correct concepts with examples."
			default:
				question = "I studied this material and took a quiz, but I struggled with some questions. Please analyze my wrong answers, explain what I misunderstood and why the correct answers are right, and ask me one follow-up question to check if I understood."
			}
		} else {
			switch tutorStyle {
			case "direct":
				question = "I am studying this material. Please give me a direct and concise overview of the key concepts."
			case "detailed":
				question = "I am studying this material. Please give me a comprehensive step-by-step overview of the key concepts with examples."
			default:
				question = "I am studying this material. Please introduce the key concept and ask me a probing question to guide my understanding."
			}
		}
	}

	var socraticInstructions string
	if failedQuestionsSummary != "" {
		switch tutorStyle {
		case "direct":
			socraticInstructions = directRescueInstructions
		case "detailed":
			socraticInstructions = detailedRescueInstructions
		default:
			socraticInstructions = socraticRescueInstructions
		}
	} else {
		switch tutorStyle {
		case "direct":
			socraticInstructions = directGeneralInstructions
		case "detailed":
			socraticInstructions = detailedGeneralInstructions
		default:
			socraticInstructions = socraticGeneralInstructions
		}
	}

	// 1. Semantic search for relevant chunks inside the notebook scope
	const topK = 5
	results, err := s.retrievalEngine.SemanticSearchNotebook(notebookID, topicID, question, topK)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// 2. Build retrieved material context blocks, citations, and chunk texts
	blocks, citations, chunkTexts := buildReaderContextBlocksWithText(results)
	if len(blocks) != len(citations) || len(blocks) != len(chunkTexts) {
		return nil, fmt.Errorf("context block length mismatch: blocks=%d citations=%d chunkTexts=%d", len(blocks), len(citations), len(chunkTexts))
	}

	// 3. Generate answer using heavy LLM provider (to ensure high quality guiding responses)
	llm := s.heavyLLMProvider
	if llm == nil {
		llm = s.fastLLMProvider
	}

	// Enforce token budget — compute available input tokens and truncate
	// the retrieved blocks to fit while preserving Socratic instructions
	// and the student question.
	limits := llm.GetLimits()

	// Build conversation history block for the prompt
	// Calculate baseline instructions tokens to establish the remaining history budget
	instructionsOnlyText := strings.Join([]string{
		socraticInstructions,
		failedQuestionsSummary,
		"",
		"Student question: " + question,
	}, "\n")
	instructionsTokens, err := embeddings.CountTokens(instructionsOnlyText)
	if err != nil {
		return nil, fmt.Errorf("error calculating prompt tokens: %w", err)
	}

	// History budget should leave space for context (e.g. 1500 tokens) and safety margin (100 tokens)
	historyBudget := limits.MaxInputTokens - instructionsTokens - 1500 - 100
	if historyBudget < 0 {
		historyBudget = 0
	}

	truncatedHistory, err := BudgetConversationHistory(conversationHistory, historyBudget)
	if err != nil {
		return nil, fmt.Errorf("error budgeting conversation history: %w", err)
	}

	historyBlock := ""
	if len(truncatedHistory) > 0 {
		var histBuilder strings.Builder
		histBuilder.WriteString("Previous conversation:\n")
		for _, msg := range truncatedHistory {
			role := "Student"
			if msg["role"] == "assistant" {
				role = "Tutor"
			}
			fmt.Fprintf(&histBuilder, "%s: %s\n", role, msg["content"])
		}
		historyBlock = histBuilder.String()
	}

	// Compute tokens for prompt overhead (instructions + history + student question + fixed labels)
	overheadText := strings.Join([]string{
		socraticInstructions,
		failedQuestionsSummary,
		"",
		historyBlock,
		"Student question: " + question,
	}, "\n")

	overheadTokens, err := embeddings.CountTokens(overheadText)
	if err != nil {
		return nil, fmt.Errorf("error counting overhead tokens: %w", err)
	}
	// Reserve a small safety margin for formatting and LLM internals
	reserved := 100
	available := limits.MaxInputTokens - overheadTokens - reserved
	if available < 0 {
		available = 0
	}

	// Include as many blocks as will fit into available tokens, truncating
	// the final block if necessary. Keep citations aligned to included blocks.
	newBlocks := make([]string, 0, len(blocks))
	newCitations := make([]string, 0, len(citations))
	newChunkTexts := make([]string, 0, len(chunkTexts))
	usedTokens := 0
	for i, blk := range blocks {
		blkTokens, err := embeddings.CountTokens(blk)
		if err != nil {
			return nil, fmt.Errorf("error counting block tokens: %w", err)
		}
		if usedTokens+blkTokens <= available {
			newBlocks = append(newBlocks, blk)
			newCitations = append(newCitations, citations[i])
			newChunkTexts = append(newChunkTexts, chunkTexts[i])
			usedTokens += blkTokens
			continue
		}
		remaining := available - usedTokens
		if remaining > 8 {
			// Try tokenizer-based truncation for the final chunk
			if truncated, err := embeddings.TruncateToTokens(blk, remaining); err == nil && strings.TrimSpace(truncated) != "" {
				newBlocks = append(newBlocks, truncated)
				newCitations = append(newCitations, citations[i])
				newChunkTexts = append(newChunkTexts, chunkTexts[i])
			}
		}
		break
	}

	// If everything was truncated away, only fall back within remaining budget.
	if len(newBlocks) == 0 && len(blocks) > 0 {
		safeLimit := available
		if safeLimit > 128 {
			safeLimit = 128
		}
		if safeLimit > 0 {
			if truncated, err := embeddings.TruncateToTokens(blocks[0], safeLimit); err == nil && strings.TrimSpace(truncated) != "" {
				newBlocks = append(newBlocks, truncated)
				newCitations = append(newCitations, citations[0])
				newChunkTexts = append(newChunkTexts, chunkTexts[0])
			}
		}
	}

	contextText := strings.TrimSpace(strings.Join(newBlocks, "\n\n"))

	// Ensure contextText label is omitted when empty
	promptParts := []string{
		socraticInstructions,
		failedQuestionsSummary,
		"",
		historyBlock,
	}
	if contextText != "" {
		promptParts = append(promptParts, "Retrieved material:", contextText, "")
	}
	promptParts = append(promptParts, "Student question: "+question)
	socraticPrompt := strings.Join(promptParts, "\n")

	if llm == s.fastLLMProvider && s.heavyLLMProvider == nil {
		utils.Warnf("[SOCRATIC] heavy LLM provider not configured; falling back to fast LLM provider for topic %s", topicID)
	}

	answer, err := llm.GenerateAnswer(socraticPrompt)
	if err != nil {
		tier := "heavy"
		if llm == s.fastLLMProvider {
			tier = "fast"
		}
		return nil, s.FormatLLMError(err, tier)
	}

	return map[string]interface{}{
		"answer":         answer,
		"cited_sections": newCitations,
		"chunk_texts":    newChunkTexts,
	}, nil
}
