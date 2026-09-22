package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"ai-tutor/internal/models"

	"github.com/google/uuid"
)

var (
	ErrInsufficientCoins    = errors.New("insufficient coins")
	ErrNoFreezes            = errors.New("no streak freezes available")
	ErrFreezeInventoryFull  = errors.New("cannot own more than 2 streak freezes")
	ErrFreezeWeeklyLimit    = errors.New("streak freeze purchase limit reached for this week (1 purchase per 7 days)")
	ErrBoxNotFound          = errors.New("loot box not found")
	ErrBoxAlreadyOpened     = errors.New("loot box already opened")
)


// Title milestones: minimum XP required for each title rank
var TitleTiers = []struct {
	BaseTitle string
	MinXP     int
}{
	{BaseTitle: "The Apprentice", MinXP: 0},
	{BaseTitle: "The Scholar", MinXP: 1500},
	{BaseTitle: "The Inquisitor", MinXP: 4500},
	{BaseTitle: "The Archivist", MinXP: 9000},
	{BaseTitle: "The Polymath", MinXP: 16000},
	{BaseTitle: "The Grandmaster", MinXP: 26000},
	{BaseTitle: "The Paragon", MinXP: 40000},
	{BaseTitle: "The Luminary", MinXP: 60000},
	{BaseTitle: "The Mythic Sage", MinXP: 90000},
}

var romanNumerals = []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"}

// ComputeLevel calculates the numerical account level (1 to 100) based on total XP.
func ComputeLevel(totalXP int) int {
	if totalXP <= 0 {
		return 1
	}
	lvl := (totalXP / 300) + 1
	if lvl > 100 {
		return 100
	}
	return lvl
}

// ComputeTitleInfo determines the title with sub-tier (e.g. "The Inquisitor III"), next title, XP boundaries, and level based on total XP.
func ComputeTitleInfo(totalXP int) (currentTitle string, nextTitle string, nextTitleXP int, currentTitleMinXP int, level int) {
	if totalXP < 0 {
		totalXP = 0
	}
	level = ComputeLevel(totalXP)

	for i := len(TitleTiers) - 1; i >= 0; i-- {
		if totalXP >= TitleTiers[i].MinXP {
			base := TitleTiers[i].BaseTitle
			minXP := TitleTiers[i].MinXP
			currentTitleMinXP = minXP

			var maxXP int
			if i < len(TitleTiers)-1 {
				maxXP = TitleTiers[i+1].MinXP
				nextTitle = TitleTiers[i+1].BaseTitle + " I"
				nextTitleXP = TitleTiers[i+1].MinXP
			} else {
				maxXP = minXP + 30000
				nextTitle = "Maximum Rank"
				nextTitleXP = minXP
			}

			xpSpan := maxXP - minXP
			if xpSpan <= 0 {
				xpSpan = 3000
			}
			step := xpSpan / 10
			if step <= 0 {
				step = 100
			}
			subIndex := (totalXP - minXP) / step
			if subIndex >= 10 {
				subIndex = 9
			} else if subIndex < 0 {
				subIndex = 0
			}

			currentTitle = fmt.Sprintf("%s %s", base, romanNumerals[subIndex])
			return
		}
	}

	return TitleTiers[0].BaseTitle + " I", TitleTiers[1].BaseTitle + " I", TitleTiers[1].MinXP, 0, 1
}

