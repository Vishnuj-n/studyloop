# Solution: Explicit Extraction Engine Tracking & Ingestion Lifecycle Isolation

## Problem
1. **Ambiguous Extraction State & UI Confusion**: The `notebooks` table stored `page_count` and `chunk_count`, but never persisted which extraction engine produced them (`pdfcpu` standard text vs `PyMuPDF` deep structured markdown).
2. **Missing Active Lane Controls**: In `NotebookCard.vue`, `canUpgradeDeep` evaluated to true for all active PDF cards for Pro users, which preempted the `Sleep` button with a `⚡ Deep Extract` button. This created the false impression that deep-extracted books needed re-extraction.
3. **Queue Risk During Background Re-Extraction**: Triggering `UpgradeNotebookToDeepPDF` kept `study_status = 'active'`, allowing active queue scheduling against unstable/rebuilding chapter boundaries while processing.

---

## Architecture & Implementation

### 1. SQLite Schema & Migration (`schema.go`, `SCHEMA.md`)
- Added `extraction_engine TEXT DEFAULT 'standard'` to `notebooks` table.
- Added automatic startup column check & migration in `alterStatements` to ensure backwards compatibility across existing user databases.
- Updated `doc/SCHEMA.md`.

### 2. Backend Models & Extraction Flow (`models.go`, `notebooks_repo.go`, `notebook_endpoints.go`)
- Added `ExtractionEngine` to `models.Notebook`.
- Updated `GetNotebooks` and `GetNotebookByID` queries to read `COALESCE(extraction_engine, 'standard')`.
- Added `UpdateNotebookExtractionEngine(notebookID, engine string)`.
- Updated `runDeepPDFExtraction` to write `extraction_engine = 'deep_structured'` upon successful completion.
- Updated `UpgradeNotebookToDeepPDF` to immediately set `study_status = 'dormant'` upon initiation, releasing the active lane slot and preventing queue tasks while rebuilding chunks.

### 3. Frontend Card Presentation (`NotebookCard.vue`)
- Added `isDeepExtracted` computed check (`props.notebook.extraction_engine === 'deep_structured'`).
- Displayed `⚡ Deep Extracted` status badge in the card metadata.
- Updated `canUpgradeDeep` to only show `⚡ Upgrade Deep` for PDF books that have not yet been deep-extracted.
- Restored permanent visibility of the `Sleep` action on active cards and `Activate` action on dormant cards.

---

## Verification
- `go test -short ./internal/...` passed with 0 errors across all internal packages.
- `npm test` in frontend passed (12 test suites, 42 tests).
- Verified `extraction_engine` column structure and persistence in SQLite `dev_data/Studyloop.db`.
