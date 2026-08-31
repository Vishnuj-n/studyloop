package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	studypkg "ai-tutor/internal/study"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

// CompleteMilestoneExam completes an active MILESTONE_EXAM task via Unified Transition Router.
// ponytail: simplest way to complete milestone task with no flashcard generation.
func (a *App) CompleteMilestoneExam(taskID string) map[string]interface{} {
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	_, err := a.studyService.TransitionTask(context.Background(), studypkg.TransitionRequest{
		TaskID: taskID,
		Event:  studypkg.EventCompleteMilestoneExam,
	})
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true}
}

func (a *App) GetTask(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": "task ID is required", "code": 400}
	}
	task, err := repo.GetTaskByID(taskID)
	if err != nil {
		return mapTaskError(err)
	}

	// Dynamic compile for MILESTONE_EXAM questions
	if task.TaskType == models.StudyTaskTypeMilestoneExam {
		if quizPayload, err := studypkg.CompileMilestonePayload(repo, &task); err == nil && len(quizPayload.Questions) > 0 {
			if quizPayloadJSON, mErr := json.Marshal(quizPayload); mErr == nil {
				task.PayloadJSON = string(quizPayloadJSON)
			}
		}
	}

	return map[string]interface{}{"task": task}
}

func (a *App) GetTaskContext(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return map[string]interface{}{"error": errTaskIDRequired, "code": 400}
	}
	task, err := repo.GetTaskByID(taskID)
	if err != nil {
		return mapTaskError(err)
	}
	externalPrompt := ""
	if task.TaskType == models.StudyTaskTypeSocraticRemedial {
		var promptErr error
		externalPrompt, promptErr = buildSocraticRemedialPrompt(repo, task)
		if promptErr != nil {
			return map[string]interface{}{"error": promptErr.Error()}
		}
	}

	return map[string]interface{}{
		"task": task,
		"topic": map[string]interface{}{
			"id": task.TopicID,
		},
		"notebook": map[string]interface{}{
			"id": task.NotebookID,
		},
		"external_prompt": externalPrompt,
	}
}

func buildSocraticRemedialPrompt(repo *db.Repository, task models.StudyQueueTask) (string, error) {
	bundle, err := repo.GetReaderTopicBundle(task.TopicID, task.NotebookID)
	if err != nil {
		return "", fmt.Errorf("failed to get reader topic bundle for task %s: %w", task.ID, err)
	}

	var sectionsContent []string
	for _, s := range bundle.Sections {
		if s.Content != "" {
			sectionsContent = append(sectionsContent, s.Content)
		}
	}
	sourceText := strings.Join(sectionsContent, "\n\n")

	bookContext := ""
	notebookTitle := bundle.NotebookTitle
	if notebookTitle != "" {
		bookContext = fmt.Sprintf("Book: %s\n", notebookTitle)
	}

	materialName := "my material"
	if notebookTitle != "" {
		materialName = notebookTitle
	}

	tutorStyle := "socratic"
	if userSettings, err := repo.GetUserSettings(); err == nil && userSettings != nil && userSettings.TutorStyle != "" {
		tutorStyle = userSettings.TutorStyle
	}

	var directive string
	switch tutorStyle {
	case "direct":
		directive = "Please act as a direct, concise AI tutor. Explain key concepts directly and clearly, point out why the failed questions occurred, and help me quickly understand without unnecessary fluff. Keep explanations focused and direct."
	case "detailed":
		directive = "Please act as a comprehensive step-by-step AI tutor. Provide detailed conceptual walkthroughs, real-world analogies, and illustrative examples to thoroughly explain the core concepts and clear up misunderstandings."
	default: // "socratic"
		directive = "Please act as a Socratic tutor — don't give me summaries or answers directly. Instead, ask me leading questions that guide me to discover the key concepts myself. Start with the most fundamental question."
	}

	promptText := fmt.Sprintf("I'm studying the following text from %s for preparation. I've encountered difficulty understanding it. %s\n\n", materialName, directive)

	promptText = appendFailedQuestionsSection(promptText, task)

	return promptText + bookContext + "---\n" + sourceText + "\n---", nil
}