// GetGamificationProfile retrieves the persistent gamification profile for the user.
func (r *Repository) GetGamificationProfile() (*models.GamificationProfile, error) {
	row := r.db.QueryRow(`
		SELECT user_id, total_xp, coins, current_title, streak_freezes_owned, COALESCE(last_freeze_purchased_at, 0), frozen_dates_json, unlocked_cosmetics_json, COALESCE(stats_json, '{}'), updated_at
		FROM user_gamification
		WHERE user_id = 1
	`)

	var prof models.GamificationProfile
	err := row.Scan(
		&prof.UserID,
		&prof.TotalXP,
		&prof.Coins,
		&prof.CurrentTitle,
		&prof.StreakFreezesOwned,
		&prof.LastFreezePurchasedAt,
		&prof.FrozenDatesJSON,
		&prof.UnlockedCosmeticsJSON,
		&prof.StatsJSON,
		&prof.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Auto-initialize if row was missing
		_, insErr := r.db.Exec(`
			INSERT INTO user_gamification (user_id, total_xp, coins, current_title, streak_freezes_owned, last_freeze_purchased_at, frozen_dates_json, unlocked_cosmetics_json, stats_json)
			VALUES (1, 0, 0, 'The Apprentice I', 1, 0, '[]', '["dark-gruvbox", "light-classic"]', '{}')
			ON CONFLICT(user_id) DO NOTHING
		`)
		if insErr != nil {
			return nil, fmt.Errorf("failed to auto-seed gamification profile: %w", insErr)
		}
		return r.GetGamificationProfile()
	} else if err != nil {
		return nil, fmt.Errorf("failed to load gamification profile: %w", err)
	}

	curTitle, nextTitle, nextXP, minXP, lvl := ComputeTitleInfo(prof.TotalXP)
	prof.Level = lvl
	prof.CurrentTitle = curTitle
	prof.NextTitle = nextTitle
	prof.NextTitleXP = nextXP
	prof.CurrentTitleMinXP = minXP

	return &prof, nil
}

// AddXPAndCoins atomically increments XP and coins, updating current_title if rank increases.
func (r *Repository) AddXPAndCoins(xp, coins int) (*models.GamificationProfile, string, error) {
	if xp < 0 {
		xp = 0
	}
	if coins < 0 {
		coins = 0
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var totalXP, coinBalance int
	var currentTitle string
	var streakFreezes int
	var frozenDatesJSON, unlockedCosmeticsJSON string
	err = tx.QueryRow(`
		SELECT total_xp, coins, current_title, streak_freezes_owned, frozen_dates_json, unlocked_cosmetics_json
		FROM user_gamification
		WHERE user_id = 1
	`).Scan(&totalXP, &coinBalance, &currentTitle, &streakFreezes, &frozenDatesJSON, &unlockedCosmeticsJSON)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query gamification state: %w", err)
	}

	oldTitle := currentTitle
	newTotalXP := totalXP + xp
	newCoins := coinBalance + coins
	newTitle, nextTitle, nextTitleXP, currentTitleMinXP, lvl := ComputeTitleInfo(newTotalXP)

	var newTitleUnlocked string
	if newTitle != oldTitle {
		newTitleUnlocked = newTitle
	}

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET total_xp = ?, coins = ?, current_title = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, newTotalXP, newCoins, newTitle)
	if err != nil {
		return nil, "", fmt.Errorf("failed to update gamification profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("failed to commit gamification update: %w", err)
	}

	updatedProf := &models.GamificationProfile{
		Level:                 lvl,
		TotalXP:               newTotalXP,
		Coins:                 newCoins,
		CurrentTitle:          newTitle,
		NextTitle:             nextTitle,
		NextTitleXP:           nextTitleXP,
		CurrentTitleMinXP:     currentTitleMinXP,
		StreakFreezesOwned:    streakFreezes,
		FrozenDatesJSON:       frozenDatesJSON,
		UnlockedCosmeticsJSON: unlockedCosmeticsJSON,
	}

	return updatedProf, newTitleUnlocked, nil
}

// CreatePendingLootBox inserts a new pending mystery chest.
func (r *Repository) CreatePendingLootBox(box models.PendingLootBox) error {
	if box.ID == "" {
		return fmt.Errorf("loot box ID cannot be empty")
	}

	_, err := r.db.Exec(`
		INSERT INTO pending_loot_boxes (id, task_id, box_tier, reward_type, reward_amount, opened, created_at)
		VALUES (?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)
	`, box.ID, box.TaskID, box.BoxTier, box.RewardType, box.RewardAmount)
	if err != nil {
		return fmt.Errorf("failed to insert pending loot box: %w", err)
	}
	return nil
}

