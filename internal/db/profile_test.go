package db

import (
	"ai-tutor/internal/models"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProfileAndSettingsLifecycle(t *testing.T) {
	tempDB := "test_profile_lifecycle.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	testRepo = repo
	defer func() {
		if err := testRepo.Close(); err != nil {
			t.Logf("Close failed: %v", err)
		}
		testRepo = nil
	}()

	// 1. Test GetUserSettings default state
	s, err := testRepo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if s.MaxFlashcardsPerSession != 30 {
		t.Errorf("expected default 30 cards, got %d", s.MaxFlashcardsPerSession)
	}
	if s.ActiveProfileID != "" {
		t.Errorf("expected empty active profile, got %q", s.ActiveProfileID)
	}

	// 2. Test CreateProfile
	deadline := time.Now().AddDate(0, 3, 0).Unix() // 3 months from now
	p1 := models.StudyProfile{
		ID:         "prof-1",
		Name:       "UPSC prep",
		DeadlineAt: deadline,
	}
	if err := testRepo.CreateProfile(p1); err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// 3. Test GetProfiles
	profiles, err := testRepo.GetProfiles()
	if err != nil {
		t.Fatalf("GetProfiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Name != "UPSC prep" {
		t.Errorf("expected UPSC prep, got %q", profiles[0].Name)
	}

	// 4. Test UpdateUserSettings with active profile
	s.ActiveProfileID = "prof-1"
	s.MaxFlashcardsPerSession = 50
	s.SkipToReadingActive = true
	s.CloudSyncURL = "http://localhost/sync"
	s.CloudAPIToken = "secret-token"

	if err := testRepo.UpdateUserSettings(*s); err != nil {
		t.Fatalf("UpdateUserSettings failed: %v", err)
	}

	sUpdated, err := testRepo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if sUpdated.MaxFlashcardsPerSession != 50 {
		t.Errorf("expected 50 cards, got %d", sUpdated.MaxFlashcardsPerSession)
	}
	if sUpdated.ActiveProfileID != "prof-1" {
		t.Errorf("expected active profile prof-1, got %q", sUpdated.ActiveProfileID)
	}
	if !sUpdated.SkipToReadingActive {
		t.Errorf("expected skip_to_reading_active to be true")
	}
	if sUpdated.CloudSyncURL != "http://localhost/sync" {
		t.Errorf("expected cloud_sync_url to be http://localhost/sync, got %q", sUpdated.CloudSyncURL)
	}

	// 5. Test Notebook Shelf Gating (limit of 4 active textbooks)
	// Create a notebook and assign to profile
	err = testRepo.CreateNotebook("nb-1", "Polity Book", "path/1.pdf", "pdf", "", "", 10, "")
	if err != nil {
		t.Fatalf("failed to create notebook: %v", err)
	}

	err = testRepo.AssignNotebookToProfile("nb-1", "prof-1")
	if err != nil {
		t.Fatalf("failed to assign notebook to profile: %v", err)
	}

	nbAfterAssign, err := testRepo.GetNotebookByID("nb-1")
	if err != nil {
		t.Fatalf("GetNotebookByID failed: %v", err)
	}
	if nbAfterAssign.ProfileID != "prof-1" {
		t.Errorf("expected ProfileID to be 'prof-1', got %q", nbAfterAssign.ProfileID)
	}

	// Activate notebook 1
	err = testRepo.UpdateNotebookStudyStatus("nb-1", "active")
	if err != nil {
		t.Fatalf("failed to activate nb-1: %v", err)
	}

	// Create 4 more notebooks and try to activate them all (bringing total active to 5)
	for i := 2; i <= 5; i++ {
		id := fmt.Sprintf("nb-%d", i)
		err = testRepo.CreateNotebook(id, fmt.Sprintf("Book %d", i), "path.pdf", "pdf", "", "", 10, "")
		if err != nil {
			t.Fatalf("failed to create notebook %s: %v", id, err)
		}
		err = testRepo.AssignNotebookToProfile(id, "prof-1")
		if err != nil {
			t.Fatalf("failed to assign notebook %s: %v", id, err)
		}
	}

	// Activate notebooks 2, 3, 4 (total active = 4)
	for i := 2; i <= 4; i++ {
		id := fmt.Sprintf("nb-%d", i)
		err = testRepo.UpdateNotebookStudyStatus(id, "active")
		if err != nil {
			t.Fatalf("failed to activate notebook %s: %v", id, err)
		}
	}

	// Notebook 5 should fail activation since limit of 4 active is hit
	err = testRepo.UpdateNotebookStudyStatus("nb-5", "active")
	if err == nil {
		t.Errorf("expected activation to fail because profile already has 4 active notebooks")
	}

	// Update max_active_notebooks to 5 and verify notebook 5 can now activate
	settings, err := testRepo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	settings.MaxActiveNotebooks = 5
	if err := testRepo.UpdateUserSettings(*settings); err != nil {
		t.Fatalf("UpdateUserSettings failed: %v", err)
	}

	err = testRepo.UpdateNotebookStudyStatus("nb-5", "active")
	if err != nil {
		t.Errorf("expected activation to succeed after bumping max_active_notebooks to 5: %v", err)
	}

	// Set max_active_notebooks to 0 (unlimited) and verify another can activate
	settings.MaxActiveNotebooks = 0
	if err := testRepo.UpdateUserSettings(*settings); err != nil {
		t.Fatalf("UpdateUserSettings failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-6", "Book 6", "path6.pdf", "pdf", "", "", 10, ""); err != nil {
		t.Fatalf("failed to create nb-6: %v", err)
	}
	if err := testRepo.AssignNotebookToProfile("nb-6", "prof-1"); err != nil {
		t.Fatalf("failed to assign nb-6: %v", err)
	}
	if err := testRepo.UpdateNotebookStudyStatus("nb-6", "active"); err != nil {
		t.Errorf("expected activation to succeed when unlimited (0): %v", err)
	}

	// 6. Test DeleteProfile cleans up references
	if err := testRepo.DeleteProfile("prof-1"); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	// Notebooks should now have profile_id = NULL
	nb1, err := testRepo.GetNotebookByID("nb-1")
	if err != nil {
		t.Fatalf("GetNotebookByID failed: %v", err)
	}
	if nb1.ProfileID != "" {
		t.Errorf("expected profile_id to be empty after profile deletion, got %q", nb1.ProfileID)
	}
}

func TestExtensionConfig(t *testing.T) {
	tempDB := "test_extension_config.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() {
		_ = repo.Close()
	}()

	// 1. Initial default state
	cfg, err := repo.GetExtensionConfig()
	if err != nil {
		t.Fatalf("GetExtensionConfig failed: %v", err)
	}
	if cfg != "{}" {
		t.Errorf("expected default '{}', got %q", cfg)
	}

	// 2. Save custom JSON config
	testJSON := `{"audio_overview":{"voice":"en-US-JennyNeural","speed":1.25},"text_simplifier":{"level":"eli10"}}`
	if err := repo.SaveExtensionConfig(testJSON); err != nil {
		t.Fatalf("SaveExtensionConfig failed: %v", err)
	}

	// 3. Read back saved config
	readCfg, err := repo.GetExtensionConfig()
	if err != nil {
		t.Fatalf("GetExtensionConfig after save failed: %v", err)
	}
	if readCfg != testJSON {
		t.Errorf("expected %q, got %q", testJSON, readCfg)
	}
}

func TestGetProfileRemainingWords_ExcludesDormant(t *testing.T) {
	tempDB := "test_remaining_words.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = repo.Close() }()

	profID := "prof-words-test"
	if err := repo.CreateProfile(models.StudyProfile{ID: profID, Name: "Test Words Profile"}); err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Create 1 Active notebook and 1 Dormant notebook
	nbActive := "nb-active-1"
	nbDormant := "nb-dormant-1"

	if err := repo.CreateNotebook(nbActive, "Active Book", "active.pdf", "pdf", "", profID, 10, ""); err != nil {
		t.Fatalf("CreateNotebook active failed: %v", err)
	}
	if err := repo.UpdateNotebookStudyStatus(nbActive, "active"); err != nil {
		t.Fatalf("UpdateNotebookStudyStatus active failed: %v", err)
	}

	if err := repo.CreateNotebook(nbDormant, "Dormant Book", "dormant.pdf", "pdf", "", profID, 10, ""); err != nil {
		t.Fatalf("CreateNotebook dormant failed: %v", err)
	}
	if err := repo.UpdateNotebookStudyStatus(nbDormant, "dormant"); err != nil {
		t.Fatalf("UpdateNotebookStudyStatus dormant failed: %v", err)
	}

	// Create topics & chunks (10 words each)
	topActive := "top-active-1"
	topDormant := "top-dormant-1"
	_ = repo.EnsureTopic(topActive, "Active Topic")
	_ = repo.EnsureTopic(topDormant, "Dormant Topic")

	chunkTextActive := "one two three four five six seven eight nine ten"
	chunkTextDormant := "one two three four five six seven eight nine ten"

	// Insert chunks into DB directly
	_, err = repo.db.Exec(`INSERT INTO chunks (id, topic_id, chunk_text, page_num) VALUES ('c-active', ?, ?, 1)`, topActive, chunkTextActive)
	if err != nil {
		t.Fatalf("insert active chunk failed: %v", err)
	}
	_, err = repo.db.Exec(`INSERT INTO notebook_chunks (notebook_id, chunk_id, page_num) VALUES (?, 'c-active', 1)`, nbActive)
	if err != nil {
		t.Fatalf("link active chunk failed: %v", err)
	}

	_, err = repo.db.Exec(`INSERT INTO chunks (id, topic_id, chunk_text, page_num) VALUES ('c-dormant', ?, ?, 1)`, topDormant, chunkTextDormant)
	if err != nil {
		t.Fatalf("insert dormant chunk failed: %v", err)
	}
	_, err = repo.db.Exec(`INSERT INTO notebook_chunks (notebook_id, chunk_id, page_num) VALUES (?, 'c-dormant', 1)`, nbDormant)
	if err != nil {
		t.Fatalf("link dormant chunk failed: %v", err)
	}

	words, err := repo.GetProfileRemainingWords(profID)
	if err != nil {
		t.Fatalf("GetProfileRemainingWords failed: %v", err)
	}

	if words != 10 {
		t.Errorf("expected 10 remaining words (active only), got %d", words)
	}
}


