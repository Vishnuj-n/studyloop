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

func TestProfileScopedSettingsPersistence(t *testing.T) {
	tempDB := "test_profile_scoped_settings.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() {
		_ = repo.Close()
	}()

	// 1. Create two profiles
	p1 := models.StudyProfile{
		ID:         "prof-a",
		Name:       "Profile A",
		DeadlineAt: time.Now().Add(24 * time.Hour).Unix(),
	}
	if err := repo.CreateProfile(p1); err != nil {
		t.Fatalf("CreateProfile A failed: %v", err)
	}

	p2 := models.StudyProfile{
		ID:         "prof-b",
		Name:       "Profile B",
		DeadlineAt: time.Now().Add(48 * time.Hour).Unix(),
	}
	if err := repo.CreateProfile(p2); err != nil {
		t.Fatalf("CreateProfile B failed: %v", err)
	}

	// 2. Set Profile A as active and update profile-scoped settings (theme, word budget, quiz, tutor_style)
	s, err := repo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	s.ActiveProfileID = "prof-a"
	s.Theme = "dark-obsidian"
	s.TargetSessionWords = 4500
	s.MinSessionWords = 1500
	s.QuizQuestionCount = 10
	s.QuizPassingScore = 80
	s.TutorStyle = "direct"
	s.DefaultRemedialStrategy = "CLASSIC"
	s.MaxFlashcardsPerSession = 40
	s.MaxActiveNotebooks = 6

	if err := repo.UpdateUserSettings(*s); err != nil {
		t.Fatalf("UpdateUserSettings for prof-a failed: %v", err)
	}

	// 3. Fetch effective settings for Profile A
	effA, err := repo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if effA.Theme != "dark-obsidian" {
		t.Errorf("expected Profile A theme 'dark-obsidian', got %q", effA.Theme)
	}
	if effA.TargetSessionWords != 4500 {
		t.Errorf("expected Profile A TargetSessionWords 4500, got %d", effA.TargetSessionWords)
	}
	if effA.TutorStyle != "direct" {
		t.Errorf("expected Profile A TutorStyle 'direct', got %q", effA.TutorStyle)
	}
	if effA.QuizQuestionCount != 10 {
		t.Errorf("expected Profile A QuizQuestionCount 10, got %d", effA.QuizQuestionCount)
	}

	// Direct DB check on study_profiles for prof-a
	var profATheme, profATutorStyle string
	var profATargetWords int
	err = repo.db.QueryRow("SELECT theme, tutor_style, target_session_words FROM study_profiles WHERE id = 'prof-a'").
		Scan(&profATheme, &profATutorStyle, &profATargetWords)
	if err != nil {
		t.Fatalf("query study_profiles for prof-a failed: %v", err)
	}
	if profATheme != "dark-obsidian" {
		t.Errorf("expected study_profiles row theme 'dark-obsidian', got %q", profATheme)
	}
	if profATutorStyle != "direct" {
		t.Errorf("expected study_profiles row tutor_style 'direct', got %q", profATutorStyle)
	}
	if profATargetWords != 4500 {
		t.Errorf("expected study_profiles row target_session_words 4500, got %d", profATargetWords)
	}

	// 4. Switch to Profile B and set different settings
	sB, err := repo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	sB.ActiveProfileID = "prof-b"
	sB.Theme = "light-warm"
	sB.TargetSessionWords = 2000
	sB.TutorStyle = "socratic"
	if err := repo.UpdateUserSettings(*sB); err != nil {
		t.Fatalf("UpdateUserSettings for prof-b failed: %v", err)
	}

	effB, err := repo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if effB.Theme != "light-warm" {
		t.Errorf("expected Profile B theme 'light-warm', got %q", effB.Theme)
	}
	if effB.TargetSessionWords != 2000 {
		t.Errorf("expected Profile B TargetSessionWords 2000, got %d", effB.TargetSessionWords)
	}

	// 5. Switch active profile back to Profile A via direct user_settings update
	_, err = repo.db.Exec("UPDATE user_settings SET active_profile_id = 'prof-a' WHERE id = 1")
	if err != nil {
		t.Fatalf("Switch active profile to prof-a failed: %v", err)
	}

	effAReturn, err := repo.GetUserSettings()
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if effAReturn.Theme != "dark-obsidian" {
		t.Errorf("expected restored Profile A theme 'dark-obsidian', got %q", effAReturn.Theme)
	}
	if effAReturn.TargetSessionWords != 4500 {
		t.Errorf("expected restored Profile A TargetSessionWords 4500, got %d", effAReturn.TargetSessionWords)
	}
	if effAReturn.TutorStyle != "direct" {
		t.Errorf("expected restored Profile A TutorStyle 'direct', got %q", effAReturn.TutorStyle)
	}
}

