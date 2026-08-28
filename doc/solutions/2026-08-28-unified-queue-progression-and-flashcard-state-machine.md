# Solution: Unified Study Queue Progression & Flashcard State Machine

**Date:** 2026-08-28  
**Module:** Backend Study Service (`internal/study/queue_transition.go`), App Handlers (`internal/app/app_study_cards.go`, `internal/app/app_study.go`), DB Repository (`internal/db/study_queue_repo.go`)

---

## 1. Problem & Root Cause

1. **Dashboard Empty State & Stalled Queue Progression**:
   - Following the refactor in commit `a95b160` enforcing read purity on `GetTodayPlan()`, `EnsurePendingReadingTasksForActiveNotebooks` was removed from the daily plan query.
   - When a user finished a reading slice and passed the subsequent Quiz, the state machine marked the quiz `COMPLETED` and generated FSRS flashcards, but **failed to explicitly enqueue the next reading slice** (`READING` task for remaining chapter pages or the next chapter) for the active notebook.
   - This left `study_queue` with 0 active/pending tasks for that notebook, causing the Dashboard to display an empty state.

2. **Flashcard Visibility Misconception**:
   - Flashcards were successfully generated in `fsrs_cards` upon passing the quiz, but were not immediately displayed on today's review dashboard.
   - This is by design under the FSRS spaced repetition model (newly generated cards have `due_at` set to tomorrow, $+1$ day). Because `due_at > now`, `QueryDueReviewCards` returned `0` for today.

---

## 2. Architectural Solution

To avoid ad-hoc side-effects during getter queries while maintaining strict state machine unification, all task transitions were consolidated under **`TransitionTask`**:

### A. Unified State Switchboard (`internal/study/queue_transition.go`)
- Updated `TransitionTask` under `EventCompleteFlashcards`:
  ```go
  case EventCompleteFlashcards:
      if err := s.repo.ResolveFlashcardGenerateTasksForTopic(req.TopicID); err != nil {
          return TransitionResult{}, fmt.Errorf("failed to complete flashcards task: %w", err)
      }

      // ponytail: seed next reading task for active notebook inside unified transition
      settings, err := s.repo.GetUserSettings()
      targetWords := 1500
      if err == nil && settings != nil && settings.TargetSessionWords > 0 {
          targetWords = settings.TargetSessionWords
      }
      if req.NotebookID != "" {
          if ensureErr := s.repo.EnsurePendingReadingTaskForNotebook(req.NotebookID, targetWords); ensureErr != nil {
              utils.Warnf("[QUEUE_TRANSITION] failed to seed next reading task for notebook %s: %v", req.NotebookID, ensureErr)
          }
      }

      return TransitionResult{
          Success:        true,
          TaskID:         req.TaskID,
          CardsScheduled: req.CardCount,
      }, nil
  ```

### B. Clean RPC Routing (`internal/app/app_study_cards.go`)
- Updated `GenerateFlashcardsForQuizTask` to execute `TransitionTask(EventCompleteFlashcards)` directly, creating a deterministic closed loop:
  $$\text{READING} \xrightarrow{\text{CompleteReading}} \text{QUIZ} \xrightarrow{\text{SubmitQuiz (Pass)}} \text{FLASHCARDS} \xrightarrow{\text{TransitionTask}} \text{Next READING}$$

### C. Read Purity Maintained (`internal/app/app_study.go`)
- `GetTodayPlan()` remains 100% pure (read-only SQLite query, zero hidden side-effect insertions).

---

## 3. Verification

1. **Compilation & Type-Checking**:
   ```bash
   go test -run=^$ ./internal/...
   ```
   Passed with 0 errors across all 13 internal packages.

2. **Unit Tests**:
   ```bash
   go test -short ./internal/...
   ```
   Passed 100% in ~1.3 seconds.

3. **Database State**:
   - Verified that `study_queue` in `dev_data/Studyloop.db` properly contains the next unblocked reading slice for *Grokking Machine Learning* (`Pages 30–36`).
