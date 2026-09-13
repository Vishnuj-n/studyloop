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

	// 5. Buy Streak Freeze
	afterFreezeProf, err := repo.BuyStreakFreeze(50)
	if err != nil {
		t.Fatalf("BuyStreakFreeze failed: %v", err)
	}
	if afterFreezeProf.Coins != 10 { // 60 - 50 = 10
		t.Fatalf("expected coins 10, got %d", afterFreezeProf.Coins)
	}
	if afterFreezeProf.StreakFreezesOwned != 2 { // 1 + 1 = 2
		t.Fatalf("expected 2 streak freezes, got %d", afterFreezeProf.StreakFreezesOwned)
	}

	// Insufficient coins test
	_, err = repo.BuyStreakFreeze(50)
	if err == nil {
		t.Fatalf("expected error buying streak freeze with insufficient coins")
	}

	// 6. Test UnlockCosmetic with coins
	// First top up coins
	_, _, err = repo.AddXPAndCoins(0, 100)
	if err != nil {
		t.Fatalf("AddXPAndCoins failed: %v", err)
	}

	afterUnlockProf, err := repo.UnlockCosmetic("dark-emerald", 75)
	if err != nil {
		t.Fatalf("UnlockCosmetic failed: %v", err)
	}
	if afterUnlockProf.Coins != 35 { // 10 + 100 - 75 = 35
		t.Fatalf("expected coins 35, got %d", afterUnlockProf.Coins)
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
