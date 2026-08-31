# Solution: Study Queue Quiz Length, Passing Score, and AI Tutor Style Settings

## Context & Motivation
Students needed fine-grained control over their daily study session intensity (number of questions per topic quiz, passing grade threshold) and how the AI remediation tutor communicates during concept rescue sessions, without overwhelming them with internal pipeline configurations.

## Solution Architecture

### 1. Database Schema & Data Models
- Updated `models.UserSettings` in `internal/models/models.go` with 3 new persisted fields:
  - `QuizQuestionCount` (`int`, default: `8`, range: `3-15`)
  - `QuizPassingScore` (`int`, default: `70`, range: `50-100`)
  - `TutorStyle` (`string`, default: `'socratic'`, options: `'socratic'`, `'direct'`, `'detailed'`)
- Updated `internal/db/schema.go` with default column definitions and non-destructive `alterStatements` for SQLite migration.
- Updated `internal/db/store.go` (`GetUserSettings` and `UpdateUserSettings`) to read, persist, and apply default fallbacks.
- Updated `doc/SCHEMA.md` single source of truth documentation.

### 2. Backend Quiz Generation & Remedial Prompt Pipelines
- **`internal/study/quiz_sync.go`**: Injected `QuizQuestionCount` and `QuizPassingScore` dynamically into `GenerateQuizSync`, ensuring generated quiz tasks and question generation prompts match the user's configured settings rather than hardcoded constants.
- **`internal/app/app_study_cards.go`**: Updated `buildSocraticRemedialPrompt` to dynamically inject the style directive based on `TutorStyle`:
  - `socratic`: Guides via leading questions without revealing answers.
  - `direct`: Explains directly and concisely where errors occurred.
  - `detailed`: Provides step-by-step walkthroughs with intuitive analogies and examples.

### 3. Frontend Settings UI & Logical Reorganization
- **`frontend/src/composables/useSettings.js`**:
  - Added reactive defaults and input validation (bounds 3–15 for quiz questions, 50–100 for passing score).
  - Explicit fallback population in `loadSettings()` so fields never render empty.
- **`frontend/src/components/SettingsStudyBudget.vue`**:
  - Placed **Questions per Quiz** and **Passing Score (%)** inside the **Study Budget & Routine** panel alongside session word count and max flashcard limits.
- **`frontend/src/components/SettingsQuizRescue.vue`**:
  - Cleaned up the **Quiz Failure Rescue** panel to focus strictly on remediation: **Classic vs. Fast Track**, **AI Tutor Style** selection cards, and **Local RAG** toggles.

## Modified Files
- `internal/models/models.go`
- `internal/db/schema.go`
- `internal/db/store.go`
- `internal/db/store_integration_test.go`
- `internal/study/quiz_sync.go`
- `internal/app/app_study_cards.go`
- `frontend/src/composables/useSettings.js`
- `frontend/src/components/SettingsStudyBudget.vue`
- `frontend/src/components/SettingsQuizRescue.vue`
- `doc/SCHEMA.md`
