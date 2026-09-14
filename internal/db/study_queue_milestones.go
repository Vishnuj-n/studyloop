package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ai-tutor/internal/models"
)

// GetLatestQuizAttemptDetailsByTopic retrieves the payload and answers for the latest quiz attempt of a topic.
func (r *Repository) GetLatestQuizAttemptDetailsByTopic(topicID string) (string, string, error) {
	var payloadJSON, answersJSON string
	err := r.db.QueryRow(`
		SELECT sq.payload_json, qa.answers_json
		FROM quiz_attempts qa
		JOIN study_queue sq ON qa.task_id = sq.id
		WHERE sq.topic_id = ? AND sq.task_type = 'QUIZ'
		ORDER BY qa.completed_at DESC LIMIT 1
	`, strings.TrimSpace(topicID)).Scan(&payloadJSON, &answersJSON)
	return payloadJSON, answersJSON, err
}

// GetActiveRemedialTaskPayloadByTopic retrieves the payload_json of the active SOCRATIC_REMEDIAL task for a topic.
func (r *Repository) GetActiveRemedialTaskPayloadByTopic(topicID string) (string, error) {
	var payloadJSON string
	err := r.db.QueryRow(`
		SELECT COALESCE(payload_json, '')
		FROM study_queue
		WHERE topic_id = ? AND task_type = 'SOCRATIC_REMEDIAL' AND status = 'ACTIVE' LIMIT 1
	`, strings.TrimSpace(topicID)).Scan(&payloadJSON)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return payloadJSON, err
}

// GetQuestionsForQuizAttempts compiles all quiz questions from the original study_queue payload_json for the given quiz attempt IDs.
func (r *Repository) GetQuestionsForQuizAttempts(attemptIDs []string) ([]models.QuizTaskQuestion, error) {
	if len(attemptIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(sq.payload_json, '')
		FROM quiz_attempts qa
		JOIN study_queue sq ON qa.task_id = sq.id
		WHERE qa.id IN (%s)
	`, strings.Repeat("?,", len(attemptIDs)-1)+"?")
	args := make([]interface{}, len(attemptIDs))
	for i, id := range attemptIDs {
		args[i] = id
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var allQuestions []models.QuizTaskQuestion
	for rows.Next() {
		var payloadJSON string
		if err := rows.Scan(&payloadJSON); err != nil {
			return nil, err
		}
		if payloadJSON != "" {
			var payload models.QuizTaskPayload
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				return nil, fmt.Errorf("failed to decode quiz payload: %w", err)
			}
			allQuestions = append(allQuestions, payload.Questions...)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allQuestions, nil
}

// GetUnexaminedPassedQuizAttemptsByTopic returns passed quiz attempts for a topic that have not yet been included in any MILESTONE_EXAM task for the given notebook.
func (r *Repository) GetUnexaminedPassedQuizAttemptsByTopic(notebookID, topicID string) ([]QuizAttemptWithPayload, error) {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, fmt.Errorf("notebook ID is required")
	}
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, fmt.Errorf("topic ID is required")
	}

	// Verify topic belongs to notebook
	var exists int
	err := r.db.QueryRow(`
		SELECT 1 FROM notebook_topics
		WHERE notebook_id = ? AND topic_id = ?
		LIMIT 1
	`, notebookID, topicID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("topic %s not found in notebook %s", topicID, notebookID)
		}
		return nil, fmt.Errorf("failed to verify notebook topic: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT
			qa.id,
			qa.score,
			qa.passed,
			qa.answers_json,
			qa.completed_at,
			COALESCE(sq.payload_json, '')
		FROM quiz_attempts qa
		JOIN study_queue sq ON qa.task_id = sq.id
		WHERE sq.notebook_id = ?
		  AND sq.topic_id = ?
		  AND sq.task_type = 'QUIZ'
		  AND qa.passed = 1
		ORDER BY qa.completed_at ASC
	`, notebookID, topicID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	allAttempts, err := scanQuizAttemptsWithPayload(rows)
	if err != nil {
		return nil, err
	}

	unexamined := make([]QuizAttemptWithPayload, 0, len(allAttempts))
	for _, attempt := range allAttempts {
		hasMilestone, err := r.HasMilestoneExamForAttemptID(notebookID, attempt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed checking milestone for attempt %s: %w", attempt.ID, err)
		}
		if hasMilestone {
			continue
		}
		unexamined = append(unexamined, attempt)
	}

	return unexamined, nil
}
