package app

import (
	"testing"

	"ai-tutor/internal/models"
)

func TestAppendFailedQuestionsSection(t *testing.T) {
	task := models.StudyQueueTask{
		ID:          "task-1",
		PayloadJSON: `{"failed_questions":[{"prompt":"What is a queue?","options":["A","B"],"user_answer":"B","correct_answer":"A"},{"prompt":"Why?","correct_answer":"Because"}]}`,
	}

	got := appendFailedQuestionsSection("Prefix\n", task)
	want := "Prefix\n" +
		"During my quiz, I failed the following questions:\n" +
		"1. Question: What is a queue?\n" +
		"   Options: A, B\n" +
		"   My Answer: B\n" +
		"   Correct Answer: A\n\n" +
		"2. Question: Why?\n" +
		"   My Answer: (No answer)\n" +
		"   Correct Answer: Because\n\n" +
		"Please focus on guiding me through the concepts behind these failed questions.\n\n"

	if got != want {
		t.Fatalf("unexpected prompt:\n%s", got)
	}
}
