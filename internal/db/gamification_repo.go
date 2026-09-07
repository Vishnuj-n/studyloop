package db

import (
	"database/sql"
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
		SELECT user_id, total_xp, coins, current_title, streak_freezes_owned, unlocked_cosmetics_json, updated_at
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
		&prof.UnlockedCosmeticsJSON,
		&prof.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Auto-initialize if row was missing
		_, insErr := r.db.Exec(`
			INSERT INTO user_gamification (user_id, total_xp, coins, current_title, streak_freezes_owned, unlocked_cosmetics_json)
			VALUES (1, 0, 0, 'The Apprentice', 1, '[]')
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

	prof, err := r.GetGamificationProfile()
	if err != nil {
		return nil, "", err
	}

	oldTitle := prof.CurrentTitle
	newTotalXP := prof.TotalXP + xp
	newCoins := prof.Coins + coins
	newTitle, _, _, _ := ComputeTitleInfo(newTotalXP)

	var newTitleUnlocked string
	if newTitle != oldTitle && newTotalXP >= prof.NextTitleXP {
		newTitleUnlocked = newTitle
	}

	_, err = r.db.Exec(`
		UPDATE user_gamification
		SET total_xp = ?, coins = ?, current_title = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1
	`, newTotalXP, newCoins, newTitle)
	if err != nil {
		return nil, "", fmt.Errorf("failed to update gamification profile: %w", err)
	}

	updatedProf, err := r.GetGamificationProfile()
	if err != nil {
		return nil, "", err
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

// ConsumeStreakFreeze decrements streak_freezes_owned by 1 when saving an interrupted streak.
func (r *Repository) ConsumeStreakFreeze() (*models.GamificationProfile, error) {
	res, err := r.db.Exec(`
		UPDATE user_gamification
		SET streak_freezes_owned = streak_freezes_owned - 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = 1 AND streak_freezes_owned > 0
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to consume streak freeze: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to verify consumed streak freeze: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("no streak freezes available to consume")
	}

	return r.GetGamificationProfile()
}
