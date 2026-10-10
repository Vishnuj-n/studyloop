package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	studypkg "ai-tutor/internal/study"
	"ai-tutor/internal/utils"
)

func (a *App) activateReadingSessionTask(taskID string) map[string]interface{} {
	repo := a.getRepo()
	qTask, qErr := repo.GetTaskByID(taskID)

	if qErr != nil {
		utils.QueueLogger.Warn("queue task pre-activate loading anomaly", "taskID", taskID, "err", qErr)
		return map[string]interface{}{"error": "failed to load task: " + qErr.Error()}
	}

	switch qTask.Status {
	case models.StudyTaskStatusPending:
		if err := repo.ActivateTask(taskID); err != nil {
			utils.QueueLogger.Warn("queue task activation failed", "taskID", taskID, "err", err)
			return map[string]interface{}{"error": "failed to activate task: " + err.Error()}
		}
		return nil
	case models.StudyTaskStatusActive:
		utils.QueueLogger.Debug("idempotent resume: task already active", "taskID", taskID, "status", qTask.Status, "type", qTask.TaskType, "notebookID", qTask.NotebookID, "topicID", qTask.TopicID)
		return nil
	default:
		utils.QueueLogger.Warn("task terminal", "status", qTask.Status, "taskID", taskID)
		return map[string]interface{}{"error": "task is in terminal status: " + string(qTask.Status), "code": 409}
	}
}

// buildReadingSessionEnvelope constructs the standard reader session response envelope.
func buildReadingSessionEnvelope(task models.ReadingTask, bundle *models.ReaderTopicBundle, currentPage int) map[string]interface{} {
	pageCount := 0
	if bundle != nil {
		pageCount = bundle.PageCount
	}
	return map[string]interface{}{
		"ok":     true,
		"task":   task,
		"bundle": bundle,
		"page_bounds": map[string]interface{}{
			"start_page":   task.StartPage,
			"end_page":     task.EndPage,
			"current_page": currentPage,
			"page_count":   pageCount,
		},
		"navigation": map[string]interface{}{
			"can_go_prev": currentPage > task.StartPage,
			"can_go_next": currentPage < task.EndPage,
		},
	}
}

// InitializeReadingSession activates and loads an active or pending reading task from the queue.
// Note: notebookID, topicID, startPage, and endPage parameters are accepted at the Wails transport boundary
// for forward compatibility / frontend callers, but the canonical truth is read from the loaded task record.
func (a *App) InitializeReadingSession(taskID, notebookID, topicID string, startPage, endPage int) map[string]interface{} {
	_ = notebookID
	_ = topicID
	_ = startPage
	_ = endPage

	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}
	utils.Debugf("[READER_INIT] InitializeReadingSession entry taskID=%s", taskID)

	if errMap := a.activateReadingSessionTask(taskID); errMap != nil {
		return errMap
	}

	// Load reading task with all context
	task, err := repo.GetReadingTask(taskID)
	if err != nil {
		if errors.Is(err, db.ErrTaskNotFound) {
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		}
		return map[string]interface{}{"error": err.Error()}
	}

	// Trigger sequential background processing on topic opening:
	// Compression runs first so study note generation consumes pre-compressed chunks (~20% smaller prompt).
	if (task.TopicID != "" || task.NotebookID != "") && a.studyService != nil {
		noteGenCallback := func() {
			if settings, sErr := repo.GetUserSettings(); sErr == nil && settings != nil && settings.AutoGenerateStudyNotes {
				existingNote, checkErr := a.studyService.GetTopicStudyNoteForRange(task.TopicID, task.StartPage, task.EndPage)
				if checkErr == nil && (existingNote == nil || strings.TrimSpace(existingNote.Content) == "") {
					a.studyService.GenerateTopicStudyNoteAsync(context.Background(), task.TopicID, task.NotebookID, task.StartPage, task.EndPage)
				}
			}
		}

		if task.TopicID != "" {
			a.studyService.CompressTopicChunksAsync(context.Background(), task.TopicID, noteGenCallback)
		} else {
			go noteGenCallback()
		}
	}

	currentPage := task.CurrentPage
	if currentPage <= 0 {
		currentPage = task.StartPage
	}

	// Get topic bundle for additional metadata
	bundle, err := repo.GetReaderTopicBundle(task.TopicID, task.NotebookID)
	if err != nil {
		utils.Warnf("[READER_INIT] bundle fetch failed for topic %s notebook %s: %v", task.TopicID, task.NotebookID, err)
		return buildReadingSessionEnvelope(task, nil, currentPage)
	}

	utils.Debugf("[READER_INIT] InitializeReadingSession response payload canonicalTaskID=%s", task.TaskID)
	return buildReadingSessionEnvelope(task, bundle, currentPage)
}