// AwardTaskRewardsTx atomically updates XP/coins and persists any pending loot box in a single transaction.
func (r *Repository) AwardTaskRewardsTx(tx *sql.Tx, xp, coins int, box *models.PendingLootBox) (string, error) {
	if xp < 0 {
		xp = 0
	}
	if coins < 0 {
		coins = 0
	}

	var totalXP, coinBalance int
	var currentTitle, statsJSON string
	err := tx.QueryRow(`
		SELECT total_xp, coins, current_title, COALESCE(stats_json, '{}')
		FROM user_gamification
		WHERE user_id = 1
	`).Scan(&totalXP, &coinBalance, &currentTitle, &statsJSON)
	if err != nil {
		return "", fmt.Errorf("failed to read user gamification: %w", err)
	}

	oldTitle := currentTitle
	newTotalXP := totalXP + xp
	newCoins := coinBalance + coins
	newTitle, _, _, _, _ := ComputeTitleInfo(newTotalXP)

	var newTitleUnlocked string
	if newTitle != oldTitle {
		newTitleUnlocked = newTitle
	}

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET total_xp = ?, coins = ?, current_title = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, newTotalXP, newCoins, newTitle)
	if err != nil {
		return "", fmt.Errorf("failed to update gamification profile: %w", err)
	}

	if box != nil {
		if box.ID == "" {
			return "", fmt.Errorf("loot box ID cannot be empty")
		}
		_, err = tx.Exec(`
			INSERT INTO pending_loot_boxes (id, task_id, box_tier, reward_type, reward_amount, opened, created_at)
			VALUES (?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)
		`, box.ID, box.TaskID, box.BoxTier, box.RewardType, box.RewardAmount)
		if err != nil {
			return "", fmt.Errorf("failed to insert pending loot box: %w", err)
		}
	}

	return newTitleUnlocked, nil
}

// GetUnopenedLootBoxes returns all unclaimed mystery chests.
func (r *Repository) GetUnopenedLootBoxes() ([]models.PendingLootBox, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, box_tier, reward_type, reward_amount, opened, created_at
		FROM pending_loot_boxes
		WHERE opened = 0
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query unopened loot boxes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	boxes := make([]models.PendingLootBox, 0)
	for rows.Next() {
		var b models.PendingLootBox
		if err := rows.Scan(&b.ID, &b.TaskID, &b.BoxTier, &b.RewardType, &b.RewardAmount, &b.Opened, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan loot box: %w", err)
		}
		boxes = append(boxes, b)
	}
	return boxes, rows.Err()
}