func appendFailedQuestionsSection(promptText string, task models.StudyQueueTask) string {
	var payload struct {
		FailedQuestions []models.FailedQuestionDetail `json:"failed_questions"`
	}
	if task.PayloadJSON != "" {
		if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
			utils.Warnf("failed to unmarshal failed questions for task %s: %v", task.ID, err)
		}
	}

	if len(payload.FailedQuestions) > 0 {
		promptText += "During my quiz, I failed the following questions:\n"
		for idx, q := range payload.FailedQuestions {
			promptText += fmt.Sprintf("%d. Question: %s\n", idx+1, q.Prompt)
			if len(q.Options) > 0 {
				promptText += fmt.Sprintf("   Options: %s\n", strings.Join(q.Options, ", "))
			}
			userAns := q.UserAnswer
			if userAns == "" {
				userAns = "(No answer)"
			}
			promptText += fmt.Sprintf("   My Answer: %s\n", userAns)
			promptText += fmt.Sprintf("   Correct Answer: %s\n\n", q.CorrectAnswer)
		}
		promptText += "Please focus on guiding me through the concepts behind these failed questions.\n\n"
	}
	return promptText
}

func (a *App) GenerateQuizForPageRange(notebookID string, startPage, endPage int) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	return a.studyService.GenerateQuizForPageRange(notebookID, startPage, endPage)
}

func (a *App) SubmitQuizAttempt(taskID string, answers []models.QuizAnswer) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	result, err := a.studyService.SubmitQuizAttempt(taskID, answers)
	if err != nil {
		return mapTaskError(err)
	}
	return map[string]interface{}{"result": result}
}

// GenerateFlashcardsForQuizTask generates flashcards based on a passed quiz task.
// Newly generated cards are future-dated and do not create an immediate review task.
func (a *App) GenerateFlashcardsForQuizTask(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}

	task, err := repo.GetTaskByID(taskID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if task.TaskType != models.StudyTaskTypeQuiz {
		return map[string]interface{}{"error": "task is not a quiz"}
	}

	utils.Warnf("[FLASHCARD_PIPELINE] continue_button_flashcard_generation_started taskID=%s topicID=%s notebookID=%s", taskID, task.TopicID, task.NotebookID)

	cardCount, err := a.studyService.GenerateFlashcardsAfterQuiz(task.NotebookID, task.TopicID, task.StartPage, task.EndPage)
	if err != nil {
		utils.Warnf("[FLASHCARD_PIPELINE] flashcard_generation_failed taskID=%s reason=%v", taskID, err)
		if ensureErr := repo.EnsurePendingFlashcardGenerateTask(task.NotebookID, task.TopicID, task.StartPage, task.EndPage, task.Title); ensureErr != nil {
			utils.Warnf("[FLASHCARD_PIPELINE] failed to insert FLASHCARD_GENERATE retry task: %v", ensureErr)
			return map[string]interface{}{"error": fmt.Sprintf("failed to generate flashcards: %v; failed to insert retry task: %v", err, ensureErr)}
		}
		return map[string]interface{}{"error": "failed to generate flashcards: " + err.Error()}
	}

	transitionRes, transitionErr := a.studyService.TransitionTask(context.Background(), studypkg.TransitionRequest{
		TaskID:     taskID,
		Event:      studypkg.EventCompleteFlashcards,
		TopicID:    task.TopicID,
		NotebookID: task.NotebookID,
		CardCount:  cardCount,
	})
	if transitionErr != nil {
		utils.Warnf("[FLASHCARD_PIPELINE] failed to execute flashcards transition: %v", transitionErr)
		return map[string]interface{}{"error": "failed to complete transition: " + transitionErr.Error()}
	}

	checkAndInsertMilestoneExam(repo, task.NotebookID)

	utils.Warnf("[FLASHCARD_PIPELINE] flashcard_generation_completed taskID=%s reviewTaskID=%s cardsScheduled=%d", taskID, "", transitionRes.CardsScheduled)
	utils.Warnf("[DASHBOARD] dashboard_redirect_after_generation taskID=%s reviewTaskID=%s cardsScheduled=%d", taskID, "", transitionRes.CardsScheduled)

	return map[string]interface{}{
		"review_task_id":    "",
		"cards_scheduled":   transitionRes.CardsScheduled,
		"flashcards_gen_ok": true,
	}
}