// fetchSessionChunks tries topic+page-range -> topic -> notebook+page-range fallback ladder.
func (a *App) fetchSessionChunks(repo *db.Repository, task models.ReadingTask) ([]models.Chunk, error) {
	var chunks []models.Chunk
	var err error

	// Bounded page range for the specific topic
	if task.TopicID != "" && task.StartPage > 0 && task.EndPage >= task.StartPage {
		chunks, err = repo.GetChunksForTopicPageRange(task.TopicID, task.StartPage, task.EndPage)
		if err != nil {
			return nil, err
		}
	}

	// Fallback to all topic chunks if page-bounded query found nothing
	if len(chunks) == 0 && task.TopicID != "" {
		chunks, err = repo.GetChunksForTopic(task.TopicID)
		if err != nil {
			return nil, err
		}
	}

	// Fallback to whole notebook page range if topic chunks are not yet indexed
	if len(chunks) == 0 && task.NotebookID != "" && task.StartPage > 0 && task.EndPage >= task.StartPage {
		chunks, err = repo.GetChunksForNotebookPageRange(task.NotebookID, task.StartPage, task.EndPage)
		if err != nil {
			return nil, err
		}
	}

	return chunks, nil
}

func (a *App) CompleteReading(taskID string, splitPage int) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}
	utils.Infof("[COMPLETE_SESSION] CompleteReading entry taskID=%s splitPage=%d", taskID, splitPage)

	// Hard fail here because completing reading requires generating and transitioning into a follow-up quiz
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}

	// 1. Verify task exists and is currently ACTIVE before any state mutation
	queueTask, qErr := repo.GetTaskByID(taskID)
	if qErr != nil {
		if errors.Is(qErr, db.ErrTaskNotFound) {
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		}
		return map[string]interface{}{"error": qErr.Error()}
	}
	if queueTask.Status != models.StudyTaskStatusActive {
		return map[string]interface{}{"error": "task is not active", "code": 409}
	}

	task, err := repo.GetReadingTask(taskID)
	if err != nil {
		if errors.Is(err, db.ErrTaskNotFound) {
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		}
		return map[string]interface{}{"error": err.Error()}
	}

	// 2. Handle split session ("Complete Here") safely now that task is confirmed ACTIVE
	if splitPage > 0 && splitPage >= task.StartPage && splitPage < task.EndPage {
		task.EndPage = splitPage
		if updateErr := repo.UpdateTaskEndPage(taskID, task.EndPage); updateErr != nil {
			utils.Warnf("[COMPLETE_SESSION] UpdateTaskEndPage failed: %v", updateErr)
			if errors.Is(updateErr, db.ErrTaskNotActive) {
				return map[string]interface{}{"error": "task is not active", "code": 409}
			}
			return map[string]interface{}{"error": updateErr.Error()}
		}
	}

	if task.TopicID == "" && task.NotebookID == "" {
		return map[string]interface{}{"error": "task has no topic or notebook", "code": 422}
	}

	// 3. Load user settings once for compression mode and word ceilings
	targetWords := defaultTargetSessionWords
	compressionMode := ""
	if settings, sErr := repo.GetUserSettings(); sErr == nil && settings != nil {
		compressionMode = settings.PromptCompressionMode
		if settings.TargetSessionWords > 0 {
			targetWords = settings.TargetSessionWords
		}
	}

	// 4. Retrieve candidate chunks using fallback ladder
	chunks, err := a.fetchSessionChunks(repo, task)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if len(chunks) == 0 {
		return map[string]interface{}{
			"error": "notebook content not yet indexed — please re-confirm your syllabus from the notebook page",
			"code":  422,
		}
	}

	// 5. Cap total chunk payload to TargetSessionWords ceiling
	maxWords := int(float64(targetWords) * 1.3)
	totalWords := 0
	cappedChunks := make([]models.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		cWords := len(strings.Fields(chunk.Text))
		if len(cappedChunks) > 0 && totalWords+cWords > maxWords {
			break
		}
		totalWords += cWords
		cappedChunks = append(cappedChunks, chunk)
	}
	if len(cappedChunks) > 0 {
		chunks = cappedChunks
	}

	chunkIDs := make([]string, 0, len(chunks))
	chunkTextByID := make(map[string]string, len(chunks))
	for _, chunk := range chunks {
		chunkIDs = append(chunkIDs, chunk.ID)
		chunkTextByID[chunk.ID] = studypkg.SelectChunkText(chunk, compressionMode)
	}

	// 6. Reserve task for two-phase completion
	if reserveErr := repo.ReserveTask(taskID); reserveErr != nil {
		return map[string]interface{}{"error": "failed to reserve task: " + reserveErr.Error(), "code": 409}
	}

	revertBase := func() map[string]interface{} {
		resp := map[string]interface{}{}
		if revErr := repo.RevertTaskReservation(taskID); revErr != nil {
			utils.QueueLogger.Error("failed to revert task reservation", "taskID", taskID, "err", revErr)
			resp["reservation_reverted"] = false
			resp["revert_error"] = revErr.Error()
		} else {
			resp["reservation_reverted"] = true
		}
		return resp
	}

	fail := func(msg string) map[string]interface{} {
		resp := revertBase()
		resp["error"] = msg
		return resp
	}

	failWithCode := func(msg string, code int) map[string]interface{} {
		resp := revertBase()
		resp["error"] = msg
		resp["code"] = code
		return resp
	}

	quizPayload, err := a.studyService.GenerateQuizSync(task.TopicID, chunkIDs, chunkTextByID)
	if err != nil {
		return fail(err.Error())
	}

	if len(quizPayload.Questions) == 0 {
		return failWithCode("quiz generation returned no questions; please retry from the reader", 422)
	}

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 90*time.Second)
	defer cancel()

	transitionRes, err := a.studyService.TransitionTask(ctx, studypkg.TransitionRequest{
		TaskID:      taskID,
		Event:       studypkg.EventCompleteReading,
		TopicID:     task.TopicID,
		NotebookID:  task.NotebookID,
		QuizPayload: &quizPayload,
	})
	if err != nil {
		return fail(err.Error())
	}


	resp := map[string]interface{}{
		"ok":           true,
		"quiz_task_id": transitionRes.NextTaskID,
		"rewards":      transitionRes.Rewards,
	}

	// Trigger note generation if auto notes enabled and not already created
	if (task.TopicID != "" || task.NotebookID != "") && a.studyService != nil {
		if settings, sErr := repo.GetUserSettings(); sErr == nil && settings != nil && settings.AutoGenerateStudyNotes {
			existingNote, checkErr := a.studyService.GetTopicStudyNoteForRange(task.TopicID, task.StartPage, task.EndPage)
			if checkErr == nil && (existingNote == nil || strings.TrimSpace(existingNote.Content) == "") {
				a.studyService.GenerateTopicStudyNoteAsync(context.Background(), task.TopicID, task.NotebookID, task.StartPage, task.EndPage)
			}
		}
	}

	// 7. Auto-seed and return next continuous reading task if available
	if task.NotebookID != "" {
		if seedErr := repo.ForceSeedPendingReadingTaskForNotebook(task.NotebookID, targetWords); seedErr != nil {
			utils.Warnf("[COMPLETE_SESSION] ForceSeedPendingReadingTaskForNotebook err: %v", seedErr)
		} else if nextTask, err := repo.GetPendingReadingTaskForNotebook(task.NotebookID); err == nil {
			resp["next_reading_task"] = map[string]interface{}{
				"id":          nextTask.ID,
				"notebook_id": nextTask.NotebookID,
				"topic_id":    nextTask.TopicID,
				"start_page":  nextTask.StartPage,
				"end_page":    nextTask.EndPage,
			}
			utils.Infof("[COMPLETE_SESSION] Next continuous reading task seeded: id=%s topicID=%s pages=%d-%d", nextTask.ID, nextTask.TopicID, nextTask.StartPage, nextTask.EndPage)
		}
	}

	return resp
}

