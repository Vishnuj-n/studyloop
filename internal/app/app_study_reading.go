package app

import (
	"strings"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

func (a *App) activateReadingSessionTask(taskID string) map[string]interface{} {
	repo := a.getRepo()
	qTask, qErr := repo.GetTaskByID(taskID)

	if qErr != nil {
		utils.Errorf("InitializeReadingSession loading anomaly: taskID=%s err=%v", taskID, qErr)
		utils.QueueLogger.Info("queue task pre-activate loading anomaly", "taskID", taskID)
		return map[string]interface{}{"error": "failed to load task: " + qErr.Error()}
	}

	switch qTask.Status {
	case models.StudyTaskStatusPending:
		if err := repo.ActivateTask(taskID); err != nil {
			utils.Errorf("InitializeReadingSession activation failed: taskID=%s err=%v", taskID, err)
			utils.QueueLogger.Info("queue task activation failed", "taskID", taskID)
			return map[string]interface{}{"error": "failed to activate task: " + err.Error()}
		} else {
			utils.QueueLogger.Info("queue task activated", "taskID", taskID)
		}
	case models.StudyTaskStatusActive:
		utils.QueueLogger.Debug("idempotent resume: task already active", "taskID", taskID, "status", qTask.Status, "type", qTask.TaskType, "notebookID", qTask.NotebookID, "topicID", qTask.TopicID)
	default:
		utils.QueueLogger.Info("task terminal", "status", qTask.Status, "taskID", taskID)
		return map[string]interface{}{"error": "task is in terminal status: " + string(qTask.Status), "code": 409}
	}
	return nil
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
	utils.Warnf("[READER_INIT] InitializeReadingSession entry taskID=%s", taskID)

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

	// Get topic bundle for additional metadata
	bundle, err := repo.GetReaderTopicBundle(task.TopicID, task.NotebookID)
	if err != nil {
		// Return task-only response if bundle fails
		return map[string]interface{}{
			"ok":   true,
			"task": task,
			"page_bounds": map[string]interface{}{
				"start_page":   task.StartPage,
				"end_page":     task.EndPage,
				"current_page": task.StartPage,
				"page_count":   0,
			},
			"navigation": map[string]interface{}{
				"can_go_prev": task.StartPage > 1,
				"can_go_next": true,
			},
		}
	}

	// Use repository-provided and clamped CurrentPage from GetReadingTask
	currentPage := task.CurrentPage
	utils.Warnf("[READER_INIT] InitializeReadingSession response payload canonicalTaskID=%s", task.TaskID)

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
		utils.Warnf("[COMPLETE_SESSION] CompleteReading entry rejected: taskID empty")
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}
	utils.Warnf("[COMPLETE_SESSION] CompleteReading entry taskID=%s", taskID)

	// Trust-based completion: just validate task exists and is active
	task, err := repo.GetReadingTask(taskID)
	if err != nil {
		switch err {
		case db.ErrTaskNotFound:
			utils.Warnf("[COMPLETE_SESSION] CompleteReading GetReadingTask error: task not found taskID=%s", taskID)
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		default:
			utils.Warnf("[COMPLETE_SESSION] CompleteReading GetReadingTask error taskID=%s err=%v", taskID, err)
			return map[string]interface{}{"error": err.Error()}
		}
	}
	utils.Warnf("[COMPLETE_SESSION] CompleteReading loaded reading task taskID=%s startPage=%d endPage=%d currentPage=%d", taskID, task.StartPage, task.EndPage, task.CurrentPage)

	queueTask, qErr := repo.GetTaskByID(taskID)
	if qErr != nil {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading GetTaskByID error taskID=%s err=%v", taskID, qErr)
		return map[string]interface{}{"error": qErr.Error()}
	}
	if queueTask.Status != models.StudyTaskStatusActive {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading task not active taskID=%s status=%s", taskID, queueTask.Status)
		return map[string]interface{}{"error": "task is not active", "code": 409}
	}

	if a.studyService == nil {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading error: study service not initialized taskID=%s", taskID)
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}

	// Generate quiz from topic chunks bounded by reading task page range when available.
	utils.Warnf("[COMPLETE_SESSION] CompleteReading chunk lookup topicID=%q startPage=%d endPage=%d", task.TopicID, task.StartPage, task.EndPage)
	var chunks []models.Chunk
	if task.StartPage > 0 && task.EndPage >= task.StartPage {
		chunks, err = repo.GetChunksForTopicPageRange(task.TopicID, task.StartPage, task.EndPage)
	} else {
		chunks, err = repo.GetChunksForTopic(task.TopicID)
	}
	if err != nil {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading chunk lookup error taskID=%s err=%v", taskID, err)
		return map[string]interface{}{"error": err.Error()}
	}
	if len(chunks) == 0 && (task.StartPage > 0 && task.EndPage >= task.StartPage) {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading 0 chunks in page range [%d-%d], falling back to topic chunks topicID=%q", task.StartPage, task.EndPage, task.TopicID)
		chunks, err = repo.GetChunksForTopic(task.TopicID)
		if err != nil {
			utils.Warnf("[COMPLETE_SESSION] CompleteReading fallback chunk lookup error taskID=%s err=%v", taskID, err)
			return map[string]interface{}{"error": err.Error()}
		}
	}
	utils.Warnf("[COMPLETE_SESSION] CompleteReading chunk lookup result: got %d chunks for topicID=%q", len(chunks), task.TopicID)

	if len(chunks) == 0 {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading no chunks found for topicID=%q — notebook content not indexed", task.TopicID)
		return map[string]interface{}{
			"error": "notebook content not yet indexed — please re-confirm your syllabus from the notebook page",
			"code":  422,
		}
	}

	chunkIDs := make([]string, 0, len(chunks))
	chunkTextByID := make(map[string]string, len(chunks))
	for i, chunk := range chunks {
		utils.Warnf("[COMPLETE_SESSION] CompleteReading chunk[%d] id=%q topicID=%q textLen=%d", i, chunk.ID, chunk.TopicID, len(chunk.Text))
		chunkIDs = append(chunkIDs, chunk.ID)
		chunkTextByID[chunk.ID] = chunk.Text
	}

	utils.Warnf("[QUIZ] CompleteReading before GenerateQuizSync taskID=%s topicID=%q chunkCount=%d chunkIDs=%v", taskID, task.TopicID, len(chunkIDs), chunkIDs)
	if reserveErr := repo.ReserveTask(taskID); reserveErr != nil {
		return map[string]interface{}{"error": "failed to reserve task: " + reserveErr.Error()}
	}

	quizPayload, err := a.studyService.GenerateQuizSync(task.TopicID, chunkIDs, chunkTextByID)
	if err != nil {
		utils.Warnf("[QUIZ] CompleteReading GenerateQuizSync error taskID=%s err=%v", taskID, err)
		if revertErr := repo.RevertTaskReservation(taskID); revertErr != nil {
			utils.Warnf("[QUIZ] failed to revert task reservation for task %s: %v", taskID, revertErr)
		}
		return map[string]interface{}{"error": err.Error()}
	}
	utils.Warnf("[QUIZ] CompleteReading after GenerateQuizSync taskID=%s questionCount=%d", taskID, len(quizPayload.Questions))

	// Complete reading task and generate follow-up quiz
	// No page completion validation required - user decides when done
	utils.Warnf("[COMPLETE_SESSION] CompleteReading before CompleteReadingWithGeneratedQuiz taskID=%s", taskID)
	quizTaskID, err := repo.CompleteReadingWithGeneratedQuiz(taskID, quizPayload)
	if err != nil {
		if revertErr := repo.RevertTaskReservation(taskID); revertErr != nil {
			utils.Warnf("[QUIZ] failed to revert task reservation on finalization failure for task %s: %v", taskID, revertErr)
		}
		switch err {
		case db.ErrTaskNotFound:
			utils.Warnf("[COMPLETE_SESSION] CompleteReading CompleteReadingWithGeneratedQuiz error: task not found taskID=%s", taskID)
			return map[string]interface{}{"error": "ErrNotFound", "code": 404}
		case db.ErrTaskNotActive:
			utils.Warnf("[COMPLETE_SESSION] CompleteReading CompleteReadingWithGeneratedQuiz error: task not active taskID=%s", taskID)
			return map[string]interface{}{"error": "ErrTaskNotActive", "code": 409}
		default:
			utils.Warnf("[COMPLETE_SESSION] CompleteReading CompleteReadingWithGeneratedQuiz error taskID=%s err=%v", taskID, err)
			return map[string]interface{}{"error": err.Error()}
		}
	}
	utils.Warnf("[COMPLETE_SESSION] CompleteReading CompleteReadingWithGeneratedQuiz result taskID=%s quizTaskID=%s", taskID, quizTaskID)
	utils.Warnf("[FLASHCARD_PIPELINE] flashcard_generation_trigger check stage=reading_completed taskID=%s topicID=%s result=not_triggered reason=no_flashcard_hook_in_complete_reading", taskID, task.TopicID)
	return map[string]interface{}{"ok": true, "quiz_task_id": quizTaskID}
}
