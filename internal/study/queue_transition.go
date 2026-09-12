package study

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

// TransitionEvent represents a deterministic queue transition trigger.
type TransitionEvent string

const (
	EventCompleteReading          TransitionEvent = "COMPLETE_READING"
	EventSubmitQuiz               TransitionEvent = "SUBMIT_QUIZ"
	EventCompleteFlashcards       TransitionEvent = "COMPLETE_FLASHCARDS"
	EventCompleteFlashcardReview  TransitionEvent = "COMPLETE_FLASHCARD_REVIEW"
	EventCompleteSocraticRescue   TransitionEvent = "COMPLETE_SOCRATIC_RESCUE"
	EventCompleteMilestoneExam    TransitionEvent = "COMPLETE_MILESTONE_EXAM"
	EventFailTask                 TransitionEvent = "FAIL_TASK"
)

// TransitionRequest encapsulates parameters for a queue task transition.
type TransitionRequest struct {
	TaskID      string
	Event       TransitionEvent
	TopicID     string
	NotebookID  string
	QuizPayload *models.QuizTaskPayload
	QuizAnswers []models.QuizAnswer
	CardCount   int
	ErrorReason string
}


// TransitionResult provides the outcome and any spawned follow-up task IDs.
type TransitionResult struct {
	Success        bool                  `json:"success"`
	TaskID         string                `json:"task_id"`
	NextTaskID     string                `json:"next_task_id,omitempty"`
	NextTaskType   string                `json:"next_task_type,omitempty"`
	QuizResult     *models.QuizResult    `json:"quiz_result,omitempty"`
	CardsScheduled int                   `json:"cards_scheduled,omitempty"`
	Rewards        *models.RewardPayload `json:"rewards,omitempty"`
	Message        string                `json:"message,omitempty"`
}

