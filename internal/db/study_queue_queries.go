package db

import (
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GetTaskByID returns one queue task by id.
func (r *Repository) GetTaskByID(taskID string) (*models.StudyQueueTask, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	task := &models.StudyQueueTask{}
	err := r.db.QueryRow(`
		SELECT
			id, notebook_id, COALESCE(topic_id, ''), task_type, status, priority,
			COALESCE(created_at, ''), COALESCE(activated_at, ''), COALESCE(completed_at, ''),
			COALESCE(payload_json, ''), COALESCE(start_page, 0), COALESCE(end_page, 0)
		FROM study_queue
		WHERE id = ?
	`, taskID).Scan(
		&task.ID, &task.NotebookID, &task.TopicID, &task.TaskType, &task.Status, &task.Priority,
		&task.CreatedAt, &task.ActivatedAt, &task.CompletedAt, &task.PayloadJSON, &task.StartPage, &task.EndPage,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

// GetAllPendingTasks returns all pending tasks ordered by deterministic queue rules.
func (r *Repository) GetAllPendingTasks() ([]models.StudyQueueTask, error) {
	activeProfileID, err := r.readActiveProfileID()
	if err != nil {
		return nil, fmt.Errorf("GetAllPendingTasks: %w", err)
	}

	if activeProfileID == "" {
		return r.getPendingTasksNoProfile()
	}
	return r.getPendingTasksWithProfile(activeProfileID)
}

// getPendingTasksNoProfile returns pending tasks without profile filtering.
func (r *Repository) getPendingTasksNoProfile() ([]models.StudyQueueTask, error) {
	query := `
		SELECT
			sq.id, sq.notebook_id, COALESCE(sq.topic_id, ''), sq.task_type, sq.status, sq.priority,
			COALESCE(sq.created_at, ''), COALESCE(sq.activated_at, ''), COALESCE(sq.completed_at, ''),
			COALESCE(sq.payload_json, ''),
			COALESCE(NULLIF(sq.start_page, 0), COALESCE(t.start_page, 0)),
			COALESCE(NULLIF(sq.end_page, 0), COALESCE(t.end_page, 0)),
			COALESCE(t.title, ''), COALESCE(n.title, ''), COALESCE(n.priority, 5)
		FROM study_queue sq
		JOIN notebooks n ON sq.notebook_id = n.id
		LEFT JOIN topics t ON sq.topic_id = t.id
		WHERE sq.status = 'PENDING'
		ORDER BY
			CASE sq.task_type
				WHEN 'FLASHCARD_GENERATE' THEN 7 WHEN 'SOCRATIC_REMEDIAL' THEN 6
				WHEN 'FLASHCARD_REVIEW' THEN 5 WHEN 'REREAD' THEN 4
				WHEN 'QUIZ' THEN 3 WHEN 'MILESTONE_EXAM' THEN 2
				WHEN 'READING' THEN 1 WHEN 'EXAMINER' THEN 0 ELSE 0
			END DESC,
			COALESCE(n.priority, 5) DESC,
			(SELECT COALESCE(MAX(sq2.completed_at), '') FROM study_queue sq2 WHERE sq2.notebook_id = sq.notebook_id AND sq2.status = 'COMPLETED') ASC,
			sq.priority ASC,
			COALESCE(sq.created_at, '') ASC, sq.id ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return r.scanPendingTaskRows(rows)
}

// getPendingTasksWithProfile returns pending tasks filtered by active profile.
func (r *Repository) getPendingTasksWithProfile(activeProfileID string) ([]models.StudyQueueTask, error) {
	query := `
		SELECT
			sq.id, sq.notebook_id, COALESCE(sq.topic_id, ''), sq.task_type, sq.status, sq.priority,
			COALESCE(sq.created_at, ''), COALESCE(sq.activated_at, ''), COALESCE(sq.completed_at, ''),
			COALESCE(sq.payload_json, ''),
			COALESCE(NULLIF(sq.start_page, 0), COALESCE(t.start_page, 0)),
			COALESCE(NULLIF(sq.end_page, 0), COALESCE(t.end_page, 0)),
			COALESCE(t.title, ''), COALESCE(n.title, ''), COALESCE(n.priority, 5)
		FROM study_queue sq
		JOIN notebooks n ON sq.notebook_id = n.id
		LEFT JOIN topics t ON sq.topic_id = t.id
		WHERE sq.status = 'PENDING'
		  AND ( ? = '' OR n.profile_id = ? )
		  AND ( ? = '' OR sq.task_type = 'FLASHCARD_REVIEW' OR sq.task_type = 'FLASHCARD_GENERATE' OR n.study_status = 'active' )
		ORDER BY
			CASE sq.task_type
				WHEN 'FLASHCARD_GENERATE' THEN 7 WHEN 'SOCRATIC_REMEDIAL' THEN 6
				WHEN 'FLASHCARD_REVIEW' THEN 5 WHEN 'REREAD' THEN 4
				WHEN 'QUIZ' THEN 3 WHEN 'MILESTONE_EXAM' THEN 2
				WHEN 'READING' THEN 1 WHEN 'EXAMINER' THEN 0 ELSE 0
			END DESC,
			COALESCE(n.priority, 5) DESC,
			(SELECT COALESCE(MAX(sq2.completed_at), '') FROM study_queue sq2 WHERE sq2.notebook_id = sq.notebook_id AND sq2.status = 'COMPLETED') ASC,
			sq.priority ASC,
			COALESCE(sq.created_at, '') ASC, sq.id ASC
	`
	rows, err := r.db.Query(query, activeProfileID, activeProfileID, activeProfileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return r.scanPendingTaskRows(rows)
}

// scanPendingTaskRows scans rows with topic_title, notebook_title, notebook_priority columns and assigns titles.
func (r *Repository) scanPendingTaskRows(rows *sql.Rows) ([]models.StudyQueueTask, error) {
	tasks := make([]models.StudyQueueTask, 0)
	for rows.Next() {
		var task models.StudyQueueTask
		var topicTitle, notebookTitle string
		var notebookPriority int
		if err := rows.Scan(
			&task.ID, &task.NotebookID, &task.TopicID, &task.TaskType, &task.Status, &task.Priority,
			&task.CreatedAt, &task.ActivatedAt, &task.CompletedAt, &task.PayloadJSON,
			&task.StartPage, &task.EndPage, &topicTitle, &notebookTitle, &notebookPriority,
		); err != nil {
			return nil, err
		}
		assignTaskTitle(&task, topicTitle, notebookTitle)
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// GetAllActiveTasks returns all active tasks ordered by activation time.
func (r *Repository) GetAllActiveTasks() ([]models.StudyQueueTask, error) {
	activeProfileID, err := r.readActiveProfileID()
	if err != nil {
		return nil, fmt.Errorf("GetAllActiveTasks: %w", err)
	}

	query := `
		SELECT
			sq.id, sq.notebook_id, COALESCE(sq.topic_id, ''), sq.task_type, sq.status, sq.priority,
			COALESCE(sq.created_at, ''), COALESCE(sq.activated_at, ''), COALESCE(sq.completed_at, ''),
			COALESCE(sq.payload_json, ''),
			COALESCE(NULLIF(sq.start_page, 0), COALESCE(t.start_page, 0)),
			COALESCE(NULLIF(sq.end_page, 0), COALESCE(t.end_page, 0)),
			COALESCE(t.title, ''), COALESCE(n.title, '')
		FROM study_queue sq
		JOIN notebooks n ON sq.notebook_id = n.id
		LEFT JOIN topics t ON sq.topic_id = t.id
		WHERE sq.status = 'ACTIVE'
		  AND ( ? = '' OR n.profile_id = ? )
		ORDER BY sq.activated_at ASC
	`

	rows, err := r.db.Query(query, activeProfileID, activeProfileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]models.StudyQueueTask, 0)
	for rows.Next() {
		var task models.StudyQueueTask
		var topicTitle, notebookTitle string
		if err := rows.Scan(
			&task.ID, &task.NotebookID, &task.TopicID, &task.TaskType, &task.Status, &task.Priority,
			&task.CreatedAt, &task.ActivatedAt, &task.CompletedAt, &task.PayloadJSON,
			&task.StartPage, &task.EndPage, &topicTitle, &notebookTitle,
		); err != nil {
			return nil, err
		}
		assignTaskTitle(&task, topicTitle, notebookTitle)
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// GetNextTask returns the next pending task ordered by deterministic queue rules.
func (r *Repository) GetNextTask(notebookID string) (*models.StudyQueueTask, error) {
	notebookID = strings.TrimSpace(notebookID)
	utils.Debugf("[QUEUE] GetNextTask filter status=PENDING notebookID=%q", notebookID)

	activeProfileID, err := r.readActiveProfileID()
	if err != nil {
		return nil, fmt.Errorf("GetNextTask: %w", err)
	}

	if activeProfileID == "" {
		return r.getNextTaskNoProfile(notebookID)
	}
	return r.getNextTaskWithProfile(notebookID, activeProfileID)
}

// getNextTaskNoProfile returns the next pending task without profile filtering.
func (r *Repository) getNextTaskNoProfile(notebookID string) (*models.StudyQueueTask, error) {
	query := `
		SELECT
			sq.id, sq.notebook_id, COALESCE(sq.topic_id, ''), sq.task_type, sq.status, sq.priority,
			COALESCE(sq.created_at, ''), COALESCE(sq.activated_at, ''), COALESCE(sq.completed_at, ''),
			COALESCE(sq.payload_json, ''),
			COALESCE(NULLIF(sq.start_page, 0), COALESCE(t.start_page, 0)),
			COALESCE(NULLIF(sq.end_page, 0), COALESCE(t.end_page, 0)),
			COALESCE(t.title, ''), COALESCE(n.title, ''), COALESCE(n.priority, 5)
		FROM study_queue sq
		JOIN notebooks n ON sq.notebook_id = n.id
		LEFT JOIN topics t ON sq.topic_id = t.id
		WHERE sq.status = 'PENDING'
	`
	args := make([]interface{}, 0, 2)
	if notebookID != "" {
		query += ` AND sq.notebook_id = ?`
		args = append(args, notebookID)
	}
	query += `
		ORDER BY
			CASE sq.task_type
				WHEN 'FLASHCARD_GENERATE' THEN 7 WHEN 'SOCRATIC_REMEDIAL' THEN 6
				WHEN 'FLASHCARD_REVIEW' THEN 5 WHEN 'REREAD' THEN 4
				WHEN 'QUIZ' THEN 3 WHEN 'MILESTONE_EXAM' THEN 2
				WHEN 'READING' THEN 1 WHEN 'EXAMINER' THEN 0 ELSE 0
			END DESC,
			COALESCE(n.priority, 5) DESC,
			(SELECT COALESCE(MAX(sq2.completed_at), '') FROM study_queue sq2 WHERE sq2.notebook_id = sq.notebook_id AND sq2.status = 'COMPLETED') ASC,
			sq.priority ASC,
			COALESCE(sq.created_at, '') ASC, sq.id ASC
		LIMIT 1
	`
	return r.scanNextPendingTask(query, args...)
}

// getNextTaskWithProfile returns the next pending task filtered by active profile.
func (r *Repository) getNextTaskWithProfile(notebookID, activeProfileID string) (*models.StudyQueueTask, error) {
	query := `
		SELECT
			sq.id, sq.notebook_id, COALESCE(sq.topic_id, ''), sq.task_type, sq.status, sq.priority,
			COALESCE(sq.created_at, ''), COALESCE(sq.activated_at, ''), COALESCE(sq.completed_at, ''),
			COALESCE(sq.payload_json, ''),
			COALESCE(NULLIF(sq.start_page, 0), COALESCE(t.start_page, 0)),
			COALESCE(NULLIF(sq.end_page, 0), COALESCE(t.end_page, 0)),
			COALESCE(t.title, ''), COALESCE(n.title, ''), COALESCE(n.priority, 5)
		FROM study_queue sq
		JOIN notebooks n ON sq.notebook_id = n.id
		LEFT JOIN topics t ON sq.topic_id = t.id
		WHERE sq.status = 'PENDING'
	`
	args := make([]interface{}, 0, 5)
	if activeProfileID != "" {
		query += ` AND n.profile_id = ?`
		args = append(args, activeProfileID)
	}
	if notebookID != "" {
		query += ` AND sq.notebook_id = ?`
		args = append(args, notebookID)
	} else {
		query += ` AND (sq.task_type = 'FLASHCARD_REVIEW' OR sq.task_type = 'FLASHCARD_GENERATE' OR n.study_status = 'active')`
	}

	query += `
		ORDER BY
			CASE sq.task_type
				WHEN 'FLASHCARD_GENERATE' THEN 7 WHEN 'SOCRATIC_REMEDIAL' THEN 6
				WHEN 'FLASHCARD_REVIEW' THEN 5 WHEN 'REREAD' THEN 4
				WHEN 'QUIZ' THEN 3 WHEN 'MILESTONE_EXAM' THEN 2
				WHEN 'READING' THEN 1 WHEN 'EXAMINER' THEN 0 ELSE 0
			END DESC,
			COALESCE(n.priority, 5) DESC,
			(SELECT COALESCE(MAX(sq2.completed_at), '') FROM study_queue sq2 WHERE sq2.notebook_id = sq.notebook_id AND sq2.status = 'COMPLETED') ASC,
			sq.priority ASC, n.title ASC,
			COALESCE(sq.created_at, '') ASC, sq.id ASC
		LIMIT 1
	`
	return r.scanNextPendingTask(query, args...)
}

// scanNextPendingTask executes a query expecting one row and returns a title-assigned task.
func (r *Repository) scanNextPendingTask(query string, args ...interface{}) (*models.StudyQueueTask, error) {
	task := &models.StudyQueueTask{}
	var topicTitle, notebookTitle string
	var notebookPriority int
	err := r.db.QueryRow(query, args...).Scan(
		&task.ID, &task.NotebookID, &task.TopicID, &task.TaskType, &task.Status, &task.Priority,
		&task.CreatedAt, &task.ActivatedAt, &task.CompletedAt, &task.PayloadJSON,
		&task.StartPage, &task.EndPage, &topicTitle, &notebookTitle, &notebookPriority,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoPendingTasks
	}
	if err != nil {
		return nil, err
	}
	assignTaskTitle(task, topicTitle, notebookTitle)
	return task, nil
}

// GetReadingTask returns one reader-compatible task with locked bounds and persisted cursor.
func (r *Repository) GetReadingTask(taskID string) (*models.ReadingTask, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	task := &models.ReadingTask{}
	err := r.db.QueryRow(`
		SELECT
			sq.id,
			sq.notebook_id,
			COALESCE(sq.topic_id, ''),
			COALESCE(sq.start_page, 0),
			COALESCE(sq.end_page, 0),
			COALESCE(rp.current_page, COALESCE(sq.start_page, 0)),
			COALESCE(nb.file_hash, '')
		FROM study_queue sq
		LEFT JOIN reading_progress rp ON rp.task_id = sq.id
		LEFT JOIN notebooks nb ON nb.id = sq.notebook_id
		WHERE sq.id = ? AND sq.task_type IN ('READING', 'REREAD')
	`, taskID).Scan(
		&task.TaskID,
		&task.NotebookID,
		&task.TopicID,
		&task.StartPage,
		&task.EndPage,
		&task.CurrentPage,
		&task.FileHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	// If task was inserted without explicit page bounds, fall back to topic page bounds.
	// This allows READING tasks created without bounds to still be initialized and completed.
	if (task.StartPage <= 0 || task.EndPage <= 0) && task.TopicID != "" {
		var topicStart, topicEnd int
		boundsErr := r.db.QueryRow(`
			SELECT COALESCE(start_page, 1), COALESCE(end_page, start_page)
			FROM topics WHERE id = ?
		`, task.TopicID).Scan(&topicStart, &topicEnd)
		if boundsErr == nil && topicStart > 0 && topicEnd >= topicStart {
			if task.StartPage <= 0 {
				task.StartPage = topicStart
			}
			if task.EndPage <= 0 {
				task.EndPage = topicEnd
			}
		}
	}
	// After fallback: if bounds are still missing or invalid, return an explicit error.
	if task.StartPage <= 0 || task.EndPage <= 0 {
		return nil, fmt.Errorf("reading task has no valid page bounds: startPage=%d, endPage=%d — set start_page/end_page on the task or ensure topic has page bounds", task.StartPage, task.EndPage)
	}
	if task.EndPage < task.StartPage {
		return nil, fmt.Errorf("reading task has invalid page bounds: endPage=%d must be >= startPage=%d", task.EndPage, task.StartPage)
	}

	// Clamp current page to bounds
	if task.CurrentPage < task.StartPage {
		task.CurrentPage = task.StartPage
	}
	if task.CurrentPage > task.EndPage {
		task.CurrentPage = task.EndPage
	}
	return task, nil
}

// GetReadingProgressPage retrieves the current page progress for a task.
func (r *Repository) GetReadingProgressPage(taskID string) (int, error) {
	var currentPage int
	err := r.db.QueryRow(`
		SELECT COALESCE(current_page, 0) FROM reading_progress WHERE task_id = ?
	`, taskID).Scan(&currentPage)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return currentPage, err
}

// CountTasksByTopicTypeAndStatus counts study_queue tasks matching the given filters.
// Pass empty string for any filter to skip it.
func (r *Repository) CountTasksByTopicTypeAndStatus(topicID, taskType, status string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM study_queue
		WHERE (? = '' OR topic_id = ?)
		  AND (? = '' OR task_type = ?)
		  AND (? = '' OR status = ?)
	`, topicID, topicID, taskType, taskType, status, status).Scan(&count)
	return count, err
}

// GetLatestQuizAttemptScoreByTopic returns the score and passed status of the latest completed quiz attempt for a topic.
func (r *Repository) GetLatestQuizAttemptScoreByTopic(topicID string) (int, bool, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return 0, false, fmt.Errorf("topic ID is required")
	}
	var score int
	var passed bool
	err := r.db.QueryRow(`
		SELECT qa.score, qa.passed
		FROM quiz_attempts qa
		JOIN study_queue sq ON qa.task_id = sq.id
		WHERE sq.topic_id = ? AND sq.task_type = 'QUIZ'
		ORDER BY qa.completed_at DESC
		LIMIT 1
	`, topicID).Scan(&score, &passed)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return score, passed, nil
}

// CountCompletedQuizzesByNotebook returns the count of passed quiz attempts for a notebook.
func (r *Repository) CountCompletedQuizzesByNotebook(notebookID string) (int, error) {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return 0, fmt.Errorf("notebook ID is required")
	}

	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM quiz_attempts qa
		JOIN study_queue sq ON qa.task_id = sq.id
		WHERE sq.notebook_id = ?
		  AND sq.task_type = 'QUIZ'
		  AND qa.passed = 1
	`, notebookID).Scan(&count)
	return count, err
}

// GetLastNQuizAttemptsWithCorrectness returns recent passed quiz attempts with original quiz payloads.
func (r *Repository) GetLastNQuizAttemptsWithCorrectness(notebookID string, n int) ([]QuizAttemptWithPayload, error) {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, fmt.Errorf("notebook ID is required")
	}
	if n <= 0 {
		return nil, fmt.Errorf("n must be positive")
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
		  AND sq.task_type = 'QUIZ'
		  AND qa.passed = 1
		ORDER BY qa.completed_at DESC
		LIMIT ?
	`, notebookID, n)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanQuizAttemptsWithPayload(rows)
}