func checkAndInsertMilestoneExam(repo *db.Repository, notebookID string) {
	count, countErr := repo.CountCompletedQuizzesByNotebook(notebookID)
	if countErr != nil {
		utils.Warnf("[MILESTONE_EXAM] quiz_count_failed notebookID=%s err=%v", notebookID, countErr)
		return
	}
	if count <= 0 || count%10 != 0 {
		return
	}

	decadeAttempts, attemptsErr := repo.GetLastNQuizAttemptsWithCorrectness(notebookID, 10)
	if attemptsErr != nil {
		utils.Warnf("[MILESTONE_EXAM] passed_quizzes_fetch_failed notebookID=%s err=%v", notebookID, attemptsErr)
		return
	}
	if len(decadeAttempts) != 10 {
		return
	}

	var representativeAttemptID string
	quizzes := make(map[string][]int, len(decadeAttempts))
	passingScore := 70
	for _, attempt := range decadeAttempts {
		flags, flagErr := studypkg.ComputeCorrectnessFlags(attempt.QuizPayload, attempt.AnswersJSON)
		if flagErr != nil || flags == nil {
			utils.Warnf("[MILESTONE_EXAM] skipped_corrupt_attempt notebookID=%s attemptID=%s err=%v", notebookID, attempt.ID, flagErr)
			continue
		}
		quizzes[attempt.ID] = flags
		if representativeAttemptID == "" {
			representativeAttemptID = attempt.ID
			if attempt.PassingScore > 0 {
				passingScore = attempt.PassingScore
			}
		}
	}

	payload := models.MilestoneExamPayload{
		Quizzes:      quizzes,
		PassingScore: passingScore,
		QuizCount:    len(quizzes),
	}
	inserted, insertErr := repo.InsertMilestoneExamTaskIfMissing(notebookID, representativeAttemptID, payload)
	if insertErr != nil {
		utils.Warnf("[MILESTONE_EXAM] insertion_failed notebookID=%s err=%v", notebookID, insertErr)
	} else if inserted {
		utils.Warnf("[MILESTONE_EXAM] inserted notebookID=%s quizCount=%d", notebookID, len(quizzes))
	}
}

// ---------- Manual Mode endpoints ----------

func (a *App) GenerateManualFlashcards(notebookID string, startPage, endPage int) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	return a.studyService.GenerateManualFlashcards(notebookID, startPage, endPage)
}

func (a *App) GenerateComprehensiveExam(notebookID string, startPage, endPage int) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	return a.studyService.GenerateComprehensiveExam(notebookID, startPage, endPage)
}

func (a *App) GetReviewSession(taskID string, notebookID string) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}

	session, err := a.studyService.GetReviewSession(taskID)
	if err != nil {
		return mapTaskError(err)
	}
	return map[string]interface{}{"session": session}
}

func (a *App) RecordCardReview(taskID, cardID string, rating int) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	remaining, err := a.studyService.RecordCardReview(taskID, cardID, rating)
	if err != nil {
		return mapTaskError(err)
	}
	return map[string]interface{}{"ok": true, "remaining": remaining}
}

// CompleteReviewSession finalizes an active review task and marks it completed via Unified Transition Router.
func (a *App) CompleteReviewSession(taskID string) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	_, err := a.studyService.TransitionTask(context.Background(), studypkg.TransitionRequest{
		TaskID: taskID,
		Event:  studypkg.EventCompleteFlashcardReview,
	})
	if err != nil {
		return mapTaskError(err)
	}
	return map[string]interface{}{"ok": true}
}

