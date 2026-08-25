package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	"ai-tutor/internal/scheduler"
	"ai-tutor/internal/utils"
)

// ---------- Helpers for GetTodayPlan ----------

func calculateDailyStudyMinutes(studyStart, studyEnd string) int {
	dailyStudyMinutes := 60 // default fallback
	var sh, sm, eh, em int
	if _, errS := fmt.Sscanf(studyStart, "%d:%d", &sh, &sm); errS == nil {
		if _, errE := fmt.Sscanf(studyEnd, "%d:%d", &eh, &em); errE == nil {
			startMins := sh*60 + sm
			endMins := eh*60 + em
			diff := endMins - startMins
			if diff < 0 {
				diff += 1440
			}
			if diff > 0 {
				dailyStudyMinutes = diff
			}
		}
	}
	return dailyStudyMinutes
}

func calculateFlashcardBudgets(dueCards, maxFlashcards int) (int, int, int) {
	materializedCards := dueCards
	if materializedCards > maxFlashcards {
		materializedCards = maxFlashcards
	}
	deferredCards := dueCards - materializedCards
	if deferredCards < 0 {
		deferredCards = 0
	}
	safeReviewBudget := int(math.Ceil(float64(materializedCards) * scheduler.ReviewMinutesPerCard))
	return materializedCards, deferredCards, safeReviewBudget
}

func aggregateQueueTasks(repo *db.Repository, active, pending []models.StudyQueueTask) ([]models.ScheduledTask, []string, int, map[string]int) {
	queueTasks := make([]models.ScheduledTask, 0, len(active)+len(pending))
	actionCounts := make(map[string]int)
	activeTopicsMap := make(map[string]bool)

	processTasks := func(tasks []models.StudyQueueTask) {
		for _, q := range tasks {
			task := queueTaskToScheduledTask(q, repo)
			queueTasks = append(queueTasks, task)
			actionCounts[task.ActionType]++
			if q.Title != "" {
				activeTopicsMap[q.Title] = true
			}
		}
	}

	processTasks(active)
	processTasks(pending)

	activeTopics := make([]string, 0, len(activeTopicsMap))
	for topicTitle := range activeTopicsMap {
		activeTopics = append(activeTopics, topicTitle)
	}

	learningMinutes := 0
	for _, task := range queueTasks {
		learningMinutes += task.EstimateMinutes
	}

	return queueTasks, activeTopics, learningMinutes, actionCounts
}

// calculateStreak computes current and longest streaks from a set of completion timestamps.
// timezoneOffsetMinutes is the JS-style offset (UTC+5:30 → -330).
func calculateStreak(times []time.Time, timezoneOffsetMinutes int) (currentStreak, longestStreak int, activeDates []string) {
	loc := time.FixedZone("ClientZone", -timezoneOffsetMinutes*60)
	nowClient := time.Now().In(loc)

	dateSet := make(map[string]bool)
	for _, t := range times {
		dateSet[t.In(loc).Format(dateFormatYYYYMMDD)] = true
	}

	sortedDates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		sortedDates = append(sortedDates, d)
	}
	sort.Strings(sortedDates)
	activeDates = sortedDates

	if len(sortedDates) == 0 {
		return 0, 0, activeDates
	}

	longestStreak = computeLongestStreak(sortedDates, loc)
	currentStreak = computeCurrentStreak(nowClient, dateSet)

	return currentStreak, longestStreak, activeDates
}

func computeLongestStreak(sortedDates []string, loc *time.Location) int {
	longestStreak := 0
	streakTemp := 0
	var prevDate time.Time
	for _, dateStr := range sortedDates {
		d, err := time.ParseInLocation(dateFormatYYYYMMDD, dateStr, loc)
		if err != nil {
			continue
		}
		if streakTemp == 0 {
			streakTemp = 1
		} else {
			daysDiff := int(d.Sub(prevDate).Hours()+0.5) / 24
			if daysDiff == 1 {
				streakTemp++
			} else if daysDiff > 1 {
				if streakTemp > longestStreak {
					longestStreak = streakTemp
				}
				streakTemp = 1
			}
		}
		prevDate = d
	}
	if streakTemp > longestStreak {
		longestStreak = streakTemp
	}
	return longestStreak
}

