package study

import (
	"path/filepath"
	"strings"
	"testing"

	"ai-tutor/internal/db"
)

type mockNotesLLM struct {
	dummyLLM
	response string
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

	// Verify prompt formatting included the topic title and compressed chunk content
	if !strings.Contains(mockLLM.lastPrompt, "Backpropagation") {
		t.Errorf("expected prompt to contain topic title 'Backpropagation'")
	}
	if !strings.Contains(mockLLM.lastPrompt, "Backpropagation: calculates loss gradients via chain rule.") {
		t.Errorf("expected prompt to prioritize compressed chunk text")
	}
}

func TestGenerateTopicStudyNoteForRange_SessionAccumulation(t *testing.T) {
	tempDir := t.TempDir()
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
	if !strings.Contains(note1.Content, "### Pages 1–10") || !strings.Contains(note1.Content, "Perceptrons") {
		t.Fatalf("expected Session 1 content with header, got: %s", note1.Content)
	}

	// 2. Generate for Session 2 (pages 11-20) -> should accumulate
	mockLLM.response = "**Backprop**: Gradient descent and chain rule."
	note2, err := svc.GenerateTopicStudyNoteForRange(topicID, "nb-dl", 11, 20)
	if err != nil {
		t.Fatalf("Session 2 generation failed: %v", err)
	}

	if !strings.Contains(note2.Content, "### Pages 1–10") {
		t.Errorf("expected Session 1 section to be preserved, got: %s", note2.Content)
	}
	if !strings.Contains(note2.Content, "### Pages 11–20") {
		t.Errorf("expected Session 2 section to be present, got: %s", note2.Content)
	}
	if !strings.Contains(note2.Content, "Perceptrons") || !strings.Contains(note2.Content, "Gradient descent") {
		t.Errorf("expected both sessions' content to accumulate in note, got: %s", note2.Content)
	}
}
