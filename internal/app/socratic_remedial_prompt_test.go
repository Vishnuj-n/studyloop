package app

import (
	"encoding/json"
	"strings"
	"testing"

	"ai-tutor/internal/models"
)

func TestBuildSocraticRemedialPrompt_AdaptiveFormat(t *testing.T) {
	app := newTestApp(t)

	if err := app.repo.EnsureTopic("topic-1", "Topic 1"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := app.repo.CreateNotebook("nb-1", "Notebook 1", "/path", "pdf", "", "", 1, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

	// Create test task with failed questions payload
	payload := struct {
		FailedQuestions []models.FailedQuestionDetail `json:"failed_questions"`
	}{
		FailedQuestions: []models.FailedQuestionDetail{
			{
				Prompt:        "What is photosynethsis?",
				Options:       []string{"A) Process A", "B) Process B"},
				UserAnswer:    "A) Process A",
				CorrectAnswer: "B) Process B",
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	task := models.StudyQueueTask{
		ID:          "task-socratic-test",
		NotebookID:  "nb-1",
		TopicID:     "topic-1",
		TaskType:    models.StudyTaskTypeSocraticRemedial,
		PayloadJSON: string(payloadBytes),
	}

	promptText, err := buildSocraticRemedialPrompt(app.repo, task)
	if err != nil {
		t.Fatalf("buildSocraticRemedialPrompt failed: %v", err)
	}

	// Verify key prompt structure and persona
	expectedPhrases := []string{
		"You are an Adaptive Concept Tutor.",
		"I studied the material provided below",
		"Here are the questions I got wrong:",
		"What is photosynethsis?",
		"My Answer: A) Process A",
		"Correct Answer: B) Process B",
		"Analyze my wrong answers against the provided material.",
		"For each mistake:",
		"Identify what I misunderstood.",
		"Explain the relevant concept clearly and simply.",
		"Explain why my answer was wrong and why the correct answer is right.",
		"After explaining a concept, ask me one short question to check whether I understood it.",
		"Focus only on what I need to understand from my mistakes.",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(promptText, phrase) {
			t.Errorf("expected prompt to contain %q, but was not found.\nPrompt:\n%s", phrase, promptText)
		}
	}
}
