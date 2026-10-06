package db

import (
	"path/filepath"
	"testing"

	"ai-tutor/internal/models"
)

func setupTestRepo(t *testing.T) *Repository {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_study_notes.db")

	repo, err := Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to initialize repository: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	return repo
}

func TestGetCompletedReadingSessionsForTopic(t *testing.T) {
	repo := setupTestRepo(t)

	// Ensure topic and notebook exist
	if err := repo.EnsureTopicWithStatus("topic-backprop", "Backpropagation", "reading"); err != nil {
		t.Fatalf("failed to ensure topic: %v", err)
	}
	if err := repo.CreateNotebook("nb-1", "Machine Learning", "/tmp/ml.pdf", "pdf", "topic-backprop", "hash1", 20, ""); err != nil {
		t.Fatalf("failed to ensure notebook: %v", err)
	}

	// Insert task in study_queue
	task := models.StudyQueueTask{
		ID:         "task-read-1",
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusCompleted,
		TopicID:    "topic-backprop",
		NotebookID: "nb-1",
		StartPage:  1,
		EndPage:    10,
	}
	if err := repo.InsertStudyTask(task); err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	sessions, err := repo.GetCompletedReadingSessionsForTopic("topic-backprop")
	if err != nil {
		t.Fatalf("failed to get completed reading sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].StartPage != 1 || sessions[0].EndPage != 10 {
		t.Errorf("unexpected session page range: %d-%d", sessions[0].StartPage, sessions[0].EndPage)
	}
}
