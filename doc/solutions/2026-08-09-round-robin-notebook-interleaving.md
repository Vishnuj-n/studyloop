# Round-Robin Notebook Interleaving Solution Document

## Context & Problem
When users had multiple active textbooks (e.g. 3 active books in the Active Lane) with equal priority (`priority = 5`), the study queue consistently served consecutive reading tasks for the same textbook.

Investigation revealed that the queue ordering query in `internal/db/study_queue_repo.go` (`getPendingTasksWithProfile`, `getPendingTasksNoProfile`, `getNextTaskWithProfile`, `getNextTaskNoProfile`) was using a static alphabetical tie-breaker (`notebook_title ASC`) when comparing equal-priority notebooks:

```sql
ORDER BY COALESCE(notebook_priority, 5) DESC, notebook_title ASC, id ASC
```

Because every completed reading task for Book A immediately created a new pending reading task for Book A's next chapter, Book A's title repeatedly won the static tie-breaker against Book B and Book C, keeping Book A locked at the top of the queue indefinitely.

## Solution
We updated the queue query tie-breaking rule across all pending task queries in `internal/db/study_queue_repo.go`. Instead of sorting equal-priority notebooks alphabetically, we sort them by their least recent task completion timestamp:

```sql
(SELECT COALESCE(MAX(sq2.completed_at), '') FROM study_queue sq2 WHERE sq2.notebook_id = sq.notebook_id AND sq2.status = 'COMPLETED') ASC
```

### Key Changes
1. **`internal/db/study_queue_repo.go`**:
   - Updated `getPendingTasksWithProfile` and `getPendingTasksNoProfile` to include the `last_completed_at` subquery in their `ORDER BY` clause.
   - Updated `getNextTaskWithProfile` and `getNextTaskNoProfile` to include the `last_completed_at` subquery in their `ORDER BY` clause.
   - Preserves explicit priority differences (e.g., Priority `10` still beats Priority `5`), but ensures equal-priority active textbooks interleave in a fair round-robin rotation.

2. **`internal/db/study_queue_repo_test.go`**:
   - Added unit test `TestStudyQueueRoundRobinInterleaving` to verify that completing a task for Notebook A causes `GetNextTask` to immediately rotate to Notebook B when both share equal notebook priority (`5`).

## Verification
- Ran `go test -v ./internal/db -run TestStudyQueueRoundRobinInterleaving` $\rightarrow$ `PASS`.
- Ran `go test ./...` $\rightarrow$ All backend test packages passed cleanly.
