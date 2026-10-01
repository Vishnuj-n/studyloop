package scheduler

import (
	"fmt"
	"time"

	"ai-tutor/internal/models"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

// FlashcardStateToCard converts FlashcardState and timestamps to a go-fsrs Card.
func FlashcardStateToCard(state models.FlashcardState, dueAt, lastReviewedAt int64) fsrs.Card {
	var dueTime, lastReviewTime time.Time
	if dueAt > 0 {
		dueTime = time.Unix(dueAt, 0)
	}
	if lastReviewedAt > 0 {
		if lastReviewedAt > 1e12 {
			lastReviewTime = time.UnixMilli(lastReviewedAt)
		} else {
			lastReviewTime = time.Unix(lastReviewedAt, 0)
		}
	}

	var fsrsState fsrs.State
	switch state.StateCode {
	case 0:
		fsrsState = fsrs.New
	case 1:
		fsrsState = fsrs.Learning
	case 2:
		fsrsState = fsrs.Review
	case 3:
		fsrsState = fsrs.Relearning
	default:
		fsrsState = fsrs.New
	}

	return fsrs.Card{
		Due:            dueTime,
		Stability:      state.Stability,
		Difficulty:     state.Difficulty,
		ScheduledDays:  uint64(state.ScheduledDays),
		Reps:           uint64(state.Reps),
		Lapses:         uint64(state.Lapses),
		State:          fsrsState,
		LastReview:     lastReviewTime,
		RemainingSteps: 0,
	}
}

// Standard FSRS rating definitions mapping straight to your app inputs
const (
	Again = int(fsrs.Again) // 1
	Hard  = int(fsrs.Hard)  // 2
	Good  = int(fsrs.Good)  // 3
	Easy  = int(fsrs.Easy)  // 4
)

// NextReviewIntervals holds human-readable relative interval labels for all 4 ratings.
type NextReviewIntervals struct {
	Again string `json:"again"`
	Hard  string `json:"hard"`
	Good  string `json:"good"`
	Easy  string `json:"easy"`
}

// formatInterval converts a duration or day count into clean, compact Anki-style intervals (e.g. "<10m", "1d", "3d", "1.5mo", "1y").
func formatInterval(diff time.Duration, scheduledDays uint64) string {
	if diff <= 0 {
		return "<10m"
	}
	if diff < 24*time.Hour {
		mins := int(diff.Minutes())
		if mins <= 10 {
			return "<10m"
		}
		if mins < 60 {
			return fmt.Sprintf("%dm", mins)
		}
		hrs := int(diff.Hours())
		return fmt.Sprintf("%dh", hrs)
	}

	days := int(scheduledDays)
	if days <= 0 {
		days = int(diff.Hours() / 24)
	}
	if days <= 0 {
		return "<10m"
	}
	if days < 30 {
		return fmt.Sprintf("%dd", days)
	}
	if days < 365 {
		months := float64(days) / 30.0
		if months < 10 {
			return fmt.Sprintf("%.1fmo", months)
		}
		return fmt.Sprintf("%dmo", int(months+0.5))
	}
	years := float64(days) / 365.0
	return fmt.Sprintf("%.1fy", years)
}

// CalculateNextIntervals computes the 4 upcoming review intervals for a given card state.
func CalculateNextIntervals(fsrsCard fsrs.Card, now time.Time) NextReviewIntervals {
	p := fsrs.DefaultParam()
	p.RequestRetention = 0.9
	engine := fsrs.NewFSRS(p)

	if fsrsCard.Reps == 0 || fsrsCard.Due.IsZero() {
		fsrsCard.Due = now
	}
	if fsrsCard.State != fsrs.New && fsrsCard.Stability < 0.001 {
		fsrsCard.Stability = 0.001
	}

	schedulingCards, err := engine.Repeat(fsrsCard, now)
	if err != nil {
		return NextReviewIntervals{
			Again: "<10m",
			Hard:  "1d",
			Good:  "3d",
			Easy:  "7d",
		}
	}

	calc := func(r fsrs.Rating) string {
		item, exists := schedulingCards[r]
		if !exists {
			return ""
		}
		return formatInterval(item.Card.Due.Sub(now), item.Card.ScheduledDays)
	}

	return NextReviewIntervals{
		Again: calc(fsrs.Again),
		Hard:  calc(fsrs.Hard),
		Good:  calc(fsrs.Good),
		Easy:  calc(fsrs.Easy),
	}
}

// NextFSRSState calls the official open-spaced-repetition engine.
func NextFSRSState(fsrsCard fsrs.Card, rating int, now time.Time) (fsrs.Card, error) {
	// 1. Initialize the official engine configuration parameters
	p := fsrs.DefaultParam()
	p.RequestRetention = 0.9 // Enforces our 90% retention profile target
	engine := fsrs.NewFSRS(p)

	// Fallback mechanism for brand new cards flowing into the scheduling window
	if fsrsCard.Reps == 0 || fsrsCard.Due.IsZero() {
		fsrsCard.Due = now
	}
	if fsrsCard.State != fsrs.New && fsrsCard.Stability < 0.001 {
		fsrsCard.Stability = 0.001
	}

	// 3. Compute all 4 timeline variations simultaneously
	schedulingCards, err := engine.Repeat(fsrsCard, now)
	if err != nil {
		return fsrsCard, err
	}

	// 4. Extract the exact button response clicked by the user
	chosenRecord := schedulingCards[fsrs.Rating(rating)]

	return chosenRecord.Card, nil
}