func computeCurrentStreak(nowClient time.Time, dateSet map[string]bool) int {
	todayStr := nowClient.Format(dateFormatYYYYMMDD)
	yesterdayStr := nowClient.AddDate(0, 0, -1).Format(dateFormatYYYYMMDD)

	anchorDate := nowClient
	if !dateSet[todayStr] && dateSet[yesterdayStr] {
		anchorDate = nowClient.AddDate(0, 0, -1)
	}
	currentStreak := 0
	if dateSet[anchorDate.Format(dateFormatYYYYMMDD)] {
		currentStreak = 1
		for {
			prevDayStr := anchorDate.AddDate(0, 0, -currentStreak).Format(dateFormatYYYYMMDD)
			if !dateSet[prevDayStr] {
				break
			}
			currentStreak++
		}
	}
	return currentStreak
}

// mapTaskError translates repository errors into API response maps.
func mapTaskError(err error) map[string]interface{} {
	switch err {
	case db.ErrTaskNotFound:
		return map[string]interface{}{"error": "ErrNotFound", "code": 404}
	case db.ErrTaskNotActive:
		return map[string]interface{}{"error": "ErrTaskNotActive", "code": 409}
	case db.ErrTaskNotPending:
		return map[string]interface{}{"error": "ErrTaskNotPending", "code": 409}
	case db.ErrReviewLinkNotPending:
		return map[string]interface{}{"error": "ErrCardAlreadyReviewed", "code": 409}
	case db.ErrReviewSessionOpen:
		return map[string]interface{}{"error": "ErrReviewSessionIncomplete", "code": 409}
	default:
		return map[string]interface{}{"error": err.Error()}
	}
}

