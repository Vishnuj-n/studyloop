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
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNoPendingTasks = errors.New("no pending tasks in queue")
	ErrTaskNotPending = errors.New("task is not in PENDING status")
	ErrTaskNotActive  = errors.New("task is not in ACTIVE status")
	ErrTaskNotFound   = errors.New("task not found")
)

// InsertStudyTask inserts one task row in study_queue.
func (r *Repository) InsertStudyTask(task models.StudyQueueTask) error {
	return r.withTx(func(tx *sql.Tx) error {
		return r.InsertStudyTaskTx(tx, task)
	})
}

// InsertStudyTaskTx inserts one task row in study_queue inside an existing transaction.
func (r *Repository) InsertStudyTaskTx(tx *sql.Tx, task models.StudyQueueTask) error {
	task.ID = strings.TrimSpace(task.ID)
	task.NotebookID = strings.TrimSpace(task.NotebookID)
	task.TopicID = strings.TrimSpace(task.TopicID)
	task.PayloadJSON = strings.TrimSpace(task.PayloadJSON)
	if task.ID == "" {
		return fmt.Errorf("task id is required")
	}
	if task.NotebookID == "" {
		return fmt.Errorf("notebook id is required")
	}
	if strings.TrimSpace(string(task.TaskType)) == "" {
		return fmt.Errorf("task type is required")
	}
	if strings.TrimSpace(string(task.Status)) == "" {
		task.Status = models.StudyTaskStatusPending
	}

	_, err := tx.Exec(`
		INSERT INTO study_queue (
			id, notebook_id, topic_id, task_type, status, priority, payload_json, start_page, end_page
		) VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), ?, ?)
	`, task.ID, task.NotebookID, task.TopicID, string(task.TaskType), string(task.Status), task.Priority, task.PayloadJSON, task.StartPage, task.EndPage)
	if err == nil {
		utils.LogQueueTaskCreated(task.ID, string(task.TaskType), task.NotebookID, task.TopicID)
	}
	return err
}