// ClaimLootBox marks a chest as opened and applies its rewards to the user.
func (r *Repository) ClaimLootBox(boxID string) (*models.PendingLootBox, *models.GamificationProfile, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin claim transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var box models.PendingLootBox
	row := tx.QueryRow(`
		SELECT id, task_id, box_tier, reward_type, reward_amount, opened, created_at
		FROM pending_loot_boxes
		WHERE id = ?
	`, boxID)
	if err := row.Scan(&box.ID, &box.TaskID, &box.BoxTier, &box.RewardType, &box.RewardAmount, &box.Opened, &box.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("%w: %s", ErrBoxNotFound, boxID)
		}
		return nil, nil, fmt.Errorf("failed to query loot box: %w", err)
	}

	if box.Opened {
		prof, err := r.GetGamificationProfile()
		return &box, prof, err
	}

	// Mark opened
	if _, err := tx.Exec(`UPDATE pending_loot_boxes SET opened = 1 WHERE id = ?`, boxID); err != nil {
		return nil, nil, fmt.Errorf("failed to mark loot box opened: %w", err)
	}

	// Apply reward
	switch box.RewardType {
	case "XP":
		if _, err := tx.Exec(`UPDATE user_gamification SET total_xp = total_xp + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = 1`, box.RewardAmount); err != nil {
			return nil, nil, fmt.Errorf("failed to apply XP reward: %w", err)
		}
	case "COINS":
		if _, err := tx.Exec(`UPDATE user_gamification SET coins = coins + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = 1`, box.RewardAmount); err != nil {
			return nil, nil, fmt.Errorf("failed to apply coins reward: %w", err)
		}
	case "STREAK_FREEZE":
		freezeAmount := box.RewardAmount
		if freezeAmount <= 0 {
			freezeAmount = 1
		}
		var currentFreezes int
		if err := tx.QueryRow(`SELECT streak_freezes_owned FROM user_gamification WHERE user_id = 1`).Scan(&currentFreezes); err != nil {
			return nil, nil, fmt.Errorf("failed to scan streak freezes: %w", err)
		}
		if currentFreezes >= 2 {
			// ponytail: if already capped at max 2 freezes, convert loot drop into 100 bonus coins
			if _, err := tx.Exec(`UPDATE user_gamification SET coins = coins + 100, updated_at = CURRENT_TIMESTAMP WHERE user_id = 1`); err != nil {
				return nil, nil, fmt.Errorf("failed to apply converted streak freeze coins: %w", err)
			}
		} else {
			newFreezes := currentFreezes + freezeAmount
			if newFreezes > 2 {
				newFreezes = 2
			}
			if _, err := tx.Exec(`UPDATE user_gamification SET streak_freezes_owned = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = 1`, newFreezes); err != nil {
				return nil, nil, fmt.Errorf("failed to apply streak freeze reward: %w", err)
			}
		}
	}

	// Refresh title based on updated total_xp inside transaction
	var currentTotalXP int
	if err := tx.QueryRow(`SELECT total_xp FROM user_gamification WHERE user_id = 1`).Scan(&currentTotalXP); err != nil {
		return nil, nil, fmt.Errorf("failed to scan total_xp for title update: %w", err)
	}
	newTitle, _, _, _, _ := ComputeTitleInfo(currentTotalXP)
	if _, err := tx.Exec(`UPDATE user_gamification SET current_title = ? WHERE user_id = 1`, newTitle); err != nil {
		return nil, nil, fmt.Errorf("failed to update current_title: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit claim transaction: %w", err)
	}

	box.Opened = true
	prof, err := r.GetGamificationProfile()
	return &box, prof, err
}

// BuyStreakFreeze spends coins (default 150) to acquire a streak freeze shield.
// Enforces max inventory cap (2) and a 1-purchase per 7-day rolling window limit.
func (r *Repository) BuyStreakFreeze(cost int, nowUnix int64) (*models.GamificationProfile, error) {
	if cost <= 0 {
		cost = 150
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var coins, freezes int
	var lastPurchased int64
	err = tx.QueryRow(`SELECT coins, streak_freezes_owned, COALESCE(last_freeze_purchased_at, 0) FROM user_gamification WHERE user_id = 1`).Scan(&coins, &freezes, &lastPurchased)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gamification balance: %w", err)
	}

	if freezes >= 2 {
		return nil, ErrFreezeInventoryFull
	}

	// 7 days = 7 * 86400 = 604800 seconds
	const weeklyWindowSeconds = int64(7 * 24 * 3600)
	if lastPurchased > 0 && (nowUnix-lastPurchased) < weeklyWindowSeconds {
		return nil, ErrFreezeWeeklyLimit
	}

	if coins < cost {
		return nil, fmt.Errorf("%w: have %d, require %d", ErrInsufficientCoins, coins, cost)
	}

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET coins = coins - ?, streak_freezes_owned = streak_freezes_owned + 1, last_freeze_purchased_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, cost, nowUnix)
	if err != nil {
		return nil, fmt.Errorf("failed to purchase streak freeze: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit streak freeze purchase: %w", err)
	}

	return r.GetGamificationProfile()
}

