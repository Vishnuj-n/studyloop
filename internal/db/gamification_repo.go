package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"ai-tutor/internal/models"
)

// Title milestones: minimum XP required for each title rank
var TitleTiers = []struct {
	Title string
	MinXP int
}{
	{Title: "The Apprentice", MinXP: 0},
	{Title: "The Scholar", MinXP: 500},
	{Title: "The Inquisitor", MinXP: 1500},
	{Title: "The Archivist", MinXP: 3000},
	{Title: "The Polymath", MinXP: 5000},
	{Title: "The Grandmaster", MinXP: 8000},
	{Title: "The Paragon", MinXP: 12000},
	{Title: "The Luminary", MinXP: 20000},
	{Title: "The Mythic Sage", MinXP: 35000},
}

// ComputeTitleInfo determines the title, next title, and XP boundaries based on total XP.
func ComputeTitleInfo(totalXP int) (currentTitle string, nextTitle string, nextTitleXP int, currentTitleMinXP int) {
	if totalXP < 0 {
		totalXP = 0
	}

	for i := len(TitleTiers) - 1; i >= 0; i-- {
		if totalXP >= TitleTiers[i].MinXP {
			currentTitle = TitleTiers[i].Title
			currentTitleMinXP = TitleTiers[i].MinXP
			if i < len(TitleTiers)-1 {
				nextTitle = TitleTiers[i+1].Title
				nextTitleXP = TitleTiers[i+1].MinXP
			} else {
				nextTitle = "Maximum Rank"
				nextTitleXP = TitleTiers[i].MinXP
			}
			return
		}
	}

	return TitleTiers[0].Title, TitleTiers[1].Title, TitleTiers[1].MinXP, 0
}

// GetGamificationProfile retrieves the persistent gamification profile for the user.
func (r *Repository) GetGamificationProfile() (*models.GamificationProfile, error) {
	row := r.db.QueryRow(`
		SELECT user_id, total_xp, coins, current_title, streak_freezes_owned, frozen_dates_json, unlocked_cosmetics_json, COALESCE(stats_json, '{}'), updated_at
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
		&prof.FrozenDatesJSON,
		&prof.UnlockedCosmeticsJSON,
		&prof.StatsJSON,
		&prof.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Auto-initialize if row was missing
		_, insErr := r.db.Exec(`
			INSERT INTO user_gamification (user_id, total_xp, coins, current_title, streak_freezes_owned, frozen_dates_json, unlocked_cosmetics_json, stats_json)
			VALUES (1, 0, 0, 'The Apprentice', 1, '[]', '["dark-gruvbox", "light-classic"]', '{}')
			ON CONFLICT(user_id) DO NOTHING
		`)
		if insErr != nil {
			return nil, fmt.Errorf("failed to auto-seed gamification profile: %w", insErr)
		}
		return r.GetGamificationProfile()
	} else if err != nil {
		return nil, fmt.Errorf("failed to load gamification profile: %w", err)
	}

	curTitle, nextTitle, nextXP, minXP := ComputeTitleInfo(prof.TotalXP)
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
	newTitle, nextTitle, nextTitleXP, currentTitleMinXP := ComputeTitleInfo(newTotalXP)

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
	var currentTitle string
	err := tx.QueryRow(`
		SELECT total_xp, coins, current_title
		FROM user_gamification
		WHERE user_id = 1
	`).Scan(&totalXP, &coinBalance, &currentTitle)
	if err != nil {
		return "", fmt.Errorf("failed to read user gamification: %w", err)
	}

	oldTitle := currentTitle
	newTotalXP := totalXP + xp
	newCoins := coinBalance + coins
	newTitle, _, _, _ := ComputeTitleInfo(newTotalXP)

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
			return nil, nil, fmt.Errorf("loot box %s not found", boxID)
		}
		return nil, nil, fmt.Errorf("failed to query loot box: %w", err)
	}

	if box.Opened {
		_ = tx.Rollback()
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
		if _, err := tx.Exec(`UPDATE user_gamification SET streak_freezes_owned = streak_freezes_owned + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = 1`, freezeAmount); err != nil {
			return nil, nil, fmt.Errorf("failed to apply streak freeze reward: %w", err)
		}
	}

	// Refresh title based on updated total_xp
	var currentTotalXP int
	if err := tx.QueryRow(`SELECT total_xp FROM user_gamification WHERE user_id = 1`).Scan(&currentTotalXP); err == nil {
		newTitle, _, _, _ := ComputeTitleInfo(currentTotalXP)
		_, _ = tx.Exec(`UPDATE user_gamification SET current_title = ? WHERE user_id = 1`, newTitle)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit claim transaction: %w", err)
	}

	box.Opened = true
	prof, err := r.GetGamificationProfile()
	return &box, prof, err
}

// BuyStreakFreeze spends coins (default 50) to acquire a streak freeze shield.
func (r *Repository) BuyStreakFreeze(cost int) (*models.GamificationProfile, error) {
	if cost <= 0 {
		cost = 50
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var coins int
	err = tx.QueryRow(`SELECT coins FROM user_gamification WHERE user_id = 1`).Scan(&coins)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch coin balance: %w", err)
	}

	if coins < cost {
		return nil, fmt.Errorf("insufficient coins: have %d, require %d", coins, cost)
	}

	_, err = tx.Exec(`
		UPDATE user_gamification
		SET coins = coins - ?, streak_freezes_owned = streak_freezes_owned + 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, cost)
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
		return nil, fmt.Errorf("no streak freezes available to consume")
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

// UnlockCosmetic unlocks a theme or title by deducting coins (if price > 0) and adding itemCode to unlocked_cosmetics_json.
func (r *Repository) UnlockCosmetic(itemCode string, price int) (*models.GamificationProfile, error) {
	if itemCode == "" {
		return nil, fmt.Errorf("item code cannot be empty")
	}

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

	// Check achievement auto-unlocks
	unlockedMap := make(map[string]bool)
	for _, u := range unlockedList {
		unlockedMap[u] = true
	}

	achievements := getAchievementDefinitions()
	for _, ach := range achievements {
		if stats[ach.StatKey] >= ach.TargetValue {
			if ach.RewardItem != "" && !unlockedMap[ach.RewardItem] {
				unlockedList = append(unlockedList, ach.RewardItem)
				unlockedMap[ach.RewardItem] = true
				if ach.RewardCoins > 0 {
					coins += ach.RewardCoins
				}
			}
		}
	}

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
			Icon:        "📖",
			StatKey:     "reading_sessions",
			TargetValue: 1,
			RewardCoins: 20,
		},
		{
			ID:          "night_scholar",
			Title:       "Night Scholar",
			Description: "Complete 5 study reading sessions",
			Icon:        "🦉",
			StatKey:     "reading_sessions",
			TargetValue: 5,
			RewardCoins: 50,
			RewardItem:  "dark-obsidian",
		},
		{
			ID:          "quiz_master",
			Title:       "Quiz Master",
			Description: "Pass 5 quizzes",
			Icon:        "🎯",
			StatKey:     "quizzes_passed",
			TargetValue: 5,
			RewardCoins: 75,
			RewardItem:  "light-monochrome",
		},
		{
			ID:          "memory_monk",
			Title:       "Memory Monk",
			Description: "Review 25 flashcards",
			Icon:        "🧠",
			StatKey:     "flashcards_reviewed",
			TargetValue: 25,
			RewardCoins: 50,
		},
	}
}

