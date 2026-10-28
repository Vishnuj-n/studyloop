package study

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

// EvaluateFeynmanReflection evaluates a student's own-words explanation of a reading chunk.
// It returns structured feedback containing strengths, gaps, calibration, and an executive summary.
func (s *StudyService) EvaluateFeynmanReflection(notebookID string, startPage, endPage int, explanation string) (*models.FeynmanReflectionFeedback, error) {
	explanation = strings.TrimSpace(explanation)
	if explanation == "" {
		return nil, nil
	}

	provider := s.fastLLMProvider
	if provider == nil {
		provider = s.heavyLLMProvider
	}
	if provider == nil {
		return nil, fmt.Errorf("no LLM provider configured for reflection evaluation")
	}

	// Fetch reading session context
	contextText := ""
	if notebookID != "" && startPage > 0 && endPage >= startPage {
		chunks, _, err := s.buildPageBoundedContext(notebookID, startPage, endPage)
		if err == nil && len(chunks) > 0 {
			var sb strings.Builder
			for _, c := range chunks {
				sb.WriteString(c.Text)
				sb.WriteString("\n\n")
			}
			contextText = strings.TrimSpace(sb.String())
		}
	}

	// Cap context length if very large
	if len(contextText) > 6000 {
		contextText = contextText[:6000] + "\n...(truncated context)..."
	}

	prompt := fmt.Sprintf(`You are an expert tutor evaluating a student's open-ended conceptual reflection (Feynman technique) after a reading session.
The student was asked to explain what they learned in their own words in 2-4 sentences.

Source Reading Context:
%s

Student's Explanation:
"%s"

Scoring / Feedback Criteria:
1. Strengths: 1-2 bullet points highlighting what concepts, mechanisms, or intuitions the student understood accurately.
2. Gaps: 1-2 bullet points highlighting any nuances missed, vague assertions, or misconceptions. If the explanation is solid with no major gaps, provide a brief suggestion for deeper exploration.
3. Calibration: Exactly one of: "Solid Understanding", "Partially Clear", "Misaligned / Needs Review".
4. Summary: 1-2 sentences summarizing how accurately the mental model reflects the reading material.

Respond ONLY with valid JSON in this exact structure:
{
  "strengths": ["string"],
  "gaps": ["string"],
  "calibration": "Solid Understanding",
  "summary": "string"
}`, contextText, explanation)

	raw, err := provider.GenerateAnswer(prompt)
	if err != nil {
		utils.Warnf("[FEYNMAN_REFLECTION] LLM generation failed: %v", err)
		return nil, err
	}

	cleanJSON := cleanLLMJSON(raw)
	var feedback models.FeynmanReflectionFeedback
	if err := json.Unmarshal([]byte(cleanJSON), &feedback); err != nil {
		utils.Warnf("[FEYNMAN_REFLECTION] Failed to parse JSON: %v, raw: %s", err, raw)
		// Fallback graceful feedback
		return &models.FeynmanReflectionFeedback{
			Strengths:   []string{"Explanation recorded."},
			Gaps:        []string{},
			Calibration: "Solid Understanding",
			Summary:     strings.TrimSpace(raw),
		}, nil
	}

	if feedback.Calibration == "" {
		feedback.Calibration = "Partially Clear"
	}
	return &feedback, nil
}

func cleanLLMJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			raw = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(raw)
}
