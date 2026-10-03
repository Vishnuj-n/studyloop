package study

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-tutor/internal/db"
)

type mockNotesLLM struct {
	dummyLLM
	response   string
	lastPrompt string
}

func (m *mockNotesLLM) GenerateAnswer(prompt string) (string, error) {
	m.lastPrompt = prompt
	if m.response != "" {
		return m.response, nil
	}
	return "**Backpropagation** is a fundamental gradient descent algorithm.\n\n- Computes loss gradients with respect to each weight using chain rule.\n- Enables deep neural network training efficiently.\n\n**Key Takeaway**: Propagate errors backwards to update weights in direction of steepest descent.", nil
}

func TestGenerateTopicStudyNote(t *testing.T) {
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	t.Setenv("STUDYLOOP_NOTES_DIR", notesDir)

	dbPath := filepath.Join(tempDir, "test_study_notes_svc.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	// Create test topic and notebook
	topicID := "topic-backprop"
	if err := repo.EnsureTopicWithStatus(topicID, "Backpropagation", "reading"); err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
	if err := repo.CreateNotebook("nb-test", "Machine Learning", "/tmp/ml.pdf", "pdf", topicID, "hash1", 20, ""); err != nil {
		t.Fatalf("failed to create notebook: %v", err)
	}

	// Insert chunk
	_, err = repo.ExecForTest(
		`INSERT INTO chunks (id, topic_id, chunk_text, compressed_text, importance_score, page_num) VALUES (?, ?, ?, ?, ?, ?)`,
		"chunk-1", topicID,
		"Backpropagation calculates gradients of loss function with respect to weights using chain rule.",
		"Backpropagation: calculates loss gradients via chain rule.",
		0.95, 12,
	)
	if err != nil {
		t.Fatalf("failed to save chunk: %v", err)
	}

	mockLLM := &mockNotesLLM{
		dummyLLM: dummyLLM{model: "gpt-4.1-mini"},
	}

	svc := &StudyService{
		repo:            repo,
		fastLLMProvider: mockLLM,
	}

	note, err := svc.GenerateTopicStudyNote(topicID, "nb-test")
	if err != nil {
		t.Fatalf("GenerateTopicStudyNote failed: %v", err)
	}

	if note == nil {
		t.Fatalf("expected generated note, got nil")
	}

	if note.TopicID != topicID {
		t.Errorf("expected topic_id %s, got %s", topicID, note.TopicID)
	}

	if !strings.Contains(note.Content, "Backpropagation") {
		t.Errorf("expected note content to contain 'Backpropagation', got: %s", note.Content)
	}

	// Verify file was written to disk
	expectedPath := filepath.Join(notesDir, "Machine Learning", "Backpropagation", "chapter_summary.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("expected markdown file on disk at %s: %v", expectedPath, err)
	}
	if !strings.Contains(string(data), "notebook_title: \"Machine Learning\"") {
		t.Errorf("expected frontmatter notebook_title in file, got: %s", string(data))
	}
	if !strings.Contains(string(data), "Backpropagation") {
		t.Errorf("expected content in file, got: %s", string(data))
	}

	// Verify assets folder was created
	assetsPath := filepath.Join(notesDir, "Machine Learning", "Backpropagation", "assets")
	if stat, err := os.Stat(assetsPath); err != nil || !stat.IsDir() {
		t.Fatalf("expected assets directory to exist at %s", assetsPath)
	}
}

func TestGenerateTopicStudyNoteForRange_SessionSlots(t *testing.T) {
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	t.Setenv("STUDYLOOP_NOTES_DIR", notesDir)

	dbPath := filepath.Join(tempDir, "test_study_notes_range.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	topicID := "topic-dl"
	if err := repo.EnsureTopicWithStatus(topicID, "Deep Learning", "reading"); err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
	if err := repo.CreateNotebook("nb-dl", "Deep Learning", "/tmp/dl.pdf", "pdf", topicID, "hash1", 50, ""); err != nil {
		t.Fatalf("failed to create notebook: %v", err)
	}

	// Insert chunk for Session 1 (pages 1-10)
	_, _ = repo.ExecForTest(
		`INSERT INTO chunks (id, topic_id, chunk_text, compressed_text, importance_score, page_num) VALUES (?, ?, ?, ?, ?, ?)`,
		"chunk-s1", topicID, "Session 1 full text", "Session 1: Neural net basics", 0.9, 5,
	)
	// Insert chunk for Session 2 (pages 11-20)
	_, _ = repo.ExecForTest(
		`INSERT INTO chunks (id, topic_id, chunk_text, compressed_text, importance_score, page_num) VALUES (?, ?, ?, ?, ?, ?)`,
		"chunk-s2", topicID, "Session 2 full text", "Session 2: Backpropagation algorithm", 0.9, 15,
	)

	mockLLM := &mockNotesLLM{
		dummyLLM: dummyLLM{model: "gpt-4.1-mini"},
	}
	svc := &StudyService{
		repo:            repo,
		fastLLMProvider: mockLLM,
	}

	// 1. Generate for Session 1 (pages 1-10)
	mockLLM.response = "**Basics**: Perceptrons and activation functions."
	note1, err := svc.GenerateTopicStudyNoteForRange(topicID, "nb-dl", 1, 10)
	if err != nil {
		t.Fatalf("Session 1 generation failed: %v", err)
	}
	if !strings.Contains(note1.Content, "Perceptrons") {
		t.Fatalf("expected Session 1 content, got: %s", note1.Content)
	}
	if note1.StartPage != 1 || note1.EndPage != 10 {
		t.Fatalf("expected StartPage=1 EndPage=10, got %d, %d", note1.StartPage, note1.EndPage)
	}

	// 2. Generate for Session 2 (pages 11-20) -> separate slot
	mockLLM.response = "**Backprop**: Gradient descent and chain rule."
	note2, err := svc.GenerateTopicStudyNoteForRange(topicID, "nb-dl", 11, 20)
	if err != nil {
		t.Fatalf("Session 2 generation failed: %v", err)
	}
	if !strings.Contains(note2.Content, "Gradient descent") {
		t.Errorf("expected Session 2 content, got: %s", note2.Content)
	}
	if note2.StartPage != 11 || note2.EndPage != 20 {
		t.Errorf("expected StartPage=11 EndPage=20, got %d, %d", note2.StartPage, note2.EndPage)
	}

	// 3. Verify slots returned from disk
	slots, err := svc.GetTopicStudyNoteSlots(topicID)
	if err != nil {
		t.Fatalf("failed to get slots: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 distinct session slots, got %d", len(slots))
	}
	if slots[0].StartPage != 1 || slots[0].EndPage != 10 {
		t.Errorf("expected first slot (1, 10), got (%d, %d)", slots[0].StartPage, slots[0].EndPage)
	}
	if slots[1].StartPage != 11 || slots[1].EndPage != 20 {
		t.Errorf("expected second slot (11, 20), got (%d, %d)", slots[1].StartPage, slots[1].EndPage)
	}

	// 4. Update note content on disk
	updatedText := "**Updated Basics**: Multi-layer perceptrons."
	if err := svc.UpdateTopicStudyNote(topicID, 1, 10, updatedText); err != nil {
		t.Fatalf("failed to update note on disk: %v", err)
	}

	refetched, err := svc.GetTopicStudyNoteForRange(topicID, 1, 10)
	if err != nil {
		t.Fatalf("failed to refetch note: %v", err)
	}
	if refetched.Content != updatedText {
		t.Errorf("expected updated content %q, got %q", updatedText, refetched.Content)
	}

	// 5. Mark reviewed
	beforeReview := time.Now().Unix() - 1
	if err := svc.MarkTopicReviewed(topicID, 1, 10); err != nil {
		t.Fatalf("failed to mark reviewed: %v", err)
	}

	afterReview, err := svc.GetTopicStudyNoteForRange(topicID, 1, 10)
	if err != nil {
		t.Fatalf("failed to refetch note: %v", err)
	}
	if afterReview.LastReviewedAt < beforeReview {
		t.Errorf("expected last_reviewed_at >= %d, got %d", beforeReview, afterReview.LastReviewedAt)
	}
}
