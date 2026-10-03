# Solution Architecture: Structured Study Notes & Knowledge Base

## Overview
This solution integrates a persistent **Study Notes Knowledge Base** into Studyloop. Study notes are generated on-demand or automatically post-reading/compression, organized under a dedicated **Notes** sidebar tab (`/notes`), viewable in a slide-over drawer while reading, and surfaced prior to Milestone Exams using an active-retrieval cognitive review modal.

---

## Architecture & Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ 1. READING SESSION & EXTRACTION                                                             │
│    [ Complete Reading Session: Pages X–Y ]                                                 │
│                     │                                                                       │
│                     ▼                                                                       │
│    [ LLMLingua-2 Compression ] ──► Chunks compressed and stored in `chunks.compressed_text` │
│                     │                                                                       │
│                     ▼ (If auto_generate_study_notes == true)                                │
│    [ Session-Scoped LLM Call ] ──► Clean, readable 100–150 word summary for Pages X–Y       │
│                     │                                                                       │
│                     ▼                                                                       │
│    [ Incremental Merge ] ──► Appended under `### Pages X–Y` in `topic_study_notes`          │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ 2. DEDICATED KNOWLEDGE BASE (`/notes`)                                                      │
│    ┌──────────────────────────────┬──────────────────────────────────────────────────────┐  │
│    │ [Notebook Dropdown ▾]        │ Chapter: Backpropagation                             │  │
│    │ 🔍 Search topics or notes... │ Last reviewed: 2 days ago                            │  │
│    ├──────────────────────────────┼──────────────────────────────────────────────────────┤  │
│    │ • Chapter 1: Foundations     │ ### Pages 1–15: Core Mechanics                       │  │
│    │ ► Chapter 2: Backprop (●)    │ Calculates gradient of loss function using chain... │  │
│    │ • Chapter 3: Optimizers      │                                                      │  │
│    │                              │ ### Pages 16–30: Vanishing Gradients                 │  │
│    │                              │ ...                                                  │  │
│    │                              │ [ ✏️ Edit Note ]  [ ⚡ Regenerate ]  [ ✓ Reviewed ]  │  │
│    └──────────────────────────────┴──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ 3. PRE-MILESTONE EXAM RETRIEVAL MODAL                                                       │
│    [ Launch Milestone Exam Task ]                                                           │
│                 │                                                                           │
│                 ▼                                                                           │
│    Step 1: Active Recall Challenge ──► "Explain the core idea of [Topic] to yourself"       │
│    Step 2: Mental Model Check     ──► Click [Reveal Note] to verify understanding           │
│    Step 3: Exam Launch            ──► Click [Ready for Exam →] (logs review timestamp)      │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Key Components

### 1. Database & Schema
- **`topic_study_notes` Table**:
  - `id` (TEXT PRIMARY KEY)
  - `topic_id` (TEXT NOT NULL UNIQUE, FK → `topics(id)` ON DELETE CASCADE)
  - `notebook_id` (TEXT NOT NULL, FK → `notebooks(id)` ON DELETE CASCADE)
  - `content` (TEXT NOT NULL DEFAULT '')
  - `last_reviewed_at` (INTEGER DEFAULT 0)
  - `created_at`, `updated_at` (TIMESTAMP)
  - **Indexes**: `idx_topic_study_notes_topic`, `idx_topic_study_notes_notebook`
- **`user_settings.auto_generate_study_notes`**: Boolean flag persisted in SQLite singleton, migrated cleanly via `internal/db/migrations.go`.

### 2. Session-Scoped Generation Engine
- **Prioritizes Pre-Compressed Text**: Uses `chunk.compressed_text` (from LLMLingua-2) first, fitting 2–3x more textbook content into the LLM prompt within context limits while generating crisp, human-readable notes.
- **Section Accumulation (`mergeSectionIntoNote`)**: Scopes generation to `start_page` and `end_page`. When multiple reading sessions are completed for a chapter, each session's summary is merged into the topic note under `### Pages X–Y` headers without overwriting prior session notes or custom user edits.
- **Whole-Chapter On-Demand Generation**: Clicking `[Generate Note]` or `[Regenerate]` from `/notes` synthesizes the entire chapter across all available chunks.

### 3. User Interfaces
- **Notes Page (`frontend/src/pages/Notes.vue`)**: Full 2-column Knowledge Base with notebook selector, live search across topics and note text, review status tags, dual Markdown view / raw editor, and AI regeneration.
- **Reader Drawer (`frontend/src/pages/Reader.vue`)**: Slide-out drawer in the top navigation bar to quickly reference and edit study notes without losing reading position.
- **Pre-Exam Review Modal (`frontend/src/components/MilestoneNoteReviewModal.vue`)**: Cognitive retrieval-first modal triggered before Milestone Exam questions begin in `Quiz.vue`.
- **Settings Toggle (`frontend/src/components/SettingsStudyBudget.vue`)**: Opt-in toggle with credit/cost warning dialog.

---

## Core Component Reference

- **Database Models & Repositories**: [`internal/db/schema.go`](../../internal/db/schema.go), [`internal/db/topics_repo.go`](../../internal/db/topics_repo.go), [`internal/db/settings_repo.go`](../../internal/db/settings_repo.go), [`doc/SCHEMA.md`](../SCHEMA.md)
- **Backend Service**: [`internal/study/study_notes_service.go`](../../internal/study/study_notes_service.go), [`internal/study/study_notes_test.go`](../../internal/study/study_notes_test.go)
- **Wails RPC Bridge**: [`internal/app/app_study.go`](../../internal/app/app_study.go), [`internal/app/app_study_reading.go`](../../internal/app/app_study_reading.go), [`frontend/src/services/appApi.js`](../../frontend/src/services/appApi.js)
- **Frontend Pages & Modals**: [`frontend/src/pages/Notes.vue`](../../frontend/src/pages/Notes.vue), [`frontend/src/pages/Reader.vue`](../../frontend/src/pages/Reader.vue), [`frontend/src/components/MilestoneNoteReviewModal.vue`](../../frontend/src/components/MilestoneNoteReviewModal.vue), [`frontend/src/components/SettingsStudyBudget.vue`](../../frontend/src/components/SettingsStudyBudget.vue)
