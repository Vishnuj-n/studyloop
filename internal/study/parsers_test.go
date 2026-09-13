package study

import (
	"testing"
)

func TestParseQuizLLMResponse_ValidComplete(t *testing.T) {
	raw := `{
		"questions": [
			{
				"prompt": "What is 2+2?",
				"options": ["3", "4", "5", "6"],
				"correct_answer": "4"
			},
			{
				"prompt": "What is the capital of France?",
				"options": ["Berlin", "Madrid", "Paris", "Rome"],
				"correct_answer": "Paris"
			}
		]
	}`

	parsed, err := parseQuizLLMResponse(raw)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(parsed.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(parsed.Questions))
	}
	if parsed.Questions[0].Prompt != "What is 2+2?" {
		t.Errorf("unexpected prompt: %s", parsed.Questions[0].Prompt)
	}
}

func TestParseQuizLLMResponse_PartialRecovery(t *testing.T) {
	// Truncated JSON simulating an LLM hitting output token limit mid-question #3
	truncatedRaw := "```json\n" + `{
		"questions": [
			{
				"prompt": "Question 1 prompt?",
				"options": ["Opt A", "Opt B", "Opt C", "Opt D"],
				"correct_answer": "Opt A"
			},
			{
				"prompt": "Question 2 prompt?",
				"options": ["Opt 1", "Opt 2", "Opt 3", "Opt 4"],
				"correct_answer": "Opt 1"
			},
			{
				"prompt": "Question 3 truncated prompt...",
				"options": ["Truncated Opt
` + "\n```"

	parsed, err := parseQuizLLMResponse(truncatedRaw)
	if err != nil {
		t.Fatalf("expected successful partial recovery, got error: %v", err)
	}
	if len(parsed.Questions) != 2 {
		t.Fatalf("expected 2 recovered questions, got %d", len(parsed.Questions))
	}
	if parsed.Questions[0].Prompt != "Question 1 prompt?" || parsed.Questions[1].Prompt != "Question 2 prompt?" {
		t.Errorf("unexpected recovered question prompts: %+v", parsed.Questions)
	}
}

func TestParseQuizLLMResponse_PartialRecoveryWithTrailingComma(t *testing.T) {
	truncatedRaw := `{
		"questions": [
			{
				"prompt": "Question 1 prompt?",
				"options": ["Opt A", "Opt B", "Opt C", "Opt D"],
				"correct_answer": "Opt A"
			},
	`

	parsed, err := parseQuizLLMResponse(truncatedRaw)
	if err != nil {
		t.Fatalf("expected successful recovery with trailing comma, got error: %v", err)
	}
	if len(parsed.Questions) != 1 {
		t.Fatalf("expected 1 recovered question, got %d", len(parsed.Questions))
	}
	if parsed.Questions[0].Prompt != "Question 1 prompt?" {
		t.Errorf("unexpected recovered question prompt: %s", parsed.Questions[0].Prompt)
	}
}
