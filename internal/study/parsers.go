package study

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ai-tutor/internal/utils"
)

var (
	reMissingComma  = regexp.MustCompile(`([\]}"\d])\s*\n\s*("[\w_]+"\s*:)`)
	reTrailingComma = regexp.MustCompile(`,(\s*[\]}])`)
)

func repairLLMJSON(raw string) string {
	raw = reMissingComma.ReplaceAllString(raw, "$1,\n$2")
	raw = reTrailingComma.ReplaceAllString(raw, "$1")
	return raw
}

func extractValidQuizQuestions(raw string) []quizLLMQuestion {
	var valid []quizLLMQuestion

	scanStart := 0
	if idx := strings.Index(raw, `"questions"`); idx != -1 {
		if arrIdx := strings.IndexByte(raw[idx:], '['); arrIdx != -1 {
			scanStart = idx + arrIdx + 1
		}
	}

	depth := 0
	startIdx := -1
	inString := false
	escaped := false

	for i := scanStart; i < len(raw); i++ {
		ch := raw[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			switch ch {
			case '{':
				if depth == 0 {
					startIdx = i
				}
				depth++
			case '}':
				if depth > 0 {
					depth--
					if depth == 0 && startIdx != -1 {
						block := raw[startIdx : i+1]
						if strings.Contains(block, `"prompt"`) && strings.Contains(block, `"options"`) {
							var q quizLLMQuestion
							repaired := repairLLMJSON(block)
							if err := json.Unmarshal([]byte(repaired), &q); err == nil && q.Prompt != "" && len(q.Options) > 0 {
								valid = append(valid, q)
							}
						}
						startIdx = -1
					}
				}
			}
		}
	}
	return valid
}

// ---------- LLM response types (shared across study sub-files) ----------

type quizLLMQuestion struct {
	SourceChunkID   string   `json:"source_chunk_id"`
	Prompt          string   `json:"prompt"`
	Options         []string `json:"options"`
	CorrectAnswer   string   `json:"correct_answer"`
	Explanation     string   `json:"explanation"`
	Hint            string   `json:"hint"`
	SourceHeading   string   `json:"source_heading"`
	SourceSnippet   string   `json:"source_snippet"`
	SourcePageStart int      `json:"source_page_start"`
	SourcePageEnd   int      `json:"source_page_end"`
}

type quizLLMResponse struct {
	Questions []quizLLMQuestion `json:"questions"`
}

type flashcardLLMCard struct {
	SourceChunkID string `json:"source_chunk_id"`
	Prompt        string `json:"prompt"`
	Answer        string `json:"answer"`
}

type flashcardLLMResponse struct {
	Cards []flashcardLLMCard `json:"cards"`
}

type shortAnswerPromptLLMResponse struct {
	Prompt string `json:"prompt"`
}

type shortAnswerScoreLLMResponse struct {
	Score    int    `json:"score"`
	Feedback string `json:"feedback"`
}

// ---------- Shared JSON Helpers ----------

func parseLLMJSON[T any](raw string) (*T, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	// Strip markdown code fences if present
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		if lastIdx := strings.LastIndex(raw, "```"); lastIdx != -1 {
			raw = raw[:lastIdx]
		}
		raw = strings.TrimSpace(raw)
	}

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func providerModelName(provider LLMProvider) string {
	if provider == nil {
		return "unknown-model"
	}
	name := strings.TrimSpace(provider.ModelName())
	if name == "" {
		return "unknown-model"
	}
	return name
}

func recoverTruncatedQuizJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		if lastIdx := strings.LastIndex(raw, "```"); lastIdx != -1 {
			raw = raw[:lastIdx]
		}
		raw = strings.TrimSpace(raw)
	}

	qIdx := strings.Index(raw, `"questions"`)
	if qIdx == -1 {
		return ""
	}

	lastBrace := strings.LastIndex(raw, "}")
	if lastBrace <= qIdx {
		return ""
	}

	sub := strings.TrimSpace(raw[:lastBrace+1])
	sub = strings.TrimSuffix(sub, ",")
	sub = strings.TrimSpace(sub)

	openBrackets := strings.Count(sub, "[")
	closeBrackets := strings.Count(sub, "]")
	openBraces := strings.Count(sub, "{")
	closeBraces := strings.Count(sub, "}")

	var sb strings.Builder
	sb.WriteString(sub)
	for i := 0; i < openBrackets-closeBrackets; i++ {
		sb.WriteString("]")
	}
	for i := 0; i < openBraces-closeBraces; i++ {
		sb.WriteString("}")
	}

	return sb.String()
}