// requireRepo returns the repository or an error map if uninitialized.
func requireRepo(a *App) (*db.Repository, map[string]interface{}) {
	repo := a.getRepo()
	if repo == nil {
		return nil, map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	return repo, nil
}

// ---------- Main App Methods ----------

func (a *App) GetTodayPlan() map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	if a.scheduler == nil {
		return map[string]interface{}{"error": "scheduler not initialized"}
	}
	now := time.Now()

	activeProfileID, err := repo.GetActiveProfileID()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if ensureErr := repo.EnsurePendingReadingTasksForActiveNotebooks(activeProfileID); ensureErr != nil {
		utils.Warnf("[QUEUE] failed to ensure reading tasks for active notebooks: %v", ensureErr)
	}

	// Canonical queue recovery/materialization path for dashboard:
	// if ACTIVE/PENDING queue tasks exist, surface those directly.
	activeQueueTasks, err := repo.GetAllActiveTasks()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	pendingQueueTasks, err := repo.GetAllPendingTasks()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// ponytail: rely 100% on study_queue and due cards, no synthetic scheduler fallback
	dueCards, err := repo.QueryDueReviewCards(now.Unix())
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	settings, err := repo.GetUserSettings()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	maxFlashcards := settings.MaxFlashcardsPerSession
	if maxFlashcards <= 0 {
		maxFlashcards = 30
	}

	dailyStudyMinutes := calculateDailyStudyMinutes(settings.StudyStartTime, settings.StudyEndTime)
	materializedCards, deferredCards, safeReviewBudget := calculateFlashcardBudgets(dueCards, maxFlashcards)
	queueTasks, activeTopics, learningMinutes, actionCounts := aggregateQueueTasks(repo, activeQueueTasks, pendingQueueTasks)

	if reviewTask, ok := buildReviewTaskForPlan(repo, now, materializedCards, safeReviewBudget); ok {
		queueTasks = append([]models.ScheduledTask{reviewTask}, queueTasks...)
		actionCounts["flashcard_review"]++
	}

	planSource := "queue-materialized"
	plan := &models.TodayPlan{
		Date:                now.Format(dateFormatYYYYMMDD),
		TotalMinutes:        dailyStudyMinutes,
		ReviewMinutes:       safeReviewBudget,
		LearningMinutes:     learningMinutes,
		DueReviewCards:      materializedCards,
		TotalDueReviewCards: dueCards,
		DeferredReviewCards: deferredCards,
		ActiveTopics:        activeTopics,
		Tasks:               queueTasks,
		IsEstimate:          false,
	}

	utils.Debugf("[TODAY_PLAN] queue materialization active=%d pending=%d merged=%d", len(activeQueueTasks), len(pendingQueueTasks), len(queueTasks))
	utils.Debugf("[TODAY_PLAN] planner aggregation dueReviewCards=%d reviewMinutes=%d queueActionCounts=%v", plan.DueReviewCards, plan.ReviewMinutes, actionCounts)
	if actionCounts["flashcard_review"] > 0 {
		utils.Debugf("[FLASHCARD_PIPELINE] today_plan_review_detected flashcard_review_count=%d", actionCounts["flashcard_review"])
	}

	// Count active notebooks for the dashboard empty-state distinction.
	activeNotebookCount, err := repo.CountActiveNotebooksForActiveProfile(activeProfileID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	utils.Debugf("[TODAY_PLAN] GetTodayPlan response tasks=%d isEstimate=%t reviewMinutes=%d learningMinutes=%d", len(plan.Tasks), plan.IsEstimate, plan.ReviewMinutes, plan.LearningMinutes)
	for idx, task := range plan.Tasks {
		utils.Debugf("[TODAY_PLAN] GetTodayPlan task[%d] taskID=%s actionType=%s topicID=%s notebookID=%s startPage=%d endPage=%d priority=%d", idx, task.ID, task.ActionType, task.TopicID, task.NotebookID, task.StartPage, task.EndPage, task.Priority)
	}
	return map[string]interface{}{
		"date": plan.Date, "total_minutes": plan.TotalMinutes,
		"review_minutes": plan.ReviewMinutes, "learning_minutes": plan.LearningMinutes,
		"due_review_cards": plan.DueReviewCards, "total_due_review_cards": plan.TotalDueReviewCards,
		"deferred_review_cards": plan.DeferredReviewCards, "active_topics": plan.ActiveTopics,
		"tasks": plan.Tasks, "is_estimate": plan.IsEstimate, "plan_source": planSource,
		"active_notebook_count": activeNotebookCount,
	}
}

func buildReviewTaskForPlan(repo *db.Repository, now time.Time, materializedCards, safeReviewBudget int) (models.ScheduledTask, bool) {
	if materializedCards <= 0 {
		return models.ScheduledTask{}, false
	}
	bestNotebookID, selectedDueCards, err := repo.GetNextDueReviewNotebook(now.Unix())
	if err != nil || bestNotebookID == "" {
		if err != nil {
			utils.Warnf("failed to get next due review notebook: %v", err)
		}
		return models.ScheduledTask{}, false
	}
	reviewCardsForTask := materializedCards
	if selectedDueCards < reviewCardsForTask {
		reviewCardsForTask = selectedDueCards
	}
	return models.ScheduledTask{
		ID:              models.ReviewTaskDailyID,
		ActionType:      "flashcard_review",
		Title:           fmt.Sprintf("Flashcard Review: %d cards", reviewCardsForTask),
		EstimateMinutes: safeReviewBudget,
		Priority:        1,
		NotebookID:      bestNotebookID,
		Meta:            fmt.Sprintf("Spaced repetition review (%d cards)", reviewCardsForTask),
	}, true
}

func queueTaskToScheduledTask(task models.StudyQueueTask, repo *db.Repository) models.ScheduledTask {
	actionType := strings.ToLower(string(task.TaskType))
	titleBase := strings.TrimSpace(task.Title)
	if titleBase == "" {
		titleBase = "Task"
	}
	titleBase = utils.CleanTopicTitle(titleBase)

	titlePrefix := "Task"
	switch task.TaskType {
	case models.StudyTaskTypeReading:
		titlePrefix = "Read"
	case models.StudyTaskTypeQuiz:
		titlePrefix = "Quiz"
	case models.StudyTaskTypeMilestoneExam:
		titlePrefix = "Milestone Exam"
	case models.StudyTaskTypeReread:
		titlePrefix = "Reread"
	case models.StudyTaskTypeFlashcardReview:
		titlePrefix = "Flashcard Review"
	case models.StudyTaskTypeExaminer:
		titlePrefix = "Examiner"
	case models.StudyTaskTypeSocraticRemedial:
		titlePrefix = "Concept Rescue"
	case models.StudyTaskTypeFlashcardGenerate:
		titlePrefix = "Generate Flashcards"
	}

	meta := ""
	if task.StartPage > 0 && task.EndPage > 0 {
		meta = fmt.Sprintf("Pages %d-%d", task.StartPage, task.EndPage)
	}

	estimateMinutes := 10
	switch {
	case task.TaskType == models.StudyTaskTypeFlashcardGenerate:
		estimateMinutes = 0
	case task.TaskType == models.StudyTaskTypeFlashcardReview:
		var payload models.ReviewSessionPayload
		if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err == nil && payload.CardCount > 0 {
			estimateMinutes = int(math.Ceil(float64(payload.CardCount) * scheduler.ReviewMinutesPerCard))
		}
	case task.StartPage > 0 && task.EndPage >= task.StartPage && task.TopicID != "" && repo != nil:
		// Word-count based estimation (words / 200 WPM).
		// Falls back to page-count if chunks have not been ingested yet.
		totalWords, _ := repo.GetTopicWordsInRange(task.TopicID, task.StartPage, task.EndPage)
		pageCount := task.EndPage - task.StartPage + 1
		estimationSource := "word_count"
		if totalWords > 0 {
			estimateMinutes = int(math.Ceil(float64(totalWords) / float64(scheduler.WordsPerMinute)))
			// safety floor: at least 1 min/page
			if pageFloor := pageCount; estimateMinutes < pageFloor {
				estimateMinutes = pageFloor
			}
		} else {
			estimationSource = "no_chunks_yet"
			estimateMinutes = int(math.Ceil(float64(pageCount) * scheduler.MinMinutesPerPage))
		}
		utils.LogReadingEstimate(task.ID, task.TopicID, task.StartPage, task.EndPage, totalWords, estimateMinutes, estimationSource)
	case task.StartPage > 0 && task.EndPage >= task.StartPage:
		estimateMinutes = int(math.Ceil(float64(task.EndPage-task.StartPage+1) * scheduler.MinMinutesPerPage))
	}

	return models.ScheduledTask{
		ID:              task.ID,
		ActionType:      actionType,
		Title:           fmt.Sprintf("%s: %s", titlePrefix, titleBase),
		TopicID:         task.TopicID,
		NotebookID:      task.NotebookID,
		StartPage:       task.StartPage,
		EndPage:         task.EndPage,
		EstimateMinutes: estimateMinutes,
		Priority:        task.Priority,
		Meta:            meta,
	}
}

func (a *App) ActivateTask(taskID string) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}
	if taskID == models.ReviewTaskDailyID {
		return map[string]interface{}{"ok": true}
	}
	task, err := repo.GetTaskByID(taskID)
	if err != nil {
		return mapTaskError(err)
	}
	if task.Status == models.StudyTaskStatusActive {
		return map[string]interface{}{"ok": true}
	}
	if task.Status == models.StudyTaskStatusCompleted {
		return map[string]interface{}{"error": "ErrTaskCompleted", "code": 409}
	}
	if err := repo.ActivateTask(taskID); err != nil {
		return mapTaskError(err)
	}
	return map[string]interface{}{"ok": true}
}

