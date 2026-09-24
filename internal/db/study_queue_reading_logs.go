package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"ai-tutor/internal/models"
)



// GetReadingTaskHistory fetches paginated reading tasks joined with notebook and topic titles for developer diagnostics.
func (r *Repository) GetReadingTaskHistory(notebookID string, limit, offset int) ([]models.ReadingTaskHistoryRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	notebookID = strings.TrimSpace(notebookID)

	var totalCount int
	var countErr error
	if notebookID != "" {
		countErr = r.db.QueryRow(`
			SELECT COUNT(*) FROM study_queue
			WHERE task_type IN ('READING', 'REREAD') AND notebook_id = ?
		`, notebookID).Scan(&totalCount)
	} else {
		countErr = r.db.QueryRow(`
			SELECT COUNT(*) FROM study_queue
			WHERE task_type IN ('READING', 'REREAD')
		`).Scan(&totalCount)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("failed to count reading history: %w", countErr)
	}

	query := `
		SELECT
			sq.id,
			sq.notebook_id,
			COALESCE(nb.title, 'Unknown Notebook'),
			COALESCE(sq.topic_id, ''),
			COALESCE(t.title, 'General Reading'),
			sq.task_type,
			sq.status,
			COALESCE(sq.start_page, 0),
			COALESCE(sq.end_page, 0),
			COALESCE(sq.current_page, COALESCE(sq.start_page, 0)),
			sq.created_at,
			COALESCE(sq.activated_at, ''),
			COALESCE(sq.completed_at, '')
		FROM study_queue sq
		LEFT JOIN notebooks nb ON nb.id = sq.notebook_id
		LEFT JOIN topics t ON t.id = sq.topic_id
		WHERE sq.task_type IN ('READING', 'REREAD')
	`
	args := []interface{}{}
	if notebookID != "" {
		query += ` AND sq.notebook_id = ? `
		args = append(args, notebookID)
	}
	query += ` ORDER BY sq.created_at DESC LIMIT ? OFFSET ? `
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed querying reading history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []models.ReadingTaskHistoryRecord
	for rows.Next() {
		var rec models.ReadingTaskHistoryRecord
		if err := rows.Scan(
			&rec.TaskID,
			&rec.NotebookID,
			&rec.NotebookTitle,
			&rec.TopicID,
			&rec.TopicTitle,
			&rec.TaskType,
			&rec.Status,
			&rec.StartPage,
			&rec.EndPage,
			&rec.CurrentPage,
			&rec.CreatedAt,
			&rec.ActivatedAt,
			&rec.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning reading history row: %w", err)
		}

		// Detect anomalies
		if rec.StartPage > rec.EndPage && rec.EndPage > 0 {
			rec.HasAnomaly = true
			rec.AnomalyReason = fmt.Sprintf("start page (%d) > end page (%d)", rec.StartPage, rec.EndPage)
		} else if rec.StartPage <= 0 || rec.EndPage <= 0 {
			rec.HasAnomaly = true
			rec.AnomalyReason = "unbound page range (0 or unassigned)"
		} else if rec.CurrentPage < rec.StartPage || (rec.EndPage > 0 && rec.CurrentPage > rec.EndPage) {
			rec.HasAnomaly = true
			rec.AnomalyReason = fmt.Sprintf("cursor page (%d) outside [%d-%d]", rec.CurrentPage, rec.StartPage, rec.EndPage)
		}

		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reading history rows iteration error: %w", err)
	}

	return records, totalCount, nil
}

// RevertReadingTaskSession rolls back a completed reading task back to ACTIVE, restores topic cursor, and deletes generated follow-ups.
func (r *Repository) RevertReadingTaskSession(taskID string) error {
	return r.withTx(func(tx *sql.Tx) error {
		return r.RevertReadingTaskSessionTx(tx, taskID)
	})
}

// RevertReadingTaskSessionTx executes the atomic reversion inside a transaction.
func (r *Repository) RevertReadingTaskSessionTx(tx *sql.Tx, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task ID is required")
	}

	var notebookID, topicID, taskType, status string
	var startPage, endPage int
	var completedAt sql.NullString

	err := tx.QueryRow(`
		SELECT notebook_id, COALESCE(topic_id, ''), task_type, status, COALESCE(start_page, 0), COALESCE(end_page, 0), completed_at
		FROM study_queue
		WHERE id = ?
	`, taskID).Scan(&notebookID, &topicID, &taskType, &status, &startPage, &endPage, &completedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to fetch task to revert: %w", err)
	}

	if taskType != string(models.StudyTaskTypeReading) && taskType != string(models.StudyTaskTypeReread) {
		return fmt.Errorf("only READING or REREAD tasks can be reverted (got %s)", taskType)
	}
	if status != string(models.StudyTaskStatusCompleted) {
		return fmt.Errorf("task is not in COMPLETED status (current: %s)", status)
	}

	// 1. Reset the reading task back to ACTIVE
	if _, err := tx.Exec(`
		UPDATE study_queue
		SET status = 'ACTIVE', completed_at = NULL, current_page = ?
		WHERE id = ?
	`, startPage, taskID); err != nil {
		return fmt.Errorf("failed resetting task to ACTIVE: %w", err)
	}

	// 2. Reset topic cursor if topic exists
	if topicID != "" && startPage > 0 {
		if _, err := tx.Exec(`
			UPDATE topics
			SET current_page_cursor = ?
			WHERE id = ?
		`, startPage, topicID); err != nil {
			return fmt.Errorf("failed resetting topic cursor: %w", err)
		}
	}

	// 3. Delete downstream generated tasks for subsequent pages in pending/active status
	if notebookID != "" && endPage > 0 {
		if topicID != "" {
			_, _ = tx.Exec(`
				DELETE FROM study_queue
				WHERE notebook_id = ? AND topic_id = ? AND start_page > ? AND status IN ('PENDING', 'ACTIVE')
			`, notebookID, topicID, endPage)
		} else {
			_, _ = tx.Exec(`
				DELETE FROM study_queue
				WHERE notebook_id = ? AND start_page > ? AND status IN ('PENDING', 'ACTIVE')
			`, notebookID, endPage)
		}
	}

	// 4. Delete quiz attempts recorded for the session
	if topicID != "" {
		_, _ = tx.Exec(`
			DELETE FROM quiz_attempts
			WHERE task_id IN (
				SELECT id FROM study_queue
				WHERE topic_id = ? AND start_page = ? AND task_type = 'QUIZ'
			)
		`, topicID, startPage)
	}

	// 5. Delete generated QUIZ and FLASHCARD_GENERATE tasks for this session range
	if topicID != "" {
		_, _ = tx.Exec(`
			DELETE FROM study_queue
			WHERE topic_id = ? AND start_page = ? AND task_type IN ('QUIZ', 'FLASHCARD_GENERATE')
		`, topicID, startPage)
	}

	// 6. Delete flashcards created at or after the completion timestamp if recorded
	if topicID != "" && completedAt.Valid && completedAt.String != "" {
		_, _ = tx.Exec(`
			DELETE FROM fsrs_cards
			WHERE topic_id = ? AND created_at >= ?
		`, topicID, completedAt.String)
	}

	return nil
}