func parseCompactQuizResponse(raw string) []quizLLMQuestion {
	var questions []quizLLMQuestion
	lines := strings.Split(raw, "\n")

	var currentPrompt string
	var currentCorrect string
	var currentCorrectCount int
	var currentOptions []string

	flush := func() {
		if strings.TrimSpace(currentPrompt) != "" && strings.TrimSpace(currentCorrect) != "" && currentCorrectCount == 1 && len(currentOptions) == 4 {
			questions = append(questions, quizLLMQuestion{
				Prompt:        strings.TrimSpace(currentPrompt),
				Options:       currentOptions,
				CorrectAnswer: strings.TrimSpace(currentCorrect),
			})
		}
		currentPrompt = ""
		currentCorrect = ""
		currentCorrectCount = 0
		currentOptions = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for Question start: "Q:", "Q1:", "Q 1:", "1.", "1)"
		upper := strings.ToUpper(trimmed)
		isQuestionStart := strings.HasPrefix(upper, "Q:") ||
			strings.HasPrefix(upper, "QUESTION") ||
			(len(trimmed) > 2 && (trimmed[0] >= '1' && trimmed[0] <= '9') && (trimmed[1] == '.' || trimmed[1] == ')' || trimmed[1] == ':')) ||
			(strings.HasPrefix(upper, "Q") && strings.Contains(upper, ":") && func() bool {
				idx := strings.Index(upper, ":")
				mid := strings.TrimSpace(upper[1:idx])
				if len(mid) == 0 {
					return false
				}
				for _, r := range mid {
					if r < '0' || r > '9' {
						return false
					}
				}
				return true
			}())

		if isQuestionStart {
			// Flush previous question if complete, otherwise discard incomplete state
			flush()
			idx := strings.IndexAny(trimmed, ":.)")
			if idx != -1 && idx < len(trimmed)-1 {
				currentPrompt = strings.TrimSpace(trimmed[idx+1:])
			} else {
				currentPrompt = trimmed
			}
			continue
		}

		// Correct option: starts with "+"
		if strings.HasPrefix(trimmed, "+") {
			opt := strings.TrimSpace(strings.TrimPrefix(trimmed, "+"))
			if opt != "" {
				currentCorrectCount++
				currentCorrect = opt
				currentOptions = append(currentOptions, opt)
			}
			continue
		}

		// Incorrect option: starts with "-"
		if strings.HasPrefix(trimmed, "-") {
			opt := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if opt != "" {
				currentOptions = append(currentOptions, opt)
			}
			continue
		}
	}
	flush()
	return questions
}

func parseQuizLLMResponse(raw string) (*quizLLMResponse, error) {
	trimmed := strings.TrimSpace(raw)
	// If raw response looks like compact text format (starts with Q: or does not start with JSON brace '{'), try compact parser first
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "```json") {
		if compactQuestions := parseCompactQuizResponse(raw); len(compactQuestions) > 0 {
			utils.Infof("[QUIZ_PARSER] successfully parsed %d questions from compact format", len(compactQuestions))
			return &quizLLMResponse{Questions: compactQuestions}, nil
		}
	}

	out, err := parseLLMJSON[quizLLMResponse](raw)
	if err == nil && len(out.Questions) > 0 {
		return out, nil
	}

	repaired := repairLLMJSON(raw)
	if repOut, repErr := parseLLMJSON[quizLLMResponse](repaired); repErr == nil && len(repOut.Questions) > 0 {
		utils.Infof("[QUIZ_PARSER] successfully parsed quiz after JSON syntax repair")
		return repOut, nil
	}

	if recoveredRaw := recoverTruncatedQuizJSON(raw); recoveredRaw != "" {
		if recOut, recErr := parseLLMJSON[quizLLMResponse](recoveredRaw); recErr == nil && len(recOut.Questions) > 0 {
			utils.Warnf("[QUIZ_RECOVERY] recovered %d valid questions from truncated LLM response", len(recOut.Questions))
			return recOut, nil
		}
		if recRepOut, recRepErr := parseLLMJSON[quizLLMResponse](repairLLMJSON(recoveredRaw)); recRepErr == nil && len(recRepOut.Questions) > 0 {
			utils.Warnf("[QUIZ_RECOVERY] recovered %d valid questions from repaired truncated LLM response", len(recRepOut.Questions))
			return recRepOut, nil
		}
	}

	if extracted := extractValidQuizQuestions(raw); len(extracted) > 0 {
		utils.Warnf("[QUIZ_RECOVERY] extracted %d valid question objects from malformed LLM response", len(extracted))
		return &quizLLMResponse{Questions: extracted}, nil
	}

	// Final fallback: try compact parser on raw in case JSON wrapping contained compact lines
	if compactQuestions := parseCompactQuizResponse(raw); len(compactQuestions) > 0 {
		utils.Infof("[QUIZ_PARSER] successfully parsed %d questions from compact fallback", len(compactQuestions))
		return &quizLLMResponse{Questions: compactQuestions}, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no questions in LLM response")
}

func parseFlashcardLLMResponse(raw string) (*flashcardLLMResponse, error) {
	out, err := parseLLMJSON[flashcardLLMResponse](raw)
	if err != nil {
		return nil, err
	}
	if len(out.Cards) == 0 {
		return nil, fmt.Errorf("no cards in LLM response")
	}
	return out, nil
}

func parseShortAnswerPromptLLMResponse(raw string) (*shortAnswerPromptLLMResponse, error) {
	out, err := parseLLMJSON[shortAnswerPromptLLMResponse](raw)
	if err != nil {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			return &shortAnswerPromptLLMResponse{Prompt: raw}, nil
		}
		return nil, err
	}
	if strings.TrimSpace(out.Prompt) == "" {
		return nil, fmt.Errorf("no prompt in LLM response")
	}
	return out, nil
}

func parseShortAnswerScoreLLMResponse(raw string) (*shortAnswerScoreLLMResponse, error) {
	out, err := parseLLMJSON[shortAnswerScoreLLMResponse](raw)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Feedback) == "" {
		out.Feedback = "Review key topic concepts and tighten your explanation."
	}
	return out, nil
}

func ResolveCorrectOption(correctAnswer string, options []string) (string, bool) {
	canonical := strings.TrimSpace(strings.ToLower(correctAnswer))
	for _, opt := range options {
		if strings.TrimSpace(strings.ToLower(opt)) == canonical {
			return strings.TrimSpace(opt), true
		}
	}
	return "", false
}
