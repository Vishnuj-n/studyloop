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
	if !stats.IsUnlocked {
		t.Errorf("expected certificate to be unlocked")
	}
}