// GetStreakFreezeUsageDates returns all dates (YYYY-MM-DD) where a streak freeze was applied.
func (r *Repository) GetStreakFreezeUsageDates() ([]string, error) {
	prof, err := r.GetGamificationProfile()
	if err != nil {
		return nil, err
	}
	if prof.FrozenDatesJSON == "" || prof.FrozenDatesJSON == "[]" {
		return []string{}, nil
	}
	var dates []string
	if err := json.Unmarshal([]byte(prof.FrozenDatesJSON), &dates); err != nil {
		return []string{}, nil
	}
	return dates, nil
}

// ConsumeStreakFreeze decrements streak_freezes_owned by 1 and records the protected date in frozen_dates_json.
func (r *Repository) ConsumeStreakFreeze(dateStr string) (*models.GamificationProfile, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var freezes int
	var frozenJSON string
	err = tx.QueryRow(`
		SELECT streak_freezes_owned, frozen_dates_json
		FROM user_gamification
		WHERE user_id = 1
	`).Scan(&freezes, &frozenJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to query gamification state: %w", err)
	}

	if freezes <= 0 {
		return nil, fmt.Errorf("%w: user has 0 streak freezes available", ErrNoFreezes)
	}

	var dates []string
	if frozenJSON != "" {
		_ = json.Unmarshal([]byte(frozenJSON), &dates)
	}
	if dateStr != "" {
		alreadyPresent := false
		for _, d := range dates {
			if d == dateStr {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			dates = append(dates, dateStr)
		}
	}
	newJSONBytes, _ := json.Marshal(dates)
	newJSON := string(newJSONBytes)

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET streak_freezes_owned = streak_freezes_owned - 1, frozen_dates_json = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, newJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update gamification profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit streak freeze consumption: %w", err)
	}

	return r.GetGamificationProfile()
}

func getCosmeticCatalog() []models.CosmeticItem {
	return []models.CosmeticItem{
		{ID: "dark-gruvbox", Name: "Gruvbox Dark", Type: "theme", Price: 0},
		{ID: "light-classic", Name: "Light Classic", Type: "theme", Price: 0},
		{ID: "dark-indigo", Name: "Deep Indigo", Type: "theme", Price: 75},
		{ID: "dark-emerald", Name: "Forest Emerald", Type: "theme", Price: 75},
		{ID: "light-warm", Name: "Warm Sepia", Type: "theme", Price: 50},
		{ID: "light-sage", Name: "Sage Garden", Type: "theme", Price: 50},
		{ID: "dark-academia", Name: "Dark Academia", Type: "theme", Price: 400},
		{ID: "dark-cyberpunk", Name: "Neon Cyberpunk", Type: "theme", Price: 600},
		{ID: "light-zen", Name: "Zen Minimalist", Type: "theme", Price: 300},
		{ID: "dark-obsidian", Name: "Obsidian Black", Type: "theme", Price: 0, UnlockCondition: "Achievement: Night Scholar"},
		{ID: "light-monochrome", Name: "Monochrome Paper", Type: "theme", Price: 0, UnlockCondition: "Achievement: Quiz Master"},
	}
}