// GetReadingTaskHistory returns historical reading tasks with pagination for developer diagnostics.
func (a *App) GetReadingTaskHistory(notebookID string, limit, offset int) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}

	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	records, totalCount, err := repo.GetReadingTaskHistory(notebookID, limit, offset)
	if err != nil {
		utils.QueueLogger.Error("failed to fetch reading task history", "err", err)
		return map[string]interface{}{"error": err.Error()}
	}

	if records == nil {
		records = []models.ReadingTaskHistoryRecord{}
	}

	hasMore := (offset + len(records)) < totalCount

	return map[string]interface{}{
		"ok":          true,
		"records":     records,
		"total_count": totalCount,
		"has_more":    hasMore,
		"limit":       limit,
		"offset":      offset,
	}
}

// RevertReadingTaskSession rolls back a completed reading session back to ACTIVE (used in developer diagnostics / testing).
func (a *App) RevertReadingTaskSession(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}

	if err := repo.RevertReadingTaskSession(taskID); err != nil {
		utils.QueueLogger.Error("failed to revert reading task session", "taskID", taskID, "err", err)
		return map[string]interface{}{"error": err.Error(), "code": 500}
	}

	utils.Infof("[REVERT_SESSION] Successfully reverted reading task session: %s", taskID)
	return map[string]interface{}{
		"ok":      true,
		"task_id": taskID,
		"message": "Session successfully reverted to active state.",
	}
}

