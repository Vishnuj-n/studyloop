package db

import (
	"ai-tutor/internal/models"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSchemaIncludesRereadAttemptsTable(t *testing.T) {
	initDBForTest(t, false, 0)

	var name string
	if err := testRepo.db.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = 'reread_attempts'
	`).Scan(&name); err != nil {
		t.Fatalf("expected reread_attempts table to exist: %v", err)
	}
	if name != "reread_attempts" {
		t.Fatalf("expected reread_attempts table, got %q", name)
	}
}

func TestSchemaIncludesReviewTaskCardsTableAndIndex(t *testing.T) {
	initDBForTest(t, false, 0)

	var tableName string
	if err := testRepo.db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = 'review_task_cards'
	`).Scan(&tableName); err != nil {
		t.Fatalf("expected review_task_cards table to exist: %v", err)
	}
	if tableName != "review_task_cards" {
		t.Fatalf("expected review_task_cards table, got %q", tableName)
	}

	var indexName string
	if err := testRepo.db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_review_task_cards_task_status'
	`).Scan(&indexName); err != nil {
		t.Fatalf("expected idx_review_task_cards_task_status index to exist: %v", err)
	}
}

func TestRereadAttemptCountHelpers(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-attempts", "Topic Attempts"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}

	count, err := testRepo.GetRereadAttemptCount("topic-attempts")
	if err != nil {
		t.Fatalf("GetRereadAttemptCount initial failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected initial reread attempt count 0, got %d", count)
	}

	tx, err := testRepo.db.Begin()
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	count, err = testRepo.IncrementRereadAttemptCountTx(tx, "topic-attempts")
	if err != nil {
		t.Fatalf("IncrementRereadAttemptCountTx first failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected first increment to return 1, got %d", count)
	}
	count, err = testRepo.IncrementRereadAttemptCountTx(tx, "topic-attempts")
	if err != nil {
		t.Fatalf("IncrementRereadAttemptCountTx second failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected second increment to return 2, got %d", count)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit increment tx failed: %v", err)
	}

	count, err = testRepo.GetRereadAttemptCount("topic-attempts")
	if err != nil {
		t.Fatalf("GetRereadAttemptCount after increment failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected persisted reread attempt count 2, got %d", count)
	}

	tx, err = testRepo.db.Begin()
	if err != nil {
		t.Fatalf("begin reset tx failed: %v", err)
	}
	if err := testRepo.ResetRereadAttemptCountTx(tx, "topic-attempts"); err != nil {
		t.Fatalf("ResetRereadAttemptCountTx failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit reset tx failed: %v", err)
	}

	count, err = testRepo.GetRereadAttemptCount("topic-attempts")
	if err != nil {
		t.Fatalf("GetRereadAttemptCount after reset failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected reread attempt count reset to 0, got %d", count)
	}
}

func TestStudyQueueLifecycleAndState(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-1", "Topic 1"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-1", "NB 1", "/tmp/nb1.pdf", "pdf", "topic-1", "", 10, ""); err != nil {
		t.Fatalf("CreateNotebook nb-1 failed: %v", err)
	}
	if err := testRepo.UpdateNotebookPriority("nb-1", 9); err != nil {
		t.Fatalf("UpdateNotebookPriority failed: %v", err)
	}

	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "task-read",
		NotebookID: "nb-1",
		TopicID:    "topic-1",
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
	}); err != nil {
		t.Fatalf("InsertStudyTask reading failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "task-review",
		NotebookID: "nb-1",
		TopicID:    "topic-1",
		TaskType:   models.StudyTaskTypeFlashcardReview,
		Status:     models.StudyTaskStatusPending,
		Priority:   10,
	}); err != nil {
		t.Fatalf("InsertStudyTask review failed: %v", err)
	}

	next, err := testRepo.GetNextTask("nb-1")
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if next.ID != "task-review" {
		t.Fatalf("expected FLASHCARD_REVIEW first, got %s", next.ID)
	}

	if err := testRepo.ActivateTask(next.ID); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}

	if err := testRepo.CompleteTask(next.ID, models.CompletionResult{
		Status: models.StudyTaskStatusCompleted,
		FollowUps: []models.StudyQueueTask{
			{
				ID:         "task-follow-up",
				NotebookID: "nb-1",
				TopicID:    "topic-1",
				TaskType:   models.StudyTaskTypeQuiz,
				Status:     models.StudyTaskStatusPending,
				Priority:   0,
			},
		},
	}); err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}
}

func TestStudyQueueErrors(t *testing.T) {
	initDBForTest(t, false, 0)

	if _, err := testRepo.GetNextTask(""); !errors.Is(err, ErrNoPendingTasks) {
		t.Fatalf("expected ErrNoPendingTasks, got %v", err)
	}
	if err := testRepo.ActivateTask("missing"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestStudyQueueDeterministicOrdering(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-a", "Topic A"); err != nil {
		t.Fatalf("EnsureTopic topic-a failed: %v", err)
	}
	if err := testRepo.EnsureTopic("topic-b", "Topic B"); err != nil {
		t.Fatalf("EnsureTopic topic-b failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-a", "NB A", "/tmp/a.pdf", "pdf", "topic-a", "", 10, ""); err != nil {
		t.Fatalf("CreateNotebook nb-a failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-b", "NB B", "/tmp/b.pdf", "pdf", "topic-b", "", 10, ""); err != nil {
		t.Fatalf("CreateNotebook nb-b failed: %v", err)
	}
	if _, err := testRepo.db.Exec(`UPDATE notebooks SET priority = 10 WHERE id = 'nb-a'`); err != nil {
		t.Fatalf("set nb-a priority failed: %v", err)
	}
	if _, err := testRepo.db.Exec(`UPDATE notebooks SET priority = 1 WHERE id = 'nb-b'`); err != nil {
		t.Fatalf("set nb-b priority failed: %v", err)
	}

	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "t-low-notebook",
		NotebookID: "nb-b",
		TopicID:    "topic-b",
		TaskType:   models.StudyTaskTypeQuiz,
		Status:     models.StudyTaskStatusPending,
		Priority:   0,
	}); err != nil {
		t.Fatalf("Insert t-low-notebook failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "t-high-notebook",
		NotebookID: "nb-a",
		TopicID:    "topic-a",
		TaskType:   models.StudyTaskTypeQuiz,
		Status:     models.StudyTaskStatusPending,
		Priority:   0,
	}); err != nil {
		t.Fatalf("Insert t-high-notebook failed: %v", err)
	}

	next, err := testRepo.GetNextTask("")
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if next.ID != "t-high-notebook" {
		t.Fatalf("expected higher notebook priority task first, got %s", next.ID)
	}
}

func TestStudyQueueTaskQueriesPreservePayloadAndExposeTitle(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-title", "Display Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-title", "Title Notebook", "/tmp/title.pdf", "pdf", "topic-title", "", 10, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

	pendingPayload := `{"kind":"pending"}`
	activePayload := `{"kind":"active"}`
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:          "task-pending",
		NotebookID:  "nb-title",
		TopicID:     "topic-title",
		TaskType:    models.StudyTaskTypeQuiz,
		Status:      models.StudyTaskStatusPending,
		Priority:    1,
		PayloadJSON: pendingPayload,
	}); err != nil {
		t.Fatalf("Insert pending task failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:          "task-active",
		NotebookID:  "nb-title",
		TopicID:     "topic-title",
		TaskType:    models.StudyTaskTypeQuiz,
		Status:      models.StudyTaskStatusPending,
		Priority:    2,
		PayloadJSON: activePayload,
	}); err != nil {
		t.Fatalf("Insert active task failed: %v", err)
	}
	if err := testRepo.ActivateTask("task-active"); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}

	pendingTasks, err := testRepo.GetAllPendingTasks()
	if err != nil {
		t.Fatalf("GetAllPendingTasks failed: %v", err)
	}
	var pendingTask *models.StudyQueueTask
	for i := range pendingTasks {
		if pendingTasks[i].ID == "task-pending" {
			pendingTask = &pendingTasks[i]
			break
		}
	}
	if pendingTask == nil {
		t.Fatalf("pending task not found in GetAllPendingTasks result: %#v", pendingTasks)
	}
	if pendingTask.PayloadJSON != pendingPayload {
		t.Fatalf("expected pending payload to remain intact, got %q", pendingTask.PayloadJSON)
	}
	if pendingTask.Title != "Display Topic" {
		t.Fatalf("expected pending task title to use topic title, got %q", pendingTask.Title)
	}

	activeTasks, err := testRepo.GetAllActiveTasks()
	if err != nil {
		t.Fatalf("GetAllActiveTasks failed: %v", err)
	}
	var activeTask *models.StudyQueueTask
	for i := range activeTasks {
		if activeTasks[i].ID == "task-active" {
			activeTask = &activeTasks[i]
			break
		}
	}
	if activeTask == nil {
		t.Fatalf("active task not found in GetAllActiveTasks result: %#v", activeTasks)
	}
	if activeTask.PayloadJSON != activePayload {
		t.Fatalf("expected active payload to remain intact, got %q", activeTask.PayloadJSON)
	}
	if activeTask.Title != "Display Topic" {
		t.Fatalf("expected active task title to use topic title, got %q", activeTask.Title)
	}
}

func TestReadingTaskProgressValidationAndCompletion(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-r", "Topic R"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-r", "NB R", "/tmp/r.pdf", "pdf", "topic-r", "", 12, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "task-reading",
		NotebookID: "nb-r",
		TopicID:    "topic-r",
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
		StartPage:  5,
		EndPage:    8,
	}); err != nil {
		t.Fatalf("InsertStudyTask reading failed: %v", err)
	}

	task, err := testRepo.GetReadingTask("task-reading")
	if err != nil {
		t.Fatalf("GetReadingTask failed: %v", err)
	}
	if task.CurrentPage != 5 {
		t.Fatalf("expected current page to initialize at start page, got %d", task.CurrentPage)
	}

	ok, err := testRepo.PersistReadingProgress("task-reading", 7)
	if err != nil {
		t.Fatalf("PersistReadingProgress failed: %v", err)
	}
	if ok {
		t.Fatalf("expected PersistReadingProgress to return false before end page")
	}

	task, err = testRepo.GetReadingTask("task-reading")
	if err != nil {
		t.Fatalf("GetReadingTask after progress failed: %v", err)
	}
	if task.CurrentPage != 7 {
		t.Fatalf("expected persisted current page 7, got %d", task.CurrentPage)
	}

	ok, err = testRepo.PersistReadingProgress("task-reading", 8)
	if err != nil {
		t.Fatalf("PersistReadingProgress at end page failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected PersistReadingProgress to return true at end page")
	}
}

func TestCompleteReadingWithGeneratedQuizAdvancesTopicCursorToTaskEnd(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-cursor", "Topic Cursor"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-cursor", "NB Cursor", "/tmp/cursor.pdf", "pdf", "topic-cursor", "", 60, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.UpdateTopicPageBounds("topic-cursor", 1, 60); err != nil {
		t.Fatalf("UpdateTopicPageBounds failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "task-cursor",
		NotebookID: "nb-cursor",
		TopicID:    "topic-cursor",
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
		StartPage:  21,
		EndPage:    49,
	}); err != nil {
		t.Fatalf("InsertStudyTask reading failed: %v", err)
	}
	if err := testRepo.ActivateTask("task-cursor"); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}

	// Persist partial progress to simulate trust-based completion without explicit final-page sync.
	if _, err := testRepo.PersistReadingProgress("task-cursor", 21); err != nil {
		t.Fatalf("PersistReadingProgress failed: %v", err)
	}

	quizTaskID, err := testRepo.CompleteReadingWithGeneratedQuiz("task-cursor", models.QuizTaskPayload{
		Questions: []models.QuizTaskQuestion{
			{
				ID:            "q1",
				Prompt:        "Prompt",
				Options:       []string{"A", "B"},
				CorrectAnswer: "A",
			},
		},
		PassingScore: 70,
	})
	if err != nil {
		t.Fatalf("CompleteReadingWithGeneratedQuiz failed: %v", err)
	}
	if quizTaskID == "" {
		t.Fatalf("expected quiz task id to be returned")
	}

	var cursor int
	if err := testRepo.db.QueryRow(`SELECT COALESCE(current_page_cursor, 0) FROM topics WHERE id = ?`, "topic-cursor").Scan(&cursor); err != nil {
		t.Fatalf("query topic cursor failed: %v", err)
	}
	if cursor != 49 {
		t.Fatalf("expected cursor advanced to task end page 49, got %d", cursor)
	}
}

func TestRereadTaskCanBeLoadedAndCompletedThroughReaderHelpers(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-reread", "Topic Reread"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.UpdateTopicPageBounds("topic-reread", 10, 14); err != nil {
		t.Fatalf("UpdateTopicPageBounds failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-reread", "NB Reread", "/tmp/reread.pdf", "pdf", "topic-reread", "", 20, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "task-reread-reader",
		NotebookID: "nb-reread",
		TopicID:    "topic-reread",
		TaskType:   models.StudyTaskTypeReread,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
		StartPage:  10,
		EndPage:    14,
	}); err != nil {
		t.Fatalf("InsertStudyTask reread failed: %v", err)
	}
	if err := testRepo.ActivateTask("task-reread-reader"); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}

	task, err := testRepo.GetReadingTask("task-reread-reader")
	if err != nil {
		t.Fatalf("GetReadingTask reread failed: %v", err)
	}
	if task.StartPage != 10 || task.EndPage != 14 {
		t.Fatalf("unexpected reread task bounds: %#v", task)
	}

	quizTaskID, err := testRepo.CompleteReadingWithGeneratedQuiz("task-reread-reader", models.QuizTaskPayload{
		PassingScore: 70,
		Questions: []models.QuizTaskQuestion{{
			ID:            "reread-q1",
			Prompt:        "Q1",
			Options:       []string{"A", "B"},
			CorrectAnswer: "A",
		}},
	})
	if err != nil {
		t.Fatalf("CompleteReadingWithGeneratedQuiz reread failed: %v", err)
	}

	var status string
	if err := testRepo.db.QueryRow(`SELECT status FROM study_queue WHERE id = ?`, "task-reread-reader").Scan(&status); err != nil {
		t.Fatalf("query reread task status failed: %v", err)
	}
	if status != "COMPLETED" {
		t.Fatalf("expected reread task status COMPLETED, got %s", status)
	}

	var quizCount int
	if err := testRepo.db.QueryRow(`
		SELECT COUNT(*)
		FROM study_queue
		WHERE topic_id = ? AND task_type = 'QUIZ' AND status = 'PENDING'
	`, "topic-reread").Scan(&quizCount); err != nil {
		t.Fatalf("query reread follow-up quiz failed: %v", err)
	}
	if quizCount != 1 {
		t.Fatalf("expected one follow-up QUIZ after reread completion, got %d", quizCount)
	}
	if quizTaskID == "" {
		t.Fatalf("expected quiz task ID to be set")
	}
	fetchedTask, err := testRepo.GetTaskByID(quizTaskID)
	if err != nil {
		t.Fatalf("failed to fetch task by ID: %v", err)
	}
	if fetchedTask.TaskType != models.StudyTaskTypeQuiz {
		t.Fatalf("expected task type %s, got %s", models.StudyTaskTypeQuiz, fetchedTask.TaskType)
	}
	if fetchedTask.ID != quizTaskID {
		t.Fatalf("expected task ID %s, got %s", quizTaskID, fetchedTask.ID)
	}
}

func TestCreateReviewSessionDueCardBatchingAndDuplicatePrevention(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-review-a", "Review Topic A"); err != nil {
		t.Fatalf("EnsureTopic A failed: %v", err)
	}
	if err := testRepo.EnsureTopic("topic-review-b", "Review Topic B"); err != nil {
		t.Fatalf("EnsureTopic B failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-review", "NB Review", "/tmp/review.pdf", "pdf", "", "", 30, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if _, err := testRepo.db.Exec(`INSERT INTO notebook_topics (notebook_id, topic_id) VALUES ('nb-review', 'topic-review-a')`); err != nil {
		t.Fatalf("link topic-review-a failed: %v", err)
	}

	cards := make([]models.Flashcard, 0, 24)
	states := make(map[string]models.FlashcardState)
	for i := 0; i < 22; i++ {
		id := "due-card-" + string(rune('a'+i))
		cards = append(cards, models.Flashcard{
			ID:        id,
			TopicID:   "topic-review-a",
			Prompt:    id,
			Answer:    "answer",
			DueAt:     int64(100 + i),
			Suspended: false,
		})
		states[id] = models.FlashcardState{}
	}
	cards = append(cards,
		models.Flashcard{ID: "future-card", TopicID: "topic-review-a", Prompt: "future", Answer: "future", DueAt: 5000, Suspended: false},
		models.Flashcard{ID: "suspended-card", TopicID: "topic-review-a", Prompt: "suspended", Answer: "suspended", DueAt: 50, Suspended: true},
		models.Flashcard{ID: "other-notebook-card", TopicID: "topic-review-b", Prompt: "other", Answer: "other", DueAt: 10, Suspended: false},
	)
	states["future-card"] = models.FlashcardState{}
	states["suspended-card"] = models.FlashcardState{}
	states["other-notebook-card"] = models.FlashcardState{}
	if err := testRepo.CreateFlashcards("topic-review-a", cards[:24], states); err != nil {
		t.Fatalf("CreateFlashcards topic-review-a failed: %v", err)
	}
	if err := testRepo.CreateFlashcards("topic-review-b", []models.Flashcard{cards[24]}, states); err != nil {
		t.Fatalf("CreateFlashcards topic-review-b failed: %v", err)
	}

	dueCards, err := testRepo.GetDueReviewCardsForNotebook("nb-review", 1000, 20)
	if err != nil {
		t.Fatalf("GetDueReviewCardsForNotebook failed: %v", err)
	}
	if len(dueCards) != 20 {
		t.Fatalf("expected due-card batch capped at 20, got %d", len(dueCards))
	}
	if dueCards[0].ID != "due-card-a" || dueCards[19].ID != "due-card-t" {
		t.Fatalf("unexpected deterministic due-card ordering: first=%s last=%s", dueCards[0].ID, dueCards[19].ID)
	}

	task, existing, err := testRepo.CreateReviewSession("nb-review")
	if err != nil {
		t.Fatalf("CreateReviewSession failed: %v", err)
	}
	if existing {
		t.Fatalf("expected first CreateReviewSession to create a new task")
	}
	if task == nil {
		t.Fatalf("expected review task to be created")
	}

	var linkedCount int
	if err := testRepo.db.QueryRow(`SELECT COUNT(*) FROM review_task_cards WHERE task_id = ?`, task.ID).Scan(&linkedCount); err != nil {
		t.Fatalf("count review_task_cards failed: %v", err)
	}
	if linkedCount != 23 {
		t.Fatalf("expected 23 linked review cards, got %d", linkedCount)
	}

	task2, existing2, err := testRepo.CreateReviewSession("nb-review")
	if err != nil {
		t.Fatalf("second CreateReviewSession failed: %v", err)
	}
	if !existing2 {
		t.Fatalf("expected second CreateReviewSession to return existing task")
	}
	if task2 == nil || task2.ID != task.ID {
		t.Fatalf("expected duplicate prevention to return task %s, got %#v", task.ID, task2)
	}
	assertCountEquals(t, `SELECT COUNT(*) FROM study_queue WHERE notebook_id = ? AND task_type = 'FLASHCARD_REVIEW'`, "nb-review", 1)
}

func TestReviewSessionRecoveryOrderingAndCompletion(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-session", "Review Session Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-session", "NB Session", "/tmp/session.pdf", "pdf", "", "", 20, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if _, err := testRepo.db.Exec(`INSERT INTO notebook_topics (notebook_id, topic_id) VALUES ('nb-session', 'topic-session')`); err != nil {
		t.Fatalf("link topic failed: %v", err)
	}
	if err := testRepo.CreateFlashcards("topic-session", []models.Flashcard{
		{ID: "card-1", TopicID: "topic-session", Prompt: "Q1", Answer: "A1", DueAt: 10},
		{ID: "card-2", TopicID: "topic-session", Prompt: "Q2", Answer: "A2", DueAt: 20},
		{ID: "card-3", TopicID: "topic-session", Prompt: "Q3", Answer: "A3", DueAt: 30},
	}, map[string]models.FlashcardState{
		"card-1": {},
		"card-2": {},
		"card-3": {},
	}); err != nil {
		t.Fatalf("CreateFlashcards failed: %v", err)
	}

	task, _, err := testRepo.CreateReviewSession("nb-session")
	if err != nil {
		t.Fatalf("CreateReviewSession failed: %v", err)
	}
	if err := testRepo.ActivateTask(task.ID); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}

	if _, err := testRepo.db.Exec(`
		UPDATE review_task_cards SET status = 'reviewed'
		WHERE task_id = ? AND card_id = 'card-1'
	`, task.ID); err != nil {
		t.Fatalf("seed reviewed link failed: %v", err)
	}

	session, err := testRepo.GetReviewSession(task.ID)
	if err != nil {
		t.Fatalf("GetReviewSession failed: %v", err)
	}
	if session.Remaining != 2 || session.ReviewedCount != 1 {
		t.Fatalf("unexpected session counts: %#v", session)
	}
	if session.NextPendingIdx != 0 || session.CurrentCard == nil || session.CurrentCard.CardID != "card-2" {
		t.Fatalf("expected next pending card-2 first, got %#v", session.CurrentCard)
	}
	if session.Cards[2].CardID != "card-1" || session.Cards[2].Status != models.ReviewTaskCardStatusReviewed {
		t.Fatalf("expected reviewed card moved after pending cards, got %#v", session.Cards)
	}

	if err := testRepo.CompleteReviewSession(task.ID); !errors.Is(err, ErrReviewSessionOpen) {
		t.Fatalf("expected ErrReviewSessionOpen before all cards reviewed, got %v", err)
	}

	if _, err := testRepo.db.Exec(`UPDATE review_task_cards SET status = 'reviewed' WHERE task_id = ?`, task.ID); err != nil {
		t.Fatalf("mark all reviewed failed: %v", err)
	}
	if err := testRepo.CompleteReviewSession(task.ID); err != nil {
		t.Fatalf("CompleteReviewSession failed: %v", err)
	}

	var status string
	if err := testRepo.db.QueryRow(`SELECT status FROM study_queue WHERE id = ?`, task.ID).Scan(&status); err != nil {
		t.Fatalf("query task status failed: %v", err)
	}
	if status != "COMPLETED" {
		t.Fatalf("expected COMPLETED task, got %s", status)
	}
}

func TestCreateReviewSessionResolvesLegacyNotebookTopicContext(t *testing.T) {
	initDBForTest(t, false, 0)

	if err := testRepo.EnsureTopic("topic-legacy-review", "Legacy Review Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook("nb-legacy-review", "Legacy NB", "/tmp/legacy.pdf", "pdf", "topic-legacy-review", "", 12, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.CreateFlashcards("topic-legacy-review", []models.Flashcard{
		{ID: "legacy-card-1", TopicID: "topic-legacy-review", Prompt: "Q1", Answer: "A1", DueAt: 10},
		{ID: "legacy-card-2", TopicID: "topic-legacy-review", Prompt: "Q2", Answer: "A2", DueAt: 20},
	}, map[string]models.FlashcardState{
		"legacy-card-1": {},
		"legacy-card-2": {},
	}); err != nil {
		t.Fatalf("CreateFlashcards failed: %v", err)
	}

	task, existing, err := testRepo.CreateReviewSession("nb-legacy-review")
	if err != nil {
		t.Fatalf("CreateReviewSession failed: %v", err)
	}
	if existing {
		t.Fatalf("expected new session for legacy-linked notebook")
	}
	if task == nil || task.NotebookID != "nb-legacy-review" {
		t.Fatalf("expected task for notebook nb-legacy-review, got %#v", task)
	}

	var linkedCount int
	if err := testRepo.db.QueryRow(`SELECT COUNT(*) FROM review_task_cards WHERE task_id = ?`, task.ID).Scan(&linkedCount); err != nil {
		t.Fatalf("count review_task_cards failed: %v", err)
	}
	if linkedCount != 2 {
		t.Fatalf("expected 2 linked review cards, got %d", linkedCount)
	}
}

func TestStudyQueueNewPriorityLevels(t *testing.T) {
	initDBForTest(t, false, 0)

	topicID := "topic-priority"
	notebookID := "nb-priority"

	if err := testRepo.EnsureTopic(topicID, "Priority Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook(notebookID, "Priority Notebook", "/tmp/priority.pdf", "pdf", topicID, "", 5, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

	taskTypes := []models.StudyTaskType{
		models.StudyTaskTypeExaminer,
		models.StudyTaskTypeSocraticRemedial,
		models.StudyTaskTypeReading,
		models.StudyTaskTypeMilestoneExam,
		models.StudyTaskTypeQuiz,
		models.StudyTaskTypeReread,
		models.StudyTaskTypeFlashcardReview,
		models.StudyTaskTypeFlashcardGenerate,
	}

	// Insert all task types in reverse-priority or arbitrary order to test queue sorting
	for i, taskType := range taskTypes {
		taskID := fmt.Sprintf("task-%d", i)
		if err := testRepo.InsertStudyTask(models.StudyQueueTask{
			ID:         taskID,
			NotebookID: notebookID,
			TopicID:    topicID,
			TaskType:   taskType,
			Status:     models.StudyTaskStatusPending,
			Priority:   1, // keep same priority to test task type precedence
		}); err != nil {
			t.Fatalf("InsertStudyTask %s failed: %v", taskType, err)
		}
	}

	// Expected order (highest priority first)
	expectedOrder := []models.StudyTaskType{
		models.StudyTaskTypeFlashcardGenerate,
		models.StudyTaskTypeSocraticRemedial,
		models.StudyTaskTypeFlashcardReview,
		models.StudyTaskTypeReread,
		models.StudyTaskTypeQuiz,
		models.StudyTaskTypeMilestoneExam,
		models.StudyTaskTypeReading,
		models.StudyTaskTypeExaminer,
	}

	for _, expectedType := range expectedOrder {
		next, err := testRepo.GetNextTask(notebookID)
		if err != nil {
			t.Fatalf("GetNextTask failed: %v", err)
		}
		if next.TaskType != expectedType {
			t.Fatalf("expected next task type to be %s, got %s", expectedType, next.TaskType)
		}
		// Activate task first
		if err := testRepo.ActivateTask(next.ID); err != nil {
			t.Fatalf("ActivateTask failed: %v", err)
		}
		// Complete task to get the next one in queue
		if err := testRepo.CompleteTask(next.ID, models.CompletionResult{Status: models.StudyTaskStatusCompleted}); err != nil {
			t.Fatalf("CompleteTask failed: %v", err)
		}
	}
}

func TestGetCompletedTaskTimes(t *testing.T) {
	initDBForTest(t, false, 0)

	notebookID := "nb-streak-test"
	topicID := "topic-streak-test"
	if err := testRepo.EnsureTopic(topicID, "Test Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook(notebookID, "Test Notebook", "/tmp/streak.pdf", "pdf", topicID, "", 5, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

	// Insert active, pending and completed tasks
	task1 := models.StudyQueueTask{
		ID:         "task-streak-1",
		NotebookID: notebookID,
		TopicID:    topicID,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
	}
	task2 := models.StudyQueueTask{
		ID:         "task-streak-2",
		NotebookID: notebookID,
		TopicID:    topicID,
		TaskType:   models.StudyTaskTypeQuiz,
		Status:     models.StudyTaskStatusPending,
	}

	if err := testRepo.InsertStudyTask(task1); err != nil {
		t.Fatalf("InsertStudyTask 1 failed: %v", err)
	}
	if err := testRepo.InsertStudyTask(task2); err != nil {
		t.Fatalf("InsertStudyTask 2 failed: %v", err)
	}

	// Fetch initial completions (should be 0)
	completions, err := testRepo.GetCompletedTaskTimes()
	if err != nil {
		t.Fatalf("GetCompletedTaskTimes initial failed: %v", err)
	}
	if len(completions) != 0 {
		t.Fatalf("expected 0 completions, got %d", len(completions))
	}

	// Activate and complete task 1
	if err := testRepo.ActivateTask(task1.ID); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}
	if err := testRepo.CompleteTask(task1.ID, models.CompletionResult{Status: models.StudyTaskStatusCompleted}); err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	// Fetch completions (should be 1)
	completions, err = testRepo.GetCompletedTaskTimes()
	if err != nil {
		t.Fatalf("GetCompletedTaskTimes after complete failed: %v", err)
	}
	if len(completions) != 1 {
		t.Fatalf("expected 1 completion, got %d", len(completions))
	}

	// Verify that the timestamp is close to now
	timeDiff := time.Since(completions[0])
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 1*time.Minute {
		t.Fatalf("expected completed time to be close to now, but diff is %v (completed time: %v, now: %v)", timeDiff, completions[0], time.Now().UTC())
	}
}

func TestMilestoneExamRepoHelpersCountOnlyPassedQuizzes(t *testing.T) {
	initDBForTest(t, false, 0)

	notebookID := "nb-milestone"
	topicID := "topic-milestone"
	if err := testRepo.EnsureTopic(topicID, "Milestone Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook(notebookID, "Milestone Notebook", "/tmp/milestone.pdf", "pdf", topicID, "", 20, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("quiz-task-%d", i)
		status := models.StudyTaskStatusPending
		if err := testRepo.InsertStudyTask(models.StudyQueueTask{
			ID:          taskID,
			NotebookID:  notebookID,
			TopicID:     topicID,
			TaskType:    models.StudyTaskTypeQuiz,
			Status:      status,
			Priority:    0,
			PayloadJSON: `{"questions":[{"id":"q1","prompt":"P","options":["A","B","C","D"],"correct_answer":"A"}],"passing_score":70}`,
		}); err != nil {
			t.Fatalf("InsertStudyTask failed: %v", err)
		}
		if err := testRepo.ActivateTask(taskID); err != nil {
			t.Fatalf("ActivateTask failed: %v", err)
		}

		passed := i != 1
		score := 100
		if !passed {
			score = 0
		}
		tx, err := testRepo.Begin()
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		if err := testRepo.SaveQuizAttemptTx(tx, models.QuizAttemptRecord{
			ID:          fmt.Sprintf("attempt-%d", i),
			TaskID:      taskID,
			Score:       score,
			Passed:      passed,
			AnswersJSON: `[{"question_id":"q1","selected":"A"}]`,
			Feedback:    "",
			CompletedAt: time.Now().Unix() + int64(i),
		}); err != nil {
			t.Fatalf("SaveQuizAttemptTx failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
	}

	count, err := testRepo.CountCompletedQuizzesByNotebook(notebookID)
	if err != nil {
		t.Fatalf("CountCompletedQuizzesByNotebook failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 passed quizzes to count toward milestone, got %d", count)
	}

	attempts, err := testRepo.GetLastNQuizAttemptsWithCorrectness(notebookID, 10)
	if err != nil {
		t.Fatalf("GetLastNQuizAttemptsWithCorrectness failed: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("expected 2 passed attempts, got %d", len(attempts))
	}
	if attempts[0].PassingScore != 70 {
		t.Fatalf("expected passing score 70, got %d", attempts[0].PassingScore)
	}
}

func TestInsertMilestoneExamTaskPersistsPayload(t *testing.T) {
	initDBForTest(t, false, 0)

	notebookID := "nb-milestone-insert"
	topicID := "topic-milestone-insert"
	if err := testRepo.EnsureTopic(topicID, "Milestone Insert Topic"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.CreateNotebook(notebookID, "Milestone Insert Notebook", "/tmp/milestone-insert.pdf", "pdf", topicID, "", 20, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}

	payload := models.MilestoneExamPayload{
		Quizzes: map[string][]int{
			"attempt-1": {1, 0, 1},
		},
		PassingScore: 70,
		QuizCount:    1,
	}
	if err := testRepo.InsertMilestoneExamTask(notebookID, payload); err != nil {
		t.Fatalf("InsertMilestoneExamTask failed: %v", err)
	}

	var taskType string
	var payloadJSON string
	if err := testRepo.db.QueryRow(`
		SELECT task_type, COALESCE(payload_json, '')
		FROM study_queue
		WHERE notebook_id = ? AND task_type = 'MILESTONE_EXAM'
		LIMIT 1
	`, notebookID).Scan(&taskType, &payloadJSON); err != nil {
		t.Fatalf("query milestone exam task failed: %v", err)
	}
	if taskType != "MILESTONE_EXAM" {
		t.Fatalf("expected MILESTONE_EXAM task type, got %s", taskType)
	}
	if payloadJSON == "" {
		t.Fatalf("expected milestone exam payload to be persisted")
	}

	exists, err := testRepo.HasMilestoneExamForAttemptID(notebookID, "attempt-1")
	if err != nil {
		t.Fatalf("HasMilestoneExamForAttemptID failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected milestone exam dedupe check to find attempt-1")
	}
}

func TestEnsurePendingReadingTaskForNotebook(t *testing.T) {
	initDBForTest(t, false, 0)

	notebookID := "nb-ensure-test"
	topicID := "topic-ensure-1"
	if err := testRepo.EnsureTopic(topicID, "Chapter 1"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.UpdateTopicPageBounds(topicID, 1, 10); err != nil {
		t.Fatalf("UpdateTopicPageBounds failed: %v", err)
	}
	if err := testRepo.CreateNotebook(notebookID, "Ensure Test Book", "/tmp/test.pdf", "pdf", topicID, "", 10, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.LinkNotebookTopics(notebookID, []string{topicID}); err != nil {
		t.Fatalf("LinkNotebookTopics failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStatus(notebookID, "chunked"); err != nil {
		t.Fatalf("UpdateNotebookStatus failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStudyStatus(notebookID, "active"); err != nil {
		t.Fatalf("UpdateNotebookStudyStatus failed: %v", err)
	}

	// First call should create the READING task in study_queue
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, 5000); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook failed: %v", err)
	}

	taskID := "task-read-" + notebookID + "-" + topicID
	task, err := testRepo.GetTaskByID(taskID)
	if err != nil {
		t.Fatalf("expected created task, got err: %v", err)
	}
	if task.NotebookID != notebookID || task.TopicID != topicID || task.TaskType != models.StudyTaskTypeReading || task.Status != models.StudyTaskStatusPending {
		t.Fatalf("unexpected task state: %+v", task)
	}

	// Second call should be idempotent and not create duplicate task
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, 5000); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook second call failed: %v", err)
	}

	// Mark created task COMPLETED while leaving topic incomplete
	if err := testRepo.ActivateTask(taskID); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}
	if err := testRepo.CompleteTask(taskID, models.CompletionResult{Status: models.StudyTaskStatusCompleted}); err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	// Call EnsurePendingReadingTaskForNotebook again; should replenish without PK collision
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, 5000); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook after completion failed: %v", err)
	}

	pendingTasks, err := testRepo.GetAllPendingTasks()
	if err != nil {
		t.Fatalf("GetAllPendingTasks failed: %v", err)
	}
	if len(pendingTasks) == 0 {
		t.Fatalf("expected pending task after replenishment, got none")
	}
}

func TestEnsurePendingReadingTasksForActiveNotebooks(t *testing.T) {
	initDBForTest(t, false, 0)

	profileID := "prof-test-batch"

	// Eligible notebook
	nbActive := "nb-batch-active"
	topicActive := "topic-batch-active"
	_ = testRepo.EnsureTopic(topicActive, "Active Topic")
	_ = testRepo.UpdateTopicPageBounds(topicActive, 1, 5)
	_ = testRepo.CreateNotebook(nbActive, "Active Book", "/tmp/active.pdf", "pdf", topicActive, profileID, 5, "")
	_ = testRepo.LinkNotebookTopics(nbActive, []string{topicActive})
	_ = testRepo.UpdateNotebookStatus(nbActive, "chunked")
	_ = testRepo.UpdateNotebookStudyStatus(nbActive, "active")

	// Ineligible notebook (dormant)
	nbDormant := "nb-batch-dormant"
	topicDormant := "topic-batch-dormant"
	_ = testRepo.EnsureTopic(topicDormant, "Dormant Topic")
	_ = testRepo.UpdateTopicPageBounds(topicDormant, 1, 5)
	_ = testRepo.CreateNotebook(nbDormant, "Dormant Book", "/tmp/dormant.pdf", "pdf", topicDormant, profileID, 5, "")
	_ = testRepo.LinkNotebookTopics(nbDormant, []string{topicDormant})
	_ = testRepo.UpdateNotebookStatus(nbDormant, "chunked")
	_ = testRepo.UpdateNotebookStudyStatus(nbDormant, "dormant")

	if err := testRepo.EnsurePendingReadingTasksForActiveNotebooks(profileID); err != nil {
		t.Fatalf("EnsurePendingReadingTasksForActiveNotebooks failed: %v", err)
	}

	tasks, err := testRepo.GetAllPendingTasks()
	if err != nil {
		t.Fatalf("GetAllPendingTasks failed: %v", err)
	}

	var hasActive, hasDormant bool
	for _, task := range tasks {
		if task.NotebookID == nbActive {
			hasActive = true
		}
		if task.NotebookID == nbDormant {
			hasDormant = true
		}
	}

	if !hasActive {
		t.Fatalf("expected eligible active notebook to receive pending task")
	}
	if hasDormant {
		t.Fatalf("expected ineligible dormant notebook not to receive pending task")
	}
}

func TestMarkTopicCompletedTx(t *testing.T) {
	initDBForTest(t, false, 0)
	topicID := "topic-test-mark-completed"
	_ = testRepo.EnsureTopic(topicID, "Topic To Complete")

	tx, err := testRepo.Begin()
	if err != nil {
		t.Fatalf("Begin tx failed: %v", err)
	}
	if err := testRepo.MarkTopicCompletedTx(tx, topicID); err != nil {
		_ = tx.Rollback()
		t.Fatalf("MarkTopicCompletedTx failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit tx failed: %v", err)
	}

	var status string
	if err := testRepo.db.QueryRow("SELECT COALESCE(status, 'unseen') FROM topics WHERE id = ?", topicID).Scan(&status); err != nil {
		t.Fatalf("Querying topic status failed: %v", err)
	}
	if status != "completed" {
		t.Fatalf("expected topic status to be 'completed', got %q", status)
	}
}

func TestEnsurePendingReadingTask_SemanticExtensionFallback(t *testing.T) {
	initDBForTest(t, false, 0)
	notebookID := "nb-semantic-ext-fallback"
	topicID := "topic-semantic-ext-fallback"

	if err := testRepo.EnsureTopic(topicID, "Chapter 1"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if err := testRepo.UpdateTopicPageBounds(topicID, 1, 10); err != nil {
		t.Fatalf("UpdateTopicPageBounds failed: %v", err)
	}
	if err := testRepo.CreateNotebook(notebookID, "Ensure Test Book", "/tmp/test.pdf", "pdf", topicID, "", 10, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.LinkNotebookTopics(notebookID, []string{topicID}); err != nil {
		t.Fatalf("LinkNotebookTopics failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStatus(notebookID, "chunked"); err != nil {
		t.Fatalf("UpdateNotebookStatus failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStudyStatus(notebookID, "active"); err != nil {
		t.Fatalf("UpdateNotebookStudyStatus failed: %v", err)
	}

	// Create 10 pages with 600 words each (default 5000 target => ~8-9 pages cutoff)
	for p := 1; p <= 10; p++ {
		cID := fmt.Sprintf("chunk-ext-fallback-p%d", p)
		_, _ = testRepo.db.Exec("INSERT INTO chunks (id, topic_id, page_num, token_count, chunk_text) VALUES (?, ?, ?, 600, 'sample text')", cID, topicID, p)
		_, _ = testRepo.db.Exec("INSERT INTO notebook_chunks (id, notebook_id, chunk_id, page_num) VALUES (?, ?, ?, ?)", cID+"-nc", notebookID, cID, p)
	}

	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, 5000); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook failed: %v", err)
	}

	tasks, err := testRepo.GetAllPendingTasks()
	if err != nil || len(tasks) == 0 {
		t.Fatalf("failed to fetch created task: %v", err)
	}
	task := tasks[0]

	// Without chunk_vectors, semantic extension gracefully falls back and creates a valid task bounds
	if task.StartPage != 1 || task.EndPage < 1 || task.EndPage > 10 {
		t.Fatalf("unexpected task page bounds fallback: start=%d end=%d", task.StartPage, task.EndPage)
	}
}

func TestStudyQueueRoundRobinInterleaving(t *testing.T) {
	initDBForTest(t, false, 0)
	profileID := "prof-rr-test"

	// Create 2 notebooks with equal priority (5)
	nbA := "nb-rr-a"
	topicA := "topic-rr-a"
	_ = testRepo.EnsureTopic(topicA, "Topic A")
	_ = testRepo.UpdateTopicPageBounds(topicA, 1, 5)
	_ = testRepo.CreateNotebook(nbA, "Book A (Alpha)", "/tmp/a.pdf", "pdf", topicA, profileID, 5, "")
	_ = testRepo.LinkNotebookTopics(nbA, []string{topicA})
	_ = testRepo.UpdateNotebookStatus(nbA, "chunked")
	_ = testRepo.UpdateNotebookStudyStatus(nbA, "active")

	nbB := "nb-rr-b"
	topicB := "topic-rr-b"
	_ = testRepo.EnsureTopic(topicB, "Topic B")
	_ = testRepo.UpdateTopicPageBounds(topicB, 1, 5)
	_ = testRepo.CreateNotebook(nbB, "Book B (Beta)", "/tmp/b.pdf", "pdf", topicB, profileID, 5, "")
	_ = testRepo.LinkNotebookTopics(nbB, []string{topicB})
	_ = testRepo.UpdateNotebookStatus(nbB, "chunked")
	_ = testRepo.UpdateNotebookStudyStatus(nbB, "active")

	// Insert READING tasks for both notebooks
	taskA1 := models.StudyQueueTask{
		ID:         "task-rr-a1",
		NotebookID: nbA,
		TopicID:    topicA,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
	}
	taskB1 := models.StudyQueueTask{
		ID:         "task-rr-b1",
		NotebookID: nbB,
		TopicID:    topicB,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
	}
	_ = testRepo.InsertStudyTask(taskA1)
	_ = testRepo.InsertStudyTask(taskB1)

	// Initially, get next task
	next1, err := testRepo.GetNextTask("")
	if err != nil {
		t.Fatalf("GetNextTask 1 failed: %v", err)
	}

	// Activate and complete next1
	_ = testRepo.ActivateTask(next1.ID)
	_ = testRepo.CompleteTask(next1.ID, models.CompletionResult{Status: models.StudyTaskStatusCompleted})

	// Insert follow-up READING task for the notebook that was just completed
	taskFollowup := models.StudyQueueTask{
		ID:         "task-rr-followup",
		NotebookID: next1.NotebookID,
		TopicID:    next1.TopicID,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		Priority:   1,
	}
	_ = testRepo.InsertStudyTask(taskFollowup)

	// Next task retrieved MUST be for the other notebook (round-robin rotation)
	next2, err := testRepo.GetNextTask("")
	if err != nil {
		t.Fatalf("GetNextTask 2 failed: %v", err)
	}

	if next2.NotebookID == next1.NotebookID {
		t.Fatalf("expected round-robin rotation to different notebook, but got same notebook %s twice", next1.NotebookID)
	}
}

func TestGetAllPendingTasks_MultipleTasksWithProfile(t *testing.T) {
	initDBForTest(t, false, 0)
	profileID := "prof-all-pending"
	_ = testRepo.CreateProfile(models.StudyProfile{ID: profileID, Name: "All Pending Profile"})
	_ = testRepo.UpdateUserSettings(models.UserSettings{ActiveProfileID: profileID})

	// Create 3 active notebooks under this profile
	for i := 1; i <= 3; i++ {
		nbID := fmt.Sprintf("nb-multi-%d", i)
		topID := fmt.Sprintf("top-multi-%d", i)
		_ = testRepo.EnsureTopic(topID, fmt.Sprintf("Topic %d", i))
		_ = testRepo.CreateNotebook(nbID, fmt.Sprintf("Book %d", i), "/tmp/b.pdf", "pdf", topID, "", 5, profileID)
		_ = testRepo.LinkNotebookTopics(nbID, []string{topID})
		_ = testRepo.UpdateNotebookStatus(nbID, "chunked")
		_ = testRepo.UpdateNotebookStudyStatus(nbID, "active")

		_ = testRepo.InsertStudyTask(models.StudyQueueTask{
			ID:         fmt.Sprintf("task-multi-%d", i),
			NotebookID: nbID,
			TopicID:    topID,
			TaskType:   models.StudyTaskTypeReading,
			Status:     models.StudyTaskStatusPending,
			Priority:   0,
		})
	}

	tasks, err := testRepo.GetAllPendingTasks()
	if err != nil {
		t.Fatalf("GetAllPendingTasks failed: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 pending tasks returned for profile with 3 active notebooks, got %d", len(tasks))
	}
}

func TestGetNextTaskWithProfile_ProfileIsolationOnExplicitNotebook(t *testing.T) {
	initDBForTest(t, false, 0)
	profileA := "prof-a"
	profileB := "prof-b"
	_ = testRepo.CreateProfile(models.StudyProfile{ID: profileA, Name: "Profile A"})
	_ = testRepo.CreateProfile(models.StudyProfile{ID: profileB, Name: "Profile B"})

	// Notebook belongs to Profile B
	nbB := "nb-profile-b"
	topB := "top-profile-b"
	_ = testRepo.EnsureTopic(topB, "Topic B")
	_ = testRepo.CreateNotebook(nbB, "Book B", "/tmp/b.pdf", "pdf", topB, "", 5, profileB)
	_ = testRepo.LinkNotebookTopics(nbB, []string{topB})
	_ = testRepo.UpdateNotebookStatus(nbB, "chunked")
	_ = testRepo.UpdateNotebookStudyStatus(nbB, "active")
	_ = testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         "task-b",
		NotebookID: nbB,
		TopicID:    topB,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
	})

	// User active profile is Profile A
	_ = testRepo.UpdateUserSettings(models.UserSettings{ActiveProfileID: profileA})

	// Requesting task for nbB while active profile is profileA must return ErrNoPendingTasks
	_, err := testRepo.GetNextTask(nbB)
	if !errors.Is(err, ErrNoPendingTasks) {
		t.Fatalf("expected ErrNoPendingTasks when requesting notebook from another profile, got %v", err)
	}
}

func TestCompleteTaskTx_PayloadPreservation(t *testing.T) {
	initDBForTest(t, false, 0)
	nbID := "nb-preserve"
	topID := "top-preserve"
	_ = testRepo.EnsureTopic(topID, "Topic Preserve")
	_ = testRepo.CreateNotebook(nbID, "Book Preserve", "/tmp/s.pdf", "pdf", topID, "", 5, "")

	taskID := "task-preserve-test"
	initialPayload := `{"initial":"payload"}`
	_ = testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:          taskID,
		NotebookID:  nbID,
		TopicID:     topID,
		TaskType:    models.StudyTaskTypeReading,
		Status:      models.StudyTaskStatusPending,
		PayloadJSON: initialPayload,
	})
	_ = testRepo.ActivateTask(taskID)

	// Complete with empty payload preserves existing payload
	tx, err := testRepo.Begin()
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	err = testRepo.CompleteTaskTx(tx, taskID, models.CompletionResult{
		Status:  models.StudyTaskStatusCompleted,
		Payload: "",
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("CompleteTaskTx failed: %v", err)
	}
	_ = tx.Commit()

	task, err := testRepo.GetTaskByID(taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.PayloadJSON != initialPayload {
		t.Fatalf("expected payload_json to be preserved as %q, got %q", initialPayload, task.PayloadJSON)
	}
}

func TestCompleteReadingWithGeneratedQuizReservedStatusFlow(t *testing.T) {
	initDBForTest(t, false, 0)

	nbID := "nb-flow-test"
	topID := "top-flow-test"
	_ = testRepo.EnsureTopic(topID, "Flow Topic")
	_ = testRepo.CreateNotebook(nbID, "Flow Notebook", "/tmp/flow.md", "md", topID, "", 5, "")

	taskID := "task-flow-read-1"
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:         taskID,
		NotebookID: nbID,
		TopicID:    topID,
		TaskType:   models.StudyTaskTypeReading,
		Status:     models.StudyTaskStatusPending,
		StartPage:  1,
		EndPage:    3,
	}); err != nil {
		t.Fatalf("InsertStudyTask failed: %v", err)
	}

	// 1. Activate task
	if err := testRepo.ActivateTask(taskID); err != nil {
		t.Fatalf("ActivateTask failed: %v", err)
	}

	// 2. Reserve task (as done during synchronous LLM quiz generation in CompleteReading)
	if err := testRepo.ReserveTask(taskID); err != nil {
		t.Fatalf("ReserveTask failed: %v", err)
	}

	// 3. CompleteReadingWithGeneratedQuiz while task is in RESERVED state
	quizPayload := models.QuizTaskPayload{
		Questions: []models.QuizTaskQuestion{
			{
				ID:            "q1",
				Prompt:        "What is database normalization?",
				Options:       []string{"A", "B", "C", "D"},
				CorrectAnswer: "A",
			},
		},
		PassingScore: 70,
	}

	quizTaskID, err := testRepo.CompleteReadingWithGeneratedQuiz(taskID, quizPayload)
	if err != nil {
		t.Fatalf("CompleteReadingWithGeneratedQuiz failed for RESERVED task: %v", err)
	}
	if quizTaskID == "" {
		t.Fatalf("expected valid quizTaskID, got empty")
	}

	// 4. Verify reading task is now COMPLETED
	readingTask, err := testRepo.GetTaskByID(taskID)
	if err != nil {
		t.Fatalf("GetTaskByID for completed reading task failed: %v", err)
	}
	if readingTask.Status != models.StudyTaskStatusCompleted {
		t.Fatalf("expected reading task status COMPLETED, got %s", readingTask.Status)
	}

	// 5. Verify follow-up QUIZ task was created and is PENDING
	quizTask, err := testRepo.GetTaskByID(quizTaskID)
	if err != nil {
		t.Fatalf("GetTaskByID for generated quiz task failed: %v", err)
	}
	if quizTask.TaskType != models.StudyTaskTypeQuiz {
		t.Fatalf("expected quiz task type QUIZ, got %s", quizTask.TaskType)
	}
	if quizTask.Status != models.StudyTaskStatusPending {
		t.Fatalf("expected quiz task status PENDING, got %s", quizTask.Status)
	}
}

func TestMultiSessionChapterSlicingAndProgression(t *testing.T) {
	initDBForTest(t, false, 0)

	notebookID := "nb-multislice"
	topicID1 := "topic-ch-1"
	topicID2 := "topic-ch-2"

	// Create Chapter 1 (pages 1-20) and Chapter 2 (pages 21-40)
	if err := testRepo.EnsureTopic(topicID1, "Chapter 1"); err != nil {
		t.Fatalf("EnsureTopic 1 failed: %v", err)
	}
	if err := testRepo.UpdateTopicPageBounds(topicID1, 1, 20); err != nil {
		t.Fatalf("UpdateTopicPageBounds 1 failed: %v", err)
	}

	if err := testRepo.EnsureTopic(topicID2, "Chapter 2"); err != nil {
		t.Fatalf("EnsureTopic 2 failed: %v", err)
	}
	if err := testRepo.UpdateTopicPageBounds(topicID2, 21, 40); err != nil {
		t.Fatalf("UpdateTopicPageBounds 2 failed: %v", err)
	}

	if err := testRepo.CreateNotebook(notebookID, "Multi-Slice Test Book", "/tmp/multislice.pdf", "pdf", topicID1, "", 40, ""); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.LinkNotebookTopics(notebookID, []string{topicID1, topicID2}); err != nil {
		t.Fatalf("LinkNotebookTopics failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStatus(notebookID, "chunked"); err != nil {
		t.Fatalf("UpdateNotebookStatus failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStudyStatus(notebookID, "active"); err != nil {
		t.Fatalf("UpdateNotebookStudyStatus failed: %v", err)
	}

	// Insert chunks: 500 words per page for pages 1-40
	for p := 1; p <= 40; p++ {
		tID := topicID1
		if p > 20 {
			tID = topicID2
		}
		cID := fmt.Sprintf("chunk-p%d", p)
		_, err := testRepo.db.Exec(`
			INSERT INTO chunks (id, topic_id, chunk_text, page_num, token_count)
			VALUES (?, ?, ?, ?, 500)
		`, cID, tID, fmt.Sprintf("Text for page %d with many words", p), p)
		if err != nil {
			t.Fatalf("insert chunk failed: %v", err)
		}
		_, err = testRepo.db.Exec(`
			INSERT INTO notebook_chunks (notebook_id, chunk_id)
			VALUES (?, ?)
		`, notebookID, cID)
		if err != nil {
			t.Fatalf("insert notebook_chunk failed: %v", err)
		}
	}

	// Target session words = 2500 -> 5 pages per session
	targetWords := 2500

	// 1. Initial replenishment -> Slice 1 should be Pages 1-5 of Chapter 1
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, targetWords); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook slice 1 failed: %v", err)
	}
	pending1, err := testRepo.GetAllPendingTasks()
	if err != nil || len(pending1) == 0 {
		t.Fatalf("GetAllPendingTasks slice 1 failed: %v", err)
	}
	task1 := pending1[0]
	if task1.StartPage != 1 || task1.EndPage != 5 || task1.TopicID != topicID1 {
		t.Fatalf("expected slice 1 (pages 1-5, topic %s), got range %d-%d topic %s", topicID1, task1.StartPage, task1.EndPage, task1.TopicID)
	}

	// Complete slice 1 reading
	if err := testRepo.ActivateTask(task1.ID); err != nil {
		t.Fatalf("activate task 1 failed: %v", err)
	}
	quizPayload := models.QuizTaskPayload{
		Questions:    []models.QuizTaskQuestion{{ID: "q1", Prompt: "P?", Options: []string{"A", "B"}, CorrectAnswer: "A"}},
		PassingScore: 70,
	}
	quizTaskID1, err := testRepo.CompleteReadingWithGeneratedQuiz(task1.ID, quizPayload)
	if err != nil {
		t.Fatalf("complete reading 1 failed: %v", err)
	}

	// Verify topic 1 cursor is now 5 and status is still reading
	var cursor1 int
	var status1 string
	if err := testRepo.db.QueryRow(`SELECT current_page_cursor, status FROM topics WHERE id = ?`, topicID1).Scan(&cursor1, &status1); err != nil {
		t.Fatalf("query topic 1 failed: %v", err)
	}
	if cursor1 != 5 {
		t.Fatalf("expected topic 1 cursor=5, got %d", cursor1)
	}
	if status1 == "completed" {
		t.Fatalf("topic 1 should NOT be completed after first slice (cursor=5, end=20)")
	}

	// Complete the quiz task for slice 1
	if err := testRepo.ActivateTask(quizTaskID1); err != nil {
		t.Fatalf("activate quiz 1 failed: %v", err)
	}
	if err := testRepo.CompleteTask(quizTaskID1, models.CompletionResult{Status: models.StudyTaskStatusCompleted}); err != nil {
		t.Fatalf("complete quiz 1 failed: %v", err)
	}

	// 2. Next replenishment -> Slice 2 should be Pages 6-10 (NO SKIPPED PAGES!)
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, targetWords); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook slice 2 failed: %v", err)
	}
	pending2, err := testRepo.GetAllPendingTasks()
	if err != nil || len(pending2) == 0 {
		t.Fatalf("GetAllPendingTasks slice 2 failed: %v", err)
	}
	task2 := pending2[0]
	if task2.StartPage != 6 || task2.EndPage != 10 || task2.TopicID != topicID1 {
		t.Fatalf("expected slice 2 (pages 6-10, topic %s), got range %d-%d topic %s", topicID1, task2.StartPage, task2.EndPage, task2.TopicID)
	}

	// Complete slice 2 reading
	if err := testRepo.ActivateTask(task2.ID); err != nil {
		t.Fatalf("activate task 2 failed: %v", err)
	}
	quizTaskID2, err := testRepo.CompleteReadingWithGeneratedQuiz(task2.ID, quizPayload)
	if err != nil {
		t.Fatalf("complete reading 2 failed: %v", err)
	}
	if err := testRepo.ActivateTask(quizTaskID2); err != nil {
		t.Fatalf("activate quiz 2 failed: %v", err)
	}
	if err := testRepo.CompleteTask(quizTaskID2, models.CompletionResult{Status: models.StudyTaskStatusCompleted}); err != nil {
		t.Fatalf("complete quiz 2 failed: %v", err)
	}

	// 3. Next replenishment -> Slice 3 should be Pages 11-15
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, targetWords); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook slice 3 failed: %v", err)
	}
	pending3, err := testRepo.GetAllPendingTasks()
	if err != nil || len(pending3) == 0 {
		t.Fatalf("GetAllPendingTasks slice 3 failed: %v", err)
	}
	task3 := pending3[0]
	if task3.StartPage != 11 || task3.EndPage != 15 || task3.TopicID != topicID1 {
		t.Fatalf("expected slice 3 (pages 11-15), got range %d-%d", task3.StartPage, task3.EndPage)
	}

	// Complete slice 3 reading & quiz
	_ = testRepo.ActivateTask(task3.ID)
	quizTaskID3, _ := testRepo.CompleteReadingWithGeneratedQuiz(task3.ID, quizPayload)
	_ = testRepo.ActivateTask(quizTaskID3)
	_ = testRepo.CompleteTask(quizTaskID3, models.CompletionResult{Status: models.StudyTaskStatusCompleted})

	// 4. Next replenishment -> Slice 4 should be Pages 16-20 (end of Chapter 1)
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, targetWords); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook slice 4 failed: %v", err)
	}
	pending4, err := testRepo.GetAllPendingTasks()
	if err != nil || len(pending4) == 0 {
		t.Fatalf("GetAllPendingTasks slice 4 failed: %v", err)
	}
	task4 := pending4[0]
	if task4.StartPage != 16 || task4.EndPage != 20 || task4.TopicID != topicID1 {
		t.Fatalf("expected slice 4 (pages 16-20), got range %d-%d", task4.StartPage, task4.EndPage)
	}

	// Complete slice 4 reading
	_ = testRepo.ActivateTask(task4.ID)
	quizTaskID4, _ := testRepo.CompleteReadingWithGeneratedQuiz(task4.ID, quizPayload)
	_ = testRepo.ActivateTask(quizTaskID4)
	_ = testRepo.CompleteTask(quizTaskID4, models.CompletionResult{Status: models.StudyTaskStatusCompleted})
	_, _ = testRepo.db.Exec(`UPDATE topics SET status = 'completed' WHERE id = ?`, topicID1)

	// 5. Next replenishment -> Should seamlessly transition to Chapter 2 (Pages 21-25)
	if err := testRepo.EnsurePendingReadingTaskForNotebook(notebookID, targetWords); err != nil {
		t.Fatalf("EnsurePendingReadingTaskForNotebook Chapter 2 failed: %v", err)
	}
	pending5, err := testRepo.GetAllPendingTasks()
	if err != nil || len(pending5) == 0 {
		t.Fatalf("GetAllPendingTasks Chapter 2 failed: %v", err)
	}
	task5 := pending5[0]
	if task5.StartPage != 21 || task5.EndPage != 25 || task5.TopicID != topicID2 {
		t.Fatalf("expected Chapter 2 slice 1 (pages 21-25, topic %s), got range %d-%d topic %s", topicID2, task5.StartPage, task5.EndPage, task5.TopicID)
	}
}
