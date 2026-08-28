package study

import (
	"context"
	"fmt"
	"strings"

	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

// TransitionEvent represents a deterministic queue transition trigger.
type TransitionEvent string

const (
	EventCompleteReading    TransitionEvent = "COMPLETE_READING"
	EventSubmitQuiz         TransitionEvent = "SUBMIT_QUIZ"
	EventCompleteFlashcards TransitionEvent = "COMPLETE_FLASHCARDS"
	EventFailTask           TransitionEvent = "FAIL_TASK"
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
	Success        bool               `json:"success"`
	TaskID         string             `json:"task_id"`
	NextTaskID     string             `json:"next_task_id,omitempty"`
	NextTaskType   string             `json:"next_task_type,omitempty"`
	QuizResult     *models.QuizResult `json:"quiz_result,omitempty"`
	CardsScheduled int                `json:"cards_scheduled,omitempty"`
	Message        string             `json:"message,omitempty"`
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
		return TransitionResult{
			Success:      true,
			TaskID:       req.TaskID,
			NextTaskID:   quizTaskID,
			NextTaskType: string(models.StudyTaskTypeQuiz),
		}, nil

	case EventSubmitQuiz:
		if len(req.QuizAnswers) == 0 {
			return TransitionResult{}, fmt.Errorf("quiz answers are required to submit quiz")
		}
		res, err := s.SubmitQuizAttempt(req.TaskID, req.QuizAnswers)
		if err != nil {
			return TransitionResult{}, fmt.Errorf("failed to process quiz submission transition: %w", err)
		}
		return TransitionResult{
			Success:    true,
			TaskID:     req.TaskID,
			QuizResult: &res,
		}, nil

	case EventCompleteFlashcards:
		if err := s.repo.ResolveFlashcardGenerateTasksForTopic(req.TopicID); err != nil {
			return TransitionResult{}, fmt.Errorf("failed to complete flashcards task: %w", err)
		}
		return TransitionResult{
			Success:        true,
			TaskID:         req.TaskID,
			CardsScheduled: req.CardCount,
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
