# Solution: Inherited Flashcard Source Notebook Resolution and Navigation

**Date**: 2026-10-22  
**Status**: Implemented  
**Scope**: Backend (`internal/models`, `internal/db`, `internal/app`) & Frontend (`Flashcards.vue`)

---

## 1. Problem Statement
When studying flashcards in a study profile with linked flashcard inheritance enabled, review session cards can originate from linked/parent study profiles. Previously:
1. `GetReviewSession` joined notebooks pinning to `n.id = task.NotebookID`, which caused cards inherited from other profiles or notebooks to fail to resolve their original notebook metadata or have `NotebookID` missing entirely.
2. `ReviewSessionCard` lacked a `notebook_id` field in the payload.
3. `GetNotebookTopicTree` only loaded notebooks matching the current profile ID (`n.profile_id = ?`), meaning inherited profiles' notebooks were absent from the tree.
4. In `Flashcards.vue`, navigation handlers (`goToTopicNotes` and `goToCardPageInReader`) defaulted to `selectedNotebookID` (the task's notebook), redirecting inherited cards to the wrong notebook context or failing navigation.

---

## 2. Changes Implemented

### Backend Changes

1. **Model Update (`internal/models/models.go`)**:
   - Added `NotebookID string \`json:"notebook_id,omitempty"\`` to `ReviewSessionCard`.

2. **Review Session Repository Query (`internal/db/review_session_repo.go`)**:
   - Removed the `n.id = ?` pin in the notebook `LEFT JOIN`.
   - The query now resolves the notebook from the card's own topic (via `n.topic_id = t.id` or `notebook_topics`), allowing inherited cards to resolve their true source notebook.
   - Added `COALESCE(n.id, '')` to the `SELECT` column list and scanned it into `&card.NotebookID`.

3. **Notebook Topic Tree Allowed Profiles (`internal/db/notebooks_repo.go`)**:
   - Updated signature from `GetNotebookTopicTree(profileID string)` to `GetNotebookTopicTree(profileIDs []string)`.
   - Uses SQL `IN (...)` to query across all allowed profiles (active + linked inherited profiles), ensuring notebooks linked via inheritance are present.

4. **App Layer Delegation (`internal/app/notebook_endpoints.go`)**:
   - In `GetNotebookTopicTree`, resolved profile IDs using `repo.GetAllowedFlashcardProfileIDs(profileID)` and passed the slice to `repo.GetNotebookTopicTree`.

5. **Test Updates (`internal/db/store_integration_test.go`)**:
   - Updated existing call sites: `GetNotebookTopicTree("")` &rarr; `GetNotebookTopicTree(nil)`.

### Frontend Changes

1. **Flashcard Navigation Target Resolution (`frontend/src/pages/Flashcards.vue`)**:
   - Introduced `currentCardNotebookID` computed property: prefers `card.notebook_id` &rarr; `currentCardTopic.value.notebook_id` &rarr; `selectedNotebookID`.
   - Updated `goToTopicNotes()` and `goToCardPageInReader()` to use `currentCardNotebookID` instead of `selectedNotebookID`.

---

## 3. Verification
- All backend packages compiled cleanly (`go test -run=^$ ./internal/...`).
- Existing integration tests updated for the new function signature.
