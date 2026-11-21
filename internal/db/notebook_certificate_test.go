package db

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"ai-tutor/internal/models"
)

func TestGetNotebookCertificateStats(t *testing.T) {
	tempDB := "test_certificate_stats.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// 1. Create Topic
	nbID := "nb-cert-1"
	topicID := "topic-cert-1"
	err = repo.EnsureTopicWithStatus(topicID, "Neural Networks", "completed")
	if err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}

	// 2. Create Notebook
	err = repo.CreateNotebook(nbID, "Deep Learning Book", "/tmp/dl.pdf", "pdf", topicID, "hash123", 100, "")
	if err != nil {
		t.Fatalf("failed to create notebook: %v", err)
	}
	_ = repo.EnsureNotebookTopic(nbID, topicID)

	// Add Reading task & complete it
	taskID := "task-read-1"
	err = repo.InsertStudyTask(models.StudyQueueTask{
		ID:         taskID,
		NotebookID: nbID,
		TopicID:    topicID,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusActive,
		StartPage:  1,
		EndPage:    100,
	})
	if err != nil {
		t.Fatalf("failed to insert reading task: %v", err)
	}
	_ = repo.CompleteTask(taskID, models.CompletionResult{Status: models.StudyTaskStatusCompleted})

	// Add Quiz & passed attempt
	quizTaskID := "task-quiz-1"
	_ = repo.InsertStudyTask(models.StudyQueueTask{
		ID:         quizTaskID,
		NotebookID: nbID,
		TopicID:    topicID,
		TaskType:   models.StudyTaskTypeQuiz,
		Status:     models.StudyTaskStatusCompleted,
	})
	_ = repo.withTx(func(tx *sql.Tx) error {
		return repo.SaveQuizAttemptTx(tx, models.QuizAttemptRecord{
			ID:          "attempt-1",
			TaskID:      quizTaskID,
			Score:       95,
			Passed:      true,
			AnswersJSON: `[{"selected":0,"correct":0}]`,
			Feedback:    "Great work!",
			CompletedAt: time.Now().Unix(),
		})
	})

	// Add Milestone Exam
	milestoneTaskID := "task-milestone-1"
	_ = repo.InsertStudyTask(models.StudyQueueTask{
		ID:         milestoneTaskID,
		NotebookID: nbID,
		TopicID:    topicID,
		TaskType:   models.StudyTaskTypeMilestoneExam,
		Status:     models.StudyTaskStatusCompleted,
	})

	// Add Flashcards with varying FSRS maturity:
	// - fc-1: New (reps 0, stability 0) -> NOT mastered
	// - fc-2: Learning (reps 2, stability 18.0) -> NOT mastered
	// - fc-3: Mature/Mastered (reps 5, stateCode 2, stability 75.0) -> MASTERED
	// - fc-4: Suspended (reps 8, stateCode 2, stability 85.0, suspended: true) -> NOT mastered
	flashcards := []models.Flashcard{
		{ID: "fc-1", TopicID: topicID, Prompt: "What is Backprop?", Answer: "Gradient calculation", Suspended: false},
		{ID: "fc-2", TopicID: topicID, Prompt: "What is ReLU?", Answer: "Activation function", Suspended: false},
		{ID: "fc-3", TopicID: topicID, Prompt: "What is Adam?", Answer: "Adaptive optimizer", Suspended: false},
		{ID: "fc-4", TopicID: topicID, Prompt: "What is Dropout?", Answer: "Regularization", Suspended: true},
	}
	states := map[string]models.FlashcardState{
		"fc-1": {Reps: 0, StateCode: 0, Stability: 0.0},
		"fc-2": {Reps: 2, StateCode: 1, Stability: 18.0},
		"fc-3": {Reps: 5, StateCode: 2, Stability: 75.0},
		"fc-4": {Reps: 8, StateCode: 2, Stability: 85.0},
	}
	if err := repo.CreateFlashcards(topicID, flashcards, states); err != nil {
		t.Fatalf("failed to create flashcards: %v", err)
	}

	// Add a review log for fc-3
	if _, err := repo.db.Exec(`INSERT INTO fsrs_review_log (id, topic_id, activity_type, reference_id, rating, reviewed_at, scheduled_days, state_before_json, state_after_json) VALUES ('log-fc-3', ?, 'flashcard', 'fc-3', 3, 1000, 1, '{}', '{}')`, topicID); err != nil {
		t.Fatalf("failed to insert review log: %v", err)
	}

	stats, err := repo.GetNotebookCertificateStats(nbID)
	if err != nil {
		t.Fatalf("unexpected error getting certificate stats: %v", err)
	}

	if stats.NotebookTitle != "Deep Learning Book" {
		t.Errorf("expected notebook title 'Deep Learning Book', got %s", stats.NotebookTitle)
	}
	if stats.QuizzesPassed != 1 {
		t.Errorf("expected 1 quiz passed, got %d", stats.QuizzesPassed)
	}
	if stats.MilestonesCleared != 1 {
		t.Errorf("expected 1 milestone cleared, got %d", stats.MilestonesCleared)
	}
	if stats.AverageQuizScore < 90 {
		t.Errorf("expected average quiz score >= 90, got %f", stats.AverageQuizScore)
	}
	if stats.FlashcardsMastered != 1 {
		t.Errorf("expected exactly 1 mastered flashcard (fc-3 with stability >= 60), got %d", stats.FlashcardsMastered)
	}
	if stats.TotalReviewsCount != 1 {
		t.Errorf("expected 1 total review log count, got %d", stats.TotalReviewsCount)
	}
	if !stats.IsUnlocked {
		t.Errorf("expected certificate to be unlocked")
	}
}
