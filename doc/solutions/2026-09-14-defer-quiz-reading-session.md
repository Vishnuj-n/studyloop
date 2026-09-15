# Deferred Quiz & Continuous In-Book Reading Flow

**Date:** 2026-09-14  
**Module:** Reader (`frontend/src/pages/Reader.vue`), Queue & Study (`internal/app/app_study_reading.go`, `internal/db/study_queue_queries.go`)

---

## Problem & Context

1. **Interrupted Flow State**: 
   Standard queue completion in `Reader.vue` forces an immediate transition to the generated `QUIZ` view after every reading session. When users clicked "Complete & Defer Quiz", the reader previously routed to `/dashboard`, which broke continuous focused reading for multi-chapter books.
2. **Multi-Book Interleaving vs Continuous Reading**:
   The study queue uses round-robin multi-book balancing and ranks `QUIZ` tasks above `READING` tasks. When a user is in the zone reading Book A, they want to read consecutive chapters of Book A without being interrupted by other books or forced quizzes.
3. **Architecture Invariants**:
   - UI holds only ephemeral state; all state mutations live in SQLite backend.
   - Preserves queue multi-book round-robin balancing for when studying from the dashboard, while supporting focused in-book sprints directly in the reader.

---

## Key Solutions Implemented

### 1. Discrete Queue Tasks for Deferred Sessions
- Each deferred session generates its `QUIZ` task and enqueues it as `PENDING` into `study_queue` in SQLite.
- Fits the **Deterministic Queue Progression** invariant. The queue accumulates quizzes naturally for review later.

### 2. Automatic In-Book Next-Task Seeding (`internal/app/app_study_reading.go`)
- On `CompleteReading(taskID)`:
  - Seeds the next reading range for the current notebook via `EnsurePendingReadingTaskForNotebook(notebookID, targetWords)`.
  - Fetches the next reading task with `repo.GetPendingReadingTaskForNotebook(notebookID)` and returns `next_reading_task` in the RPC response:
    ```json
    {
      "ok": true,
      "quiz_task_id": "...",
      "next_reading_task": {
        "id": "task-read-...",
        "notebook_id": "...",
        "topic_id": "...",
        "start_page": 11,
        "end_page": 20
      }
    }
    ```

### 3. Seamless In-Reader Progression (`frontend/src/pages/Reader.vue`)
- In `completeSession(deferQuiz = true)`:
  - If `next_reading_task` is returned:
    - Shows notice: *"Quiz saved to queue! Continuing with next reading session..."*
    - Updates route query and triggers `resolveTaskContext()` for the next section.
    - User stays inside the reader without page reload or dashboard kickout.
  - If all reading for the book is complete:
    - Shows notice and navigates to `/dashboard`.

---

## Architecture & Invariants Preserved

- **Zero Schema Mutations**: Uses existing `study_queue` and task lifecycle events (`COMPLETE_READING`).
- **Thin Frontend Compliance**: State transitions and next-task seeding are handled transactionally in the Go backend.
- **Queue Interleaving Untouched**: Global multi-book queue balance remains intact for normal dashboard study.

---

## Verification

- **Go Backend Tests**: `go test -short ./internal/...` → **PASS** (100% across `internal/app`, `internal/db`, `internal/study`).
- **Frontend Integration Tests**: `npm test` → **PASS** (14 test files, 49 tests passed).