func (a *App) getStreakState(timezoneOffsetMinutes int) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}

	times, err := repo.GetCompletedTaskTimes()
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to get completed times: %v", err)}
	}

	currentStreak, longestStreak, activeDates := calculateStreak(times, timezoneOffsetMinutes)

	loc := time.FixedZone("ClientZone", -timezoneOffsetMinutes*60)
	todayStr := time.Now().In(loc).Format("2006-01-02")
	todayCompleted := false
	for _, d := range activeDates {
		if d == todayStr {
			todayCompleted = true
			break
		}
	}

	return map[string]interface{}{
		"current_streak":  currentStreak,
		"longest_streak":  longestStreak,
		"active_dates":    activeDates,
		"today_completed": todayCompleted,
	}
}

// GetDashboardOverview consolidates settings, profiles, today plan, streak state, and pending ingestion info into a single IPC payload.
func (a *App) GetDashboardOverview(timezoneOffsetMinutes int) map[string]interface{} {
	settings := a.GetUserSettings()
	profiles := a.GetProfiles()
	todayPlan := a.GetTodayPlan()
	streakState := a.getStreakState(timezoneOffsetMinutes)

	var pendingNotebook map[string]interface{}
	var pendingNotebookError string

	if repo := a.getRepo(); repo != nil {
		activeProfileID, _ := repo.GetActiveProfileID()
		notebooks, err := repo.GetNotebooks("", activeProfileID)
		if err != nil {
			pendingNotebookError = err.Error()
		} else {
			for _, nb := range notebooks {
				if (nb.ChunkCount == 0 || nb.Status == "uploaded" || nb.Status == "draft_ready" || nb.Status == "") && nb.Status != "indexing" && nb.Status != "indexed" && nb.Status != "failed" {
					pendingNotebook = map[string]interface{}{
						"id":              nb.ID,
						"title":           nb.Title,
						"file_type":       nb.FileType,
						"topic_id":        nb.TopicID,
						"status":          nb.Status,
						"indexing_status": nb.IndexingStatus,
						"page_count":      nb.PageCount,
						"chunk_count":     nb.ChunkCount,
						"priority":        nb.Priority,
					}
					break
				}
			}
		}
	} else {
		pendingNotebookError = "database not initialized"
	}

	return map[string]interface{}{
		"settings":               settings,
		"profiles":               profiles,
		"today_plan":             todayPlan,
		"streak_state":           streakState,
		"pending_notebook":       pendingNotebook,
		"pending_notebook_error": pendingNotebookError,
	}
}

