package app

import (
	"strings"
	"testing"

	"ai-tutor/internal/models"
)

func TestBuildSocraticRemedialPrompt_Integration(t *testing.T) {
	app := newTestApp(t)

	task := models.StudyQueueTask{
		ID:          "task-1",
		TopicID:     "topic-1",
		NotebookID:  "nb-1",
		PayloadJSON: `{"failed_questions":[{"prompt":"What is a queue?","options":["A","B"],"user_answer":"B","correct_answer":"A"},{"prompt":"Why?","correct_answer":"Because"}]}`,
	}

	got, err := buildSocraticRemedialPrompt(app.repo, task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPhrases := []string{
		"You are an Adaptive Concept Tutor.",
		"1. Question: What is a queue?",
		"   Options: A, B",
		"   My Answer: B",
		"   Correct Answer: A",
		"2. Question: Why?",
		"   My Answer: (No answer)",
		"   Correct Answer: Because",
		"Analyze my wrong answers against the provided material.",
	}

	for _, phrase := range wantPhrases {
		if !strings.Contains(got, phrase) {
			t.Errorf("expected prompt to contain %q, but was not found.\nGot:\n%s", phrase, got)
		}
	}
}
