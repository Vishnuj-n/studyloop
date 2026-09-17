package app

import (
	"strings"
	"testing"
	"time"

	"ai-tutor/internal/models"
)

func TestBuildSocraticRemedialPrompt_Integration(t *testing.T) {
	app := newTestApp(t)

	if err := app.repo.EnsureTopic("topic-1", "Topic 1"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := app.repo.CreateNotebook("nb-1", "Notebook 1", "/path", "pdf", "", "", 1, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

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

func TestDayBoundaryReviewScheduling(t *testing.T) {
	app := newTestApp(t)

	// Verify clientEndOfDayUnix calculates a future cutoff for today
	tzOffset := -330 // IST is UTC+5:30 -> offset -330
	cutoff := clientEndOfDayUnix(tzOffset)
	nowUnix := time.Now().Unix()
	if cutoff <= nowUnix {
		t.Fatalf("expected cutoff %d to be strictly in future relative to %d", cutoff, nowUnix)
	}

	// Create notebook and cards with due_at set within today's window
	if err := app.repo.EnsureTopic("topic-day-boundary", "Topic Day Boundary"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := app.repo.CreateNotebook("nb-day-boundary", "Notebook Day Boundary", "/path", "pdf", "", "", 1, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := app.repo.EnsureNotebookTopic("nb-day-boundary", "topic-day-boundary"); err != nil {
		t.Fatalf("EnsureNotebookTopic failed: %v", err)
	}

	// Card due strictly between current Unix time and cutoff
	cardID := "card-day-boundary-1"
	cardDue := nowUnix + (cutoff-nowUnix)/2
	if cardDue <= nowUnix {
		cardDue = nowUnix + 1
	}
	_, err := testRepo.ExecForTest(`
		INSERT INTO fsrs_cards (id, topic_id, prompt, answer, due_at, state_json)
		VALUES (?, ?, ?, ?, ?, ?)
	`, cardID, "topic-day-boundary", "Q", "A", cardDue, `{"state_code":2,"stability":1,"difficulty":5}`)
	if err != nil {
		t.Fatalf("insert card failed: %v", err)
	}

	// GetTodayPlan with timezoneOffsetMinutes should detect card due today
	planRes := app.GetTodayPlan(tzOffset)
	dueCards, _ := planRes["due_review_cards"].(int)
	if dueCards < 1 {
		t.Fatalf("expected at least 1 due card under day-boundary, got %d (plan: %#v)", dueCards, planRes)
	}
}

