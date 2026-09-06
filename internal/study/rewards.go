package study

import (
	"math/rand"
	"time"

	"ai-tutor/internal/models"

	"github.com/google/uuid"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RollTaskRewards calculates guaranteed base XP & coins plus a variable-tier mystery chest.
func RollTaskRewards(taskType models.StudyTaskType, quizScore int, isAce bool) (int, int, *models.PendingLootBox) {
	var baseXP int
	var baseCoins int
	var tier string

	switch taskType {
	case models.StudyTaskTypeReading:
		baseXP = 20
		baseCoins = 5
		tier = "BRONZE"

	case models.StudyTaskTypeQuiz:
		if isAce || quizScore >= 100 {
			baseXP = 100
			baseCoins = 25
			tier = "GOLD"
		} else {
			baseXP = 50
			baseCoins = 10
			tier = "SILVER"
		}

	case models.StudyTaskTypeFlashcardReview, models.StudyTaskTypeFlashcardGenerate:
		baseXP = 30
		baseCoins = 8
		tier = "SILVER"

	case models.StudyTaskTypeMilestoneExam, models.StudyTaskTypeSocraticRemedial:
		baseXP = 150
		baseCoins = 35
		tier = "MYTHIC"

	default:
		baseXP = 15
		baseCoins = 3
		tier = "BRONZE"
	}

	box := generateLootBox(tier)
	return baseXP, baseCoins, box
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
