package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// NotebookCertificateStats contains all verifiable aggregated stats for a completed notebook.
type NotebookCertificateStats struct {
	NotebookID         string  `json:"notebook_id"`
	NotebookTitle      string  `json:"notebook_title"`
	FileType           string  `json:"file_type"`
	PageCount          int     `json:"page_count"`
	CompletionPercent  int     `json:"completion_percent"`
	QuizzesPassed      int     `json:"quizzes_passed"`
	AverageQuizScore   float64 `json:"average_quiz_score"`
	MilestonesCleared  int     `json:"milestones_cleared"`
	FlashcardsMastered int     `json:"flashcards_mastered"`
	TotalReviewsCount  int     `json:"total_reviews_count"`
	TotalStudyMinutes  int     `json:"total_study_minutes"`
	CompletedAt        string  `json:"completed_at"`
	UserTitle          string  `json:"user_title"`
	UserLevel          int     `json:"user_level"`
	IsUnlocked         bool    `json:"is_unlocked"`
}

// GetNotebookCertificateStats aggregates all verified study metrics for a notebook certificate.
func (r *Repository) GetNotebookCertificateStats(notebookID string) (*NotebookCertificateStats, error) {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, fmt.Errorf("notebook id is required")
	}

	// 1. Fetch Notebook info and completion percent
	notebooks, err := r.GetNotebooks("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notebooks: %w", err)
	}

	var targetNb *NotebookCertificateStats
	for _, nb := range notebooks {
		if nb.ID == notebookID {
			targetNb = &NotebookCertificateStats{
				NotebookID:        nb.ID,
				NotebookTitle:     nb.Title,
				FileType:          nb.FileType,
				PageCount:         nb.PageCount,
				CompletionPercent: nb.CompletionPercent,
			}
			break
		}
	}

	if targetNb == nil {
		return nil, fmt.Errorf("notebook %s not found", notebookID)
	}

	// 2. Fetch User Gamification Title & Level
	gamificationProf, err := r.GetGamificationProfile()
	if err == nil && gamificationProf != nil {
		targetNb.UserTitle = gamificationProf.CurrentTitle
		targetNb.UserLevel = gamificationProf.Level
	} else {
		targetNb.UserTitle = "The Apprentice I"
		targetNb.UserLevel = 1
	}

	// 3. Count Quizzes Passed & Average Score for this notebook
	var quizCount int
	var avgScore sql.NullFloat64
	err = r.db.QueryRow(`
		SELECT COUNT(qa.id), AVG(qa.score)
		FROM quiz_attempts qa
		JOIN study_queue sq ON qa.task_id = sq.id
		WHERE sq.notebook_id = ? AND sq.task_type = 'QUIZ' AND qa.passed = 1
	`, notebookID).Scan(&quizCount, &avgScore)
	if err == nil {
		targetNb.QuizzesPassed = quizCount
		if avgScore.Valid {
			targetNb.AverageQuizScore = avgScore.Float64
		}
	}

	// 4. Count Milestones Cleared
	var milestonesCleared int
	err = r.db.QueryRow(`
		SELECT COUNT(DISTINCT sq.id)
		FROM study_queue sq
		WHERE sq.notebook_id = ? AND sq.task_type = 'MILESTONE_EXAM' AND sq.status = 'COMPLETED'
	`, notebookID).Scan(&milestonesCleared)
	if err == nil {
		targetNb.MilestonesCleared = milestonesCleared
	}

	// 5. Count Flashcards Mastered & Review Logs for this notebook's topics
	var masteredCount int
	var reviewLogsCount int
	err = r.db.QueryRow(`
		SELECT 
			COUNT(DISTINCT fc.id),
			COALESCE((
				SELECT COUNT(r.id)
				FROM fsrs_review_log r
				WHERE r.activity_type = 'flashcard' AND r.reference_id IN (
					SELECT id FROM fsrs_cards WHERE topic_id = n.topic_id 
					OR topic_id IN (SELECT topic_id FROM notebook_topics WHERE notebook_id = n.id)
				)
			), 0)
		FROM fsrs_cards fc
		JOIN notebooks n ON (fc.topic_id = n.topic_id OR fc.topic_id IN (SELECT topic_id FROM notebook_topics WHERE notebook_id = n.id))
		WHERE n.id = ? AND fc.suspended = 0
	`, notebookID).Scan(&masteredCount, &reviewLogsCount)
	if err == nil {
		targetNb.FlashcardsMastered = masteredCount
		targetNb.TotalReviewsCount = reviewLogsCount
	}

	// 6. Estimate Study Minutes (reading completions, quizzes, reviews)
	var readingTasksCompleted int
	_ = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM study_queue
		WHERE notebook_id = ? AND task_type IN ('READING', 'REREAD') AND status = 'COMPLETED'
	`, notebookID).Scan(&readingTasksCompleted)

	// Approximate: 15 min per reading session + 5 min per quiz + 1 min per flashcard review
	targetNb.TotalStudyMinutes = (readingTasksCompleted * 15) + (targetNb.QuizzesPassed * 5) + (targetNb.TotalReviewsCount * 1)
	if targetNb.TotalStudyMinutes == 0 && targetNb.CompletionPercent > 0 {
		targetNb.TotalStudyMinutes = targetNb.PageCount * 2 // fallback estimate
	}

	// 7. Find completion date (or fallback to latest task completion / now)
	var latestCompletedAt sql.NullString
	_ = r.db.QueryRow(`
		SELECT MAX(completed_at)
		FROM study_queue
		WHERE notebook_id = ? AND status = 'COMPLETED'
	`, notebookID).Scan(&latestCompletedAt)

	if latestCompletedAt.Valid && latestCompletedAt.String != "" {
		targetNb.CompletedAt = latestCompletedAt.String
	} else {
		targetNb.CompletedAt = time.Now().Format("2006-01-02 15:04:05")
	}

	// 8. Determine unlock status (100% completion or no pending/active tasks if started)
	var remainingTasks int
	_ = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM study_queue
		WHERE notebook_id = ? AND status IN ('PENDING', 'ACTIVE')
	`, notebookID).Scan(&remainingTasks)

	targetNb.IsUnlocked = targetNb.CompletionPercent >= 100 || (remainingTasks == 0 && targetNb.QuizzesPassed > 0)

	return targetNb, nil
}

// DevUnlockNotebookCertificate is a developer bypass that completes remaining reading tasks & topic cursors for a notebook so the user can immediately test 100% completion & certificates.
func (r *Repository) DevUnlockNotebookCertificate(notebookID string) (*NotebookCertificateStats, error) {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, fmt.Errorf("notebook id is required")
	}

	err := r.withTx(func(tx *sql.Tx) error {
		// 1. Advance all topics linked to this notebook to their end_page and mark 'completed'
		_, err := tx.Exec(`
			UPDATE topics
			SET current_page_cursor = end_page, status = 'completed', updated_at = CURRENT_TIMESTAMP
			WHERE id IN (
				SELECT topic_id FROM notebook_topics WHERE notebook_id = ?
				UNION
				SELECT topic_id FROM notebooks WHERE id = ? AND topic_id IS NOT NULL AND topic_id != ''
			)
		`, notebookID, notebookID)
		if err != nil {
			return err
		}

		// 2. Mark any pending/active tasks for this notebook as COMPLETED
		_, err = tx.Exec(`
			UPDATE study_queue
			SET status = 'COMPLETED', completed_at = CURRENT_TIMESTAMP
			WHERE notebook_id = ? AND status IN ('PENDING', 'ACTIVE')
		`, notebookID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return r.GetNotebookCertificateStats(notebookID)
}


