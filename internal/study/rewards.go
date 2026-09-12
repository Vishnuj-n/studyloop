package study

import (
	"math/rand"

	"ai-tutor/internal/models"

	"github.com/google/uuid"
)

// RollTaskRewards calculates guaranteed base XP & coins plus a variable-tier mystery chest.
func RollTaskRewards(taskType models.StudyTaskType, quizScore int, isAce bool) (int, int, *models.PendingLootBox) {
	var baseXP int
	var baseCoins int
	var tier string

	// Determine base XP and coins
	switch taskType {
	case models.StudyTaskTypeReading:
		baseXP = 20
		baseCoins = 5

	case models.StudyTaskTypeQuiz:
		if isAce || quizScore >= 100 {
			baseXP = 100
			baseCoins = 25
		} else {
			baseXP = 50
			baseCoins = 10
		}

	case models.StudyTaskTypeFlashcardReview, models.StudyTaskTypeFlashcardGenerate:
		baseXP = 30
		baseCoins = 8

	case models.StudyTaskTypeMilestoneExam, models.StudyTaskTypeSocraticRemedial:
		baseXP = 150
		baseCoins = 35

	default:
		baseXP = 15
		baseCoins = 3
	}

	// Probabilistic Gacha Chest Engine:
	// Achievements & Milestones (Ace, Socratic/Viva, Milestone Exam) get 100% drop with boosted odds.
	// Routine tasks get 30% drop with standard rare odds.
	isAchievement := isAce || quizScore >= 100 || taskType == models.StudyTaskTypeMilestoneExam || taskType == models.StudyTaskTypeSocraticRemedial

	var box *models.PendingLootBox
	if isAchievement || rand.Intn(100) < 30 {
		tier = rollChestTier(isAchievement)
		box = generateLootBox(tier)
	}
	return baseXP, baseCoins, box
}

// rollChestTier uses transparent probabilistic gacha odds out of 1000.
// Routine: MYTHIC 0.5% (5/1000), GOLD 5.5% (55/1000), SILVER 34% (340/1000), BRONZE 60% (600/1000)
// Achievement (Viva/Milestone/Ace): MYTHIC 3.0% (30/1000), GOLD 37% (370/1000), SILVER 45% (450/1000), BRONZE 15% (150/1000)
func rollChestTier(isAchievement bool) string {
	roll := rand.Intn(1000)

	if isAchievement {
		if roll < 30 { // 3.0%
			return "MYTHIC"
		} else if roll < 400 { // 37.0%
			return "GOLD"
		} else if roll < 850 { // 45.0%
			return "SILVER"
		}
		return "BRONZE"
	}

	// Routine odds
	if roll < 5 { // 0.5% Ultra-rare
		return "MYTHIC"
	} else if roll < 60 { // 5.5% Rare
		return "GOLD"
	} else if roll < 400 { // 34.0%
		return "SILVER"
	}
	return "BRONZE"
}

func generateLootBox(tier string) *models.PendingLootBox {
	boxID := uuid.NewString()
	var rewardType string
	var amount int

	roll := rand.Intn(100)

	switch tier {
	case "BRONZE":
		if roll < 60 {
			rewardType = "XP"
			amount = 15 + rand.Intn(26) // 15 - 40 XP
		} else {
			rewardType = "COINS"
			amount = 5 + rand.Intn(6) // 5 - 10 coins
		}

	case "SILVER":
		if roll < 65 {
			rewardType = "XP"
			amount = 40 + rand.Intn(61) // 40 - 100 XP
		} else {
			rewardType = "COINS"
			amount = 15 + rand.Intn(16) // 15 - 30 coins
		}

	case "GOLD":
		if roll < 50 {
			rewardType = "XP"
			amount = 100 + rand.Intn(151) // 100 - 250 XP
		} else if roll < 85 {
			rewardType = "COINS"
			amount = 30 + rand.Intn(21) // 30 - 50 coins
		} else {
			rewardType = "STREAK_FREEZE"
			amount = 1
		}

	case "MYTHIC":
		if roll < 45 {
			rewardType = "XP"
			amount = 200 + rand.Intn(301) // 200 - 500 XP
		} else if roll < 80 {
			rewardType = "COINS"
			amount = 50 + rand.Intn(51) // 50 - 100 coins
		} else {
			rewardType = "STREAK_FREEZE"
			amount = 1
		}

	default:
		rewardType = "XP"
		amount = 20
	}

	return &models.PendingLootBox{
		ID:           boxID,
		BoxTier:      tier,
		RewardType:   rewardType,
		RewardAmount: amount,
		Opened:       false,
	}
}
