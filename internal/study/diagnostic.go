package study

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-tutor/internal/models"
)

const diagnosticPromptSystem = `You are an expert diagnostic AI tutor.
Analyze the following set of missed multiple-choice assessment questions to identify the underlying conceptual breakdown and pattern of misunderstanding.

Respond ONLY with valid JSON matching this exact structure:
{
  "core_misconception": "1-2 sentences explaining the root mental model error connecting these missed questions.",
  "golden_rule": "A clear, memorable principle or rule from the material to remember.",
  "actionable_tip": "A concise, actionable piece of study advice.",
  "sub_concepts": [
    {
      "name": "Subtopic/Concept Name",
      "mastery_score": 35,
      "status": "Vulnerable"
    }
  ]
}

Rules:
- Do not output markdown code fences (like ` + "```json" + `). Output pure JSON.
- Synthesize the shared root cause across the questions rather than just repeating each answer.
- "status" in sub_concepts can be "Vulnerable" (score < 50), "Needs Review" (50-70), or "Mastered" (> 70).
`

// AnalyzeQuizFailure performs an LLM-based root-cause analysis on a set of missed quiz questions.
func (s *StudyService) AnalyzeQuizFailure(bookContent string, failedQuestions []models.FailedQuestionDetail) (*models.QuizDiagnosticResult, error) {
	if len(failedQuestions) == 0 {
		return &models.QuizDiagnosticResult{
			CoreMisconception: "No missed questions detected. Full mastery demonstrated.",
			GoldenRule:        "Continue regular spaced repetition to consolidate retention.",
			ActionableTip:     "Proceed to the next chapter or challenge yourself with the viva examiner.",
			SubConcepts: []models.DiagnosticSubConcept{
				{
					Name:         "Overall Material",
					MasteryScore: 100,
					Status:       "Mastered",
				},
			},
		}, nil
	}

	if s.fastLLMProvider == nil {
		return nil, fmt.Errorf("fast LLM provider not initialized")
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString(diagnosticPromptSystem)
	promptBuilder.WriteString("\n\n")

	if strings.TrimSpace(bookContent) != "" {
		promptBuilder.WriteString("Textbook Reference Material:\n")
		promptBuilder.WriteString(strings.TrimSpace(bookContent))
		promptBuilder.WriteString("\n\n")
	}

	promptBuilder.WriteString("Missed Questions:\n")
	for i, fq := range failedQuestions {
		fmt.Fprintf(&promptBuilder, "Question %d:\nPrompt: %s\n", i+1, fq.Prompt)
		if len(fq.Options) > 0 {
			fmt.Fprintf(&promptBuilder, "Options: %s\n", strings.Join(fq.Options, " | "))
		}
		fmt.Fprintf(&promptBuilder, "Student's Answer: %s\nCorrect Answer: %s\n\n", fq.UserAnswer, fq.CorrectAnswer)
	}

	rawResponse, err := s.fastLLMProvider.GenerateAnswer(promptBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate diagnostic from LLM: %w", err)
	}

	cleaned := cleanDiagnosticJSON(rawResponse)
	var result models.QuizDiagnosticResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse diagnostic JSON: %w (raw response: %s)", err, rawResponse)
	}

	return &result, nil
}

func cleanDiagnosticJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start != -1 && end != -1 && end > start {
		return raw[start : end+1]
	}
	return raw
}
