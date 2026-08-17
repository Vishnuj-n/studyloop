package db

import (
	"ai-tutor/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type QuizAttemptWithPayload struct {
	ID           string
	Score        int
	Passed       bool
	AnswersJSON  string
	CompletedAt  int64
	QuizPayload  string
	PassingScore int
}

func scanQuizAttemptsWithPayload(rows *sql.Rows) ([]QuizAttemptWithPayload, error) {
	var attempts []QuizAttemptWithPayload
	for rows.Next() {
		var attempt QuizAttemptWithPayload
		if err := rows.Scan(
			&attempt.ID,
			&attempt.Score,
			&attempt.Passed,
			&attempt.AnswersJSON,
			&attempt.CompletedAt,
			&attempt.QuizPayload,
		); err != nil {
			return nil, err
		}

		var payload models.QuizTaskPayload
		if err := json.Unmarshal([]byte(attempt.QuizPayload), &payload); err == nil && payload.PassingScore > 0 {
			attempt.PassingScore = payload.PassingScore
		} else {
			attempt.PassingScore = 70
		}

		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attempts, nil
}

// readActiveProfileID fetches only the active_profile_id from user_settings.
func (r *Repository) readActiveProfileID() (string, error) {
	var activeProfileID sql.NullString
	if err := r.db.QueryRow(`
		SELECT COALESCE(active_profile_id, '') FROM user_settings WHERE id = 1
	`).Scan(&activeProfileID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("reading user_settings: %w", err)
	}
	if activeProfileID.Valid {
		return activeProfileID.String, nil
	}
	return "", nil
}

// assignTaskTitle sets task.Title based on task type and available topic/notebook titles.
func assignTaskTitle(task *models.StudyQueueTask, topicTitle, notebookTitle string) {
	if task.TaskType == models.StudyTaskTypeFlashcardReview {
		task.Title = notebookTitle
	} else if topicTitle != "" {
		task.Title = topicTitle
	} else if notebookTitle != "" {
		task.Title = notebookTitle
	} else {
		task.Title = "Task"
	}
}

func parseSQLiteTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
