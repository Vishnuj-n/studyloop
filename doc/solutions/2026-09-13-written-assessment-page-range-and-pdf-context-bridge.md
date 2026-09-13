# Solution: Written Assessment Page Range & PDF Reader Context Bridge

## Overview
This document records the root cause analysis, architectural improvements, and verification for two interrelated study flow bugs:
1. **Written Assessment Page Range Reset**: Examiner questions were generated for pages 25–30 instead of the target page window passed in the URL (e.g., pages 80–90).
2. **Reading Task Page Cursor Stale Leakage**: Launching a reading task (e.g. Pages 68–99) opened the PDF at Page 82 because a shared topic cursor left over from earlier reading/indexing was loaded instead of starting at Page 68.

---

## 1. Written Assessment Page Range Fix

### Root Cause
In [`frontend/src/pages/WrittenAssessment.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/WrittenAssessment.vue#L189-L242), a Vue watcher on `selectedNotebookID` executed during component mounting whenever `selectedNotebookID` was set from router query parameters. The watcher unconditionally reset `startPage` and `endPage` to the notebook's default bounds (pages 25–30), overwriting explicit query parameters (`startPage` / `endPage`) passed from the Reader or Quiz router navigation.

Additionally, `Dashboard.vue` passed `notebookId` (lowercase `d`) while `Quiz.vue` passed `notebookID` (capital `ID`).

### Solution
1. **Initial Mount Guard**: Added an `isInitialLoad` flag to `WrittenAssessment.vue` to prevent `watch(selectedNotebookID)` from overwriting explicit route query parameters during initial load.
2. **Query Parameter Normalization**: Supported both `notebookID` and `notebookId` query keys.
3. **Auto-Generation Trigger**: Updated `Dashboard.vue` task routing for `examiner`/`written` action items to pass `query.autoGenerate = 'true'`.

---

## 2. Reading Task Page Cursor Isolation & PDF Context Bridge

### Root Cause
Previously, `GetReadingTask` retrieved the reading position from a shared `topics.current_page_cursor` column:

```sql
SELECT
    sq.id,
    sq.notebook_id,
    COALESCE(sq.topic_id, ''),
    COALESCE(sq.start_page, 0),
    COALESCE(sq.end_page, 0),
    COALESCE(t.current_page_cursor, COALESCE(sq.start_page, 0)),
    COALESCE(nb.file_hash, '')
FROM study_queue sq
LEFT JOIN topics t ON t.id = sq.topic_id
WHERE sq.id = ?
```

Because `topics.current_page_cursor` was shared across all tasks for a topic, any progress or indexing on that topic left a stale cursor (e.g. Page 82). When a user launched a task bounded to Pages 68–99, `GetReadingTask` returned `CurrentPage = 82` because `68 <= 82 <= 99`.

### Architectural Improvements

#### A. Per-Task Page Cursor (`study_queue.current_page`)
1. **Database Schema (`internal/db/schema.go`)**:
   Added `current_page INTEGER` column to `study_queue` and added a clean schema migration statement in `alterStatements`:
   ```go
   {"study_queue", "current_page", "ALTER TABLE study_queue ADD COLUMN current_page INTEGER"}
   ```

2. **Task Query (`internal/db/study_queue_queries.go`)**:
   Updated `GetReadingTask` to select the task-specific cursor `sq.current_page`:
   ```sql
   COALESCE(sq.current_page, COALESCE(sq.start_page, 0))
   ```

3. **Task Activation & Progress Updates (`internal/db/study_queue_repo.go`)**:
   - `ActivateTaskTx`: Initializes `study_queue.current_page = sq.start_page` upon task activation.
   - `PersistReadingProgress`: Updates `study_queue.current_page = finalPage` as the user reads, while maintaining `topics.current_page_cursor` for topic-level analytics.

#### B. PDF Reader Context Bridge (`useReaderBase.js`)
To provide cognitive continuity between reading sessions without semantic cuts, the Reader component implements a **PDF Context Bridge**:

```javascript
// Validate bounds have valid values
const validStart = Number(navigationBounds.start_page) || 1
const validEnd = Number(navigationBounds.end_page) || validStart
const validCurrent = Number(navigationBounds.current_page) || validStart

// PDF Context Bridge: allow scrolling back 1 page (start_page - 1) for visual continuity
const isPdfFormat = isPdf.value || fileType.value === 'pdf'
const minPage = validStart > 1 && isPdfFormat ? validStart - 1 : validStart

navigationMinPage.value = minPage
navigationMaxPage.value = validEnd

// Context bridge: if launching a new task at start_page, initialize view at start_page - 1 for PDFs
const displayCurrent =
  validCurrent === validStart && validStart > 1 && isPdfFormat
    ? validStart - 1
    : validCurrent

currentPage.value = Math.min(Math.max(displayCurrent, minPage), validEnd)
```

#### Edge-Case Safety Rules
- **Page 1 Guard**: Clamped to Page 1 when `start_page == 1` (`Math.max(1, validStart - 1)`).
- **Non-PDF Formats**: Disabled for Markdown, Audio, and YouTube formats.
- **Resumed Sessions**: When resuming an in-progress task (e.g. user left off at Page 75), `validCurrent !== validStart`, so the reader resumes directly at Page 75 without context bridging.

---

## 3. Verification

1. **Go Unit Test**:
   Created `TestActivatePendingReadingTaskAlignsTopicCursor` in [`internal/db/study_queue_repo_test.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/db/study_queue_repo_test.go#L438-L473) to verify that activating a pending reading task aligns both task and topic cursors to `start_page`.
   - `go test -v -run TestActivatePendingReadingTaskAlignsTopicCursor ./internal/db` $\rightarrow$ **PASS (0.05s)**.

2. **Backend Short Test Suite**:
   Ran `go test -short ./internal/...` $\rightarrow$ **12/12 packages passed**.

3. **Frontend Vitest Suite**:
   Ran `cd frontend; npx vitest run` $\rightarrow$ **14/14 test files passed, 49/49 unit tests passed**.
