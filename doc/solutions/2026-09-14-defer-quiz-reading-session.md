# Deferred Quiz Reading Session Flow ("Complete & Defer Quiz")

**Date:** 2026-09-14  
**Module:** Reader (`frontend/src/pages/Reader.vue`), Queue & Study (`internal/app/app_study_reading.go`)

---

## Problem & Context

1. **Interrupted Flow State**: Standard queue completion in `Reader.vue` forces an immediate transition to the generated `QUIZ` view after every reading session. For users engaged in deep, continuous reading (e.g. multi-chapter books), this forces high-frequency context switching.
2. **Architecture & Queue Constraints**: 
   - Holding un-taken quiz states in frontend memory violates the **Thin Frontend** invariant (UI holds only ephemeral state; all mutations live in Go backend).
   - Aggregating $N$ deferred sessions into a single composite mega-quiz at the end introduces heavy LLM token latency, RAG context explosion, and question deduplication overhead.

---

## Key Solutions Implemented

### 1. Discrete Queue Tasks for Deferred Sessions
- Instead of complex multi-session quiz merging or custom DB states, each deferred session enqueues its standard generated `QUIZ` task as `PENDING` into `study_queue` in SQLite (`dev_data/Studyloop.db`).
- Fits the core **Deterministic Queue Progression** invariant. The queue handles Quiz 1, Quiz 2, and Quiz 3 naturally via task priority and FIFO ordering.
- If the application closes or crashes during a reading spree, no quiz state is lost.

### 2. Split Button UI (`Reader.vue`)
- Converted the single `[ Complete Session ]` button into a split button:
  - **Primary Button**: `[ Complete Session ]` (default: completes session and immediately starts quiz).
  - **Chevron Button (`▾`)**: Opens a dropdown menu containing `"Complete & Defer Quiz"`.
- Handled dropdown toggle with auto-closing on outside clicks (`handleDocumentClick`).

### 3. Navigation & Error Resilience
- When `"Complete & Defer Quiz"` is selected:
  - `completeReading(taskID)` generates the quiz and persists it as `PENDING` in SQLite.
  - The frontend redirects to `/dashboard` so the user can continue their study queue without taking the quiz immediately.
  - If quiz generation fails (e.g. LLM timeout), `RevertTaskReservation(taskID)` is invoked in Go backend, returning the reading task to `ACTIVE` state without losing progress, and an inline toast displays the error for retry.

---

## Architecture & Design Rationale

- **Zero DB Schema Mutations**: Reuses existing `study_queue` table and task lifecycle events (`COMPLETE_READING`).
- **Thin Frontend Compliance**: Enqueued quiz items are written directly to SQLite by the Go backend before navigation.
- **Unbroken Reading Flow**: Reader can complete multiple sessions in succession while building a clean backlog of active recall quizzes in the study queue.

---

## Verification

- **Go Backend Test Verification**: `go test -short ./internal/...` → **PASS** (100% success across `internal/app`, `internal/db`, `internal/study`).
