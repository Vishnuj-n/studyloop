package study

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

func parseQuizLLMResponse(raw string) (*quizLLMResponse, error) {
	out, err := parseLLMJSON[quizLLMResponse](raw)
	if err != nil {
		return nil, err
	}
	if len(out.Questions) == 0 {
		return nil, fmt.Errorf("no questions in LLM response")
	}
	return out, nil
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
