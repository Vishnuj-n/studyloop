package study

import (
	"testing"

	"ai-tutor/internal/models"
)

func TestRollTaskRewards(t *testing.T) {
	// 1. Reading (guaranteed XP/coins, 30% chest drop chance)
	xp, coins, box := RollTaskRewards(models.StudyTaskTypeReading, 0, false)
	if xp != 20 || coins != 5 {
		t.Fatalf("unexpected reading rewards: xp=%d coins=%d", xp, coins)
	}
	if box != nil && (box.BoxTier != "BRONZE" || box.RewardAmount <= 0) {
		t.Fatalf("unexpected bronze chest when dropped: %+v", box)
	}

	// 2. Quiz Ace
	xp, coins, box = RollTaskRewards(models.StudyTaskTypeQuiz, 100, true)
	if xp != 100 || coins != 25 {
		t.Fatalf("unexpected quiz ace rewards: xp=%d coins=%d", xp, coins)
	}
	if box == nil || box.BoxTier != "GOLD" || box.RewardAmount <= 0 {
		t.Fatalf("unexpected gold chest: %+v", box)
	}

	// 3. Milestone Exam
	xp, coins, box = RollTaskRewards(models.StudyTaskTypeMilestoneExam, 0, false)
	if xp != 150 || coins != 35 {
		t.Fatalf("unexpected milestone rewards: xp=%d coins=%d", xp, coins)
	}
	if box == nil || box.BoxTier != "MYTHIC" || box.RewardAmount <= 0 {
		t.Fatalf("unexpected mythic chest: %+v", box)
	}
}
