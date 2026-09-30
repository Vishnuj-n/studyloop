# Book Completion Certificate & Study Mastery Summary

## Overview & Background
In self-directed desktop study apps like Studyloop, completion certificates provide high intrinsic psychological reward, closure, and milestone validation.

This document details the architectural decisions, unlock criteria, backend stat aggregation queries, zero-dependency high-DPI HTML5 Canvas PNG rendering engine, and developer bypass mechanisms.

---

## Core Architecture & Invariants

1. **Single Source of Truth in Go SQLite**:
   - The frontend never computes or guesses completion statistics.
   - All study metrics (quizzes passed, average quiz score, milestone exams cleared, flashcards mastered, study time, and unlock criteria) are queried and aggregated directly from SQLite via `repo.GetNotebookCertificateStats(notebookID)`.
2. **Zero Backend Dependency / No CGO PDF Bloat**:
   - We avoid heavy CGO-based or headless Chrome PDF compilers in the Go backend.
   - High-resolution certificates ($1920 \times 1200$ px at $2\times$ pixel ratio) are rendered dynamically via an HTML5 Canvas 2D engine in Vue 3 for instant PNG download or clipboard copy.
3. **Deterministic Queue & Progress Alignment**:
   - Unlock check: `completion_percent >= 100` (derived from topic page cursors) OR `remaining_tasks == 0` (no pending/active tasks for that notebook).
   - Multi-topic and multi-chapter textbooks linked through `notebook_topics` are aggregated seamlessly using table joins.

---

## Data Model & Aggregation Schema

### Backend Record Contract (`internal/db/notebook_certificate.go`)
```go
type NotebookCertificateStats struct {
	NotebookID         string  `json:"notebook_id"`
	NotebookTitle      string  `json:"notebook_title"`
	FileType           string  `json:"file_type"`
	PageCount          int     `json:"page_count"`
	CompletionPercent  int     `json:"completion_percent"`
	QuizzesPassed      int     `json:"quizzes_passed"`
	AverageQuizScore   float64 `json:"average_quiz_score"`
	MilestonesCleared  int     `json:"milestones_cleared"`
	FlashcardsMastered int     `json:"flashcards_mastered"`
	TotalReviewsCount  int     `json:"total_reviews_count"`
	TotalStudyMinutes  int     `json:"total_study_minutes"`
	CompletedAt        string  `json:"completed_at"`
	UserTitle          string  `json:"user_title"`
	UserLevel          int     `json:"user_level"`
	IsUnlocked         bool    `json:"is_unlocked"`
}
```

### Metrics Computation Logic
- **Quizzes Passed**: `COUNT(qa.id)` where `sq.task_type = 'QUIZ'` and `qa.passed = 1`.
- **Milestones Cleared**: `COUNT(DISTINCT sq.id)` where `sq.task_type = 'MILESTONE_EXAM'` and `sq.status = 'COMPLETED'`.
- **Flashcards Mastered**: `COUNT(DISTINCT fc.id)` for all topics linked to the notebook with `suspended = 0`.
- **Total Study Minutes**: `(reading_sessions * 15m) + (quizzes * 5m) + (reviews * 1m)`.
- **Gamification Link**: Stamps user title (e.g. *The Scholar I*) and numerical account level.

---

## User Experience & UI Triggers

1. **Notebook Card Trigger (`frontend/src/components/NotebookCard.vue`)**:
   - When a textbook reaches `completionPercent >= 100`, a golden **`Certificate`** badge button appears beside the completion progress ring.
   - Clicking the badge emits `@view-certificate` to open `NotebookCertificateModal.vue`.

2. **Celebration Modal (`frontend/src/components/NotebookCertificateModal.vue`)**:
   - Displays a certificate preview card with gold borders, Studyloop watermark, user title, book title, and verified metric badges.
   - Triggers gold confetti and chest opening audio fanfare on fresh unlock.
   - Offers two zero-friction export actions:
     - **Download PNG Certificate**: Generates a high-res $1920 \times 1200$ PNG file directly to local downloads.
     - **Copy to Clipboard**: Copies the PNG image blob to system clipboard for sharing on Discord/X/LinkedIn.

---

## Developer Testing Bypass

To test 100% completion and certificate rendering without manually reading through hundreds of book pages:
1. Open **Settings → Developer Mode** (toggle on).
2. Locate the **Certificate Testing Bypass** card.
3. Select any textbook from the dropdown and click **"Unlock 100% & Certificate"**.
4. This calls `app.DevUnlockNotebookCertificate(notebookID)` which:
   - Advances all topic `current_page_cursor` to `end_page` and marks topics `completed`.
   - Closes remaining active/pending queue tasks for that textbook.
   - Returns the updated stats immediately.