// GenerateLootBox produces a randomized reward loot box based on tier.
func GenerateLootBox(tier string) *models.PendingLootBox {
	boxID := uuid.NewString()
	var rewardType string
	var amount int

	roll := rand.Intn(100)

	switch tier {
	case "BRONZE":
		if roll < 60 {
			rewardType = "XP"
			amount = 15 + rand.Intn(26)
		} else {
			rewardType = "COINS"
			amount = 5 + rand.Intn(6)
		}

	case "SILVER":
		if roll < 65 {
			rewardType = "XP"
			amount = 40 + rand.Intn(61)
		} else {
			rewardType = "COINS"
			amount = 15 + rand.Intn(16)
		}

	case "GOLD":
		if roll < 50 {
			rewardType = "XP"
			amount = 100 + rand.Intn(151)
		} else if roll < 85 {
			rewardType = "COINS"
			amount = 30 + rand.Intn(21)
		} else {
			rewardType = "STREAK_FREEZE"
			amount = 1
		}

	case "MYTHIC":
		if roll < 45 {
			rewardType = "XP"
			amount = 200 + rand.Intn(301)
		} else if roll < 80 {
			rewardType = "COINS"
			amount = 50 + rand.Intn(51)
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



func getCosmeticDefinition(itemCode string) (*models.CosmeticItem, bool) {
	for _, item := range getCosmeticCatalog() {
		if item.ID == itemCode {
			return &item, true
		}
	}
	return nil, false
}

// UnlockCosmetic unlocks a theme or title by deducting catalog price and adding itemCode to unlocked_cosmetics_json.
func (r *Repository) UnlockCosmetic(itemCode string, _ int) (*models.GamificationProfile, error) {
	if itemCode == "" {
		return nil, fmt.Errorf("item code cannot be empty")
	}

	def, found := getCosmeticDefinition(itemCode)
	if !found {
		return nil, fmt.Errorf("unknown cosmetic item: %s", itemCode)
	}
	if def.UnlockCondition != "" {
		return nil, fmt.Errorf("item %s is achievement-gated and cannot be purchased directly", itemCode)
	}

	price := def.Price

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var coins int
	var unlockedJSON string
	err = tx.QueryRow(`SELECT coins, unlocked_cosmetics_json FROM user_gamification WHERE user_id = 1`).Scan(&coins, &unlockedJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to read user gamification state: %w", err)
	}

	var unlockedList []string
	if unlockedJSON != "" {
		_ = json.Unmarshal([]byte(unlockedJSON), &unlockedList)
	}

	for _, item := range unlockedList {
		if item == itemCode {
			// Already unlocked
			_ = tx.Rollback()
			return r.GetGamificationProfile()
		}
	}

	if price > 0 {
		if coins < price {
			return nil, fmt.Errorf("insufficient coins: have %d, require %d", coins, price)
		}
		coins -= price
	}

	unlockedList = append(unlockedList, itemCode)
	newUnlockedBytes, _ := json.Marshal(unlockedList)

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET coins = ?, unlocked_cosmetics_json = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, coins, string(newUnlockedBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to update unlocked cosmetics: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit cosmetic unlock: %w", err)
	}

	return r.GetGamificationProfile()
}

// AddDevCoins boosts user coin balance by specified amount for dev testing.
func (r *Repository) AddDevCoins(amount int) (*models.GamificationProfile, error) {
	if amount <= 0 {
		amount = 10000
	}
	_, err := r.db.Exec(`UPDATE user_gamification SET coins = coins + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = 1`, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to add dev coins: %w", err)
	}
	return r.GetGamificationProfile()
}

// reconcileAchievementsTx checks achievement thresholds and unlocks rewards.
func reconcileAchievementsTx(stats map[string]int, unlockedList []string, coins int) ([]string, int, bool) {
	unlockedMap := make(map[string]bool, len(unlockedList))
	for _, u := range unlockedList {
		unlockedMap[u] = true
	}

	changed := false
	achievements := getAchievementDefinitions()
	for _, ach := range achievements {
		if stats[ach.StatKey] >= ach.TargetValue {
			claimKey := "achievement:" + ach.ID
			if !unlockedMap[claimKey] {
				unlockedList = append(unlockedList, claimKey)
				unlockedMap[claimKey] = true
				changed = true
				if ach.RewardItem != "" && !unlockedMap[ach.RewardItem] {
					unlockedList = append(unlockedList, ach.RewardItem)
					unlockedMap[ach.RewardItem] = true
				}
				if ach.RewardCoins > 0 {
					coins += ach.RewardCoins
				}
			}
		}
	}
	return unlockedList, coins, changed
}

// IncrementStat increments a counter in stats_json and auto-unlocks any completed achievements.
func (r *Repository) IncrementStat(statKey string, delta int) error {
	if statKey == "" || delta <= 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin stat transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var statsJSON, unlockedJSON string
	var coins int
	err = tx.QueryRow(`SELECT COALESCE(stats_json, '{}'), unlocked_cosmetics_json, coins FROM user_gamification WHERE user_id = 1`).Scan(&statsJSON, &unlockedJSON, &coins)
	if err != nil {
		return fmt.Errorf("failed to query stats: %w", err)
	}

	stats := make(map[string]int)
	if statsJSON != "" {
		_ = json.Unmarshal([]byte(statsJSON), &stats)
	}
	stats[statKey] = stats[statKey] + delta

	var unlockedList []string
	if unlockedJSON != "" {
		_ = json.Unmarshal([]byte(unlockedJSON), &unlockedList)
	}

	unlockedList, coins, _ = reconcileAchievementsTx(stats, unlockedList, coins)

	newStatsBytes, _ := json.Marshal(stats)
	newUnlockedBytes, _ := json.Marshal(unlockedList)

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET stats_json = ?, unlocked_cosmetics_json = ?, coins = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, string(newStatsBytes), string(newUnlockedBytes), coins)
	if err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}

	return tx.Commit()
}

func getAchievementDefinitions() []models.Achievement {
	return []models.Achievement{
		{
			ID:          "first_step",
			Title:       "First Steps",
			Description: "Complete 1 study reading session",
			Icon:        "book",
			StatKey:     "reading_sessions",
			TargetValue: 1,
			RewardCoins: 20,
		},
		{
			ID:          "night_scholar",
			Title:       "Night Scholar",
			Description: "Complete 5 study reading sessions",
			Icon:        "owl",
			StatKey:     "reading_sessions",
			TargetValue: 5,
			RewardCoins: 50,
			RewardItem:  "dark-obsidian",
		},
		{
			ID:          "deep_diver",
			Title:       "Deep Diver",
			Description: "Complete 15 study reading sessions",
			Icon:        "diving",
			StatKey:     "reading_sessions",
			TargetValue: 15,
			RewardCoins: 150,
		},
		{
			ID:          "quiz_starter",
			Title:       "Quiz Starter",
			Description: "Pass 1 quiz",
			Icon:        "target",
			StatKey:     "quizzes_passed",
			TargetValue: 1,
			RewardCoins: 25,
		},
		{
			ID:          "quiz_master",
			Title:       "Quiz Master",
			Description: "Pass 5 quizzes",
			Icon:        "trophy",
			StatKey:     "quizzes_passed",
			TargetValue: 5,
			RewardCoins: 75,
			RewardItem:  "light-monochrome",
		},
		{
			ID:          "quiz_ace",
			Title:       "Quiz Ace",
			Description: "Pass 15 quizzes",
			Icon:        "star",
			StatKey:     "quizzes_passed",
			TargetValue: 15,
			RewardCoins: 200,
		},
		{
			ID:          "flashcard_initiate",
			Title:       "Flashcard Initiate",
			Description: "Review 10 flashcards",
			Icon:        "card",
			StatKey:     "flashcards_reviewed",
			TargetValue: 10,
			RewardCoins: 30,
		},
		{
			ID:          "memory_monk",
			Title:       "Memory Monk",
			Description: "Review 25 flashcards",
			Icon:        "brain",
			StatKey:     "flashcards_reviewed",
			TargetValue: 25,
			RewardCoins: 50,
		},
		{
			ID:          "memory_master",
			Title:       "Memory Master",
			Description: "Review 100 flashcards",
			Icon:        "crown",
			StatKey:     "flashcards_reviewed",
			TargetValue: 100,
			RewardCoins: 250,
		},
		{
			ID:          "wager_winner",
			Title:       "Goal Conqueror",
			Description: "Win your first daily study wager",
			Icon:        "flame",
			StatKey:     "wagers_won",
			TargetValue: 1,
			RewardCoins: 150,
		},
	}
}

// GetGamificationStore returns profile, available themes with unlock status, and achievements progress.
func (r *Repository) GetGamificationStore() (*models.GamificationStore, error) {
	// Reconcile stats from SQLite source of truth tables
	var countReading, countQuizzes, countCards int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM study_queue WHERE task_type IN ('READING', 'AUDIO_LECTURE', 'VIDEO_LECTURE') AND status = 'COMPLETED'`).Scan(&countReading)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM quiz_attempts WHERE passed = 1`).Scan(&countQuizzes)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM fsrs_review_log`).Scan(&countCards)

	prof, err := r.GetGamificationProfile()
	if err != nil {
		return nil, err
	}

	var unlockedList []string
	if prof.UnlockedCosmeticsJSON != "" {
		_ = json.Unmarshal([]byte(prof.UnlockedCosmeticsJSON), &unlockedList)
	}
	unlockedMap := make(map[string]bool)
	for _, u := range unlockedList {
		unlockedMap[u] = true
	}

	stats := make(map[string]int)
	if prof.StatsJSON != "" {
		_ = json.Unmarshal([]byte(prof.StatsJSON), &stats)
	}

	changed := false
	if countReading > stats["reading_sessions"] {
		stats["reading_sessions"] = countReading
		changed = true
	}
	if countQuizzes > stats["quizzes_passed"] {
		stats["quizzes_passed"] = countQuizzes
		changed = true
	}
	if countCards > stats["flashcards_reviewed"] {
		stats["flashcards_reviewed"] = countCards
		changed = true
	}

	achDefs := getAchievementDefinitions()
	coins := prof.Coins
	var recChanged bool
	unlockedList, coins, recChanged = reconcileAchievementsTx(stats, unlockedList, coins)
	if recChanged {
		changed = true
	}

	if changed {
		tx, err := r.db.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin reconciliation transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		newStatsBytes, _ := json.Marshal(stats)
		newUnlockedBytes, _ := json.Marshal(unlockedList)
		_, err = tx.Exec(`
			UPDATE user_gamification
			SET stats_json = ?, unlocked_cosmetics_json = ?, coins = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = 1
		`, string(newStatsBytes), string(newUnlockedBytes), coins)
		if err != nil {
			return nil, fmt.Errorf("failed to update reconciled stats: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit reconciliation transaction: %w", err)
		}

		prof, err = r.GetGamificationProfile()
		if err != nil {
			return nil, err
		}
	}

	catalog := getCosmeticCatalog()
	allThemes := make([]models.CosmeticItem, len(catalog))
	for i, c := range catalog {
		c.Unlocked = unlockedMap[c.ID]
		if c.ID == "dark-gruvbox" || c.ID == "light-classic" {
			c.Unlocked = true
		}
		allThemes[i] = c
	}

	achievements := make([]models.Achievement, 0, len(achDefs))
	for _, a := range achDefs {
		curr := stats[a.StatKey]
		a.CurrentValue = curr

		// ponytail: dynamic exponential target scaling (doubles target per tier: e.g. 5 -> 10 -> 20 -> 40...)
		if a.TargetValue > 0 && curr >= a.TargetValue {
			tier := 1
			tVal := a.TargetValue
			for curr >= tVal {
				tier++
				tVal *= 2
			}
			a.Title = fmt.Sprintf("%s %s", a.Title, toRoman(tier))
			a.TargetValue = tVal
			a.Completed = false
			if tier > 1 {
				a.RewardItem = ""
			}
		} else {
			a.Completed = curr >= a.TargetValue
		}

		achievements = append(achievements, a)
	}

	return &models.GamificationStore{
		Profile:      prof,
		Themes:       allThemes,
		Achievements: achievements,
	}, nil
}

func toRoman(num int) string {
	if num <= 0 {
		return "I"
	}
	vals := []int{10, 9, 5, 4, 1}
	syms := []string{"X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i := 0; i < len(vals); i++ {
		for num >= vals[i] {
			num -= vals[i]
			b.WriteString(syms[i])
		}
	}
	return b.String()
}
