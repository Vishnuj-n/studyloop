package app

import (
	"fmt"
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
		"You are an expert, encouraging, and highly adaptive AI Tutor (Feynman / Socratic method).",
		"1. Question: What is a queue?",
		"   Options: A, B",
		"   My Answer: B",
		"   Correct Answer: A",
		"2. Question: Why?",
		"   My Answer: (No answer)",
		"   Correct Answer: Because",
		"1. Identify the Patterns:",
		"2. Teach the Missing Concepts:",
		"3. Check for Understanding (Active Recall):",
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

func TestTodayPlanReflectsRemainingReviewCardsAfterPartialReview(t *testing.T) {
	app := newTestApp(t)

	if err := app.repo.EnsureTopic("topic-partial", "Topic Partial"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := app.repo.CreateNotebook("nb-partial", "Notebook Partial", "/path", "pdf", "", "", 1, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := app.repo.EnsureNotebookTopic("nb-partial", "topic-partial"); err != nil {
		t.Fatalf("EnsureNotebookTopic failed: %v", err)
	}

	nowUnix := time.Now().Unix()
	cards := []string{"card-p1", "card-p2", "card-p3"}
	for i, cid := range cards {
		_, err := testRepo.ExecForTest(`
			INSERT INTO fsrs_cards (id, topic_id, prompt, answer, due_at, state_json)
			VALUES (?, ?, ?, ?, ?, ?)
		`, cid, "topic-partial", fmt.Sprintf("Q%d", i+1), "A", nowUnix-10, `{"state_code":2,"stability":1,"difficulty":5}`)
		if err != nil {
			t.Fatalf("insert card failed: %v", err)
		}
	}

	// 1. Initial plan creates the review task with 3 cards
	plan1 := app.GetTodayPlan(0)
	if dueCards, _ := plan1["due_review_cards"].(int); dueCards != 3 {
		t.Fatalf("expected 3 due cards initially, got %d", dueCards)
	}

	taskID := ""
	tasks, ok := plan1["tasks"].([]models.ScheduledTask)
	if !ok || len(tasks) == 0 {
		t.Fatalf("expected tasks in plan: %#v", plan1)
	}
	for _, tk := range tasks {
		if tk.ActionType == "flashcard_review" {
			taskID = tk.ID
			break
		}
	}
	if taskID == "" {
		t.Fatalf("expected flashcard_review task in plan")
	}

	// 2. User reviews 1 card out of 3
	revRes := app.RecordCardReview(taskID, "card-p1", 3)
	if revRes["error"] != nil {
		t.Fatalf("RecordCardReview failed: %v", revRes["error"])
	}

	// 3. Re-querying GetTodayPlan should immediately reflect 2 cards remaining
	plan2 := app.GetTodayPlan(0)
	if dueCards, _ := plan2["due_review_cards"].(int); dueCards != 2 {
		t.Fatalf("expected 2 due cards after reviewing 1 card, got %d", dueCards)
	}
}