// GetGamificationStore returns profile, available themes with unlock status, and achievements progress.
func (r *Repository) GetGamificationStore() (*models.GamificationStore, error) {
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

	allThemes := []models.CosmeticItem{
		{ID: "dark-gruvbox", Name: "Gruvbox Dark", Type: "theme", Price: 0, Unlocked: true},
		{ID: "light-classic", Name: "Light Classic", Type: "theme", Price: 0, Unlocked: true},
		{ID: "dark-indigo", Name: "Deep Indigo", Type: "theme", Price: 75, Unlocked: unlockedMap["dark-indigo"]},
		{ID: "dark-emerald", Name: "Forest Emerald", Type: "theme", Price: 75, Unlocked: unlockedMap["dark-emerald"]},
		{ID: "light-warm", Name: "Warm Sepia", Type: "theme", Price: 50, Unlocked: unlockedMap["light-warm"]},
		{ID: "light-sage", Name: "Sage Garden", Type: "theme", Price: 50, Unlocked: unlockedMap["light-sage"]},
		{ID: "dark-obsidian", Name: "Obsidian Black", Type: "theme", Price: 0, Unlocked: unlockedMap["dark-obsidian"], UnlockCondition: "Achievement: Night Scholar"},
		{ID: "light-monochrome", Name: "Monochrome Paper", Type: "theme", Price: 0, Unlocked: unlockedMap["light-monochrome"], UnlockCondition: "Achievement: Quiz Master"},
	}

	achDefs := getAchievementDefinitions()
	achievements := make([]models.Achievement, 0, len(achDefs))
	for _, a := range achDefs {
		curr := stats[a.StatKey]
		a.CurrentValue = curr
		a.Completed = curr >= a.TargetValue
		achievements = append(achievements, a)
	}

	return &models.GamificationStore{
		Profile:      prof,
		Themes:       allThemes,
		Achievements: achievements,
	}, nil
}
