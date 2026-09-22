package db

import (
	"os"
	"testing"

	"ai-tutor/internal/models"
)

func TestGamificationRepo(t *testing.T) {
	tempDB := "test_gamification_repo.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// 1. Get initial profile
	prof, err := repo.GetGamificationProfile()
	if err != nil {
		t.Fatalf("GetGamificationProfile failed: %v", err)
	}
	if prof.UserID != 1 || prof.TotalXP != 0 || prof.CurrentTitle != "The Apprentice I" || prof.Level != 1 || prof.StreakFreezesOwned != 1 {
		t.Fatalf("unexpected default profile: %+v", prof)
	}

	// 2. Add XP and Coins (1500 XP unlocks The Scholar I)
	updatedProf, unlockedTitle, err := repo.AddXPAndCoins(1500, 60)
	if err != nil {
		t.Fatalf("AddXPAndCoins failed: %v", err)
	}
	if updatedProf.TotalXP != 1500 || updatedProf.Coins != 60 || updatedProf.Level != 6 {
		t.Fatalf("unexpected updated values: %+v", updatedProf)
	}
	if updatedProf.CurrentTitle != "The Scholar I" {
		t.Fatalf("expected title 'The Scholar I', got %q", updatedProf.CurrentTitle)
	}
	if unlockedTitle != "The Scholar I" {
		t.Fatalf("expected unlockedTitle 'The Scholar I', got %q", unlockedTitle)
	}

	// 3. Create Pending Loot Box
	box := models.PendingLootBox{
		ID:           "box-123",
		TaskID:       "task-1",
		BoxTier:      "SILVER",
		RewardType:   "XP",
		RewardAmount: 75,
	}
	if err := repo.CreatePendingLootBox(box); err != nil {
		t.Fatalf("CreatePendingLootBox failed: %v", err)
	}

	boxes, err := repo.GetUnopenedLootBoxes()
	if err != nil {
		t.Fatalf("GetUnopenedLootBoxes failed: %v", err)
	}
	if len(boxes) != 1 || boxes[0].ID != "box-123" {
		t.Fatalf("expected 1 unopened box, got: %+v", boxes)
	}

	// 4. Claim Loot Box
	claimedBox, afterClaimProf, err := repo.ClaimLootBox("box-123")
	if err != nil {
		t.Fatalf("ClaimLootBox failed: %v", err)
	}
	if !claimedBox.Opened {
		t.Fatalf("claimed box opened flag should be true")
	}
	if afterClaimProf.TotalXP != 1500+75 {
		t.Fatalf("expected total XP %d, got %d", 1500+75, afterClaimProf.TotalXP)
	}

	// 5. Buy Streak Freeze (Cost: 150)
	// Add enough coins for testing
	_, _, err = repo.AddXPAndCoins(0, 500)
	if err != nil {
		t.Fatalf("AddXPAndCoins failed: %v", err)
	}

	nowUnix := int64(1700000000)
	afterFreezeProf, err := repo.BuyStreakFreeze(150, nowUnix)
	if err != nil {
		t.Fatalf("BuyStreakFreeze failed: %v", err)
	}
	if afterFreezeProf.Coins != 560-150 { // (60 + 500) - 150 = 410
		t.Fatalf("expected coins 410, got %d", afterFreezeProf.Coins)
	}
	if afterFreezeProf.StreakFreezesOwned != 2 { // 1 + 1 = 2
		t.Fatalf("expected 2 streak freezes, got %d", afterFreezeProf.StreakFreezesOwned)
	}

	// Max capacity test (already has 2)
	_, err = repo.BuyStreakFreeze(150, nowUnix+8*86400)
	if err != ErrFreezeInventoryFull {
		t.Fatalf("expected ErrFreezeInventoryFull, got %v", err)
	}

	// Consume one freeze
	consumedProf, err := repo.ConsumeStreakFreeze("2026-09-20")
	if err != nil {
		t.Fatalf("ConsumeStreakFreeze failed: %v", err)
	}
	if consumedProf.StreakFreezesOwned != 1 {
		t.Fatalf("expected 1 streak freeze owned after consumption, got %d", consumedProf.StreakFreezesOwned)
	}

	// Weekly rate limit test (buying within 7 days of last purchase)
	_, err = repo.BuyStreakFreeze(150, nowUnix+2*86400)
	if err != ErrFreezeWeeklyLimit {
		t.Fatalf("expected ErrFreezeWeeklyLimit, got %v", err)
	}

	// Successful purchase after 7 days
	afterWeeklyProf, err := repo.BuyStreakFreeze(150, nowUnix+8*86400)
	if err != nil {
		t.Fatalf("expected successful purchase after 7 days, got: %v", err)
	}
	if afterWeeklyProf.StreakFreezesOwned != 2 {
		t.Fatalf("expected 2 streak freezes after second buy, got %d", afterWeeklyProf.StreakFreezesOwned)
	}

	// 6. Test UnlockCosmetic with coins
	afterUnlockProf, err := repo.UnlockCosmetic("dark-emerald", 75)
	if err != nil {
		t.Fatalf("UnlockCosmetic failed: %v", err)
	}
	if afterUnlockProf.Coins != 410-150-75 { // 410 - 150 - 75 = 185
		t.Fatalf("expected coins 185, got %d", afterUnlockProf.Coins)
	}

	// 7. Test IncrementStat & Achievement auto-unlock
	if err := repo.IncrementStat("quizzes_passed", 5); err != nil {
		t.Fatalf("IncrementStat failed: %v", err)
	}

	store, err := repo.GetGamificationStore()
	if err != nil {
		t.Fatalf("GetGamificationStore failed: %v", err)
	}

	// Check if light-monochrome was auto unlocked by quiz_master achievement
	foundMonochrome := false
	for _, th := range store.Themes {
		if th.ID == "light-monochrome" && th.Unlocked {
			foundMonochrome = true
			break
		}
	}
	if !foundMonochrome {
		t.Fatalf("expected light-monochrome to be unlocked by quiz_master achievement")
	}
}

