# Solution: Milestone Exam Telemetry Refactor & Candidate Validation

## Overview
Refactored milestone exam telemetry and quiz attempt eligibility checks to eliminate fragile client-side tracking, enforce clean candidate validation, and correctly isolate notebook queries.

1. **Durable Backend Telemetry**: Telemetry for `milestone_exam_complete` moved into Go `TransitionTask(EventCompleteMilestoneExam)`. Persists directly into SQLite `analytics_events` outbox upon task completion rather than relying on frontend lifecycle hooks.
2. **Corrupt Quiz Filtering Invariant**: Enforces at least 3 valid quizzes after corrupt/unparseable attempts are discarded (`len(quizzes) < 3 || representativeAttemptID == ""`).
3. **Notebook & Topic Query Scoping**: `GetUnexaminedPassedQuizAttemptsByTopic` now scopes directly by `(notebookID, topicID)`, validates `notebook_topics` membership, and propagates lookup and `HasMilestoneExamForAttemptID` errors cleanly.
4. **Documentation Alignment**: SSoT docs updated to reflect the chapter completion $\ge 3$ unexamined passed quiz requirement and obsolete 10th-quiz notebook triggers removed.

---

## Changes Made

### 1. Backend Telemetry & Transition
- **[internal/study/queue_transition.go](../../internal/study/queue_transition.go)**:
  - Inside `EventCompleteMilestoneExam`, records `milestone_exam_complete` event directly to `s.repo.TrackAnalyticsEvent` with notebook file hash and metadata before returning.
- **[frontend/src/pages/Quiz.vue](../../frontend/src/pages/Quiz.vue)**:
  - Removed client-side `trackAnalyticsEvent('milestone_exam_complete', ...)` call in `submitMilestone()`. Kept UI limited to task completion execution and navigation.

### 2. Candidate Filtering & Repository
- **[internal/app/app_study_cards.go](../../internal/app/app_study_cards.go)**:
  - Passes `task.NotebookID` to `repo.GetUnexaminedPassedQuizAttemptsByTopic`.
  - In `insertMilestoneForAttempts`, checks `len(quizzes) < 3 || representativeAttemptID == ""` so no milestone exam is inserted if corrupt attempts bring valid count below 3.
- **[internal/db/study_queue_queries.go](../../internal/db/study_queue_queries.go)**:
  - `GetUnexaminedPassedQuizAttemptsByTopic(notebookID, topicID string)` queries `study_queue` constrained by both `notebook_id` and `topic_id`.
  - Validates `notebook_topics` relationship and returns explicit error if missing.
  - Propagates errors from `HasMilestoneExamForAttemptID` instead of treating errors as eligible.

### 3. Tests & Documentation
- **[internal/app/quiz_flashcard_test.go](../../internal/app/quiz_flashcard_test.go)**:
  - Added regression scenario inside `TestMilestoneExamTriggers` verifying 3 candidates with 1 corrupt attempt does not produce a 2-quiz milestone exam.
- **[internal/db/study_queue_repo_test.go](../../internal/db/study_queue_repo_test.go)**:
  - Updated test callers to pass `(notebookID, topicID)`.
- **[doc/DATA_API.md](../../doc/DATA_API.md)** & **[doc/APP_FLOW.md](../../doc/APP_FLOW.md)**:
  - Updated trigger definition and removed obsolete trigger documentation.

---

## Verification
- `go test -short ./internal/...` -> **PASS**
- `go test -v ./internal/app -run "TestMilestoneExamTriggers"` -> **PASS**
- `go test -v ./internal/db -run "TestGetUnexaminedPassedQuizAttempts"` -> **PASS**
