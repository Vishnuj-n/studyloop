package app

import (
	"context"
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
		utils.QueueLogger.Info("queue task pre-activate loading anomaly", "taskID", taskID, "err", qErr)
		return map[string]interface{}{"error": "failed to load task: " + qErr.Error()}
	}

	switch qTask.Status {
	case models.StudyTaskStatusPending:
		if err := repo.ActivateTask(taskID); err != nil {
			utils.QueueLogger.Info("queue task activation failed", "taskID", taskID, "err", err)
			return map[string]interface{}{"error": "failed to activate task: " + err.Error()}
		}
		utils.QueueLogger.Info("queue task activated", "taskID", taskID)
		return nil
	case models.StudyTaskStatusActive:
		utils.QueueLogger.Debug("idempotent resume: task already active", "taskID", taskID, "status", qTask.Status, "type", qTask.TaskType, "notebookID", qTask.NotebookID, "topicID", qTask.TopicID)
		return nil
	default:
		utils.QueueLogger.Info("task terminal", "status", qTask.Status, "taskID", taskID)
		return map[string]interface{}{"error": "task is in terminal status: " + string(qTask.Status), "code": 409}
	}
}

// InitializeReadingSession activates and loads an active or pending reading task from the queue.
func (a *App) InitializeReadingSession(taskID, notebookID, topicID string, startPage, endPage int) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}
	utils.Infof("[READER_INIT] InitializeReadingSession entry taskID=%s", taskID)

	if errMap := a.activateReadingSessionTask(taskID); errMap != nil {
		return errMap
	}

	// Load reading task with all context
	task, err := repo.GetReadingTask(taskID)
	if err != nil {
		if err == db.ErrTaskNotFound {
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		}
		return map[string]interface{}{"error": err.Error()}
	}

	currentPage := task.CurrentPage
	if currentPage <= 0 {
		currentPage = task.StartPage
	}

	// Get topic bundle for additional metadata
	bundle, err := repo.GetReaderTopicBundle(task.TopicID, task.NotebookID)
	if err != nil {
		utils.Warnf("[READER_INIT] bundle fetch failed for topic %s notebook %s: %v", task.TopicID, task.NotebookID, err)
		return map[string]interface{}{
			"ok":     true,
			"task":   task,
			"bundle": nil,
			"page_bounds": map[string]interface{}{
				"start_page":   task.StartPage,
				"end_page":     task.EndPage,
				"current_page": currentPage,
				"page_count":   0,
			},
			"navigation": map[string]interface{}{
				"can_go_prev": currentPage > task.StartPage,
				"can_go_next": currentPage < task.EndPage,
			},
		}
	}

	utils.Infof("[READER_INIT] InitializeReadingSession response payload canonicalTaskID=%s", task.TaskID)

	return map[string]interface{}{
		"ok":     true,
		"task":   task,
		"bundle": bundle,
		"page_bounds": map[string]interface{}{
			"start_page":   task.StartPage,
			"end_page":     task.EndPage,
			"current_page": currentPage,
			"page_count":   bundle.PageCount,
		},
		"navigation": map[string]interface{}{
			"can_go_prev": currentPage > task.StartPage,
			"can_go_next": currentPage < task.EndPage,
		},
	}
}

func (a *App) CompleteReading(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}
	utils.Infof("[COMPLETE_SESSION] CompleteReading entry taskID=%s", taskID)

	task, err := repo.GetReadingTask(taskID)
	if err != nil {
		if err == db.ErrTaskNotFound {
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		}
		return map[string]interface{}{"error": err.Error()}
	}

	queueTask, qErr := repo.GetTaskByID(taskID)
	if qErr != nil {
		return map[string]interface{}{"error": qErr.Error()}
	}
	if queueTask.Status != models.StudyTaskStatusActive {
		return map[string]interface{}{"error": "task is not active", "code": 409}
	}

	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}

	if task.TopicID == "" && task.NotebookID == "" {
		return map[string]interface{}{"error": "task has no topic or notebook", "code": 422}
	}

	// Generate quiz from topic chunks bounded by reading task page range when available.
	var chunks []models.Chunk
	if task.StartPage > 0 && task.EndPage >= task.StartPage {
		chunks, err = repo.GetChunksForTopicPageRange(task.TopicID, task.StartPage, task.EndPage)
	} else {
		chunks, err = repo.GetChunksForTopic(task.TopicID)
	}
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if len(chunks) == 0 && (task.StartPage > 0 && task.EndPage >= task.StartPage) {
		chunks, err = repo.GetChunksForTopic(task.TopicID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
	}
	if len(chunks) == 0 && task.NotebookID != "" && task.StartPage > 0 && task.EndPage >= task.StartPage {
		chunks, err = repo.GetChunksForNotebookPageRange(task.NotebookID, task.StartPage, task.EndPage)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
	}

	if len(chunks) == 0 {
		return map[string]interface{}{
			"error": "notebook content not yet indexed — please re-confirm your syllabus from the notebook page",
			"code":  422,
		}
	}

	// Cap total chunk payload to TargetSessionWords user setting ceiling
	if settings, sErr := repo.GetUserSettings(); sErr == nil && settings != nil && settings.TargetSessionWords > 0 {
		maxWords := int(float64(settings.TargetSessionWords) * 1.3)
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
	}

	chunkIDs := make([]string, 0, len(chunks))
	chunkTextByID := make(map[string]string, len(chunks))
	for _, chunk := range chunks {
		chunkIDs = append(chunkIDs, chunk.ID)
		chunkTextByID[chunk.ID] = chunk.Text
	}

	if reserveErr := repo.ReserveTask(taskID); reserveErr != nil {
		return map[string]interface{}{"error": "failed to reserve task: " + reserveErr.Error(), "code": 409}
	}

	quizPayload, err := a.studyService.GenerateQuizSync(task.TopicID, chunkIDs, chunkTextByID)
	if err != nil {
		_ = repo.RevertTaskReservation(taskID)
		return map[string]interface{}{"error": err.Error()}
	}

	if len(quizPayload.Questions) == 0 {
		_ = repo.RevertTaskReservation(taskID)
		return map[string]interface{}{
			"error": "quiz generation returned no questions; please retry from the reader",
			"code":  422,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	transitionRes, err := a.studyService.TransitionTask(ctx, studypkg.TransitionRequest{
		TaskID:      taskID,
		Event:       studypkg.EventCompleteReading,
		TopicID:     task.TopicID,
		NotebookID:  task.NotebookID,
		QuizPayload: &quizPayload,
	})
	if err != nil {
		_ = repo.RevertTaskReservation(taskID)
		return map[string]interface{}{"error": err.Error()}
	}

	resp := map[string]interface{}{
		"ok":           true,
		"quiz_task_id": transitionRes.NextTaskID,
		"rewards":      transitionRes.Rewards,
	}

	// Auto-seed and return the next continuous reading task for this notebook if available
	if task.NotebookID != "" {
		targetWords := 600
		if settings, sErr := repo.GetUserSettings(); sErr == nil && settings != nil && settings.TargetSessionWords > 0 {
			targetWords = settings.TargetSessionWords
		}
		if seedErr := repo.EnsurePendingReadingTaskForNotebook(task.NotebookID, targetWords); seedErr != nil {
			utils.Warnf("[COMPLETE_SESSION] EnsurePendingReadingTaskForNotebook err: %v", seedErr)
		} else if nextTask, err := repo.GetPendingReadingTaskForNotebook(task.NotebookID); err == nil && nextTask.ID != "" {
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