func TestGamificationStoreAutoReconcilesFromSQL(t *testing.T) {
	tempDB := "test_gamification_reconcile.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := Init(tempDB, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Seed completed reading task in study_queue
	_, err = repo.db.Exec(`
		INSERT INTO notebooks (id, title, file_path) VALUES ('nb-1', 'Test NB', '/path/test.pdf');
		INSERT INTO study_queue (id, notebook_id, task_type, status) VALUES ('sq-1', 'nb-1', 'READING', 'COMPLETED');
	`)
	if err != nil {
		t.Fatalf("failed to seed reading task: %v", err)
	}

	// Seed passed quiz attempt
	_, err = repo.db.Exec(`
		INSERT INTO quiz_attempts (id, task_id, score, passed, answers_json, completed_at) VALUES ('qa-1', 'sq-1', 100, 1, '[]', 1000);
	`)
	if err != nil {
		t.Fatalf("failed to seed quiz attempt: %v", err)
	}

	store, err := repo.GetGamificationStore()
	if err != nil {
		t.Fatalf("GetGamificationStore failed: %v", err)
	}

	var firstStepAch, quizStarterAch *models.Achievement
	for i := range store.Achievements {
		if store.Achievements[i].ID == "first_step" {
			firstStepAch = &store.Achievements[i]
		}
		if store.Achievements[i].ID == "quiz_starter" {
			quizStarterAch = &store.Achievements[i]
		}
	}

	if firstStepAch == nil || firstStepAch.CurrentValue != 1 || firstStepAch.TargetValue != 2 || firstStepAch.Title != "First Steps II" {
		t.Fatalf("expected First Steps II achievement (1/2), got %+v", firstStepAch)
	}
	if quizStarterAch == nil || quizStarterAch.CurrentValue != 1 || quizStarterAch.TargetValue != 2 || quizStarterAch.Title != "Quiz Starter II" {
		t.Fatalf("expected Quiz Starter II achievement (1/2), got %+v", quizStarterAch)
	}

	// Verify Tier II achievements clear RewardItem
	_ = repo.IncrementStat("reading_sessions", 5)
	storeAfterTier2, err := repo.GetGamificationStore()
	if err != nil {
		t.Fatalf("GetGamificationStore after tier 2 failed: %v", err)
	}
	for _, a := range storeAfterTier2.Achievements {
		if a.ID == "night_scholar" {
			if a.Title != "Night Scholar II" {
				t.Fatalf("expected Night Scholar II, got %s", a.Title)
			}
			if a.RewardItem != "" {
				t.Fatalf("expected empty RewardItem on Night Scholar II, got %s", a.RewardItem)
			}
		}
	}
}

