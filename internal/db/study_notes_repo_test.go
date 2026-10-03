package db

import (
	"path/filepath"
	"testing"
	"time"

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

func TestTopicStudyNotes_CRUD(t *testing.T) {
	repo := setupTestRepo(t)

	// Ensure topic and notebook exist
	if err := repo.EnsureTopicWithStatus("topic-backprop", "Backpropagation", "reading"); err != nil {
		t.Fatalf("failed to ensure topic: %v", err)
	}
	if err := repo.CreateNotebook("nb-1", "Machine Learning", "/tmp/ml.pdf", "pdf", "topic-backprop", "hash1", 20, ""); err != nil {
		t.Fatalf("failed to ensure notebook: %v", err)
	}

	// 1. Initial Get should return nil
	initialNote, err := repo.GetTopicStudyNote("topic-backprop")
	if err != nil {
		t.Fatalf("unexpected error getting non-existent note: %v", err)
	}
	if initialNote != nil {
		t.Fatalf("expected nil note for new topic, got %+v", initialNote)
	}

	// 2. Insert via UpsertTopicStudyNote
	noteToInsert := models.TopicStudyNote{
		TopicID:        "topic-backprop",
		NotebookID:     "nb-1",
		Content:        "**Backpropagation** calculates the gradient of the loss function with respect to weights using the chain rule.",
		LastReviewedAt: 0,
	}
	if err := repo.UpsertTopicStudyNote(noteToInsert); err != nil {
		t.Fatalf("failed to upsert study note: %v", err)
	}

	// 3. Fetch and verify note
	fetched, err := repo.GetTopicStudyNote("topic-backprop")
	if err != nil {
		t.Fatalf("failed to get study note: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected note to exist, got nil")
	}
	if fetched.TopicID != "topic-backprop" || fetched.NotebookID != "nb-1" {
		t.Errorf("expected topic_id=topic-backprop notebook_id=nb-1, got topic_id=%s notebook_id=%s", fetched.TopicID, fetched.NotebookID)
	}
	if fetched.Content != noteToInsert.Content {
		t.Errorf("content mismatch: got %q, want %q", fetched.Content, noteToInsert.Content)
	}

	// 4. Update note content
	updatedContent := "**Backpropagation Updated**: Uses backward pass with reverse-mode automatic differentiation."
	if err := repo.UpdateTopicStudyNoteContent("topic-backprop", updatedContent); err != nil {
		t.Fatalf("failed to update note content: %v", err)
	}

	fetchedUpdated, err := repo.GetTopicStudyNote("topic-backprop")
	if err != nil {
		t.Fatalf("failed to get updated study note: %v", err)
	}
	if fetchedUpdated.Content != updatedContent {
		t.Errorf("updated content mismatch: got %q, want %q", fetchedUpdated.Content, updatedContent)
	}

	// 5. Query by Notebook
	notesList, err := repo.GetNotesByNotebook("nb-1")
	if err != nil {
		t.Fatalf("failed to get notes by notebook: %v", err)
	}
	if len(notesList) != 1 {
		t.Fatalf("expected 1 note for notebook nb-1, got %d", len(notesList))
	}
	if notesList[0].TopicID != "topic-backprop" {
		t.Errorf("expected topic-backprop, got %s", notesList[0].TopicID)
	}

	// 6. Mark reviewed
	beforeReview := time.Now().Unix() - 1
	if err := repo.MarkTopicStudyNoteReviewed("topic-backprop"); err != nil {
		t.Fatalf("failed to mark note reviewed: %v", err)
	}

	afterReviewNote, err := repo.GetTopicStudyNote("topic-backprop")
	if err != nil {
		t.Fatalf("failed to get note after review: %v", err)
	}
	if afterReviewNote.LastReviewedAt < beforeReview {
		t.Errorf("expected last_reviewed_at >= %d, got %d", beforeReview, afterReviewNote.LastReviewedAt)
	}
}
