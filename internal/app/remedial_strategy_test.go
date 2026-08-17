package app

import (
	"testing"

	"ai-tutor/internal/models"
)

func submitFailedQuiz(t *testing.T, app *App, taskID string) models.QuizResult {
	t.Helper()
	resp := app.SubmitQuizAttempt(taskID, []models.QuizAnswer{
		{QuestionID: "quiz-q1", Selected: "B"},
		{QuestionID: "quiz-q2", Selected: "C"},
	})
	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("expected submit success, got error: %v", resp["error"])
	}

	result, ok := resp["result"].(models.QuizResult)
	if !ok {
		t.Fatalf("expected QuizResult payload, got %#v", resp["result"])
	}
	if result.Passed {
		t.Fatalf("expected failed quiz result")
	}
	return result
}

func TestFastTrackSkipsReread(t *testing.T) {
	app := newTestApp(t)

	// Set strategy to FAST
	if _, err := testRepo.ExecForTest("UPDATE user_settings SET default_remedial_strategy = ? WHERE id = 1", "FAST"); err != nil {
		t.Fatalf("failed to set strategy to FAST: %v", err)
	}

	mustInsertActiveQuizTask(t, "nb-fast-track", "topic-fast-track", "task-fast-track", 100)
	result := submitFailedQuiz(t, app, "task-fast-track")

	// Verify no REREAD task was created
	if result.RereadTaskID != "" {
		t.Fatalf("expected no reread task, but got ID: %s", result.RereadTaskID)
	}

	// Verify SOCRATIC_REMEDIAL task was created
	pendingSocratic, err := testRepo.CountTasksByTopicTypeAndStatus("topic-fast-track", "SOCRATIC_REMEDIAL", "PENDING")
	if err != nil {
		t.Fatalf("query pending socratic failed: %v", err)
	}
	if pendingSocratic != 1 {
		t.Fatalf("expected 1 pending SOCRATIC_REMEDIAL task, got %d", pendingSocratic)
	}

	// Verify no REREAD task exists in DB
	pendingRereads, err := testRepo.CountTasksByTopicTypeAndStatus("topic-fast-track", "REREAD", "PENDING")
	if err != nil {
		t.Fatalf("query pending rereads failed: %v", err)
	}
	if pendingRereads != 0 {
		t.Fatalf("expected 0 pending REREAD tasks, got %d", pendingRereads)
	}
}

func TestClassicTrackInsertsReread(t *testing.T) {
	app := newTestApp(t)

	// Set strategy to CLASSIC
	if _, err := testRepo.ExecForTest("UPDATE user_settings SET default_remedial_strategy = ? WHERE id = 1", "CLASSIC"); err != nil {
		t.Fatalf("failed to set strategy to CLASSIC: %v", err)
	}

	mustInsertActiveQuizTask(t, "nb-classic-track", "topic-classic-track", "task-classic-track", 100)
	result := submitFailedQuiz(t, app, "task-classic-track")

	// Verify REREAD task was created
	if result.RereadTaskID == "" {
		t.Fatalf("expected reread task to be created")
	}

	// Verify pending REREAD exists in DB
	pendingRereads, err := testRepo.CountTasksByTopicTypeAndStatus("topic-classic-track", "REREAD", "PENDING")
	if err != nil {
		t.Fatalf("query pending rereads failed: %v", err)
	}
	if pendingRereads != 1 {
		t.Fatalf("expected 1 pending REREAD task, got %d", pendingRereads)
	}
}

func TestDefaultIsClassic(t *testing.T) {
	app := newTestApp(t)

	// Do not set strategy explicitly; it should default to CLASSIC
	mustInsertActiveQuizTask(t, "nb-default-track", "topic-default-track", "task-default-track", 100)
	result := submitFailedQuiz(t, app, "task-default-track")

	// Verify REREAD task was created (default behavior)
	if result.RereadTaskID == "" {
		t.Fatalf("expected reread task to be created by default")
	}
}
