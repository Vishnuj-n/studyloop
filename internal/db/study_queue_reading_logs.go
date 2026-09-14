package db

import (
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

	return records, totalCount, nil
}
