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
	if prof.UserID != 1 || prof.TotalXP != 0 || prof.CurrentTitle != "The Apprentice" || prof.StreakFreezesOwned != 1 {
		t.Fatalf("unexpected default profile: %+v", prof)
	}

	// 2. Add XP and Coins
	updatedProf, unlockedTitle, err := repo.AddXPAndCoins(550, 60)
	if err != nil {
		t.Fatalf("AddXPAndCoins failed: %v", err)
	}
	if updatedProf.TotalXP != 550 || updatedProf.Coins != 60 {
		t.Fatalf("unexpected updated values: %+v", updatedProf)
	}
	if updatedProf.CurrentTitle != "The Scholar" {
		t.Fatalf("expected title 'The Scholar', got %q", updatedProf.CurrentTitle)
	}
	if unlockedTitle != "The Scholar" {
		t.Fatalf("expected unlockedTitle 'The Scholar', got %q", unlockedTitle)
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
	if afterClaimProf.TotalXP != 550+75 {
		t.Fatalf("expected total XP %d, got %d", 550+75, afterClaimProf.TotalXP)
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
}
