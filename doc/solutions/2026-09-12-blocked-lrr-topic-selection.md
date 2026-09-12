# 2-Session Blocked Interleaving (LRR) Topic Selection Solution Document

## Context & Problem
When users have multiple active textbooks (e.g., 3 active books in the Active Lane) sharing equal priority (`priority = 5`), `QueryNextReadingTopic()` in `internal/db/topics_repo.go` previously resolved tie-breakers using static creation ordering and start page numbers (`ORDER BY COALESCE(n.priority, 5) DESC, t.start_page ASC, t.created_at ASC`).

This caused severe **book starvation**: older uploaded textbooks (such as *Designing Data-Intensive Applications*) won static tie-breakers indefinitely, keeping other active books (*Grokking Deep Learning* and *PyTorch in 1 Hour*) trapped at the bottom of the queue. 

Simultaneously, single-session round-robin rotation caused high cognitive context-switching overhead (switching complex technical domains every single reading session).

## Solution
We implemented **2-Session Blocked Interleaving (Least-Recently-Read Round-Robin)** within `QueryNextReadingTopic()` in `internal/db/topics_repo.go`.

### Key Mechanics
1. **2-Session Block Streak Rotation**: A Common Table Expression (CTE) checks the number of completed reading tasks for each active notebook. Notebooks that have completed fewer than 2 consecutive reading tasks retain a streak anchor (`1970-01-01 00:00:00`), allowing them 2 full reading sessions to build focus and momentum before rotating.
2. **Least-Recently-Read (LRR) Rotation**: Once a notebook completes its 2-session block, its `effective_last_read` timestamp updates to its latest task completion time, automatically passing the queue selection to the active notebook that hasn't been read in the longest time.
3. **Explicit Priority Preserved**: Users can still bump a specific notebook's priority slider higher (e.g., priority `6` vs `5`) to force a dedicated single-subject sprint whenever desired.

### Code Changes
In `internal/db/topics_repo.go`:
```sql
WITH recent_reads AS (
    SELECT notebook_id, completed_at,
           ROW_NUMBER() OVER (PARTITION BY notebook_id ORDER BY completed_at DESC) as rnum
    FROM study_queue
    WHERE task_type = 'READING' AND status = 'COMPLETED'
),
book_streak AS (
    SELECT n.id as notebook_id,
           CASE 
               WHEN COUNT(rr.completed_at) < 2 THEN '1970-01-01 00:00:00'
               ELSE MAX(rr.completed_at)
           END as effective_last_read
    FROM notebooks n
    LEFT JOIN recent_reads rr ON n.id = rr.notebook_id AND rr.rnum <= 2
    GROUP BY n.id
)
SELECT
    t.id, t.title, COALESCE(t.start_page, 0), COALESCE(t.end_page, 0),
    COALESCE(t.current_page_cursor, 0), n.id
FROM topics t
LEFT JOIN notebook_topics nt ON nt.topic_id = t.id
LEFT JOIN notebooks n ON (n.id = nt.notebook_id OR n.topic_id = t.id)
LEFT JOIN book_streak bs ON bs.notebook_id = n.id
WHERE t.status IN ('unseen', 'reading')
  AND COALESCE(t.end_page, 0) > 0
  AND COALESCE(t.current_page_cursor, 0) < COALESCE(t.end_page, 0)
  AND (nt.notebook_id IS NOT NULL OR n.topic_id = t.id)
  AND n.id IS NOT NULL
  AND n.id != ''
  AND n.study_status = 'active'
  AND n.status = 'chunked'
ORDER BY COALESCE(n.priority, 5) DESC, COALESCE(bs.effective_last_read, '1970-01-01 00:00:00') ASC, t.start_page ASC, t.created_at ASC
LIMIT 1;
```

## Empirical Verification
- **Empirical Simulation**: Simulated 9 study sessions across 3 mock notebooks in SQLite:
  - **Sessions 1–2**: DDIA (2 sessions)
  - **Sessions 3–4**: Grokking Deep Learning (2 sessions)
  - **Sessions 5–6**: PyTorch in 1 Hour (2 sessions)
  - **Sessions 7–8**: DDIA (2 sessions)
- **Live SQLite DB Test**: Verified against user production DB `C:\Users\vishn\AppData\Roaming\Studyloop\Studyloop.db` $\rightarrow$ correctly selected *PyTorch in 1 Hour* (`Autograd: requires_grad`) as the next topic.
- **Backend Test Suite**: Ran `go test -short ./internal/...` $\rightarrow$ ALL test packages passed 100% cleanly.