// RevertTopicToReadingFromQuiz is the user-facing "Restart Reading Session" action available on the
// quiz result screen after a final rescue-quiz failure. Given the quiz task ID it:
//  1. Looks up the quiz task to obtain topic_id and start_page.
//  2. Finds the most recently completed READING / REREAD task for that topic range.
//  3. Atomically reverts that reading task back to ACTIVE (resetting the topic cursor, purging the
//     failed quiz / tutor / flashcard tasks, and deleting any quiz attempts for this range).
//  4. Returns the reverted reading task ID so the frontend can navigate directly to /reader.
func (a *App) RevertTopicToReadingFromQuiz(quizTaskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}

	quizTaskID = strings.TrimSpace(quizTaskID)
	if quizTaskID == "" {
		return map[string]interface{}{"error": "quiz task ID is required", "code": 400}
	}

	// 1. Load the quiz task to get topic_id and start_page.
	qTask, err := repo.GetTaskByID(quizTaskID)
	if err != nil {
		utils.Warnf("[REVERT_TO_READING] failed to load quiz task %s: %v", quizTaskID, err)
		return map[string]interface{}{"error": "failed to load quiz task: " + err.Error(), "code": 404}
	}
	if qTask.TopicID == "" {
		return map[string]interface{}{"error": "quiz task has no associated topic; cannot revert", "code": 422}
	}
	if qTask.StartPage <= 0 {
		return map[string]interface{}{"error": "quiz task has no valid start page; cannot revert", "code": 422}
	}

	// 2. Find the most recently completed reading task for this topic + page range.
	readingTaskID, findErr := repo.FindCompletedReadingTaskForTopic(qTask.TopicID, qTask.StartPage)
	if findErr != nil {
		utils.Warnf("[REVERT_TO_READING] no completed reading task found for topicID=%s startPage=%d: %v", qTask.TopicID, qTask.StartPage, findErr)
		return map[string]interface{}{"error": "no completed reading session found for this topic range", "code": 404}
	}

	// 3. Revert the reading task atomically.
	if revertErr := repo.RevertReadingTaskSession(readingTaskID); revertErr != nil {
		utils.QueueLogger.Error("RevertTopicToReadingFromQuiz revert failed", "readingTaskID", readingTaskID, "err", revertErr)
		return map[string]interface{}{"error": revertErr.Error(), "code": 500}
	}

	utils.Infof("[REVERT_TO_READING] Reverted reading task %s for quizTaskID=%s topicID=%s startPage=%d",
		readingTaskID, quizTaskID, qTask.TopicID, qTask.StartPage)

	return map[string]interface{}{
		"ok":               true,
		"reading_task_id":  readingTaskID,
		"topic_id":         qTask.TopicID,
		"start_page":       qTask.StartPage,
		"message":          "Reading session restarted. You can now re-read from the beginning of this topic.",
	}
}

