# Reading Task Estimate: Word-Count Fix

**Date:** 2026-08-08

---

## Problem

The dashboard displayed a stale READING task as **"Reading 605 min | Pages 1-242"**.

Root cause: `queueTaskToScheduledTask` in `app_study.go` used a legacy formula:

```
estimate = (EndPage - StartPage + 1) * 2.5 min/page
```

For a topic spanning pages 1–242 this produced `242 * 2.5 = 605 minutes`, regardless of how many words were actually on those pages.

---

## Why It Was Not the Scheduler

Two separate paths serve tasks to the dashboard:

| Path | Source | Already correct? |
|------|--------|-----------------|
| `BuildTodayPlan` (scheduler) | Queries `topics` table, applies `ResolvePageWindow` | ✅ Yes |
| `aggregateQueueTasks` → `queueTaskToScheduledTask` | Reads `study_queue` rows directly | ❌ Used legacy formula |

The "605 min" task was a persisted `PENDING` row in `study_queue` served via the second path. The scheduler never touched it.

---

## What Was NOT the Fix

**Bookmark extraction and reading session sizing are independent layers.**  
Adding `ResolvePageWindow` inside `EnsurePendingReadingTaskForNotebook` (inside a `withTx` transaction) was attempted but:

1. It caused a **SQLite deadlock** — `GetUserSettings()` called inside an open transaction tried to acquire a second connection.
2. It was **dead code** anyway — the early-return `if count > 0 { return nil }` meant it never ran while the stale task existed.

That change was fully reverted.

---

## Actual Fix

**File:** `internal/app/app_study.go` — `queueTaskToScheduledTask`

For `READING` and `REREAD` tasks with a valid `TopicID`, replaced the legacy formula with:

```go
tokenMap, _ := repo.GetTokensPerPageMap(task.TopicID, task.StartPage, task.EndPage)
for _, w := range tokenMap { totalWords += w }

if totalWords > 0 {
    estimateMinutes = ceil(totalWords / 200 WPM)  // same engine as scheduler
} else {
    estimateMinutes = ceil(pageCount * MinutesPerPage)  // fallback
}
```

- `repo` is passed into `aggregateQueueTasks` (was a pure function before).
- Fallback to `MinutesPerPage` when no chunks have been ingested yet (safe for new notebooks).
- Safety floor: `estimateMinutes >= pageCount` (at least 1 min/page).

---

## Regression Guard

A `WARN` log is emitted every time the estimate is computed:

```
[READING_ESTIMATE] taskID=... topicID=... pages=1-242 word_count=87450 estimate_minutes=437 source=word_count
```

| `source` value | Meaning |
|----------------|---------|
| `word_count` | Chunks present, estimate is accurate |
| `no_chunks_yet` | Chunks not ingested yet — estimate is approximate |

If `word_count=0` or `source=legacy_page_count_fallback` appears on a fully-ingested notebook, something broke in `GetTokensPerPageMap`.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/app/app_study.go` | `queueTaskToScheduledTask` uses `GetTokensPerPageMap`; `aggregateQueueTasks` receives `repo *db.Repository`; `[READING_ESTIMATE]` log added |
| `internal/app/app_test.go` | Updated 3 `aggregateQueueTasks` call sites to pass `nil` repo |

## Files Reverted

| File | Reason |
|------|--------|
| `internal/db/study_queue_repo.go` | `GetUserSettings` inside `withTx` caused SQLite deadlock; also dead code |
