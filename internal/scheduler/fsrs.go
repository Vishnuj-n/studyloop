package scheduler

import (
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