// GetAppEnv returns the current value of the APP_ENV environment variable.
func (a *App) GetAppEnv() map[string]interface{} {
	return map[string]interface{}{
		"env": os.Getenv("APP_ENV"),
	}
}

type FlashcardDuePoint struct {
	Date      string `json:"date"`
	DayLabel  string `json:"day_label"`
	CardCount int    `json:"card_count"`
}

// GetFlashcardDueTimeline returns the review card load over the next 7 days.
func (a *App) GetFlashcardDueTimeline(timezoneOffsetMinutes int) map[string]interface{} {
	repo, errMap := requireRepo(a)
	if errMap != nil {
		return errMap
	}

	loc := time.FixedZone("ClientZone", -timezoneOffsetMinutes*60)
	now := time.Now().In(loc)
	y, m, d := now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, loc)
	endOfToday := midnight.Add(24 * time.Hour).Unix()

	counts, err := repo.QueryDueReviewCardsTimeline(endOfToday)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	timeline := make([]FlashcardDuePoint, 7)
	timeline[0] = FlashcardDuePoint{
		Date:      midnight.Format(dateFormatYYYYMMDD),
		DayLabel:  "Today",
		CardCount: counts[0],
	}

	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for i := 1; i < 7; i++ {
		targetDay := midnight.Add(time.Duration(i*24) * time.Hour)
		dayLabel := ""
		if i == 1 {
			dayLabel = "Tomorrow"
		} else {
			dayLabel = dayNames[targetDay.Weekday()]
		}

		timeline[i] = FlashcardDuePoint{
			Date:      targetDay.Format(dateFormatYYYYMMDD),
			DayLabel:  dayLabel,
			CardCount: counts[i],
		}
	}

	return map[string]interface{}{
		"timeline": timeline,
	}
}