// HasMilestoneExamForAttemptID reports whether a milestone exam already includes the given quiz attempt.
func (r *Repository) HasMilestoneExamForAttemptID(notebookID, attemptID string) (bool, error) {
	notebookID = strings.TrimSpace(notebookID)
	attemptID = strings.TrimSpace(attemptID)
	if notebookID == "" {
		return false, fmt.Errorf("notebook ID is required")
	}
	if attemptID == "" {
		return false, fmt.Errorf("attempt ID is required")
	}

	rows, err := r.db.Query(`
		SELECT COALESCE(payload_json, '')
		FROM study_queue
		WHERE notebook_id = ?
		  AND task_type = 'MILESTONE_EXAM'
	`, notebookID)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var payloadJSON string
		if err := rows.Scan(&payloadJSON); err != nil {
			return false, err
		}
		if payloadJSON == "" {
			continue
		}
		var payload models.MilestoneExamPayload
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return false, err
		}
		if _, ok := payload.Quizzes[attemptID]; ok {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

// GetPassedQuizAttempts returns all passed quiz attempts for a notebook, ordered by completed_at ASC.
func (r *Repository) GetPassedQuizAttempts(notebookID string) ([]QuizAttemptWithPayload, error) {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, fmt.Errorf("notebook ID is required")
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
		  AND sq.task_type = 'QUIZ'
		  AND qa.passed = 1
		ORDER BY qa.completed_at ASC
	`, notebookID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanQuizAttemptsWithPayload(rows)
}

// GetCompletedTaskTimes returns a list of completion times in UTC.
func (r *Repository) GetCompletedTaskTimes() ([]time.Time, error) {
	rows, err := r.db.Query(`
		SELECT completed_at
		FROM study_queue
		WHERE status = 'COMPLETED' AND completed_at IS NOT NULL AND completed_at != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("GetCompletedTaskTimes query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var times []time.Time
	for rows.Next() {
		var completedAtStr string
		if err := rows.Scan(&completedAtStr); err != nil {
			return nil, fmt.Errorf("GetCompletedTaskTimes scan: %w", err)
		}
		t, err := parseSQLiteTimestamp(completedAtStr)
		if err != nil {
			utils.Warnf("[QUEUE] GetCompletedTaskTimes failed to parse completed_at %q: %v", completedAtStr, err)
			continue
		}
		times = append(times, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetCompletedTaskTimes rows error: %w", err)
	}
	return times, nil
}

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