func (a *App) SuspendFlashcard(taskID, cardID string) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	remaining, err := a.studyService.SuspendFlashcard(taskID, cardID)
	if err != nil {
		return mapTaskError(err)
	}
	return map[string]interface{}{"ok": true, "remaining": remaining}
}

func (a *App) ForceDueFlashcardsNow() map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	profileID, _ := repo.GetActiveProfileID()
	updated, err := repo.MakeAllFlashcardsDueNow(profileID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "updated_cards": updated}
}

func (a *App) ScoreShortAnswer(questionID, userAnswer string) map[string]interface{} {
	if _, errMap := requireRepo(a); errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	return a.studyService.ScoreShortAnswer(questionID, userAnswer)
}

// CompleteSocraticRescue completes the socratic rescue session and inserts a re-quiz via Unified Transition Router.
func (a *App) CompleteSocraticRescue(taskID string) map[string]interface{} {
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	res, err := a.studyService.TransitionTask(context.Background(), studypkg.TransitionRequest{
		TaskID: taskID,
		Event:  studypkg.EventCompleteSocraticRescue,
	})
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "quiz_task_id": res.NextTaskID}
}

// DevForceSocraticRescue forces a topic into the SOCRATIC_REMEDIAL queue task state.
// Only accessible when APP_ENV = dev.
func (a *App) DevForceSocraticRescue(notebookID, topicID string) map[string]interface{} {
	if os.Getenv("APP_ENV") != "dev" {
		return map[string]interface{}{"error": "forbidden: dev mode only"}
	}
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}

	tx, err := repo.Begin()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer func() { _ = tx.Rollback() }()

	// Wipe FSRS flashcards for this topic to protect purity
	if err := repo.DeleteFSRSCardsByTopicIDTx(tx, topicID); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	feedback := "Concept rescue activated. Complete the Socratic session to retry."
	socraticTaskID := uuid.NewString()
	socraticPayload, _ := json.Marshal(map[string]string{
		"feedback": feedback,
		"lane":     "socratic_rescue",
		"mode":     "external_prompt",
	})

	// Note: the hardcoded start_page value of 1 and end_page value of 10 are placeholder bounds used only for this dev helper function.
	socraticTask := models.StudyQueueTask{
		ID:          socraticTaskID,
		NotebookID:  notebookID,
		TopicID:     topicID,
		TaskType:    models.StudyTaskTypeSocraticRemedial,
		Status:      models.StudyTaskStatusPending,
		Priority:    0,
		PayloadJSON: string(socraticPayload),
		StartPage:   1,
		EndPage:     10,
	}
	err = repo.InsertStudyTaskTx(tx, socraticTask)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	if err := tx.Commit(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"ok": true, "task_id": socraticTaskID}
}

// DevForceFlashcardGenerate forces a FLASHCARD_GENERATE task into the pending queue.
// Only accessible when APP_ENV = dev.
func (a *App) DevForceFlashcardGenerate(notebookID string) map[string]interface{} {
	if os.Getenv("APP_ENV") != "dev" {
		return map[string]interface{}{"error": "forbidden: dev mode only"}
	}
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	topics, err := repo.GetNotebookTopicsWithBounds(notebookID)
	if err != nil || len(topics) == 0 {
		if err := repo.EnsurePendingFlashcardGenerateTask(notebookID, "dev-dummy-topic", 1, 10, "Dev Dummy Topic"); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{"ok": true}
	}
	firstTopic := topics[0]
	startPage := firstTopic.StartPage
	if startPage <= 0 {
		startPage = 1
	}
	endPage := firstTopic.EndPage
	if endPage <= startPage {
		endPage = startPage + 10
	}
	if err := repo.EnsurePendingFlashcardGenerateTask(notebookID, firstTopic.TopicID, startPage, endPage, firstTopic.Title); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true}
}