// ActivateTaskTx moves one task from PENDING to ACTIVE within a transaction.
func (r *Repository) ActivateTaskTx(tx *sql.Tx, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	var beforeStatus string
	var taskType string
	if err := tx.QueryRow(`SELECT COALESCE(status, ''), COALESCE(task_type, '') FROM study_queue WHERE id = ?`, taskID).Scan(&beforeStatus, &taskType); err == nil {
		utils.Debugf("[QUEUE] ActivateTaskTx before update taskID=%s status=%s taskType=%s", taskID, beforeStatus, taskType)
	} else {
		utils.Warnf("[QUEUE] ActivateTaskTx before update taskID=%s statusLoadErr=%v", taskID, err)
	}
	res, err := tx.Exec(`
		UPDATE study_queue
		SET status = 'ACTIVE', activated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'PENDING'
	`, taskID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		utils.LogQueueTransition(taskID, taskType, string(models.StudyTaskStatusPending), string(models.StudyTaskStatusActive), "task_activated")
		return nil
	}
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM study_queue WHERE id = ?`, taskID).Scan(&exists); err != nil {
		utils.Warnf("[QUEUE] ActivateTaskTx existence check error taskID=%s err=%v", taskID, err)
		return err
	}
	if exists == 0 {
		utils.Warnf("[QUEUE] ActivateTaskTx rejected taskID=%s reason=not_found", taskID)
		return ErrTaskNotFound
	}
	utils.Warnf("[QUEUE] ActivateTaskTx rejected taskID=%s reason=not_pending status=%s", taskID, beforeStatus)
	return ErrTaskNotPending
}

// ActivateTask moves one task from PENDING to ACTIVE.
func (r *Repository) ActivateTask(taskID string) error {
	return r.withTx(func(tx *sql.Tx) error {
		return r.ActivateTaskTx(tx, taskID)
	})
}

// CompleteTaskTx marks ACTIVE task as terminal and inserts explicit follow-up tasks transactionally.
func (r *Repository) CompleteTaskTx(tx *sql.Tx, taskID string, result models.CompletionResult) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	utils.Debugf("[QUEUE] CompleteTaskTx reading task completion update start taskID=%s", taskID)
	status := strings.TrimSpace(string(result.Status))
	if status == "" {
		status = string(models.StudyTaskStatusCompleted)
	}
	if status != string(models.StudyTaskStatusCompleted) && status != string(models.StudyTaskStatusFailed) {
		return fmt.Errorf("completion status must be COMPLETED or FAILED")
	}

	// Empty string preserves existing payload; non-empty string overwrites it.
	payloadVal := strings.TrimSpace(result.Payload)
	res, err := tx.Exec(`
		UPDATE study_queue
		SET status = ?, completed_at = CURRENT_TIMESTAMP,
		    payload_json = CASE WHEN ? = '' THEN payload_json ELSE ? END
		WHERE id = ? AND status IN ('ACTIVE', 'RESERVED')
	`, status, payloadVal, payloadVal, taskID)
	if err != nil {
		utils.Warnf("[QUEUE] CompleteTaskTx reading task completion update error taskID=%s err=%v", taskID, err)
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		utils.Warnf("[QUEUE] CompleteTaskTx reading task completion rows affected error taskID=%s err=%v", taskID, err)
		return err
	}
	if affected == 0 {
		var exists int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM study_queue WHERE id = ?`, taskID).Scan(&exists); err != nil {
			utils.Warnf("[QUEUE] CompleteTaskTx reading task completion existence check error taskID=%s err=%v", taskID, err)
			return err
		}
		if exists == 0 {
			utils.Warnf("[QUEUE] CompleteTaskTx reading task completion task not found taskID=%s", taskID)
			return ErrTaskNotFound
		}
		utils.Warnf("[QUEUE] CompleteTaskTx reading task completion task not active taskID=%s", taskID)
		return ErrTaskNotActive
	}
	var taskType string
	if err := tx.QueryRow(`SELECT COALESCE(task_type, '') FROM study_queue WHERE id = ?`, taskID).Scan(&taskType); err != nil {
		utils.Warnf("[QUEUE] CompleteTaskTx task_type lookup error taskID=%s err=%v", taskID, err)
		return err
	}
	utils.LogQueueTransition(taskID, taskType, "ACTIVE", status, "task_completed")

	for _, followUp := range result.FollowUps {
		followUp.ID = strings.TrimSpace(followUp.ID)
		followUp.NotebookID = strings.TrimSpace(followUp.NotebookID)
		followUp.TopicID = strings.TrimSpace(followUp.TopicID)
		followUp.PayloadJSON = strings.TrimSpace(followUp.PayloadJSON)
		if followUp.ID == "" {
			return fmt.Errorf("follow-up task id is required")
		}
		if followUp.NotebookID == "" {
			return fmt.Errorf("follow-up notebook id is required")
		}
		if strings.TrimSpace(string(followUp.TaskType)) == "" {
			return fmt.Errorf("follow-up task type is required")
		}
		if strings.TrimSpace(string(followUp.Status)) == "" {
			followUp.Status = models.StudyTaskStatusPending
		}

		if _, err := tx.Exec(`
			INSERT INTO study_queue (
				id, notebook_id, topic_id, task_type, status, priority, payload_json, start_page, end_page
			) VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), ?, ?)
		`, followUp.ID, followUp.NotebookID, followUp.TopicID, string(followUp.TaskType), string(followUp.Status), followUp.Priority, followUp.PayloadJSON, followUp.StartPage, followUp.EndPage); err != nil {
			utils.Warnf("[QUEUE] CompleteTaskTx follow-up insertion error taskID=%s followUpID=%s err=%v", taskID, followUp.ID, err)
			return err
		}
		utils.Warnf("[FLASHCARD_PIPELINE] queue_insertion source=completion_followup parentTaskID=%s followUpID=%s taskType=%s notebookID=%s topicID=%s", taskID, followUp.ID, followUp.TaskType, followUp.NotebookID, followUp.TopicID)
		utils.LogQueueTaskCreated(followUp.ID, string(followUp.TaskType), followUp.NotebookID, followUp.TopicID)
	}

	return nil
}