func TestMultiProfileFlashcardInheritance(t *testing.T) {
	tempDB := "test_profile_inheritance.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = repo.Close() }()

	now := time.Now().Unix()

	// 1. Create profiles: November (active), October, Kannada
	profNov := models.StudyProfile{ID: "prof-nov", Name: "November Placement", DeadlineAt: now + 3600*24*30}
	profOct := models.StudyProfile{ID: "prof-oct", Name: "October Midterms", DeadlineAt: now + 3600*24*60}
	profKan := models.StudyProfile{ID: "prof-kan", Name: "Kannada Language", DeadlineAt: now + 3600*24*90}

	for _, p := range []models.StudyProfile{profNov, profOct, profKan} {
		if err := repo.CreateProfile(p); err != nil {
			t.Fatalf("CreateProfile %s failed: %v", p.ID, err)
		}
	}

	// 2. Set active profile to November
	if err := repo.UpdateUserSettings(models.UserSettings{ActiveProfileID: "prof-nov", MaxFlashcardsPerSession: 30}); err != nil {
		t.Fatalf("UpdateUserSettings failed: %v", err)
	}

	// 3. Create notebooks and topics for each profile
	topNov, topOct, topKan := "top-nov", "top-oct", "top-kan"
	for _, top := range []string{topNov, topOct, topKan} {
		if err := repo.EnsureTopic(top, top+" Title"); err != nil {
			t.Fatalf("EnsureTopic %s failed: %v", top, err)
		}
	}

	if err := repo.CreateNotebook("nb-nov", "Nov Book", "/path/nov.pdf", "pdf", topNov, "", 10, "prof-nov"); err != nil {
		t.Fatalf("CreateNotebook Nov failed: %v", err)
	}
	_ = repo.EnsureNotebookTopic("nb-nov", topNov)
	if err := repo.CreateNotebook("nb-oct", "Oct Book", "/path/oct.pdf", "pdf", topOct, "", 8, "prof-oct"); err != nil {
		t.Fatalf("CreateNotebook Oct failed: %v", err)
	}
	_ = repo.EnsureNotebookTopic("nb-oct", topOct)
	if err := repo.CreateNotebook("nb-kan", "Kan Book", "/path/kan.pdf", "pdf", topKan, "", 5, "prof-kan"); err != nil {
		t.Fatalf("CreateNotebook Kan failed: %v", err)
	}
	_ = repo.EnsureNotebookTopic("nb-kan", topKan)

	// 4. Create due flashcards: 10 Nov cards, 12 Oct cards, 5 Kan cards
	var novCards, octCards, kanCards []models.Flashcard
	stateMap := make(map[string]models.FlashcardState)
	for i := 1; i <= 10; i++ {
		cid := fmt.Sprintf("c-nov-%d", i)
		novCards = append(novCards, models.Flashcard{ID: cid, TopicID: topNov, Prompt: cid, Answer: "ans", DueAt: now - 100})
		stateMap[cid] = models.FlashcardState{StateCode: 1}
	}
	for i := 1; i <= 12; i++ {
		cid := fmt.Sprintf("c-oct-%d", i)
		octCards = append(octCards, models.Flashcard{ID: cid, TopicID: topOct, Prompt: cid, Answer: "ans", DueAt: now - 200})
		stateMap[cid] = models.FlashcardState{StateCode: 1}
	}
	for i := 1; i <= 5; i++ {
		cid := fmt.Sprintf("c-kan-%d", i)
		kanCards = append(kanCards, models.Flashcard{ID: cid, TopicID: topKan, Prompt: cid, Answer: "ans", DueAt: now - 300})
		stateMap[cid] = models.FlashcardState{StateCode: 1}
	}

	if err := repo.CreateFlashcards(topNov, novCards, stateMap); err != nil {
		t.Fatalf("CreateFlashcards nov failed: %v", err)
	}
	if err := repo.CreateFlashcards(topOct, octCards, stateMap); err != nil {
		t.Fatalf("CreateFlashcards oct failed: %v", err)
	}
	if err := repo.CreateFlashcards(topKan, kanCards, stateMap); err != nil {
		t.Fatalf("CreateFlashcards kan failed: %v", err)
	}

	// 5. Without inheritance: only Nov cards (10) are due for active profile
	dueInitial, err := repo.QueryDueReviewCards(now)
	if err != nil {
		t.Fatalf("QueryDueReviewCards initial failed: %v", err)
	}
	if dueInitial != 10 {
		t.Fatalf("expected 10 due cards without inheritance, got %d", dueInitial)
	}

	// 6. Inherit October into November
	if err := repo.SetProfileFlashcardInheritance("prof-nov", []string{"prof-oct"}); err != nil {
		t.Fatalf("SetProfileFlashcardInheritance failed: %v", err)
	}

	// 7. Verify allowed profile IDs helper
	allowed, err := repo.GetAllowedFlashcardProfileIDs("prof-nov")
	if err != nil {
		t.Fatalf("GetAllowedFlashcardProfileIDs failed: %v", err)
	}
	if len(allowed) != 2 || allowed[0] != "prof-nov" || allowed[1] != "prof-oct" {
		t.Fatalf("unexpected allowed profiles: %+v", allowed)
	}

	// 8. With October inherited: due cards = 10 (Nov) + 12 (Oct) = 22 cards. Never Kannada (5)!
	dueWithInheritance, err := repo.QueryDueReviewCards(now)
	if err != nil {
		t.Fatalf("QueryDueReviewCards with inheritance failed: %v", err)
	}
	if dueWithInheritance != 22 {
		t.Fatalf("expected 22 due cards with October inherited (10 Nov + 12 Oct, excluding Kan), got %d", dueWithInheritance)
	}

	// 9. Verify next due notebook selects from the inherited set (October has 12 due vs Nov 10)
	nextNB, dueCount, err := repo.GetNextDueReviewNotebook(now)
	if err != nil {
		t.Fatalf("GetNextDueReviewNotebook failed: %v", err)
	}
	if nextNB != "nb-oct" || dueCount != 12 {
		t.Fatalf("expected next notebook nb-oct with 12 cards, got notebook=%s dueCount=%d", nextNB, dueCount)
	}

	// 10. Verify Deck Overview with inheritance
	overview, err := repo.GetAllFlashcardsDeckOverview("prof-nov")
	if err != nil {
		t.Fatalf("GetAllFlashcardsDeckOverview failed: %v", err)
	}
	if overview.Metrics.TotalCards != 22 {
		t.Fatalf("expected 22 total cards in overview, got %d", overview.Metrics.TotalCards)
	}
	if len(overview.Profiles) != 3 {
		t.Fatalf("expected 3 profiles in summary breakdown, got %d", len(overview.Profiles))
	}

	// Verify profile summary flags
	var foundNov, foundOct, foundKan bool
	for _, ps := range overview.Profiles {
		if ps.ProfileID == "prof-nov" {
			foundNov = true
			if !ps.IsActive {
				t.Errorf("expected prof-nov to be active")
			}
			if ps.DueCards != 10 {
				t.Errorf("expected prof-nov due cards 10, got %d", ps.DueCards)
			}
		}
		if ps.ProfileID == "prof-oct" {
			foundOct = true
			if !ps.IsInherited {
				t.Errorf("expected prof-oct to be marked inherited")
			}
			if ps.DueCards != 12 {
				t.Errorf("expected prof-oct due cards 12, got %d", ps.DueCards)
			}
		}
		if ps.ProfileID == "prof-kan" {
			foundKan = true
			if ps.IsInherited {
				t.Errorf("expected prof-kan to NOT be marked inherited")
			}
			if ps.DueCards != 5 {
				t.Errorf("expected prof-kan due cards 5, got %d", ps.DueCards)
			}
		}
	}
	if !foundNov || !foundOct || !foundKan {
		t.Fatalf("missing profile in overview summary")
	}

	// 11. Disable inheritance (toggle off October)
	if err := repo.SetProfileFlashcardInheritance("prof-nov", []string{}); err != nil {
		t.Fatalf("SetProfileFlashcardInheritance clear failed: %v", err)
	}
	dueAfterRemoval, err := repo.QueryDueReviewCards(now)
	if err != nil {
		t.Fatalf("QueryDueReviewCards after removal failed: %v", err)
	}
	if dueAfterRemoval != 10 {
		t.Fatalf("expected 10 due cards after clearing inheritance, got %d", dueAfterRemoval)
	}
}



