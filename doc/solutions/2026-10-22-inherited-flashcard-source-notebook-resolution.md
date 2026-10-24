# Solution: Inherited Flashcard Source Notebook Resolution and Profile Isolation

**Date**: 2026-10-22 (Updated 2026-10-24)  
**Status**: Implemented  
**Scope**: Backend (`internal/models`, `internal/db`, `internal/app`) & Frontend (`Flashcards.vue`, `Notes.vue`)

---

## 1. Problem Statement
When studying flashcards in a study profile with linked flashcard inheritance enabled, review session cards can originate from linked/parent study profiles.
1. `GetReviewSession` previously joined notebooks pinning to `n.id = task.NotebookID`, which caused cards inherited from other profiles or notebooks to fail to resolve their original notebook metadata or have `NotebookID` missing entirely.
2. `ReviewSessionCard` lacked a `notebook_id` field in the payload.
3. In `Flashcards.vue`, navigation handlers (`goToTopicNotes` and `goToCardPageInReader`) defaulted to `selectedNotebookID` (the task's notebook), redirecting inherited cards to the wrong notebook context or failing navigation.
4. **Profile Isolation Regression**: An initial attempt widened `GetNotebookTopicTree` to fetch all `allowedProfileIDs` (active + inherited). While this allowed inherited notebooks to be resolved, it caused all notebooks from linked profiles to appear in the active profile's **Study Notes** sidebar tree (`Notes.vue`), breaking profile isolation.

---

## 2. Changes Implemented

### Backend Changes

1. **Model Update (`internal/models/models.go`)**:
   - Added `NotebookID string \`json:"notebook_id,omitempty"\`` to `ReviewSessionCard`.

2. **Review Session Repository Query (`internal/db/review_session_repo.go`)**:
   - Removed the `n.id = ?` pin in the notebook `LEFT JOIN`.
   - The query now resolves the notebook from the card's own topic (via `n.topic_id = t.id` or `notebook_topics`), allowing inherited cards to resolve their true source notebook.
   - Added `COALESCE(n.id, '')` to the `SELECT` column list and scanned it into `&card.NotebookID`.

3. **Repository Multi-Profile Support (`internal/db/notebooks_repo.go`)**:
   - Signature: `GetNotebookTopicTree(profileIDs []string)` supporting `IN (...)` for flexible scoping.
   - Tested in `internal/db/store_integration_test.go`.

4. **App Layer Scoping Restored (`internal/app/notebook_endpoints.go`)**:
   - `GetNotebookTopicTree()` strictly queries only the **active profile ID** (and unassigned notebooks).
   - Keeps the Study Notes explorer sidebar clean and strictly isolated to the current profile.

### Frontend Changes

1. **Flashcard Navigation Target Resolution (`frontend/src/pages/Flashcards.vue`)**:
   - Introduced `currentCardNotebookID` computed property: prefers `card.notebook_id` &rarr; `currentCardTopic.value.notebook_id` &rarr; `selectedNotebookID`.
   - Updated `goToTopicNotes()` and `goToCardPageInReader()` to pass `currentCardNotebookID` via query parameters (`?notebookId=...&topicId=...`).
   - When viewing notes or jumping to the reader from an inherited card, the target note loads directly via query parameters without needing the source notebook to be in the active profile's sidebar tree.

---

## 3. Verification
- Backend packages compile cleanly (`go test -run=^$ ./internal/...`).
- Store integration tests passing (`GetNotebookTopicTree(nil)`).
- Flashcards unit tests passing (`npm --prefix frontend run test -- src/pages/Flashcards.spec.js --run`).
