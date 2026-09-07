package app

import (
	"os"
	"path/filepath"
	"testing"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	"ai-tutor/internal/notebook"
)

func TestImportAnkiDeckStandalone(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "studyloop-test.db")
	repo, err := db.Init(tempDB, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	uploadDir := t.TempDir()
	externalDir := t.TempDir()
	mockApkg := filepath.Join(externalDir, "sample.apkg")
	if err := os.WriteFile(mockApkg, []byte("fake-anki-content"), 0o644); err != nil {
		t.Fatalf("failed to write mock apkg: %v", err)
	}

	// Verify the database ingestion and FSRS initial states
	now := int64(1700000000)
	cards := []models.Flashcard{
		{ID: "c1", TopicID: "topic-anki", Prompt: "Hola", Answer: "Hello", DueAt: now},
		{ID: "c2", TopicID: "topic-anki", Prompt: "Adios", Answer: "Goodbye", DueAt: now},
	}
	states := map[string]models.FlashcardState{
		"c1": {Stability: 2.0, Difficulty: 5.0, StateCode: 2},
		"c2": {Stability: 2.0, Difficulty: 5.0, StateCode: 2},
	}

	// Create topic as completed so it never generates reading tasks
	_ = repo.EnsureTopicWithStatus("topic-anki", "Test Spanish Deck", "completed")
	createdCards, existing, err := repo.GetOrCreateFlashcardsForTopic("topic-anki", cards, states)
	if err != nil {
		t.Fatalf("GetOrCreateFlashcardsForTopic failed: %v", err)
	}
	if existing {
		t.Errorf("expected existing=false, got true")
	}
	if len(createdCards) != 2 {
		t.Errorf("expected 2 cards, got %d", len(createdCards))
	}

	// Verify that standalone notebook with page_count = 0 creates 0 reading tasks
	err = repo.CreateNotebook("nb-anki", "Test Spanish Deck", mockApkg, "anki", "topic-anki", "hash123", 0, "")
	if err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	_ = repo.LinkNotebookTopics("nb-anki", []string{"topic-anki"})

	// Ensure reading tasks: should be 0 because page_count is 0, topic is completed, and topic end_page is 0
	_ = repo.EnsurePendingReadingTaskForNotebook("nb-anki", 300)
	tasks, err := repo.GetAllPendingTasks()
	if err != nil {
		t.Fatalf("GetAllPendingTasks failed: %v", err)
	}
	for _, task := range tasks {
		if task.NotebookID == "nb-anki" && task.TaskType == models.StudyTaskTypeReading {
			t.Errorf("unexpected reading task generated for standalone anki deck: %+v", task)
		}
	}

	// Create test App to verify DeleteNotebook safety
	app := &App{
		repo:            repo,
		notebookService: notebook.NewService(uploadDir),
	}
	// Verify deleting notebook leaves the external mockApkg untouched
	delResp := app.DeleteNotebook("nb-anki")
	if delResp["error"] != nil {
		t.Fatalf("DeleteNotebook failed: %v", delResp["error"])
	}
	if _, err := os.Stat(mockApkg); err != nil {
		t.Fatalf("external file in user directory was wrongly deleted: %v", err)
	}
}

func TestImportAnkiDeckPreserveHistory(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "studyloop-anki-history.db")
	repo, err := db.Init(tempDB, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	targetDue := int64(1800000000)
	cards := []models.Flashcard{
		{ID: "card-mature", TopicID: "topic-anki-hist", Prompt: "Mature Prompt", Answer: "Mature Answer", DueAt: targetDue},
		{ID: "card-new", TopicID: "topic-anki-hist", Prompt: "New Prompt", Answer: "New Answer", DueAt: 0},
	}
	states := map[string]models.FlashcardState{
		"card-mature": {
			Stability:  35.5,
			Difficulty: 4.2,
			Reps:       5,
			Lapses:     1,
			StateCode:  2,
		},
		"card-new": {
			Stability:  2.0,
			Difficulty: 5.0,
			Reps:       0,
			Lapses:     0,
			StateCode:  0,
		},
	}

	_ = repo.EnsureTopicWithStatus("topic-anki-hist", "History Deck", "completed")
	createdCards, _, err := repo.GetOrCreateFlashcardsForTopic("topic-anki-hist", cards, states)
	if err != nil {
		t.Fatalf("GetOrCreateFlashcardsForTopic failed: %v", err)
	}
	if len(createdCards) != 2 {
		t.Fatalf("expected 2 created cards, got %d", len(createdCards))
	}

	// Verify that the persisted FSRS state maintained stability and reps
	stateMap, err := repo.GetFlashcardStatesByIDs([]string{"card-mature", "card-new"})
	if err != nil {
		t.Fatalf("GetFlashcardStatesByIDs failed: %v", err)
	}
	matureState, ok := stateMap["card-mature"]
	if !ok || matureState.Stability != 35.5 || matureState.Reps != 5 {
		t.Errorf("mature card state not preserved properly: %+v", matureState)
	}
}