// TransitionTask serves as the unified switchboard for all study queue task state transitions.
func (s *StudyService) TransitionTask(ctx context.Context, req TransitionRequest) (TransitionResult, error) {
	req.TaskID = strings.TrimSpace(req.TaskID)
	if req.TaskID == "" {
		return TransitionResult{}, fmt.Errorf("task ID is required")
	}

	utils.Infof("[QUEUE_TRANSITION] executing transition taskID=%s event=%s", req.TaskID, req.Event)

	switch req.Event {
	case EventCompleteReading:
		if req.QuizPayload == nil || len(req.QuizPayload.Questions) == 0 {
			return TransitionResult{}, fmt.Errorf("quiz payload is required to complete reading")
		}
		quizTaskID, err := s.repo.CompleteReadingWithGeneratedQuiz(req.TaskID, *req.QuizPayload)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to complete reading transition: %w", err)
		}
		if err := s.repo.IncrementStat("reading_sessions", 1); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to increment reading_sessions stat: %w", err)
		}
		rewards := s.awardCompletionRewards(req.TaskID, models.StudyTaskTypeReading, 0, false)
		return TransitionResult{
			Success:      true,
			TaskID:       req.TaskID,
			NextTaskID:   quizTaskID,
			NextTaskType: string(models.StudyTaskTypeQuiz),
			Rewards:      rewards,
		}, nil

	case EventSubmitQuiz:
		if len(req.QuizAnswers) == 0 {
			return TransitionResult{}, fmt.Errorf("quiz answers are required to submit quiz")
		}
		res, err := s.SubmitQuizAttempt(req.TaskID, req.QuizAnswers)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to process quiz submission transition: %w", err)
		}
		var rewards *models.RewardPayload
		if res.Passed {
			if err := s.repo.IncrementStat("quizzes_passed", 1); err != nil {
				return TransitionResult{}, fmt.Errorf("failed to increment quizzes_passed stat: %w", err)
			}
			rewards = s.awardCompletionRewards(req.TaskID, models.StudyTaskTypeQuiz, res.Score, res.Score >= 100)
		}
		return TransitionResult{
			Success:    true,
			TaskID:     req.TaskID,
			QuizResult: &res,
			Rewards:    rewards,
		}, nil

	case EventCompleteFlashcards:
		if err := s.repo.ResolveFlashcardGenerateTasksForTopic(req.TopicID); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to complete flashcards task: %w", err)
		}

		// ponytail: seed next reading task for active notebook inside unified transition
		settings, err := s.repo.GetUserSettings()
		targetWords := 3000
		minWords := 0
		if err == nil && settings != nil && settings.TargetSessionWords > 0 {
			targetWords = settings.TargetSessionWords
			minWords = settings.MinSessionWords
		}
		if req.NotebookID != "" {
			if ensureErr := s.repo.EnsurePendingReadingTaskForNotebook(req.NotebookID, targetWords, minWords); ensureErr != nil {
				utils.Warnf("[QUEUE_TRANSITION] failed to seed next reading task for notebook %s: %v", req.NotebookID, ensureErr)
			}
		}

		if err := s.repo.IncrementStat("flashcards_reviewed", req.CardCount); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to increment flashcards_reviewed stat: %w", err)
		}
		rewards := s.awardCompletionRewards(req.TaskID, models.StudyTaskTypeFlashcardGenerate, 0, false)
		return TransitionResult{
			Success:        true,
			TaskID:         req.TaskID,
			CardsScheduled: req.CardCount,
			Rewards:        rewards,
		}, nil

	case EventCompleteFlashcardReview:
		if err := s.repo.CompleteReviewSession(req.TaskID); err != nil {
			return TransitionResult{}, err
		}
		if err := s.repo.IncrementStat("flashcards_reviewed", 10); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to increment flashcards_reviewed stat: %w", err)
		}
		rewards := s.awardCompletionRewards(req.TaskID, models.StudyTaskTypeFlashcardReview, 0, false)
		return TransitionResult{
			Success: true,
			TaskID:  req.TaskID,
			Rewards: rewards,
		}, nil

	case EventCompleteMilestoneExam:
		task, err := s.repo.GetTaskByID(req.TaskID)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to load milestone exam task: %w", err)
		}
		if task.TaskType != models.StudyTaskTypeMilestoneExam {
			return TransitionResult{}, fmt.Errorf("task %s is not a MILESTONE_EXAM task", req.TaskID)
		}
		if err := s.repo.CompleteTask(req.TaskID, models.CompletionResult{
			Status: models.StudyTaskStatusCompleted,
		}); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to complete milestone exam: %w", err)
		}

		// Seed the next reading task so completing a milestone does not leave
		// the queue dependent on whichever notebook was already seeded.
		settings, settingsErr := s.repo.GetUserSettings()
		targetWords := 3000
		minWords := 0
		if settingsErr == nil && settings != nil && settings.TargetSessionWords > 0 {
			targetWords = settings.TargetSessionWords
			minWords = settings.MinSessionWords
		}
		if task.NotebookID != "" {
			if ensureErr := s.repo.EnsurePendingReadingTaskForNotebook(task.NotebookID, targetWords, minWords); ensureErr != nil {
				utils.Warnf("[QUEUE_TRANSITION] failed to seed next reading task for notebook %s: %v", task.NotebookID, ensureErr)
			}
		}

		// Persist milestone_exam_complete telemetry to durable SQLite outbox
		fileHash := ""
		if nb, nbErr := s.repo.GetNotebookByID(task.NotebookID); nbErr == nil && nb != nil {
			fileHash = nb.FileHash
		}
		metaBytes, _ := json.Marshal(map[string]interface{}{
			"task_id": req.TaskID,
			"score":   100,
			"passed":  true,
		})
		if trackErr := s.repo.TrackAnalyticsEvent("milestone_exam_complete", fileHash, task.StartPage, string(metaBytes)); trackErr != nil {
			utils.Warnf("[QUEUE_TRANSITION] failed to record milestone analytics event: %v", trackErr)
		}

		rewards := s.awardCompletionRewards(req.TaskID, models.StudyTaskTypeMilestoneExam, 100, true)
		return TransitionResult{
			Success: true,
			TaskID:  req.TaskID,
			Rewards: rewards,
		}, nil

	case EventCompleteSocraticRescue:
		task, err := s.repo.GetTaskByID(req.TaskID)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to load socratic task: %w", err)
		}
		if task.TaskType != models.StudyTaskTypeSocraticRemedial {
			return TransitionResult{}, fmt.Errorf("task %s is not SOCRATIC_REMEDIAL", req.TaskID)
		}
		if task.Status != models.StudyTaskStatusActive {
			return TransitionResult{}, fmt.Errorf("task %s is not ACTIVE (status=%s)", req.TaskID, task.Status)
		}

		chunks, err := s.repo.GetChunksForTopicPageRange(task.TopicID, task.StartPage, task.EndPage)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to load chunks for socratic task: %w", err)
		}

		chunkIDs := make([]string, 0, len(chunks))
		chunkTextByID := make(map[string]string, len(chunks))
		for _, chunk := range chunks {
			chunkIDs = append(chunkIDs, chunk.ID)
			chunkTextByID[chunk.ID] = chunk.Text
		}

		generatedQuiz, err := s.GenerateQuizSync(task.TopicID, chunkIDs, chunkTextByID)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to generate quiz: %w", err)
		}

		tx, err := s.repo.Begin()
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		quizTaskID := uuid.NewString()
		quizPayload, err := json.Marshal(map[string]interface{}{
			"source":        "socratic_rescue_requiz",
			"topic_id":      task.TopicID,
			"questions":     generatedQuiz.Questions,
			"passing_score": generatedQuiz.PassingScore,
		})
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to marshal quiz payload: %w", err)
		}

		followUps := []models.StudyQueueTask{
			{
				ID:          quizTaskID,
				NotebookID:  task.NotebookID,
				TopicID:     task.TopicID,
				TaskType:    models.StudyTaskTypeQuiz,
				Status:      models.StudyTaskStatusPending,
				Priority:    0,
				PayloadJSON: string(quizPayload),
				StartPage:   task.StartPage,
				EndPage:     task.EndPage,
			},
		}

		if err := s.repo.CompleteTaskTx(tx, req.TaskID, models.CompletionResult{
			Status:    models.StudyTaskStatusCompleted,
			FollowUps: followUps,
		}); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to complete socratic task: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to commit socratic rescue completion: %w", err)
		}

		rewards := s.awardCompletionRewards(req.TaskID, models.StudyTaskTypeSocraticRemedial, 100, true)
		utils.Warnf("[SOCRATIC_RESCUE] rescue_completed taskID=%s topicID=%s requizTaskID=%s", req.TaskID, task.TopicID, quizTaskID)
		return TransitionResult{
			Success:      true,
			TaskID:       req.TaskID,
			NextTaskID:   quizTaskID,
			NextTaskType: string(models.StudyTaskTypeQuiz),
			Rewards:      rewards,
		}, nil

	case EventFailTask:
		if err := s.repo.CompleteTask(req.TaskID, models.CompletionResult{
			Status: models.StudyTaskStatusFailed,
		}); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to mark task as failed: %w", err)
		}
		return TransitionResult{
			Success: true,
			TaskID:  req.TaskID,
			Message: req.ErrorReason,
		}, nil

	default:
		return TransitionResult{}, fmt.Errorf("unrecognized transition event: %s", req.Event)
	}
}

// awardCompletionRewards calculates deterministic XP/coins and persists any mystery chest to SQLite atomically.
func (s *StudyService) awardCompletionRewards(taskID string, taskType models.StudyTaskType, score int, isAce bool) *models.RewardPayload {
	if s.repo == nil {
		return nil
	}

	xp, coins, box := RollTaskRewards(taskType, score, isAce)
	if box != nil {
		box.TaskID = taskID
	}

	tx, err := s.repo.Begin()
	if err != nil {
		utils.Warnf("[GAMIFICATION] failed to begin reward transaction for task %s: %v", taskID, err)
		return nil
	}
	defer func() { _ = tx.Rollback() }()

	newTitle, err := s.repo.AwardTaskRewardsTx(tx, xp, coins, box)
	if err != nil {
		utils.Warnf("[GAMIFICATION] failed to award completion rewards for task %s: %v", taskID, err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		utils.Warnf("[GAMIFICATION] failed to commit reward transaction for task %s: %v", taskID, err)
		return nil
	}

	return &models.RewardPayload{
		XPEarned:         xp,
		CoinsEarned:      coins,
		LootBox:          box,
		NewTitleUnlocked: newTitle,
	}
}
