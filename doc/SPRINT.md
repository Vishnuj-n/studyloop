# Sprint Roadmap — AI Tutor

**Status:** Active roadmap
**Last Updated:** 2026-08-08
**Architecture:** SQLite-backed deterministic queue (NOT autonomous orchestration)

---

## Active Roadmap

### Sprint 10: RAG Setup, Asset Management & Environment Verification [DONE]
- [x] Dynamic CGO/vec0 verification at startup
- [x] Asset downloader script for missing embedding models

### Sprint 11: 2-Strike Socratic Rescue Pipeline [DONE]
- [x] Track consecutive quiz failures via `reread_attempts`
- [x] On 2nd failure: delete FSRS cards, insert `SOCRATIC_REMEDIAL` task
- [x] Queue interleaving at priority tier 6
- [x] Dual-lane rescue UI (in-app + external prompt)
- [x] Dev bypass endpoint + tests

### Sprint 12: Cloud Dashboard & Sync [DONE]
- [x] Schema audit: UUID keys, `is_locked` column on `user_accounts`
- [x] Dirty flags + `updated_at` on core tables
- [x] Delta sync + handover payload endpoint
- [x] Student account creation and cloud sync
- [x] Classroom lock/unlock and student removal
- [x] PDF page-range ingestion from cloud dashboard
- [x] Ingestion notification banner + one-click ingestion workflow
- [x] Multi-source data fetching for classroom dashboard and assignments
- [x] Cloud assignment metadata integration
- [x] Automatic classroom study profile isolation and lifecycle management
- [x] Cloud dashboard refactored into multi-page application
- [x] Design system + theme support for cloud dashboard
- [x] `cmd/cloud-server` — Go proxy server for Supabase service-role operations

### Sprint 13: User Asset Provisioning [DONE]
- [x] Detect missing RAG assets, download from GitHub Releases, progress UI, hash verification

### Sprint 14: User-Configurable Remediation Strategy [DONE]
- [x] `default_remedial_strategy` in `user_settings` (CLASSIC/FAST)
- [x] Quiz logic branching based on strategy
- [x] Frontend settings toggle
- [x] Tests

### Sprint 15: Simplified FSRS Calibration & Enhanced Features [DONE]
- [x] Clean Review state initialization, day-based offsets
- [x] Cloud sync with stable identifiers
- [x] Streak tracking + calendar widget
- [x] UI enhancements (flip-back, sidebar, scroll progress)
- [x] Delta sync + settings

### Sprint 16: Milestone Exams, AI Cleanup & Dashboard Metrics [DONE]
- [x] `MILESTONE_EXAM` task type — aggregate exam every 10 quizzes per notebook
- [x] Milestone exam payload with correctness flags + deduplication
- [x] `CleanTopicTitle` utility — formats raw topic IDs to human-readable titles
- [x] AI cleanup fallback — graceful degradation when LLM fails during chapter cleanup
- [x] Dashboard `sessions_per_day` computed metric for pacing display
- [x] `GetAllTopics` enhanced with page ranges + clean titles
- [x] `GetQuestionsForQuizAttempts` repo method for milestone exam question retrieval
- [x] Flashcard generation retry logic with topic/page validation
- [x] FLASHCARD_SYNC → FLASHCARD_GENERATE rename across codebase

---

## Archive (Sprints 1-9)

<details>
<summary>Click to expand</summary>

### Sprint 1: Queue Foundation [DONE]
Schema: `study_queue` table. Deliverables: table + indexes, queue repo, task lifecycle, Wails bindings.

### Sprint 2: Reading Flow [DONE]
Schema: `reading_progress`. Deliverables: page-range locking, progress persistence, quiz task auto-insertion.

### Sprint 3: Sync Quiz Generation [DONE]
Deliverables: sync endpoint, quiz payload, UI with loading state, scoring, conditional reread.

### Sprint 4: Reread Remediation [DONE]
Schema: `reread_attempts`. Deliverables: reread counter, max attempt enforcement, Reader reuse.

### Sprint 5: Core Foundation [DONE]
Deliverables: SQL sorting, bootstrap `boot.go`, settings table, flattened schema.

### Sprint 6: Reading + Quiz Pipelines [DONE]
Deliverables: page validation, context-locked quiz, deadline config, dashboard telemetry.

### Sprint 7: Memory Engine [DONE]
Deliverables: FSRS type-safe integration, consistent `GetTodayPlan`.

### Sprint 8: Study Groups [DONE]
Deliverables: `study_groups` schema, feasibility verification, priority multiplier, capacity UI.

### Sprint 9: Socratic Tutor + Examiner Gate [DONE]
Deliverables: backend Socratic endpoint, milestone gate.
</details>

---

## Architecture Foundation

**This app is: A Persistent Guided Study Queue**

NOT: autonomous AI tutor, mission engine, hidden orchestrator, proactive scheduler.

**Core Principle:** Learning systems are **Data, not Engines**. They create queue tasks, not orchestration. SQLite = source of truth.

## Queue Model

All progression: `study_queue`

**Task Lifecycle:** `PENDING → ACTIVE → COMPLETED` (or `FAILED`/`SKIPPED`)

**Priority Order:**
1. `FLASHCARD_GENERATE` — cloud sync recovery
2. `SOCRATIC_REMEDIAL` — rescue lane
3. `FLASHCARD_REVIEW` — spaced repetition
4. `REREAD` — remediation
5. `QUIZ` — assessment
6. `MILESTONE_EXAM` — cumulative mastery exam (after 10 quizzes)
7. `READING` — content
8. `EXAMINER` — mastery verification

**Ordering:** task type priority → notebook priority → task priority → creation time (FIFO).

## Terminology

| Use This | NOT This |
|----------|----------|
| `study_queue` | DailyAgenda |
| Task type | Mission type |
| Queue ordering | Scheduling engine |
| Deterministic | AI-driven |
| FSRS algorithm | FSRS orchestrator |

## Definition of Done

1. Schema migrations applied
2. Repository layer with tests
3. Wails bindings exposed
4. Frontend wired (if applicable)
5. `go test ./...` passes
6. `wails dev` smoke test passes
7. No deprecated orchestration terminology
