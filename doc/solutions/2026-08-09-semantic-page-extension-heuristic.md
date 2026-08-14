# Solution Document — Semantic Page-Extension Logic for Reading Tasks

**Date:** 2026-08-09  
**Feature:** Deterministic Semantic Page-Extension Heuristic for Reading Task Generation  
**Module:** `internal/db/reader_repo.go`, `internal/db/study_queue_repo.go`

---

## Overview

When generating `READING` tasks in `study_queue`, the system calculates a soft-capped page range target (~5,000 words). Previously, tasks cut off strictly at the last page reaching that word budget. 

This feature adds a **smart semantic page-extension heuristic**:
- After determining the base page cutoff $E$, it inspects up to 3 additional adjacent pages ($E+1 \dots E+3$).
- If page $E+k$ has high semantic similarity ($\ge 0.85$) with page $E+k-1$ and total words $\le 6,500$, the page is absorbed into the reading task window.
- This prevents awkward concept cliffhangers without introducing un-bounded session inflation.

---

## Architecture & Compliance Highlights

1. **100% Deterministic & SQLite-Based**: Computes cosine similarity using existing local embeddings stored in `chunk_vectors` via `sqlite-vec` (`vec_distance_cosine`).
2. **Zero Runtime AI Latency**: Runs entirely as a single synchronous local SQLite transaction. No external LLM calls or background daemons.
3. **Safe Fallback**: If vector embeddings are missing, un-ingested, or `sqlite-vec` is unavailable, `GetPageBoundaryCosineSimilarityTx` returns `0.0` distance / similarity gracefully and falls back to base cutoff page $E$ without throwing errors.

---

## Strict Safety Guards

- **Hard Word Limit Cap**: Total task word count cannot exceed **6,500 words** (vs. 5,000 standard target).
- **Max Page Cap**: Maximum extension of **+3 extra pages**.
- **Topic Boundary Limit**: Never extends beyond `topic.end_page`.

---

## Logging & Audit Trail

When extra pages are absorbed into the task bounds, the system emits an explicit log:
```text
[SEMANTIC_EXTENSION] Absorbed 2 extra page(s) into reading task for topicID="topic-123" (baseEnd=8 -> extendedEnd=10, totalWords=5840)
```

---

## Key Files Modified

- `internal/db/reader_repo.go`: Added `GetPageBoundaryCosineSimilarityTx` to query `vec_distance_cosine` across boundary chunks.
- `internal/db/study_queue_repo.go`: Updated `EnsurePendingReadingTaskForNotebook` with semantic extension loop and logging.
- `internal/db/study_queue_repo_test.go`: Added `TestEnsurePendingReadingTask_SemanticExtensionFallback` for regression testing.
