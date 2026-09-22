# Solution: Dynamic Reading Session Splitting ("Complete Here")

## Overview
Added the ability to complete an active reading session prematurely at the current page ("Complete Here") to solve cognitive fatigue and friction when studying dense technical textbooks (e.g., DDIA, Deep Learning). 

Following the **Ponytail** principle of absolute minimal implementation:
1. **Reused Existing State Machine**: 0 new tables, 0 schema migrations, and 0 new endpoints.
2. **Pedagogical Integrity**: Scopes quiz generation strictly to read chunks ($[start\_page \dots splitPage]$).
3. **Queue Continuity**: Advances topic cursor to $splitPage$ and leverages existing scheduler replenishment to seed the remainder ($[splitPage + 1 \dots]$) as the next continuous task.

---

## Changes Made

### 1. Backend & Repository
- **[internal/db/study_queue_repo.go](../../internal/db/study_queue_repo.go)**:
  - Added `UpdateTaskEndPage(taskID string, endPage int)` to truncate the `end_page` boundary on active queue tasks.
- **[internal/app/app_study_reading.go](../../internal/app/app_study_reading.go)**:
  - Extended `CompleteReading(taskID string, splitPage int)`:
    - If `splitPage > 0` ($start\_page \le splitPage < end\_page$), truncates `task.EndPage` and saves to SQLite before quiz generation.
    - Binds quiz generation chunk retrieval to the truncated page range.
    - Synchronizes topic cursor and seeds the remainder via standard queue replenishment.
- **[doc/DATA_API.md](../../doc/DATA_API.md)**:
  - Documented `splitPage` parameter and behavior for `CompleteReading`.

### 2. Frontend (Vue 3)
- **[frontend/src/services/appApi.js](../../frontend/src/services/appApi.js)**:
  - Updated `completeReading(taskID, splitPage = null)` to forward the optional `splitPage` integer argument to the Go backend bridge.
- **[frontend/src/pages/Reader.vue](../../frontend/src/pages/Reader.vue)**:
  - Added `canSplitHere` computed property (`currentPage >= minPage && currentPage < maxPage`).
  - Added **"Complete Here (Page X)"** action in the completion split-dropdown with descriptive helper copy.
  - Added `onCompleteHereClick` handler to trigger `completeSession(false, currentPage)`.

---

## Verification
- `go test -v ./internal/app -run TestCompleteReading_SplitPage` $\to$ **PASS** (verified task boundary truncation, completion status, and payload scope)
- `go test -short ./internal/...` $\to$ **PASS** (all internal packages compile and pass)
