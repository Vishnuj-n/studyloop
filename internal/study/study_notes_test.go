package study

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
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
	expectedPath := svc.GetNoteFilePath(topicID, "nb-test", 0, 0)
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
	_, assetsPath := svc.GetTopicNoteFolder(topicID, "nb-test")
	if stat, err := os.Stat(assetsPath); err != nil || !stat.IsDir() {
		t.Fatalf("expected assets directory to exist at %s", assetsPath)
	}
}

func TestMarkTopicReviewed_NoExistingNote(t *testing.T) {
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	t.Setenv("STUDYLOOP_NOTES_DIR", notesDir)

	dbPath := filepath.Join(tempDir, "test_study_notes_empty.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	svc := &StudyService{repo: repo}
	err = svc.MarkTopicReviewed("non-existent-topic", 1, 10)
	if err != nil {
		t.Fatalf("expected nil error on missing note, got %v", err)
	}

	note, err := svc.GetTopicStudyNoteForRange("non-existent-topic", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error getting note: %v", err)
	}
	if note != nil {
		t.Fatalf("expected no note created, got: %#v", note)
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

	// 6. Test User Notes Preservation on AI Regeneration
	userNoteContent := "**Basics**: Multi-layer perceptrons.\n\n### My Notes\n- Remember to review Figure 9.1\n- ![Pasted Diagram](assets/screenshot_123.png)"
	if err := svc.UpdateTopicStudyNote(topicID, 1, 10, userNoteContent); err != nil {
		t.Fatalf("failed to save custom user notes: %v", err)
	}

	mockLLM.response = "**Regenerated AI Basics**: Convolutional neural networks."
	regenNote, err := svc.GenerateTopicStudyNoteForRange(topicID, "nb-dl", 1, 10)
	if err != nil {
		t.Fatalf("regenerate failed: %v", err)
	}
	if !strings.Contains(regenNote.Content, "Convolutional neural networks") {
		t.Errorf("expected regenerated AI content, got: %s", regenNote.Content)
	}
	if !strings.Contains(regenNote.Content, "### My Notes") || !strings.Contains(regenNote.Content, "Figure 9.1") || !strings.Contains(regenNote.Content, "screenshot_123.png") {
		t.Errorf("expected user notes to be preserved across regeneration, got: %s", regenNote.Content)
	}

	// 7. Test SaveNoteImage and RelativeFolderPath
	relPath, relFolder, err := svc.SaveNoteImage(topicID, "nb-dl", "test_diagram.png", []byte("fake png content"))
	if err != nil {
		t.Fatalf("SaveNoteImage failed: %v", err)
	}
	if !strings.HasPrefix(relPath, "assets/test_diagram_") || !strings.HasSuffix(relPath, ".png") {
		t.Errorf("expected relPath 'assets/test_diagram_*.png', got: %s", relPath)
	}
	if relFolder == "" {
		t.Errorf("expected non-empty relative folder path, got empty")
	}
}

func TestExtractUserNotesSection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name: "ai note with My Notes",
			input: "### Core Concept & Motivation\nDistributed consensus.\n\n### My Notes\n- Custom point 1\n- Custom point 2",
			expected: "### My Notes\n- Custom point 1\n- Custom point 2",
		},
		{
			name: "pure manual user note",
			input: "- Read pages 10-20\n- Diagram pasted: ![alt](assets/pic.png)",
			expected: "### My Notes\n- Read pages 10-20\n- Diagram pasted: ![alt](assets/pic.png)",
		},
		{
			name: "ai note without user notes",
			input: "### Core Concept & Motivation\nLinearizability.\n\n### Key Mechanisms & Principles\n- 2PC\n- Raft",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractUserNotesSection(tc.input)
			if result != tc.expected {
				t.Errorf("ExtractUserNotesSection() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestReorderTopicStudyNotes(t *testing.T) {
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	t.Setenv("STUDYLOOP_NOTES_DIR", notesDir)

	dbPath := filepath.Join(tempDir, "test_reorder.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	defer repo.Close()

	topicID := "topic-reorder-test"
	_ = repo.EnsureTopicWithStatus(topicID, "Reorder Chapter", "reading")
	_ = repo.CreateNotebook("nb-reorder", "Reorder Book", "/tmp/reorder.pdf", "pdf", topicID, "hash_reorder", 100, "")

	svc := &StudyService{
		repo: repo,
	}

	// Create 3 notes: (0,0), (1, 10), (11, 20)
	if err := svc.UpdateTopicStudyNote(topicID, 0, 0, "# Chapter Summary"); err != nil {
		t.Fatalf("failed creating note 0-0: %v", err)
	}
	if err := svc.UpdateTopicStudyNote(topicID, 1, 10, "Pages 1 to 10 notes"); err != nil {
		t.Fatalf("failed creating note 1-10: %v", err)
	}
	if err := svc.UpdateTopicStudyNote(topicID, 11, 20, "Pages 11 to 20 notes"); err != nil {
		t.Fatalf("failed creating note 11-20: %v", err)
	}

	// Initial order should be: (0,0), (1,10), (11,20)
	slots, err := svc.GetTopicStudyNoteSlots(topicID)
	if err != nil || len(slots) != 3 {
		t.Fatalf("expected 3 slots, got %d, err: %v", len(slots), err)
	}
	if slots[0].StartPage != 0 || slots[1].StartPage != 1 || slots[2].StartPage != 11 {
		t.Errorf("unexpected initial slot ordering: %d, %d, %d", slots[0].StartPage, slots[1].StartPage, slots[2].StartPage)
	}

	// Reorder: Move (11,20) first, then (0,0), then (1,10)
	reordered := []models.NoteSlotRange{
		{StartPage: 11, EndPage: 20},
		{StartPage: 0, EndPage: 0},
		{StartPage: 1, EndPage: 10},
	}
	if err := svc.ReorderTopicStudyNotes(topicID, reordered); err != nil {
		t.Fatalf("ReorderTopicStudyNotes failed: %v", err)
	}

	// Fetch again and verify new order
	afterSlots, err := svc.GetTopicStudyNoteSlots(topicID)
	if err != nil || len(afterSlots) != 3 {
		t.Fatalf("expected 3 slots after reorder, got %d, err: %v", len(afterSlots), err)
	}
	if afterSlots[0].StartPage != 11 || afterSlots[0].EndPage != 20 {
		t.Errorf("expected first slot to be 11-20, got %d-%d", afterSlots[0].StartPage, afterSlots[0].EndPage)
	}
	if afterSlots[1].StartPage != 0 || afterSlots[1].EndPage != 0 {
		t.Errorf("expected second slot to be 0-0, got %d-%d", afterSlots[1].StartPage, afterSlots[1].EndPage)
	}
	if afterSlots[2].StartPage != 1 || afterSlots[2].EndPage != 10 {
		t.Errorf("expected third slot to be 1-10, got %d-%d", afterSlots[2].StartPage, afterSlots[2].EndPage)
	}
}