// RetryFlashcardGeneration retries generating flashcards for a failed FLASHCARD_GENERATE task.
func (a *App) RetryFlashcardGeneration(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	if a.studyService == nil {
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}

	task, err := repo.GetTaskByID(taskID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if task.TaskType != models.StudyTaskTypeFlashcardGenerate {
		return map[string]interface{}{"error": "task is not a flashcard generation retry task"}
	}

	topicID, startPage, endPage, resolveErr := resolveRetryTopicAndBounds(repo, task)
	if resolveErr != nil {
		utils.Warnf("[FLASHCARD_PIPELINE] retry_flashcard_generation_failed taskID=%s reason=no_topics_for_notebook notebookID=%s", taskID, task.NotebookID)
		return map[string]interface{}{"error": resolveErr.Error()}
	}

	utils.Warnf("[FLASHCARD_PIPELINE] retry_flashcard_generation_started taskID=%s topicID=%s notebookID=%s", taskID, topicID, task.NotebookID)

	var cardCount int
	var existingCardsExist bool

	if topicID != "" {
		count, countErr := repo.CountFlashcardsForTopic(topicID)
		if countErr == nil && count > 0 {
			utils.Infof("[FLASHCARD_PIPELINE] retry_flashcard_generation_skipped reason=cards_already_exist taskID=%s topicID=%s cardCount=%d", taskID, topicID, count)
			cardCount = count
			existingCardsExist = true
		}
	}

	if !existingCardsExist {
		cardCount, err = a.studyService.GenerateFlashcardsAfterQuiz(task.NotebookID, topicID, startPage, endPage)
	}

	if err != nil {
		utils.Warnf("[FLASHCARD_PIPELINE] retry_flashcard_generation_failed taskID=%s reason=%v", taskID, err)
		_, _ = a.studyService.TransitionTask(context.Background(), studypkg.TransitionRequest{
			TaskID:      taskID,
			Event:       studypkg.EventFailTask,
			ErrorReason: err.Error(),
		})
		return map[string]interface{}{"error": "failed to generate flashcards: " + err.Error()}
	}

	if _, completeErr := a.studyService.TransitionTask(context.Background(), studypkg.TransitionRequest{
		TaskID:    taskID,
		Event:     studypkg.EventCompleteFlashcards,
		TopicID:   topicID,
		CardCount: cardCount,
	}); completeErr != nil {
		utils.Warnf("[FLASHCARD_PIPELINE] retry_flashcard_completion_failed taskID=%s reason=%v", taskID, completeErr)
		return map[string]interface{}{"error": "failed to complete flashcard task: " + completeErr.Error()}
	}
	utils.Infof("[FLASHCARD_PIPELINE] retry_flashcard_generation_completed taskID=%s topicID=%s cardsScheduled=%d", taskID, topicID, cardCount)

	return map[string]interface{}{
		"ok":              true,
		"cards_scheduled": cardCount,
	}
}

func resolveRetryTopicAndBounds(repo *db.Repository, task models.StudyQueueTask) (string, int, int, error) {
	topicID := task.TopicID
	startPage := task.StartPage
	endPage := task.EndPage
	if topicID != "" && startPage > 0 && endPage > 0 && endPage >= startPage {
		return topicID, startPage, endPage, nil
	}

	topics, topicsErr := repo.GetNotebookTopicsWithBounds(task.NotebookID)
	if topicsErr != nil || len(topics) == 0 {
		return "", 0, 0, fmt.Errorf("no topics found for notebook, cannot generate flashcards")
	}
	firstTopic := topics[0]
	if topicID == "" {
		topicID = firstTopic.TopicID
	}
	if startPage <= 0 {
		startPage = firstTopic.StartPage
		if startPage <= 0 {
			startPage = 1
		}
	}
	if endPage <= 0 || endPage < startPage {
		endPage = firstTopic.EndPage
		if endPage <= 0 || endPage < startPage {
			endPage = startPage
		}
	}
	return topicID, startPage, endPage, nil
}