// CompleteTask marks ACTIVE task as terminal and inserts explicit follow-up tasks transactionally.
func (r *Repository) CompleteTask(taskID string, result models.CompletionResult) error {
	utils.Warnf("[QUEUE] CompleteTask transaction start taskID=%s", strings.TrimSpace(taskID))
	err := r.withTx(func(tx *sql.Tx) error {
		if err := r.CompleteTaskTx(tx, taskID, result); err != nil {
			utils.Warnf("[QUEUE] CompleteTask transaction error taskID=%s err=%v", strings.TrimSpace(taskID), err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	utils.Warnf("[QUEUE] CompleteTask tx commit success taskID=%s", strings.TrimSpace(taskID))
	return nil
}

// PersistReadingProgress persists page progress without validating completion.
// Used in trust-based completion model where user decides when reading is complete.
func (r *Repository) PersistReadingProgress(taskID string, finalPage int) (bool, error) {
	task, err := r.GetReadingTask(taskID)
	if err != nil {
		return false, err
	}
	reachedEnd := finalPage >= task.EndPage
	if finalPage < task.StartPage {
		finalPage = task.StartPage
	}
	if finalPage > task.EndPage {
		finalPage = task.EndPage
	}

	err = r.withTx(func(tx *sql.Tx) error {
		_, err = tx.Exec(`
			INSERT INTO reading_progress (task_id, current_page, last_accessed_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(task_id) DO UPDATE
			SET current_page = excluded.current_page,
			    last_accessed_at = CURRENT_TIMESTAMP
		`, task.TaskID, finalPage)
		if err != nil {
			return err
		}

		// Synchronize topics.current_page_cursor to keep both cursor systems aligned
		if task.TopicID != "" {
			_, err = tx.Exec(`
				UPDATE topics
				SET current_page_cursor = ?,
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = ? AND current_page_cursor < ?
			`, finalPage, task.TopicID, finalPage)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return false, err
	}
	return reachedEnd, nil
}

// CompleteReadingWithGeneratedQuiz completes an ACTIVE reader-compatible task and inserts a QUIZ follow-up with payload.
func (r *Repository) CompleteReadingWithGeneratedQuiz(taskID string, quizPayload models.QuizTaskPayload) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", fmt.Errorf("task id is required")
	}
	if len(quizPayload.Questions) == 0 {
		return "", fmt.Errorf("quiz payload must include questions")
	}
	if quizPayload.PassingScore <= 0 {
		quizPayload.PassingScore = 70
	}
	payloadBytes, err := json.Marshal(quizPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal quiz payload: %w", err)
	}

	type completionSeed struct {
		ID         string
		NotebookID string
		TopicID    string
		StartPage  int
		EndPage    int
	}
	seed := completionSeed{}
	var currentPage int
	var status string

	err = r.db.QueryRow(`
		SELECT
			sq.id,
			sq.notebook_id,
			COALESCE(sq.topic_id, ''),
			COALESCE(sq.start_page, 0),
			COALESCE(sq.end_page, 0),
			sq.status,
			COALESCE(rp.current_page, COALESCE(sq.start_page, 0))
		FROM study_queue sq
		LEFT JOIN reading_progress rp ON rp.task_id = sq.id
		WHERE sq.id = ? AND sq.task_type IN ('READING', 'REREAD')
	`, taskID).Scan(
		&seed.ID,
		&seed.NotebookID,
		&seed.TopicID,
		&seed.StartPage,
		&seed.EndPage,
		&status,
		&currentPage,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrTaskNotFound
	}
	if err != nil {
		return "", err
	}
	if status != string(models.StudyTaskStatusActive) && status != string(models.StudyTaskStatusReserved) {
		return "", ErrTaskNotActive
	}

	quizTaskID := uuid.NewString()
	err = r.withTx(func(tx *sql.Tx) error {
		// Synchronize topics.current_page_cursor to keep both cursor systems aligned.
		// Completion is authoritative for the assigned reading window, so cursor must
		// advance to at least end_page to prevent scheduler rematerializing the same window.
		if seed.TopicID != "" {
			cursorAfterCompletion := currentPage
			if seed.EndPage > cursorAfterCompletion {
				cursorAfterCompletion = seed.EndPage
			}
			_, err = tx.Exec(`
				UPDATE topics
				SET current_page_cursor = ?,
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = ? AND current_page_cursor < ?
			`, cursorAfterCompletion, seed.TopicID, cursorAfterCompletion)
			if err != nil {
				return fmt.Errorf("failed to synchronize topic cursor: %w", err)
			}
		}

		err = r.CompleteTaskTx(tx, seed.ID, models.CompletionResult{
			Status: models.StudyTaskStatusCompleted,
			FollowUps: []models.StudyQueueTask{
				{
					ID:          quizTaskID,
					NotebookID:  seed.NotebookID,
					TopicID:     seed.TopicID,
					TaskType:    models.StudyTaskTypeQuiz,
					Status:      models.StudyTaskStatusPending,
					Priority:    0,
					PayloadJSON: string(payloadBytes),
					StartPage:   seed.StartPage,
					EndPage:     seed.EndPage,
				},
			},
		})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	return quizTaskID, nil
}

// SaveQuizAttemptTx saves one quiz attempt record transactionally.
func (r *Repository) SaveQuizAttemptTx(tx *sql.Tx, attempt models.QuizAttemptRecord) error {
	attempt.ID = strings.TrimSpace(attempt.ID)
	attempt.TaskID = strings.TrimSpace(attempt.TaskID)
	attempt.AnswersJSON = strings.TrimSpace(attempt.AnswersJSON)
	if attempt.ID == "" {
		return fmt.Errorf("attempt id is required")
	}
	if attempt.TaskID == "" {
		return fmt.Errorf("task id is required")
	}
	if attempt.AnswersJSON == "" {
		return fmt.Errorf("answers json is required")
	}
	if attempt.CompletedAt <= 0 {
		return fmt.Errorf("completed at is required")
	}
	if tx == nil {
		return fmt.Errorf("nil tx passed to SaveQuizAttemptTx")
	}
	_, err := tx.Exec(`
		INSERT INTO quiz_attempts (id, task_id, score, passed, answers_json, feedback, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, attempt.ID, attempt.TaskID, attempt.Score, boolToInt(attempt.Passed), attempt.AnswersJSON, attempt.Feedback, attempt.CompletedAt)
	return err
}

// InsertMilestoneExamTask inserts a MILESTONE_EXAM task into the queue.
func (r *Repository) InsertMilestoneExamTask(notebookID string, payload models.MilestoneExamPayload) error {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return fmt.Errorf("notebook ID is required")
	}
	if len(payload.Quizzes) == 0 {
		return fmt.Errorf("milestone exam payload must include quizzes")
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal milestone exam payload: %w", err)
	}

	task := models.StudyQueueTask{
		ID:          uuid.NewString(),
		NotebookID:  notebookID,
		TaskType:    models.StudyTaskTypeMilestoneExam,
		Status:      models.StudyTaskStatusPending,
		Priority:    0,
		PayloadJSON: string(payloadJSON),
	}
	return r.InsertStudyTask(task)
}

// InsertMilestoneExamTaskIfMissing checks representativeAttemptID and inserts a milestone exam task atomically if missing.
func (r *Repository) InsertMilestoneExamTaskIfMissing(notebookID, representativeAttemptID string, payload models.MilestoneExamPayload) (bool, error) {
	notebookID = strings.TrimSpace(notebookID)
	representativeAttemptID = strings.TrimSpace(representativeAttemptID)
	if notebookID == "" {
		return false, fmt.Errorf("notebook ID is required")
	}
	if representativeAttemptID == "" {
		return false, fmt.Errorf("attempt ID is required")
	}
	if len(payload.Quizzes) == 0 {
		return false, fmt.Errorf("milestone exam payload must include quizzes")
	}

	var inserted bool
	err := r.withTx(func(tx *sql.Tx) error {
		rows, err := tx.Query(`
			SELECT COALESCE(payload_json, '')
			FROM study_queue
			WHERE notebook_id = ?
			  AND task_type = 'MILESTONE_EXAM'
		`, notebookID)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var payloadJSON string
			if err := rows.Scan(&payloadJSON); err != nil {
				return err
			}
			if payloadJSON == "" {
				continue
			}
			var existingPayload models.MilestoneExamPayload
			if err := json.Unmarshal([]byte(payloadJSON), &existingPayload); err != nil {
				return err
			}
			if _, ok := existingPayload.Quizzes[representativeAttemptID]; ok {
				return nil
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		_ = rows.Close()

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal milestone exam payload: %w", err)
		}

		task := models.StudyQueueTask{
			ID:          uuid.NewString(),
			NotebookID:  notebookID,
			TaskType:    models.StudyTaskTypeMilestoneExam,
			Status:      models.StudyTaskStatusPending,
			Priority:    0,
			PayloadJSON: string(payloadJSON),
		}
		if err := r.InsertStudyTaskTx(tx, task); err != nil {
			return err
		}
		inserted = true
		return nil
	})
	return inserted, err
}

// EnsurePendingFlashcardGenerateTask inserts a new FLASHCARD_GENERATE task if none exists in PENDING or ACTIVE status for the topic.
func (r *Repository) EnsurePendingFlashcardGenerateTask(notebookID, topicID string, startPage, endPage int, title string) error {
	notebookID = strings.TrimSpace(notebookID)
	topicID = strings.TrimSpace(topicID)
	if notebookID == "" {
		return fmt.Errorf("notebook ID is required")
	}

	return r.withTx(func(tx *sql.Tx) error {
		var count int
		var err error
		if topicID == "" {
			err = tx.QueryRow(`
				SELECT COUNT(*) FROM study_queue
				WHERE task_type = 'FLASHCARD_GENERATE' AND (topic_id IS NULL OR topic_id = '') AND status IN ('PENDING', 'ACTIVE')
			`).Scan(&count)
		} else {
			err = tx.QueryRow(`
				SELECT COUNT(*) FROM study_queue
				WHERE task_type = 'FLASHCARD_GENERATE' AND topic_id = ? AND status IN ('PENDING', 'ACTIVE')
			`, topicID).Scan(&count)
		}
		if err != nil {
			return err
		}
		if count > 0 {
			return nil // Already exists
		}

		task := models.StudyQueueTask{
			ID:         uuid.NewString(),
			NotebookID: notebookID,
			TopicID:    topicID,
			TaskType:   models.StudyTaskTypeFlashcardGenerate,
			Status:     models.StudyTaskStatusPending,
			Priority:   0,
			Title:      title,
			StartPage:  startPage,
			EndPage:    endPage,
		}
		return r.InsertStudyTaskTx(tx, task)
	})
}

// ResolveFlashcardGenerateTasksForTopic marks all pending/active FLASHCARD_GENERATE tasks for a topic as COMPLETED.
func (r *Repository) ResolveFlashcardGenerateTasksForTopic(topicID string) error {
	topicID = strings.TrimSpace(topicID)
	return r.withTx(func(tx *sql.Tx) error {
		var rows *sql.Rows
		var err error
		if topicID == "" {
			rows, err = tx.Query(`
				SELECT id, status FROM study_queue
				WHERE task_type = 'FLASHCARD_GENERATE' AND (topic_id IS NULL OR topic_id = '') AND status IN ('PENDING', 'ACTIVE')
			`)
		} else {
			rows, err = tx.Query(`
				SELECT id, status FROM study_queue
				WHERE task_type = 'FLASHCARD_GENERATE' AND topic_id = ? AND status IN ('PENDING', 'ACTIVE')
			`, topicID)
		}
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()

		type taskInfo struct {
			id     string
			status string
		}
		var tasks []taskInfo
		for rows.Next() {
			var t taskInfo
			if err := rows.Scan(&t.id, &t.status); err != nil {
				return err
			}
			tasks = append(tasks, t)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		for _, t := range tasks {
			if t.status == string(models.StudyTaskStatusPending) {
				if err := r.ActivateTaskTx(tx, t.id); err != nil {
					return err
				}
			}

			if err := r.CompleteTaskTx(tx, t.id, models.CompletionResult{
				Status: models.StudyTaskStatusCompleted,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// EnsurePendingReadingTaskForNotebook ensures at least one PENDING/ACTIVE READING task exists in study_queue for an active notebook.
func (r *Repository) EnsurePendingReadingTaskForNotebook(notebookID string, targetSessionWords int) error {
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return fmt.Errorf("notebook id is required")
	}

	if targetSessionWords <= 0 {
		return fmt.Errorf("targetSessionWords must be greater than 0, got %d", targetSessionWords)
	}

	return r.withTx(func(tx *sql.Tx) error {
		var count int
		err := tx.QueryRow(`
			SELECT COUNT(*) FROM study_queue
			WHERE notebook_id = ? AND status IN ('PENDING', 'ACTIVE')
		`, notebookID).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil // Task already exists in queue
		}

		var topicID, topicTitle, notebookTitle string
		var topicStartPage, topicEndPage int
		err = tx.QueryRow(`
			SELECT
				t.id,
				COALESCE(t.title, ''),
				COALESCE(n.title, ''),
				COALESCE(NULLIF(t.start_page, 0), 1),
				COALESCE(NULLIF(t.end_page, 0), 1)
			FROM topics t
			JOIN notebook_topics nt ON t.id = nt.topic_id
			JOIN notebooks n ON n.id = nt.notebook_id
			WHERE nt.notebook_id = ? AND COALESCE(t.status, 'unseen') != 'completed'
			ORDER BY t.start_page ASC, t.id ASC
			LIMIT 1
		`, notebookID).Scan(&topicID, &topicTitle, &notebookTitle, &topicStartPage, &topicEndPage)
		if errors.Is(err, sql.ErrNoRows) {
			return nil // No uncompleted topics found
		}
		if err != nil {
			return err
		}

		// Calculate soft-capped page window based on targetSessionWords
		topicCursor := models.ReadingTopicCursor{
			ID:         topicID,
			Title:      topicTitle,
			StartPage:  topicStartPage,
			EndPage:    topicEndPage,
			NotebookID: notebookID,
		}

		var queryErr error
		startPage, endPage, ok, wordMap := ResolvePageWindow(topicCursor, targetSessionWords, func(tID string, sp int, ep int) (map[int]int, error) {
			query := `
				SELECT page_num, COALESCE(token_count, 0), COALESCE(chunk_text, '')
				FROM chunks
				WHERE topic_id = ? AND page_num BETWEEN ? AND ?
				ORDER BY page_num
			`
			rows, qErr := tx.Query(query, tID, sp, ep)
			if qErr != nil {
				queryErr = qErr
				return nil, qErr
			}
			defer func() { _ = rows.Close() }()

			wMap := make(map[int]int)
			for rows.Next() {
				var pNum, tCount int
				var cText string
				if sErr := rows.Scan(&pNum, &tCount, &cText); sErr == nil {
					if tCount <= 0 {
						tCount = utf8.RuneCountInString(cText) / 4
					}
					wMap[pNum] += tCount
				}
			}
			if rErr := rows.Err(); rErr != nil {
				queryErr = rErr
				return nil, rErr
			}
			return wMap, nil
		})
		if queryErr != nil {
			return queryErr
		}

		if !ok {
			startPage = topicStartPage
			endPage = topicEndPage
		} else {
			// Calculate current total word count for startPage..endPage using wordMap
			var currentWords int
			for p := startPage; p <= endPage; p++ {
				pWords := wordMap[p]
				if pWords <= 0 {
					pWords = FallbackWordsPerPage
				}
				currentWords += pWords
			}

			// Semantic extension: check up to +3 additional pages
			const maxExtensionPages = 3
			const minSimilarityThreshold = 0.85
			maxTotalWords := targetSessionWords + int(float64(targetSessionWords)*0.3)

			baseEndPage := endPage
			stopReason := "max_pages"
			for ext := 1; ext <= maxExtensionPages; ext++ {
				nextPage := endPage + 1
				if nextPage > topicEndPage {
					stopReason = "topic_end"
					break
				}

				// Get word count for nextPage from wordMap
				nextPageWords := wordMap[nextPage]
				if nextPageWords <= 0 {
					nextPageWords = FallbackWordsPerPage
				}

				if currentWords+nextPageWords > maxTotalWords {
					stopReason = "word_limit"
					break
				}

				sim, simErr := r.GetPageBoundaryCosineSimilarityTx(tx, topicID, endPage, nextPage)
				if simErr != nil {
					stopReason = "query_err"
					break
				}
				if sim < minSimilarityThreshold {
					stopReason = fmt.Sprintf("similarity_low(%.2f)", sim)
					break
				}

				// Absorbed! Extend endPage
				endPage = nextPage
				currentWords += nextPageWords
			}

			if endPage > baseEndPage {
				utils.Warnf("[SEMANTIC_EXTENSION] absorbed=%d topicID=%q range=%d-%d words=%d reason=%s", endPage-baseEndPage, topicID, startPage, endPage, currentWords, stopReason)
			} else {
				utils.Warnf("[SEMANTIC_EXTENSION] absorbed=0 topicID=%q endPage=%d words=%d reason=%s", topicID, baseEndPage, currentWords, stopReason)
			}
		}

		taskID := fmt.Sprintf("task-read-%s-%s", notebookID, topicID)
		var idExists int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM study_queue WHERE id = ?`, taskID).Scan(&idExists); err == nil && idExists > 0 {
			taskID = fmt.Sprintf("task-read-%s-%s-%d", notebookID, topicID, time.Now().UnixNano())
		}
		task := models.StudyQueueTask{
			ID:         taskID,
			NotebookID: notebookID,
			TopicID:    topicID,
			TaskType:   models.StudyTaskTypeReading,
			Status:     models.StudyTaskStatusPending,
			Priority:   1,
			StartPage:  startPage,
			EndPage:    endPage,
		}
		assignTaskTitle(&task, topicTitle, notebookTitle)
		return r.InsertStudyTaskTx(tx, task)
	})
}

// EnsurePendingReadingTasksForActiveNotebooks ensures all active notebooks for a profile have at least one PENDING/ACTIVE task.
func (r *Repository) EnsurePendingReadingTasksForActiveNotebooks(activeProfileID string) error {
	settings, err := r.GetUserSettings()
	if err != nil {
		return fmt.Errorf("failed to load user settings for reading tasks: %w", err)
	}
	if settings == nil || settings.TargetSessionWords <= 0 {
		return fmt.Errorf("invalid target_session_words in user settings")
	}
	targetWords := settings.TargetSessionWords

	rows, err := r.db.Query(`
		SELECT n.id FROM notebooks n
		WHERE n.study_status = 'active'
		  AND n.status = 'chunked'
		  AND (? = '' OR n.profile_id = ? OR n.profile_id IS NULL OR n.profile_id = '')
		  AND NOT EXISTS (
			SELECT 1 FROM study_queue sq
			WHERE sq.notebook_id = n.id
			  AND sq.status IN ('PENDING', 'ACTIVE')
		  )
	`, activeProfileID, activeProfileID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var notebookIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		notebookIDs = append(notebookIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, nID := range notebookIDs {
		if err := r.EnsurePendingReadingTaskForNotebook(nID, targetWords); err != nil {
			utils.Warnf("[QUEUE] failed to ensure reading task for active notebook %s: %v", nID, err)
		}
	}
	return nil
}

// ReserveTask updates a task status from ACTIVE to RESERVED.
func (r *Repository) ReserveTask(taskID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRow(`SELECT COALESCE(status, '') FROM study_queue WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}
	if status != string(models.StudyTaskStatusActive) {
		return fmt.Errorf("task %s cannot be reserved because it is in status %s (expected ACTIVE)", taskID, status)
	}

	_, err = tx.Exec(`UPDATE study_queue SET status = ? WHERE id = ?`, models.StudyTaskStatusReserved, taskID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RevertTaskReservation reverts a task status from RESERVED to ACTIVE.
func (r *Repository) RevertTaskReservation(taskID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRow(`SELECT COALESCE(status, '') FROM study_queue WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}
	if status != string(models.StudyTaskStatusReserved) {
		return fmt.Errorf("task %s is not reserved (status: %s)", taskID, status)
	}

	_, err = tx.Exec(`UPDATE study_queue SET status = ? WHERE id = ?`, models.StudyTaskStatusActive, taskID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
