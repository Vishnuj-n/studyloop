package study

import (
	"strings"
	"testing"

	"ai-tutor/internal/models"
)

func TestBuildContextTextFromQuizQuestions(t *testing.T) {
	questions := []models.QuizTaskQuestion{
		{
			ID:            "q1",
			Prompt:        "What is gradient descent?",
			CorrectAnswer: "An optimization algorithm to minimize loss",
		},
		{
			ID:            "q2",
			Prompt:        "Why are non-linear activations necessary?",
			CorrectAnswer: "To allow multi-layer neural networks to learn non-linear decision boundaries",
		},
	}

	res := buildContextTextFromQuizQuestions(questions)
	if !strings.Contains(res, "Question 1: What is gradient descent?") {
		t.Errorf("expected Question 1 prompt in context, got: %s", res)
	}
	if !strings.Contains(res, "Correct Answer: An optimization algorithm to minimize loss") {
		t.Errorf("expected Question 1 answer in context, got: %s", res)
	}
	if !strings.Contains(res, "Question 2: Why are non-linear activations necessary?") {
		t.Errorf("expected Question 2 prompt in context, got: %s", res)
	}
}

func TestBuildVivaFromQuizPrompt(t *testing.T) {
	notebookTitle := "Machine Learning 101"
	quizContext := "Question 1: What is backpropagation?\nCorrect Answer: Reverse-mode automatic differentiation"

	prompt := buildVivaFromQuizPrompt(notebookTitle, 1, 5, quizContext)
	if !strings.Contains(prompt, "Machine Learning 101") {
		t.Errorf("expected notebook title in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "pages 1-5") {
		t.Errorf("expected page range in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "QUIZ QUESTIONS AND CONCEPTS") {
		t.Errorf("expected quiz context header in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Reverse-mode automatic differentiation") {
		t.Errorf("expected quiz context content in prompt, got: %s", prompt)
	}
}
